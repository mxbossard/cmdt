package daemon

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/gofrs/flock"

	"mby.fr/cmdtest/model"
	"mby.fr/cmdtest/repo"
	"mby.fr/cmdtest/service"
	"mby.fr/utils/filez"
)

const (
	DaemonLockFilename = "daemon.lock"
	DaemonPidFilename  = "daemon.pid"
	LockWatingSecs     = 5
	ExtraRunningSecs   = 10
)

var logger = slog.New(slog.NewTextHandler(os.Stderr, model.DefaultLoggerOpts))

/*
Keys:
- Only one daemon running at once
- Do not miss queued tests

Ideas:
- Init process
  - acquire lockfile (daemon is starting) or wait
  - if PID file is present => return
  - Write PID file
  - Start daemon process
  - Release lock file
  - Start daemon

- Running daemon
  - Loop Unqueuing tests
  - If no test for x seconds => acquire lock file (daemon is stopping)
  - Last unqueue (May take a lot of time testing)
  - Stop the daemon

- Stop daemon
  - Rm PID file
  - Release lock file
  - Exit 0

*/

type daemon struct {
	token string
	repo  repo.FileRepo
}

func (d daemon) unqueue() (ok bool) {
	op, err := d.repo.UnqueueOperation()
	if err != nil {
		panic(err)
	}
	defer func() {
		err = d.repo.Done(op)
		if err != nil {
			panic(err)
		}
	}()
	if op != nil {
		logger.Debug("daemon: unqueued operation.", "kind", op.Kind(), "id", op.Id())
		switch o := op.(type) {
		case *repo.TestOp:
			op.SetExitCode(uint16(d.performTest(o.Definition)))
		case *repo.ReportOp:
			op.SetExitCode(uint16(d.report(o.Definition)))
		case *repo.ReportAllOp:
			op.SetExitCode(uint16(d.reportAll(o.Definition)))
		default:
			err = fmt.Errorf("unknown operation %T", op)
			panic(err)
		}
		ok = true
	}
	return
}

func (d daemon) run() {
	logger.Info("daemon: starting ...", "token", d.token)
	lastUnqueue := time.Now()
	for {
		//logger.Warn("daemon: unqueueing ...")
		if ok := d.unqueue(); !ok {
			// nothing to unqueue wait 1ms
			duration := time.Since(lastUnqueue)
			if duration > ExtraRunningSecs*time.Second {
				logger.Info("daemon: nothing to unqueue", "duration", duration, "token", d.token)
				// More than ExtraRunningSecs since last unqueue
				break
			}
			time.Sleep(time.Millisecond)
			continue
		}

		lastUnqueue = time.Now()
	}
	logger.Debug("daemon: stopping ...", "token", d.token)
}

func (d daemon) performTest(testDef model.TestDefinition) (exitCode int16) {
	//logger.Warn("daemon performing test", "testDef", testDef)
	logger.Debug("daemon: processing test...")
	exitCode = service.ProcessTestDef(testDef)
	logger.Debug("daemon: test done.")
	return
}

func (d daemon) report(def model.ReportDefinition) (exitCode int16) {
	logger.Debug("daemon: reporting test suite...")
	exitCode = service.ProcessReportDef(def)
	logger.Debug("daemon: reporting done.")
	return
}

func (d daemon) reportAll(def model.ReportDefinition) (exitCode int16) {
	logger.Debug("daemon: reporting all test suites...")
	exitCode = service.ProcessReportAllDef(def)
	logger.Debug("daemon: reporting done.")
	return
}

func (d daemon) ReadPid() string {
	pidFilepath := filepath.Join(d.repo.BackingFilepath(), DaemonPidFilename)
	pid, err := filez.ReadString(pidFilepath)
	if os.IsNotExist(err) {
		return ""
	} else if err != nil {
		panic(err)
	}
	return pid
}

func (d daemon) WritePid() {
	pidFilepath := filepath.Join(d.repo.BackingFilepath(), DaemonPidFilename)
	err := filez.WriteString(pidFilepath, fmt.Sprintf("%d", os.Getpid()), 0600)
	if err != nil {
		panic(err)
	}
}

func (d daemon) ClearPid() {
	pidFilepath := filepath.Join(d.repo.BackingFilepath(), DaemonPidFilename)
	err := os.Remove(pidFilepath)
	if err != nil && !os.IsNotExist(err) {
		panic(err)
	}
}

func TakeOver() {
	//logger.Warn("daemon: should I take over ?", "args", os.Args)
	if len(os.Args) > 1 && os.Args[1] == "@_daemon" {
		//fmt.Printf("@_daemon args: %s", os.Args)
		if len(os.Args) != 3 {
			panic("bad usage of @_daemon")
		}
	} else {
		return
	}

	logger.Debug("daemon: taking over ...")

	token := os.Args[2]
	repo := repo.New(token)
	d := daemon{token: token, repo: repo}
	lockFilepath := filepath.Join(repo.BackingFilepath(), DaemonLockFilename)
	fileLock := flock.New(lockFilepath)

	// Wait to acquire file lock
	lockCtx, cancel := context.WithTimeout(context.Background(), LockWatingSecs*time.Second)
	defer cancel()
	locked, err := fileLock.TryLockContext(lockCtx, time.Millisecond)
	if err != nil {
		panic(err)
	}
	if !locked {
		os.Exit(2)
	}

	// If PID file already exists exit => already running
	if d.ReadPid() != "" {
		fileLock.Unlock()
		os.Exit(3)
	}

	// Write PID file
	d.WritePid()

	// Release file lock
	err = fileLock.Unlock()
	if err != nil {
		panic(err)
	}

	// Run daemon
	d.run()

	// Lock prior last unqueue
	locked, err = fileLock.TryLockContext(lockCtx, time.Millisecond)
	if err != nil && err != context.DeadlineExceeded {
		panic(err)
	}
	if !locked {
		os.Exit(2)
	}

	// Last unqueue
	_ = d.unqueue()

	// Clear PID file
	d.ClearPid()

	// Release file lock
	fileLock.Unlock()

	os.Exit(0)
}

func LanchProcessIfNeeded(token string) error {
	logger.Info("daemon: should I launch daemon ?", "token", token)
	if token == "" {
		// No token => no daemon to launch
		return nil
	}
	// FIXME: add retries ?
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}
	cmd := exec.Command(os.Args[0], "@_daemon", token)
	cmd.Dir = cwd
	cmd.Env = os.Environ()
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err = cmd.Start()
	if err != nil {
		return err
	}
	err = cmd.Process.Release()
	if err != nil {
		return err
	}
	logger.Debug("daemon process released")
	return nil
}
