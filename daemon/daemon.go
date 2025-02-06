package daemon

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"time"

	"github.com/gofrs/flock"

	"mby.fr/cmdtest/asyncdisplay"
	"mby.fr/cmdtest/facade"
	"mby.fr/cmdtest/model"
	"mby.fr/cmdtest/repo"
	"mby.fr/cmdtest/service"
	"mby.fr/utils/filez"
	"mby.fr/utils/printz"
	_ "mby.fr/utils/screen"
	"mby.fr/utils/zlog"
)

const (
	DaemonLockFilename         = "daemon.lock"
	DaemonPidFilename          = "daemon.pid"
	LockWatingSecs             = 5
	ExtraRunningSecs           = 5
	AsyncPollingSleepInMs      = 50
	WaitAsyncReportTestTimeout = 2 * time.Second
)

var logger = zlog.New() //slog.New(slog.NewTextHandler(os.Stderr, model.DefaultLoggerOpts))

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
	token, isolation string
	repo             repo.Repo
	display          *asyncdisplay.AsyncDisplay
	openedSuite      string
}

func (d *daemon) run() {
	logger.Warn("DAEMON: starting ...", "token", d.token, "isolation", d.isolation)
	startTime := time.Now()
	debugTime := time.Now()
	lastUnqueue := time.Now()

	outs := printz.NewDiscardingOutputs() // Daemon shoud not write on stdouts by default
	d.display = asyncdisplay.New(d.repo.BackingFilepath(), true, outs)
	service.Dpl = d.display

	for {
		if time.Since(debugTime) > time.Second {
			debugTime = time.Now()
			logger.Trace("DAEMON: running", "token", d.token, "for", time.Since(startTime))
		}

		if op, err := d.unqueue(); err != nil {
			panic(err)
		} else if op != nil {
			_, err := d.process(op)
			if err != nil {
				return
			}
		} else {
			// nothing to unqueue wait some period
			duration := time.Since(lastUnqueue)
			if duration > ExtraRunningSecs*time.Second {
				logger.Debug("DAEMON: nothing to unqueue", "duration", duration, "token", d.token)
				// More than ExtraRunningSecs since last unqueue
				break
			}
			time.Sleep(AsyncPollingSleepInMs * time.Millisecond)
			continue
		}

		lastUnqueue = time.Now()

	}
	logger.Warn("DAEMON: stopping ...", "token", d.token, "after", time.Since(startTime))
}

func (d daemon) unqueue() (op model.Operater, err error) {
	op, err = d.repo.UnqueueOperation()
	if err != nil {
		return
	}
	return
}

func (d *daemon) process(op model.Operater) (ok bool, err error) {
	defer func() {
		if op != nil {
			//logger.Warn("doning op ...", "op", op)
			err = d.repo.Done(op)
			if err != nil {
				err = fmt.Errorf("unable to done op: [%s] : %w", op, err)
				panic(err)
			}
		}
	}()

	if op != nil {
		logger.Debug("DAEMON: unqueued operation.", "kind", op.Kind(), "id", op.Id(), "suite", op.Suite(), "seq", op.Seq())
		if d.openedSuite == "" {
			d.openedSuite = op.Suite()
			logger.Debug("Initializing test suite", "token", d.token, "isolation", d.isolation, "openedSuite", d.openedSuite)
			ctx := facade.NewSuiteContext(d.token, d.isolation, d.openedSuite, false, model.InitAction, model.Config{})
			d.display.OpenSuite(ctx)
			d.display.SuiteTitle(ctx)
		}
		switch o := op.(type) {
		case *model.TestOp:
			ec := d.performTest(o.Definition)
			op.SetExitCode(uint16(ec))
		case *model.ReportOp:
			exitCode, err := d.report(o.Definition)
			if err != nil {
				return false, err
			}
			op.SetExitCode(uint16(exitCode))
		case *model.ReportAllOp:
			op.SetExitCode(uint16(d.reportAll(o.Definition)))
		default:
			err = fmt.Errorf("unknown operation %T", op)
			return
		}
		ok = true
	}
	return
}

func (d *daemon) performTest(testDef model.TestDefinition) (exitCode int16) {
	perf := logger.PerfTimer()
	defer perf.End()
	exitCode = service.ProcessTestDef(testDef)
	return
}

func (d *daemon) report(def model.ReportDefinition) (exitCode int16, err error) {
	perf := logger.PerfTimer()
	defer perf.End()

	// Check if suite was started or wait some time
	start := time.Now()
	cfg, err := d.repo.GetSuiteConfig(def.TestSuite, true)
	if err != nil {
		return 1, err
	}
	for cfg.SuiteStartTime.IsEmpty() {
		// Wait until suite is started
		if time.Since(start) > WaitAsyncReportTestTimeout {
			return 1, fmt.Errorf("no test to report")
		}
		time.Sleep(time.Millisecond)
		cfg, err = d.repo.GetSuiteConfig(def.TestSuite, true)
		if err != nil {
			return 1, err
		}
	}

	// Wait for some test until suite timeout
	testCount := d.repo.TestCount(def.TestSuite)
	for testCount == 0 {
		if time.Since(start) > cfg.SuiteTimeout.Get() {
			return 1, fmt.Errorf("reached suite timeout")
		}
		time.Sleep(time.Millisecond)
		testCount = d.repo.TestCount(def.TestSuite)
	}

	//d.display.DisplayRecorded(def.TestSuite, def.Config.Timeout.Get())
	go func() {
		err := d.display.TailBlocking(def.TestSuite, def.Config.Timeout.Get())
		if err != nil {
			panic(err)
		}
	}()
	exitCode, err = service.ProcessReportDef(def)
	logger.Debug("Closing test suite", "token", def.Token, "isolation", def.Isolation, "openedSuite", d.openedSuite)
	d.openedSuite = ""
	ctx := facade.NewSuiteContext(d.token, d.isolation, def.TestSuite, false, model.InitAction, model.Config{})
	d.display.CloseSuite(ctx)
	return
}

func (d *daemon) reportAll(def model.ReportDefinition) (exitCode int16) {
	perf := logger.PerfTimer()
	defer perf.End()
	//d.display.DisplayAllRecorded(def.Config.Timeout.Get())
	go func() {
		err := d.display.TailAllBlocking(def.Config.Timeout.Get())
		if err != nil {
			panic(err)
		}
	}()
	exitCode = service.ProcessReportAllDef(def)
	logger.Debug("Closing test suite", "token", def.Token, "isolation", def.Isolation, "openedSuite", d.openedSuite)
	d.openedSuite = ""
	d.display.Clear()
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
	defaultLogLevel := slog.LevelDebug
	model.LoggerLevel.Set(defaultLogLevel)

	//logger.Warn("daemon: should I take over ?", "args", os.Args)
	if len(os.Args) > 1 && os.Args[1] == "@_daemon" {
		if len(os.Args) != 5 {
			panic("bad usage of @_daemon")
		}
	} else {
		return
	}

	zlog.ColoredConfig(slog.Int("pid", os.Getpid()))
	zlog.SetPart("daemon")
	zlog.SetDefaultAppendingFileOutput(model.DefaultDebugDaemonLogFilepath)
	token := os.Args[2]
	isolation := os.Args[3]
	debugLevel, err := strconv.Atoi(os.Args[4])
	if err != nil {
		panic("bad debug level")
	}
	zlog.SetLogLevelThreshold(slog.Level(debugLevel))

	logger.Debug("daemon prechecks", "token", token, "isolation", isolation, "debugLevel", debugLevel, "args", os.Args[1:])

	repo := repo.New(token, isolation)
	//defer repo.Close()

	d := daemon{token: token, isolation: isolation, repo: &repo}
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
	pid := d.ReadPid()
	if pid != "" {
		logger.Info("daemon already running")
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

	/*
		stdout, err := os.OpenFile("/dev/stdout", os.O_WRONLY+os.O_CREATE+os.O_APPEND, 0644)
		if err != nil {
			panic(err)
		}
		stderr, err := os.OpenFile("/dev/stderr", os.O_WRONLY+os.O_CREATE+os.O_APPEND, 0644)
		if err != nil {
			panic(err)
		}
	*/

	/*
		stdout := os.NewFile(uintptr(syscall.Stdout), "/dev/stdout")
		stderr := os.NewFile(uintptr(syscall.Stderr), "/dev/stderr")

		os.Stdout = stdout
		os.Stderr = stderr
		defer stdout.Close()
		defer stderr.Close()
		logger = slog.New(slog.NewTextHandler(os.Stderr, model.DefaultLoggerOpts))
	*/

	logger.Info("daemon taking over")

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
	_, err = d.unqueue()
	if err != nil {
		panic(err)
	}

	// Clear PID file
	d.ClearPid()

	// Release file lock
	fileLock.Unlock()

	os.Exit(0)
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
