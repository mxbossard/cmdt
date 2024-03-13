package mock

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"mby.fr/cmdtest/model"
	"mby.fr/cmdtest/utils"
)

func MockWrapperPath(mockDir string) (path string) {
	// get test dir
	path = filepath.Join(mockDir, "mockWrapper.sh")
	return
}

func ProcessMocking(mockDir string, rootMocks, mocks []model.CmdMock) (err error) {
	wrapperFilepath := MockWrapperPath(mockDir)

	// write the mock wrapper
	allMocks := rootMocks
	allMocks = append(allMocks, mocks...)
	err = writeMockWrapperScript(wrapperFilepath, allMocks)
	if err != nil {
		return
	}
	// for each cmd mocked add link to the mock wrapper
	for _, mock := range mocks {
		var ok bool
		ok, err = utils.IsShellBuiltin(mock.Cmd)
		if err != nil {
			return
		}
		if ok {
			err = fmt.Errorf("command %s is not mockable (shell builtin)", mock.Cmd)
			return
		}

		linkName := filepath.Join(mockDir, mock.Cmd)
		err = os.RemoveAll(linkName)
		if err != nil {
			return
		}
		linkSource := wrapperFilepath
		err = os.Symlink(linkSource, linkName)
		if err != nil {
			return
		}
	}

	log.Printf("mock wrapper: %s\n", wrapperFilepath)
	return
}

func writeMockWrapperScript(wrapperFilepath string, mocks []model.CmdMock) (err error) {
	// By default run all args
	//wrapper.sh CMD ARG_1 ARG_2 ... ARG_N
	// Pour chaque CmdMock
	// if "$@" match CmdMock
	wrapperScript := "#! /bin/sh\nset -e\n"
	wrapperScript += `if [ -n "$ORIGINAL_PATH" ]; then` + "\n"
	wrapperScript += "\t" + `export PATH="$ORIGINAL_PATH"` + "\n"
	wrapperScript += `fi` + "\n"
	//wrapperScript += ">&2 echo PATH:$PATH\n"
	wrapperScript += `cmd=$( basename "$0" )` + "\n"

	for _, mock := range mocks {
		wrapperScript += "if "
		wildcard := false
		if mock.Op == "=" {
			// args must exactly match mock config
			for pos, arg := range mock.Args {
				if arg != "*" {
					wrapperScript += fmt.Sprintf(`[ "$%d" = "%s" ] && `, pos+1, arg)
				} else {
					wildcard = true
					break
				}
			}
		} else if mock.Op == ":" {
			// args must contains mock config disorderd
			// all mock args must be in $@
			// if multiple same mock args must all be present in $@
			mockArgsCount := make(map[string]int, 8)
			for _, arg := range mock.Args {
				if arg != "*" {
					mockArgsCount[arg]++
				} else {
					wildcard = true
				}
			}
			for arg, count := range mockArgsCount {
				wrapperScript += fmt.Sprintf(`[ %d -eq $( echo "$@" | grep -c "%s" ) ] && `, count, arg)
			}
		}
		if !wildcard {
			wrapperScript += fmt.Sprintf(`[ "$#" -eq %d ] && `, len(mock.Args))
		}
		wrapperScript += fmt.Sprintf(`[ "$cmd" = "%s" ] || [ "$0" = "%s" ]`, mock.Cmd, mock.Cmd)

		wrapperScript += `; then` + "\n"
		if mock.Stdin != nil {
			wrapperScript += fmt.Sprintf("\t" + `stdin="$( cat )"` + "\n")
			wrapperScript += fmt.Sprintf("\t"+`expected="%s"`+"\n", *mock.Stdin)
			if mock.StdinOp == "=" {
				wrapperScript += fmt.Sprintf("\t" + `if [ "$stdin" = "$( echo $expected )" ]; then` + "\n")
			} else if mock.StdinOp == ":" {
				wrapperScript += fmt.Sprintf("\t" + `if echo "$stdin" | grep "$expected" > /dev/null; then` + "\n")
			} else {
				err = fmt.Errorf("not supported stdin op: %s", mock.StdinOp)
				panic(err)
			}
		}

		// Add at least one command
		wrapperScript += "true\n"

		if mock.Stdout != "" {
			wrapperScript += fmt.Sprintf("\t"+`echo -n "%s"`+"\n", mock.Stdout)
		}
		if mock.Stderr != "" {
			wrapperScript += fmt.Sprintf("\t"+` >&2 echo -n "%s"`+"\n", mock.Stderr)
		}
		if len(mock.OnCallCmdAndArgs) > 0 {
			wrapperScript += fmt.Sprintf("\t"+`%s`+"\n", strings.Join(mock.OnCallCmdAndArgs, " "))
		}
		if !mock.Delegate {
			wrapperScript += fmt.Sprintf("\t"+`exit %d`+"\n", mock.ExitCode)
		}
		if mock.Stdin != nil {
			wrapperScript += fmt.Sprintf("\t" + `fi` + "\n")
		}
		wrapperScript += fmt.Sprintf(`fi` + "\n")
	}
	wrapperScript += `if [ -e "/mocked$0" ]; then` + "\n"
	//wrapperScript += "\t" + `>&2 echo calling "${0}.mocked" "$@" ...` + "\n"
	//wrapperScript += "\t" + ` ls -l "${0}.mocked"` + "\n"
	//wrapperScript += "\t" + `alias foo=${0}.mocked` + "\n"
	wrapperScript += "\t" + `"/mocked$0" "$@"` + "\n"
	wrapperScript += `else` + "\n"
	//wrapperScript += "\t" + `>&2 echo calling "$cmd" "$@" ...` + "\n"
	wrapperScript += "\t" + `"$cmd" "$@"` + "\n"
	wrapperScript += `fi` + "\n"

	err = os.Remove(wrapperScript)
	if err != nil && !os.IsNotExist(err) {
		return
	}
	err = os.WriteFile(wrapperFilepath, []byte(wrapperScript), 0755)
	return
}
