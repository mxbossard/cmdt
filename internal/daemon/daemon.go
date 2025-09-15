package daemon

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"runtime/debug"
	"slices"
	"strconv"
	"time"

	"github.com/gofrs/flock"

	"cmdt/internal/asyncdisplay"
	"cmdt/internal/facade"
	"cmdt/internal/fork"
	"cmdt/internal/model"
	"cmdt/internal/repo"
	"cmdt/internal/service"

	"github.com/mxbossard/utilz/collectionz"
	"github.com/mxbossard/utilz/printz"
	_ "github.com/mxbossard/utilz/zcreen"
	"github.com/mxbossard/utilz/zlog"
)

const (
	DaemonLockFilename         = "daemon.lock"
	DaemonPidFilename          = "daemon.pid"
	LockWatingDuration         = 5 * time.Second
	MaxNoOpToUnqueueDuration   = 10 * time.Second // Max time to wait for an ooperation to unqueue before sopping the daemon.
	AsyncPollingSleep          = 1 * time.Millisecond
	WaitAsyncReportTestTimeout = 2 * time.Second
	daemonTryLockPeriod        = 200 * time.Microsecond
	daemonWatcherPeriod        = 10 * time.Millisecond
	maxWatcherSuccessiveErrors = 5
	maxDaemonRestart           = 3
	maxWatcherInactivityPeriod = (maxWatcherSuccessiveErrors * 2) * daemonWatcherPeriod
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
	openedSuites     []string
}

func (d *daemon) run() {
	logger.Warn("DAEMON: starting ...", "token", d.token, "isolation", d.isolation)
	// fmt.Printf("\n<<>> Daemon running ...\n")
	startTime := time.Now()
	debugTime := time.Now()
	lastUnqueue := time.Now()

	startHeartBeat(d.repo, "starting daemon run")
	defer stopHeartBeat("finished daemon run")

	outs := printz.NewDiscardingOutputs() // Daemon shoud not write on stdouts by default
	d.display = asyncdisplay.New(d.repo.BackingFilepath(), true, outs)
	service.Dpl = d.display

	for {
		if time.Since(debugTime) > time.Second {
			debugTime = time.Now()
			logger.Trace("DAEMON: running", "token", d.token, "for", time.Since(startTime))
		}

		if _, done := d.unqueueAndProcess(); done {
			lastUnqueue = time.Now()
		} else {
			n, err := fork.WorkersCount()
			if err != nil {
				logger.Error("Error", "err", err)
			}
			if n == 0 {
				// nothing to unqueue wait some period
				duration := time.Since(lastUnqueue)
				if duration > MaxNoOpToUnqueueDuration {
					logger.Debug("DAEMON: nothing to unqueue", "duration", duration, "token", d.token)
					fmt.Printf("\n<<>> No Op to unqueue for %s\n", time.Since(lastUnqueue))
					break
				}
			}
			time.Sleep(AsyncPollingSleep)
			continue
		}
	}
	logger.Warn("DAEMON: stopping ...", "token", d.token, "after", time.Since(startTime))
	// fmt.Printf("\n<<>> Daemon stopped\n")
}

// Process Operation and trap panic to continue processing
func (d *daemon) unqueueAndProcess() (op model.Operater, done bool) {
	defer func() {
		err := recover()
		if err != nil {
			logger.Error("DAEMON ERROR: trapped a panic", "error", err)
			fmt.Printf("\n/!\\ DAEMON ERROR [%d]: trapped a panic /!\\\n%v\n", os.Getpid(), err)
			fmt.Printf("\nstack :%s\n", string(debug.Stack()))
			d.display.Errors(fmt.Errorf("%s", err))
			err2 := d.repo.NotDone(op)
			if err2 != nil {
				panic(fmt.Errorf("cannot put back on queue failed operation: %w", err2))
			}
			time.Sleep(time.Second)
		}
	}()

	var err error
	if op, err = d.repo.UnqueueOperation(); err != nil {
		panic(err)
	} else if op != nil {
		// fmt.Printf("\n<<>> processing op: %d (%s %d) [%s] ... \n", op.Id(), op.Kind(), op.Seq(), d.token)
		_, err := d.process(op)
		if err != nil {
			panic(err)
		}
		done = true
	}
	return
}

func (d daemon) unqueue() (op model.Operater, err error) {
	op, err = d.repo.UnqueueOperation()
	if err != nil {
		return
	}
	return
}

func (d *daemon) process(op model.Operater) (ok bool, err error) {
	if op == nil {
		return
	}

	fmt.Printf("\n<<>> Processing op: %s ...\n", op)

	onDone := func() {
		//logger.Warn("doning op ...", "op", op)
		err = d.repo.Done(op)
		if err != nil {
			err = fmt.Errorf("unable to done op: [%s] : %w", op, err)
			logger.Error(err.Error())
			panic(err)
		} else {
			logger.Info("op done", "ok", ok, "op", op)
			// fmt.Printf("\n<<>> op done: %d (%s %d) [%s] ... \n", op.Id(), op.Kind(), op.Seq(), d.token)
		}
		fmt.Printf("\n<<>> Op: %s DONE.\n", op)
	}

	onDoneSavingSuiteCfg := func() {
		onDone()
	}

	suite := op.Suite()
	logger.Info("DAEMON: unqueued operation.", "kind", op.Kind(), "id", op.Id(), "suite", op.Suite(), "seq", op.Seq())

	// Automagicaly open suite on first operation
	var saveCfgOnDone bool
	if suite != model.GlobalConfigTestSuiteName && !slices.Contains(d.openedSuites, suite) {
		// Do not open *special* global suite
		d.openedSuites = append(d.openedSuites, suite)
		logger.Debug("Initializing test suite", "token", d.token, "isolation", d.isolation, "openedSuite", suite)
		fmt.Printf("\n<<>> [%d] opening suite: %s ; openedSuites: %s ; op: %s\n", os.Getpid(), suite, d.openedSuites, op)
		ctx := facade.NewSuiteContext(d.token, d.isolation, suite, false, model.InitAction, model.Config{}, true)
		defer ctx.Close()
		err = fork.ClearQueue(suite)
		if err != nil {
			return false, err
		}
		d.display.OpenSuite(ctx)
		if ctx.Config.SuiteTitled.Is(false) {
			// Display suite title once and record it was done.
			d.display.SuiteTitle(ctx)
			ctx.Config.SuiteTitled.Set(true)
			// The title will be flushed with first test display.
			// The SuiteTitled state must be recorded with outcome.
			// FIXME: for now it is recorded after outcome in onDoneSavingSuiteCfg().
			onDoneSavingSuiteCfg = func() {
				// Replace onDone() func to save the config on test done.
				onDone()
				err := d.repo.SaveSuiteConfig(ctx.Config)
				if err != nil {
					logger.Error(err.Error())
					panic(err)
				}
			}
			// fork.QueueTestDef(def, o, onDoneSavingSuiteCfg)
			saveCfgOnDone = true
		}
	} else {
		logger.Debug("Test suite already opened", "token", d.token, "isolation", d.isolation, "openedSuite", suite)
	}

	switch o := op.(type) {
	case *model.TestOp:
		// FIXME: must override bad token & isolation inside ReportDefinition !
		def := o.Definition
		def.Token = d.token
		def.Isolation = d.isolation

		// var forked bool
		// // Automagicaly open suite on first test
		// if !slices.Contains(d.openedSuites, suite) {
		// 	d.openedSuites = append(d.openedSuites, suite)
		// 	logger.Debug("Initializing test suite", "token", d.token, "isolation", d.isolation, "openedSuite", suite)
		// 	fmt.Printf("\n<<>> [%d] opening suite: %s ; openedSuites: %s\n", os.Getpid(), suite, d.openedSuites)
		// 	ctx := facade.NewSuiteContext(d.token, d.isolation, suite, false, model.InitAction, model.Config{}, true)
		// 	defer ctx.Close()
		// 	fork.ClearQueue(suite)
		// 	d.display.OpenSuite(ctx)
		// 	if ctx.Config.SuiteTitled.Is(false) {
		// 		// Display suite title once and record it was done.
		// 		d.display.SuiteTitle(ctx)
		// 		ctx.Config.SuiteTitled.Set(true)
		// 		// The title will be flushed with first test display.
		// 		// The SuiteTitled state must be recorded with outcome.
		// 		// FIXME: for now it is recorded after outcome in onDoneSavingSuiteCfg().

		// 		fork.QueueTestDef(def, o, onDoneSavingSuiteCfg)
		// 		forked = true
		// 	}
		// } else {
		// 	logger.Debug("Test suite already opened", "token", d.token, "isolation", d.isolation, "openedSuite", suite)
		// }

		// exitCode := service.ProcessTestDef(def)
		// o.SetExitCode(uint16(exitCode))
		// fmt.Printf("\n<<>> queued test: #%d", def.Seq)
		if saveCfgOnDone {
			fork.QueueTestDef(def, o, onDoneSavingSuiteCfg)
		} else {
			fork.QueueTestDef(def, o, onDone)
		}

	case *model.ReportOp:
		// FIXME: must override bad token & isolation inside ReportDefinition !
		def := o.Definition
		def.Token = d.token
		def.Isolation = d.isolation

		fork.WaitQueueComplete(def.TestSuite)

		err = d.repo.WaitSuiteOperationsDoneBefore(op, def.Config.SuiteTimeout.Get())
		if err != nil {
			return false, err
		}

		exitCode, err2 := d.report(o)
		op.SetExitCode(uint16(exitCode))
		op.SetErr(err2)
		onDone()
	case *model.GlobalReportOp:
		// FIXME: must override bad token & isolation inside ReportDefinition !
		def := o.Definition
		def.Token = d.token
		def.Isolation = d.isolation

		fork.WaitAllQueuesComplete()

		// FIXME: which timeout for global report ?
		err = d.repo.WaitAllOperationsDoneBefore(op, def.Config.SuiteTimeout.GetOr(10*time.Second))
		if err != nil {
			return false, err
		}
		// time.Sleep(1 * time.Second)

		exitCode, err2 := d.globalReport(o)
		op.SetExitCode(uint16(exitCode))
		op.SetErr(err2)
		onDone()
	default:
		err = fmt.Errorf("unknown operation %T", op)
		return
	}
	ok = true
	return
}

func (d *daemon) performTest0(testDef model.TestDefinition) (exitCode int16) {
	perf := logger.PerfTimer()
	defer perf.End()
	exitCode = service.ProcessTestDef(testDef, true)
	return
}

func (d *daemon) report(op *model.ReportOp) (exitCode int16, err error) {
	perf := logger.PerfTimer()
	defer perf.End()

	def := op.Definition
	cfg, err := d.repo.GetSuiteConfig(def.TestSuite, true)
	if err != nil {
		return 1, err
	}

	// // Check if suite was started or wait some time
	// start := time.Now()
	// cfg, err := d.repo.GetSuiteConfig(def.TestSuite, true)
	// if err != nil {
	// 	return 1, err
	// }
	// for cfg.SuiteStartTime.IsEmpty() {
	// 	// Wait until suite is started
	// 	if time.Since(start) > WaitAsyncReportTestTimeout {
	// 		return 1, fmt.Errorf("you must perform some test prior to report")
	// 	}
	// 	time.Sleep(time.Millisecond)
	// 	cfg, err = d.repo.GetSuiteConfig(def.TestSuite, true)
	// 	if err != nil {
	// 		return 1, err
	// 	}
	// }

	// Wait for some test to report until suite timeout
	// FIXME: Daemon never report @all ?
	// FIXME: must wait all previous operations in suite are done !
	// Add a WaitAllOperationsDone(seq) error
	// Add a WaitSuiteOperationsDone(suite, seq) error
	// On first suite test Op done update suite start time

	// testCount := d.repo.ToReportTestCountBySuiteAndMode(def.TestSuite, true, false)
	// for testCount == 0 {
	// 	if time.Since(start) > cfg.SuiteTimeout.Get() {
	// 		return 1, fmt.Errorf("timeouted suite report waiting for test")
	// 	}
	// 	time.Sleep(time.Millisecond)
	// 	testCount = d.repo.TestCount(def.TestSuite)
	// }

	// FIXME: must use right timeout
	err = d.repo.WaitSuiteOperationsDoneBefore(&op.OperationBase, cfg.SuiteTimeout.GetOr(model.DefaultSuiteTimeout))
	if err != nil {
		return 1, err
	}

	// Attempt to perform report on cli side only
	// exitCode, err = service.ProcessReportDef(def)
	ctx := facade.NewSuiteContext(def.Token, def.Isolation, def.TestSuite, false, model.ReportAction, def.Config, false) // FIXME ? removing def.Config ?
	defer ctx.Close()
	d.display.CloseSuite(ctx, "following suite report")

	logger.Debug("Closing test suite", "token", def.Token, "isolation", def.Isolation, "openedSuite", def.TestSuite)
	d.openedSuites = collectionz.Delete(d.openedSuites, def.TestSuite)
	fmt.Printf("\n<<>> [%d] deleted opened suite: %s ; openedSuites: %s\n", os.Getpid(), def.TestSuite, d.openedSuites)
	return
}

func (d *daemon) globalReport(op *model.GlobalReportOp) (exitCode int16, err error) {
	perf := logger.PerfTimer()
	defer perf.End()

	def := op.Definition
	cfg, err := d.repo.GetGlobalConfig()
	if err != nil {
		return 1, err
	}

	// start := time.Now()
	// Wait for some test to report until suite timeout.
	// FIXME: Daemon never report @all ?
	// testCount := d.repo.ToReportTestCountByMode(true, false)

	// // FIXME: should not need to wait for test count > 0
	// for testCount == 0 {
	// 	if time.Since(start) > WaitAsyncReportTestTimeout {
	// 		// FIXME: why not return an error ?
	// 		return 1, fmt.Errorf("timeouted global report waiting for test")
	// 	}
	// 	time.Sleep(time.Millisecond)
	// 	testCount = d.repo.NotReportedTestCount()
	// }

	// FIXME: must use right timeout
	err = d.repo.WaitAllOperationsDoneBefore(&op.OperationBase, cfg.SuiteTimeout.GetOr(model.DefaultSuiteTimeout))
	if err != nil {
		return 1, err
	}

	// Attempt to perform report on cli side only
	// exitCode, err = service.ProcessGlobalReportDef(def, true)
	// if err != nil {
	// 	return 1, err
	// }
	ctx := facade.NewGlobalContext(def.Token, def.Isolation, model.Config{}, false)
	defer ctx.Close()
	reportableSuites, err := ctx.Repo.ListReportableSuites()
	if err != nil {
		return 1, err
	}
	for _, suite := range reportableSuites {
		sctx := facade.NewSuiteContext(def.Token, def.Isolation, suite, false, model.ReportAction, def.Config, false) // FIXME ? removing def.Config ?
		d.display.CloseSuite(sctx, "following global report")
		sctx.Close()
	}

	logger.Debug("Closing all test suites", "token", def.Token, "isolation", def.Isolation)
	d.openedSuites = []string{}
	fmt.Printf("\n<<>> [%d] deleted all opened suite\n", os.Getpid())
	//d.display.Clear()
	return
}

/*
func (d daemon) ReadPid() string {
	pidFilepath := filepath.Join(d.repo.BackingFilepath(), DaemonPidFilename)
	//fmt.Printf("reading PID file: %s ...", pidFilepath)
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
*/

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
	token := os.Args[2]
	isolation := os.Args[3]
	debugLevel, err := strconv.Atoi(os.Args[4])
	if err != nil {
		panic("bad debug level")
	}

	zlog.ColoredConfig(slog.Int("pid", os.Getpid()))
	zlog.SetPart("daemon")
	var loggingQualifier string
	if isolation != "" {
		loggingQualifier = "-" + isolation
	}
	loggingFilepath := fmt.Sprintf(model.DefaultDebugDaemonLogFilepath, loggingQualifier)
	zlog.SetDefaultAppendingFileOutput(loggingFilepath)
	zlog.SetLogLevelThreshold(slog.Level(debugLevel))

	logger.Debug("daemon prechecks", "token", token, "isolation", isolation, "debugLevel", debugLevel, "args", os.Args[1:])

	repo := repo.New(token, isolation)
	defer repo.Close()

	d := daemon{token: token, isolation: isolation, repo: &repo}
	lockFilepath := filepath.Join(repo.BackingFilepath(), DaemonLockFilename)
	fileLock := flock.New(lockFilepath)

	// Wait to acquire file lock
	lockCtx, cancel := context.WithTimeout(context.Background(), LockWatingDuration)
	defer cancel()
	locked, err := fileLock.TryLockContext(lockCtx, daemonTryLockPeriod)
	if err != nil {
		panic(err)
	}
	if !locked {
		fmt.Printf("Cannot locked file to start daemon properly !")
		os.Exit(2)
	}

	// If PID file already exists exit => already running
	pid, err := repo.GetDaemonPid()
	if err != nil {
		panic(err)
	}
	if pid > 0 {
		logger.Info("daemon already running")
		fileLock.Unlock()
		os.Exit(3)
	}

	err = repo.SaveDaemonPid(os.Getpid())
	if err != nil {
		panic(err)
	}

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
	//fmt.Printf("\n<<>> Running new daemon ; pid: %d ; isol: %s ; token: %s\n", os.Getpid(), isolation, token)
	d.run()

	// Lock prior last unqueue
	lockCtx, cancel = context.WithTimeout(context.Background(), LockWatingDuration)
	defer cancel()
	locked, err = fileLock.TryLockContext(lockCtx, daemonTryLockPeriod)
	if err != nil && err != context.DeadlineExceeded {
		panic(err)
	}
	if !locked {
		fmt.Printf("Cannot locked file to stop daemon properly !")
		os.Exit(2)
	}

	// Last unqueue
	_, err = d.unqueue()
	if err != nil {
		panic(err)
	}

	// Clear Daemon PID in DB
	err = repo.ClearDaemonPid(pid)
	if err != nil {
		panic(err)
	}

	// Release file lock
	fileLock.Unlock()

	//fmt.Printf("\n<<>> Stopped daemon ; isol: %s ; token: %s\n", isolation, token)

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
