package background

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/mxbossard/utilz/zlog"
)

func RequireDaemonRunning(token, isol string) {
	go watchDaemonActivity(token, isol)
	// FIXME: delegate the daemon launch to the demon activity watcher. Which should launch a daemon only if necessary.
	err := LanchProcessIfNeeded(token, isol)
	if err != nil {
		panic(err)
	}
}

func LanchProcessIfNeeded(token, isolation string) error {
	logger.Debug("daemon: should I launch daemon ?", "token", token, "isolation", isolation)
	if token == "" {
		// No token => no daemon to launch
		return nil
	}
	// FIXME: add retries ?
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}

	/*
		stdout := os.NewFile(uintptr(syscall.Stdout), "/dev/stdout")
		stderr := os.NewFile(uintptr(syscall.Stderr), "/dev/stderr")
	*/

	ppid := os.Getppid()
	stdout, err := os.OpenFile(fmt.Sprintf("/proc/%d/fd/1", ppid), os.O_WRONLY, 0644)
	if err != nil {
		panic(err)
	}
	stderr, err := os.OpenFile(fmt.Sprintf("/proc/%d/fd/2", ppid), os.O_WRONLY, 0644)
	if err != nil {
		panic(err)
	}

	debugLevel := int(zlog.GetLogLevelThreshold())

	cmd := exec.Command(os.Args[0], "@_daemon", token, isolation, fmt.Sprintf("%d", debugLevel))
	cmd.Dir = cwd
	cmd.Env = os.Environ()
	//cmd.Stdout = os.Stdout
	//cmd.Stderr = os.Stderr
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	// FIXME: daemon should produce outputs in buffers and post it witin done op if waiting.
	err = cmd.Start()
	if err != nil {
		return err
	}
	err = cmd.Process.Release()
	if err != nil {
		return err
	}

	/*
		argv := []string{os.Args[0], "@_daemon", token}
		//procattr := os.ProcAttr{Dir: cwd, Env: os.Environ(), Files: []*os.File{nil, os.Stdout, os.Stderr}}
		procattr := os.ProcAttr{Dir: cwd, Env: os.Environ(), Files: []*os.File{nil, nil, nil}}
		proc, err := os.StartProcess(os.Args[0], argv, &procattr)
		if err != nil {
			return err
		}
		err = proc.Release()
	*/

	logger.Info("daemon process released", "cmd", cmd)
	return err
}
