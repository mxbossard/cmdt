package service

import (
	"fmt"
	"os"
	"strings"

	"mby.fr/cmdtest/facade"
	"mby.fr/cmdtest/mock"
	"mby.fr/cmdtest/model"
	"mby.fr/cmdtest/utils"
	"mby.fr/utils/cmdz"
	"mby.fr/utils/container"
)

const ()

func StartContainer(testCtx facade.TestContext) (id string, err error) {
	image := testCtx.Config.ContainerImage.Get()
	mocks := testCtx.Config.Mocks
	// FIXME: implements Mocking in caontainer
	_ = mocks

	// Start container with :
	// - cmdtest
	// - configured mock
	id, err = utils.ForgeUuid()
	if err != nil {
		return
	}
	repoDir := testCtx.Repo.BackingFilepath()

	cmdtestVol := os.Args[0] + ":/opt/cmdtest:ro"
	ctxDirVol := repoDir + ":" + repoDir + ":rw"
	dr := container.DockerRunner{
		Name:       id,
		Image:      image,
		Entrypoint: "/bin/sh",
		CmdArgs:    []string{"-c", "sleep 120"}, //FIXME use suite timeout or test timeout
		Remove:     true,
		Detach:     true,
		Volumes:    []string{cmdtestVol, ctxDirVol},
	}
	//stdout := &inout.RecordingWriter{}
	//stderr := &inout.RecordingWriter{}
	stdout := os.Stdout
	stderr := os.Stderr
	err = dr.Wait(stdout, stderr)

	/*
		args = []string{"docker", "run", "--rm", "-d", "--entrypoint=/bin/sh", "--name="+id, image, "-c" "sleep inf"}
		c := cmdz.Cmd("docker", "run", "--rm", id)
		var exitCode int
		exitCode, err = c.BlockRun()
		if err != nil {
			return
		}
	*/

	return
}

func MockInContainer(testCtx facade.TestContext, id string) (err error) {
	var script string
	var mockDir string
	mockDir, err = testCtx.MockDirectoryPath(testCtx.Repo.TestCount(testCtx.Config.TestSuite.Get()) + 1)
	if err != nil {
		return
	}
	mockWrapperPath := mock.MockWrapperPath(mockDir)
	for _, mock := range testCtx.Config.RootMocks {
		var ok bool
		ok, err = IsShellBuiltin(testCtx, id, mock.Cmd)
		if err != nil {
			return
		}
		if ok {
			err = fmt.Errorf("command %s is not mockable (shell builtin)", mock.Cmd)
			return
		}
		script += fmt.Sprintf(`mkdir -p $( dirname /mocked%[1]s ) ; mv "%[1]s" "/mocked/%[1]s" ; ln -s "%[2]s" "%[1]s"`, mock.Cmd, mockWrapperPath)
	}
	if script != "" {
		c := cmdz.Cmd("docker", "exec", "-u", "0", id, "sh", "-e", "-c", script).ErrorOnFailure(true)
		_, err = c.BlockRun()
	}
	return
}

func UnmockInContainer(testCtx facade.TestContext, id string) (err error) {
	var script string
	for _, mock := range testCtx.Config.RootMocks {
		if strings.HasPrefix(mock.Cmd, string(os.PathSeparator)) {
			script += fmt.Sprintf(`test -f "/mocked%[1]s" && rm -f -- "%[1]s" || true ; mv "/mocked%[1]s" "%[1]s" || true`, mock.Cmd)
		}
	}
	if script != "" {
		c := cmdz.Cmd("docker", "exec", "-u", "0", id, "sh", "-e", "-c", script).ErrorOnFailure(true)
		_, err = c.BlockRun()
	}
	return
}

func RemoveContainer(id string) (err error) {
	c := cmdz.Cmd("docker", "rm", "-f", id)
	var exitCode int
	exitCode, err = c.BlockRun()
	if err != nil {
		return
	}
	if exitCode != 0 {
		err = fmt.Errorf("error stopping container: RC=%d", exitCode)
	}
	return
}

func ExecInContainer(testCtx facade.TestContext, id string, cmdAndArgs []string) (exec cmdz.Executer, err error) {
	var envArgs []string
	envArgs = append(envArgs, "-e", model.ContextTokenEnvVarName+"="+testCtx.Token)
	envArgs = append(envArgs, "-e", model.EnvContainerScopeKey+"="+fmt.Sprintf("%d", testCtx.Config.ContainerScope.Get()))
	envArgs = append(envArgs, "-e", model.EnvContainerImageKey+"="+testCtx.Config.ContainerImage.Get())
	envArgs = append(envArgs, "-e", model.EnvContainerIdKey+"="+id)

	user := fmt.Sprintf("%d", os.Getuid())
	//workDir := "/tmp"
	args := []string{"docker", "exec", "-u", user}
	args = append(args, envArgs...)
	args = append(args, id)
	args = append(args, cmdAndArgs...)
	exec = cmdz.Cmd(args...)
	//c.AddEnviron(os.Environ()...)
	exec.SetOutputs(os.Stdout, os.Stderr)
	//log.Printf("Will execute: [%s]\n", c)
	_, err = exec.BlockRun()
	return
}

func IsShellBuiltin(testCtx facade.TestContext, id, cmd string) (ok bool, err error) {
	cmdAndArgs := []string{"sh", "-c", "type " + cmd}
	exec, err := ExecInContainer(testCtx, id, cmdAndArgs)
	if err != nil {
		err = fmt.Errorf("cannot evaluate if command %s is a shell builtin: %w", cmd, err)
		return
	} else if exec.ExitCode() > 0 {
		if strings.Contains(exec.StdoutRecord(), "not found") {
			err = fmt.Errorf("command %s not found in path: %s", cmd, exec.StdoutRecord())
			return
		} else {
			err = fmt.Errorf("cannot evaluate if command %s is a shell builtin: %s", cmd, exec.StdoutRecord())
			return
		}
	}
	ok = strings.Contains(exec.StdoutRecord(), "shell builtin")
	return
}

func PerformTestInEphemeralContainer(testCtx facade.TestContext) (exitCode int, err error) {
	// Launch test in new container
	var ctId string
	ctId, err = StartContainer(testCtx)
	if err != nil {
		return
	}

	cmdAndArgs := []string{"/opt/cmdtest"}
	if len(os.Args) > 1 {
		for _, arg := range os.Args[1:] {
			if !strings.HasPrefix(arg, "@container") {
				cmdAndArgs = append(cmdAndArgs, arg)
			}
		}

	}

	var exec cmdz.Executer
	exec, err = ExecInContainer(testCtx, ctId, cmdAndArgs)
	if err != nil {
		return
	}

	exitCode = exec.ExitCode()
	if exitCode == 0 {
		err = RemoveContainer(ctId)
	}
	return
}

func PerformTestInContainer(testCtx facade.TestContext) (ctId string, exitCode int, err error) {
	if testCtx.Config.ContainerId.IsPresent() {
		ctId = testCtx.Config.ContainerId.Get()
	}

	// If container dirty before test
	if ctId != "" && testCtx.Config.ContainerDirties.Is(model.DirtyBeforeTest) || testCtx.Config.ContainerDirties.Is(model.DirtyBeforeRun) {
		err = RemoveContainer(ctId)
		if err != nil {
			return
		}
		ctId = ""
	}

	// If container not already exists, create a new one
	if ctId == "" {
		ctId, err = StartContainer(testCtx)
		if err != nil {
			return
		}
	}

	err = MockInContainer(testCtx, ctId)
	if err != nil {
		return
	}

	defer func() {
		//err = UnmockInContainer(testCtx, ctId)
		if err != nil {
			return
		}

		if testCtx.Config.ContainerDirties.Is(model.DirtyAfterTest) || testCtx.Config.ContainerDirties.Is(model.DirtyAfterRun) {
			err = RemoveContainer(ctId)
			if err != nil {
				return
			}
			ctId = ""
		}
	}()

	//log.Printf("ctId: %s ; dirty: %s\n", ctId, testCtx.ContainerDirties)
	//log.Printf("PerformTestInContainer #%s (%v)\n", ctId, *testCtx.ContainerScope)
	// FIXME: Test if container is up and running

	// Launch test in container
	cmdAndArgs := []string{"/opt/cmdtest", "@container=false"}
	if len(os.Args) > 1 {
		for _, arg := range os.Args[1:] {
			// FIXME: how to not pass root mocks params ?
			if !strings.HasPrefix(arg, "@container") { //&&
				//!strings.HasPrefix(arg, "@mock="+string(os.PathSeparator)) {
				cmdAndArgs = append(cmdAndArgs, arg)
			}
		}

	}

	var exec cmdz.Executer
	exec, err = ExecInContainer(testCtx, ctId, cmdAndArgs)
	if err != nil {
		return
	}
	exitCode = exec.ExitCode()

	return
}
