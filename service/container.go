package service

import (
	"fmt"
	"os"
	"strings"

	"mby.fr/cmdtest/facade"
	"mby.fr/cmdtest/model"
	"mby.fr/cmdtest/utils"
	"mby.fr/utils/cmdz"
	"mby.fr/utils/container"
)

func StartContainer(testCtx facade.Context) (id string, err error) {
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

func ExecInContainer(token, id string, cmdAndArgs []string) (exitCode int, err error) {
	user := fmt.Sprintf("%d", os.Getuid())
	//workDir := "/tmp"
	args := []string{"docker", "exec", "-u", user, "-e", model.ContextTokenEnvVarName + "=" + token, id}
	args = append(args, cmdAndArgs...)
	c := cmdz.Cmd(args...)
	//c.AddEnviron(os.Environ()...)
	c.SetOutputs(os.Stdout, os.Stderr)
	//log.Printf("Will execute: [%s]\n", c)
	exitCode, err = c.BlockRun()
	return
}

func PerformTestInEphemeralContainer(testCtx facade.Context) (exitCode int, err error) {
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
	exitCode, err = ExecInContainer(testCtx.Token, ctId, cmdAndArgs)
	if err != nil {
		return
	}

	if exitCode == 0 {
		err = RemoveContainer(ctId)
	}
	return
}

func PerformTestInContainer(testCtx facade.Context) (ctId string, exitCode int, err error) {
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

	//log.Printf("ctId: %s ; dirty: %s\n", ctId, testCtx.ContainerDirties)
	//log.Printf("PerformTestInContainer #%s (%v)\n", ctId, *testCtx.ContainerScope)
	// FIXME: Test if container is up and running

	// Launch test in container
	cmdAndArgs := []string{"/opt/cmdtest", "@container=false"}
	if len(os.Args) > 1 {
		for _, arg := range os.Args[1:] {
			if !strings.HasPrefix(arg, "@container") {
				cmdAndArgs = append(cmdAndArgs, arg)
			}
		}

	}
	exitCode, err = ExecInContainer(testCtx.Token, ctId, cmdAndArgs)
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

	return
}
