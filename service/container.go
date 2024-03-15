package service

import (
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"mby.fr/cmdtest/facade"
	"mby.fr/cmdtest/mock"
	"mby.fr/cmdtest/model"
	"mby.fr/cmdtest/utils"
	"mby.fr/utils/cmdz"
	"mby.fr/utils/ctnrz"
)

const ()

func StartContainer(testCtx facade.TestContext) (id string, err error) {
	start := time.Now()
	image := testCtx.Config.ContainerImage.Get()

	id, err = utils.ForgeUuid()
	if err != nil {
		return
	}
	repoDir := testCtx.Repo.BackingFilepath()

	cmdtestVol := os.Args[0] + ":/opt/cmdtest:ro"
	ctxDirVol := repoDir + ":" + repoDir + ":rw,z"

	// FIXME: sleep test or suite timeout
	e := ctnrz.Engine().Container(id).Run(image, "-c", "sleep 300").
		Entrypoint("/bin/sh").
		Rm().
		Detach().
		User("1000:1000").
		AddVolumes(cmdtestVol, ctxDirVol).
		Executer()

	// discard stdout which should contain only container id (because of Detach option)
	e.SetOutputs(io.Discard, os.Stderr)

	logger.Debug("starting container", "id", id, "image", image)
	_, err = e.BlockRun()
	duration := time.Since(start)
	logger.Debug("started container", "duration", duration, "id", id, "image", image)
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
		logger.Debug("mocking container root cmd", "cmd", mock.Cmd)
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
		cmdAndArgs := []string{"sh", "-e", "-c", script}
		e := ctnrz.Engine().Container(id).Exec(cmdAndArgs...).User("0").Executer()
		_, err = e.BlockRun()
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
		cmdAndArgs := []string{"sh", "-e", "-c", script}
		e := ctnrz.Engine().Container(id).Exec(cmdAndArgs...).User("0").Executer()
		_, err = e.BlockRun()
	}
	return
}

func RemoveContainer(id string) (err error) {
	start := time.Now()
	e := ctnrz.Engine().Container(id).Stop().Timeout(0 * time.Second).Executer()
	e.SetOutputs(os.Stdout, os.Stderr)

	go func() {
		logger.Debug("removing container", "id", id)
		exitCode, err := e.BlockRun()
		if err != nil {
			logger.Error("error stopping container", "error", err)
		}
		if exitCode != 0 {
			logger.Error("error stopping container", "exitCode", exitCode)
		}
		duration := time.Since(start)
		logger.Debug("removed container", "duration", duration, "id", id)
	}()
	return
}

func ExecInContainer(testCtx facade.TestContext, id string, cmdAndArgs []string) (exec cmdz.Executer, err error) {
	start := time.Now()
	envArgs := make(map[string]string)
	envArgs[model.ContextTokenEnvVarName] = testCtx.Token
	envArgs[model.EnvContainerScopeKey] = fmt.Sprintf("%d", testCtx.Config.ContainerScope.Get())
	envArgs[model.EnvContainerImageKey] = testCtx.Config.ContainerImage.Get()
	envArgs[model.EnvContainerIdKey] = id

	user := fmt.Sprintf("%d", os.Getuid())
	exec = ctnrz.Engine().Container(id).Exec(cmdAndArgs...).AddEnvMap(envArgs).User(user).Executer()
	exec.SetOutputs(os.Stdout, os.Stderr)
	logger.Debug("executing in container", "id", id, "cmdAndArgs", cmdAndArgs)
	_, err = exec.BlockRun()
	duration := time.Since(start)
	logger.Debug("executed in container", "duration", duration, "id", id, "cmdAndArgs", cmdAndArgs)
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

	// FIXME: Test if container is up and running

	// Launch test in container
	cmdAndArgs := []string{"/opt/cmdtest", "@container=false"}
	if len(os.Args) > 1 {
		for _, arg := range os.Args[1:] {
			// FIXME: how to not pass root mocks params ?
			if !strings.HasPrefix(arg, "@container") { //&&
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
