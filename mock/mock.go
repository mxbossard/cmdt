package mock

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"mby.fr/cmdtest/model"
)

func mockDirectoryPath(testWorkDir string) (mockDir string) {
	mockDir = filepath.Join(testWorkDir, "mock")
	return
}

func ProcessMocking(testWorkDir string, mocks []model.CmdMock) (mockDir string, err error) {
	// get test dir
	// create a mock dir
	mockDir = mockDirectoryPath(testWorkDir)
	if err != nil {
		return
	}
	err = os.MkdirAll(mockDir, 0755)
	if err != nil {
		return
	}
	wrapperFilepath := filepath.Join(mockDir, "mockWrapper.sh")
	// add mockdir to PATH TODO in perform test

	// write the mocke wrapper
	err = writeMockWrapperScript(wrapperFilepath, mocks)
	if err != nil {
		return
	}
	// for each cmd mocked add link to the mock wrapper
	for _, mock := range mocks {
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
	wrapperScript += `export PATH="$ORIGINAL_PATH"` + "\n"
	//wrapperScript += ">&2 echo PATH:$PATH\n"
	wrapperScript += `cmd=$( basename "$0" )` + "\n"

	for _, mock := range mocks {
		wrapperScript += fmt.Sprintf(`if [ "$cmd" = "%s" ]`, mock.Cmd)
		wildcard := false
		if mock.Op == "=" {
			// args must exactly match mock config
			for pos, arg := range mock.Args {
				if arg != "*" {
					wrapperScript += fmt.Sprintf(` && [ "$%d" = "%s" ] `, pos+1, arg)
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
				wrapperScript += fmt.Sprintf(` && [ %d -eq $( echo "$@" | grep -c "%s" ) ]`, count, arg)
			}
		}
		if !wildcard {
			wrapperScript += fmt.Sprintf(` && [ "$#" -eq %d ] `, len(mock.Args))
		}

		wrapperScript += `; then` + "\n"
		if mock.Stdin != nil {
			wrapperScript += fmt.Sprintf("\t" + `stdin="$( cat )"` + "\n")
			wrapperScript += fmt.Sprintf("\t"+`if [ "$stdin" = "%s" ]; then`+"\n", *mock.Stdin)
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
	wrapperScript += `"$cmd" "$@"` + "\n"

	err = os.WriteFile(wrapperFilepath, []byte(wrapperScript), 0755)
	return
}
