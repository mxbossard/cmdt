package service

import (
	"cmdt/internal/asyncdisplay"
	"cmdt/internal/background"
	"cmdt/internal/facade"
	"cmdt/internal/model"
	"cmdt/internal/utils"
	"fmt"
	"os"
	"time"

	"github.com/mxbossard/utilz/errorz"
	"github.com/mxbossard/utilz/printz"
)

const defaultGlobalTimeout = 5 * time.Minute

func globalAction(token, isolation string, inputConfig model.Config, parseArgsErrors errorz.Aggregated) (exitCode int16, err error) {
	exitCode = 1
	logger.Debug("Executing Global action")
	if parseArgsErrors.GotError() {
		errorz.Fatal(parseArgsErrors)
	}
	globalCtx := facade.NewGlobalContext(token, isolation, inputConfig, false)
	defer globalCtx.Close()

	ProcessGlobalError(globalCtx, parseArgsErrors.Return())
	logger.Trace("Forged context", "ctx", globalCtx)
	logger.Info("Processing global action", "token", token)
	Dpl.Quiet(globalCtx.Config.Quiet.Is(true))
	exitCode, err = GlobalConfig(globalCtx)

	return
}

func initAction(token, isolation string, inputConfig model.Config, parseArgsErrors errorz.Aggregated) (exitCode int16, err error) {
	exitCode = 1
	testSuite := inputConfig.TestSuite.Get()
	logger.Debug("Executing Init action", "suite", testSuite)

	// Check if suite exists and it's status
	rep := facade.CachedRepo(token, isolation)
	defer rep.PoolClose()

	exists, reported, err := rep.SuiteStatus(testSuite)

	// If @async is missing but @fork is present => Add @async
	if !inputConfig.Async.IsSet() {
		if inputConfig.ForkCount.IsSet() {
			if inputConfig.ForkCount.Is(0) {
				inputConfig.Async.Set(false)
			} else {
				// Forked suite MUST be async
				inputConfig.Async.Set(true)
			}
		}
	}

	suiteCtx := facade.NewSuiteContext(token, isolation, testSuite, false, model.InitAction, inputConfig, false)
	ProcessSuiteError(suiteCtx, err)
	defer suiteCtx.Close()

	// Store ignore at suite level
	suiteCtx.Config.IgnoreSuite = suiteCtx.Config.Ignore
	if suiteCtx.Config.IgnoreSuite.GetOr(false) {
		// Ignored suite should not be async
		suiteCtx.Config.Async.Set(false)
	}

	logger.Debug("repo suite status", "testSuite", testSuite, "exists", exists, "reported", reported)

	if exists {
		n := rep.TestCount(testSuite)
		if n > 0 && !reported {
			err = fmt.Errorf("cannot erase test suite: [%s] (isol: %s) which contains %d test(s) not reported yet", testSuite, isolation, n)
			ProcessSuiteError(suiteCtx, err)
		}
	}

	ProcessSuiteError(suiteCtx, parseArgsErrors.Return())
	logger.Trace("Forged context", "ctx", suiteCtx)
	logger.Info("Processing init action", "token", token, "suite", testSuite, "async", suiteCtx.Config.Async.Get())
	Dpl.Quiet(suiteCtx.Config.Quiet.Is(true))

	exitCode, err = cliInitTestSuite(suiteCtx)
	ProcessSuiteError(suiteCtx, err)

	return
}

func reportAllAction(token, isolation string, inputConfig model.Config, parseArgsErrors errorz.Aggregated) (exitCode int16, wait func() int16, err error) {
	// Report all sync suites then all async suites
	exitCode = 1
	wait = func() int16 { return exitCode }

	logger.Debug("Executing Report all action")
	// Reporting All test suite
	if parseArgsErrors.GotError() {
		errorz.Fatal(parseArgsErrors)
	}
	globalCtx := facade.NewGlobalContext(token, isolation, inputConfig, false)
	defer globalCtx.Close()
	globalCfg := globalCtx.Config
	rep := facade.CachedRepo(token, isolation)
	defer rep.PoolClose()
	reportAll := globalCfg.ReportAll.GetOr(model.DefaultReportAll)

	wait = func() int16 {
		err = globalCtx.Repo.MarkSuitesReported()
		ProcessGlobalError(globalCtx, err)

		return exitCode
	}

	// Process report all without daemon
	logger.Trace("Forged context", "ctx", globalCtx)
	// logger.Info("executing report all in sync (not queueing report)")
	Dpl.Quiet(globalCfg.Quiet.Is(true))

	var asyncExitCode int16

	ignoredSuiteCount := rep.IgnoredSuiteCount(globalCfg.ReportAll.Get())
	emptySuiteCount := rep.EmptySuiteCount(globalCfg.ReportAll.Get())
	//syncSuites, err := rep.ListSyncSuites()
	syncSuites, err := rep.ListReportableSuitesByMode(false, reportAll)
	ProcessGlobalError(globalCtx, err)
	//asyncSuites, err := rep.ListAsyncSuites()
	asyncSuites, err := rep.ListReportableSuitesByMode(true, reportAll)
	ProcessGlobalError(globalCtx, err)

	// 1- Global Report sync suites
	toReportSyncTestCount := rep.ToReportTestCountByMode(false, reportAll)
	if reportAll || ignoredSuiteCount+emptySuiteCount+toReportSyncTestCount > 0 {
		exitCode, err = globalReport(globalCtx, false)
		ProcessGlobalError(globalCtx, err)
		for _, suite := range syncSuites {
			err = cliAfterSuiteReport(globalCtx.Token, globalCtx.Isolation, suite, Dpl)
			ProcessGlobalError(globalCtx, err)
		}
	} else {
		exitCode = 0
	}

	// 2- Global Report All already reported async suites
	reportedAsyncSuites, err := rep.ListReportedAsyncSuites()
	ProcessGlobalError(globalCtx, err)
	if reportAll && len(reportedAsyncSuites) > 0 {
		// Report All async suites already reported

		exitCode, err = globalReport(globalCtx, true)
		ProcessGlobalError(globalCtx, err)
		for _, suite := range reportedAsyncSuites {
			err = cliAfterSuiteReport(globalCtx.Token, globalCtx.Isolation, suite, Dpl)
			ProcessGlobalError(globalCtx, err)
		}
	}

	// 3- Global report async suites which are not reported yet
	toReportAsyncTestCount := rep.ToReportTestCountByMode(true, reportAll)
	if toReportAsyncTestCount > 0 {
		if len(asyncSuites) > 0 {
			// Delegate report all processing to daemon
			//logger.Info("executing report all on async display")
			logger.Info("executing report all (queueing report)")

			def := model.ReportDefinition{
				Token:     token,
				Isolation: isolation,
				Config:    globalCtx.Config,
			}
			op := model.GlobalReportOperation(true, def) // FIXME should not block if test can be run simultaneously
			err = globalCtx.Repo.QueueOperation(&op)
			if err != nil {
				errorz.Fatal(err)
			}

			// Report always launch a daemon, just in case daemon was not running.
			background.RequireDaemonRunning(globalCtx.Isolation, globalCtx.Token)

			asyncDpl := asyncdisplay.New(globalCtx.Repo.BackingFilepath(), false, printz.NewStandardOutputs())

			// always wait
			// Replace wait func for async processing
			// if globalCtx.Config.Wait.Is(true) {
			wait = func() int16 {
				// FIXME: bad timeout
				var opErr error

				asyncExitCode, opErr, err = globalCtx.Repo.WaitOperationDone(&op, globalCtx.Config.SuiteTimeout.GetOr(defaultGlobalTimeout))
				if err != nil {
					//panic(err)
					Dpl.Errors(err)
				} else if opErr != nil {
					Dpl.Errors(fmt.Errorf("daemon error: %w", opErr))
				}
				logger.Info("op done", "opId", op.Id(), "opKind", op.Kind(), "suite", op.TestSuite, "asyncExitCode", asyncExitCode)

				for _, suite := range asyncSuites {
					err = cliAfterSuiteReport(globalCtx.Token, globalCtx.Isolation, suite, asyncDpl)
					ProcessGlobalError(globalCtx, err)
				}

				err = globalCtx.Repo.MarkSuitesReported()
				ProcessGlobalError(globalCtx, err)

				// FIXME: why clear all suites on report all ?
				// Clear all reported suite async display
				// suites, err := globalCtx.Repo.ListReportedAsyncSuites()
				// ProcessGlobalError(globalCtx, err)
				// for _, suite := range suites {
				// 	asyncdisplay.ClearSuite(globalCtx.Repo.BackingFilepath(), suite)
				// }
				return max(exitCode, asyncExitCode)
			}

			go func() {
				fmt.Fprintf(os.Stderr, "\n<<>> tailing suites: %s ... \n", asyncSuites)
				err = asyncDpl.TailSuppliedBlocking(asyncSuites, globalCtx.Config.SuiteTimeout.GetOr(model.DefaultSuiteTimeout))
				ProcessGlobalError(globalCtx, err)
				fmt.Fprintf(os.Stderr, "\n<<>> tailing suites: %s finished. \n", asyncSuites)
				logger.Info("finished async TailAllBlocking", "opId", op.Id())
			}()
		}
	}

	if len(syncSuites)+len(asyncSuites) == 0 || !reportAll && ignoredSuiteCount+toReportSyncTestCount+toReportAsyncTestCount == 0 {
		exitCode = 1
		err := fmt.Errorf("you must perform some test prior to report globaly")
		ProcessGlobalError(globalCtx, err)
	}

	// Display report all footer before wait is called and then before suites are marked reported for accurate timings
	Dpl.ReportAllFooter(globalCtx)

	return
}

func syncReportAllAction(token, isolation string, inputConfig model.Config, parseArgsErrors errorz.Aggregated) (exitCode int16, wait func() int16, err error) {
	// Report all sync suites then all async suites
	exitCode = 1
	wait = func() int16 { return exitCode }

	logger.Debug("Executing Report all action")
	// Reporting All test suite
	if parseArgsErrors.GotError() {
		errorz.Fatal(parseArgsErrors)
	}
	globalCtx := facade.NewGlobalContext(token, isolation, inputConfig, false)
	defer globalCtx.Close()
	globalCfg := globalCtx.Config
	rep := facade.CachedRepo(token, isolation)
	defer rep.PoolClose()
	reportAll := globalCfg.ReportAll.GetOr(model.DefaultReportAll)

	defer func() {
		err = globalCtx.Repo.MarkSuitesReported()
		ProcessGlobalError(globalCtx, err)
	}()

	// Process report all without daemon
	logger.Trace("Forged context", "ctx", globalCtx)
	// logger.Info("executing report all in sync (not queueing report)")
	Dpl.Quiet(globalCfg.Quiet.Is(true))

	var suites []string
	if reportAll {
		suites, err = rep.ListAllSuites()
	} else {
		suites, err = rep.ListReportableSuites()
	}
	ProcessGlobalError(globalCtx, err)

	//if len(syncSuites)+len(asyncSuites) == 0 { // !reportAll && ignoredSuiteCount+toReportSyncTestCount+toReportAsyncTestCount == 0
	if len(suites) == 0 {
		exitCode = 1
		err := fmt.Errorf("you must perform some test prior to report globaly")
		ProcessGlobalError(globalCtx, err)
	}

	suitesChan, err := WatchAllTestsPerformed(rep, reportAll, defaultGlobalTimeout)
	ProcessGlobalError(globalCtx, err)

	reportPassed := true
	var testsPassed uint32
	for suite := range suitesChan {
		if suite.Err != nil {
			ProcessGlobalError(globalCtx, err)
		}
		testSuite := suite.Val

		suiteCtx := facade.NewSuiteContext(token, isolation, testSuite, false, model.ReportAction, model.Config{}, false)
		// if suiteCtx.Config.TestSuite.IsEmpty() {
		// 	fmt.Printf("\n<<>> empty suite name in ctx !!! \nctx: %v ; \ncfg: %v\n", suiteCtx, suiteCtx.Config)
		// }

		def := model.ReportDefinition{Token: token, Isolation: isolation, TestSuite: testSuite, Config: suiteCtx.Config}
		ctx := facade.NewSuiteContext(def.Token, def.Isolation, def.TestSuite, false, model.ReportAction, def.Config, false) // FIXME ? removing def.Config ?

		if ctx.Config.Async.Is(true) {
			// Manage async display
			asyncDpl := asyncdisplay.NewWaitingTailer(globalCtx.Repo.BackingFilepath(), true, printz.NewStandardOutputs())
			asyncDpl.CloseSuite(ctx, "all tests performed")
			err = asyncDpl.TailBlocking(testSuite, globalCtx.Config.SuiteTimeout.GetOr(1*time.Second)) // model.DefaultSuiteTimeout
			ProcessGlobalError(globalCtx, err)
		}

		// Report Suite
		suiteOutcome, _, err := reportTestSuite(ctx, true, reportAll)
		ProcessGlobalError(globalCtx, err)

		if suiteOutcome.Duration < 0 {
			// FIXME: report should save an endTime for suite duration to be saved
			suiteOutcome.Duration = time.Since(ctx.Config.SuiteStartTime.Get())
		}

		Dpl.ReportSuite(suiteOutcome)
		Dpl.CloseSuite(ctx, "following report")

		err = cliAfterSuiteReport(token, isolation, testSuite, Dpl)
		ProcessGlobalError(globalCtx, err)
		reportPassed = reportPassed && (suiteOutcome.Outcome == model.PASSED || suiteOutcome.Outcome == model.IGNORED || suiteOutcome.Outcome == model.EMPTY)
		testsPassed += suiteOutcome.PassedCount

		ctx.Close()
	}

	// err = globalCtx.Repo.MarkSuitesReported()
	// ProcessGlobalError(globalCtx, err)

	if reportPassed {
		exitCode = 0
	} else {
		exitCode = 1
	}

	// Display report all footer before wait is called and then before suites are marked reported for accurate timings
	Dpl.ReportAllFooter(globalCtx)

	if testsPassed == 0 && !reportAll {
		// When not reporting @all, reporting no tests passed should fail.
		exitCode = 1
		err := fmt.Errorf("you must perform some test prior to report globaly")
		ProcessGlobalError(globalCtx, err)
	}

	return
}

func reportSuiteAction(token, isolation string, inputConfig model.Config, parseArgsErrors errorz.Aggregated) (exitCode int16, wait func() int16, err error) {
	// Reporting One test suite
	exitCode = 1
	wait = func() int16 { return exitCode }

	testSuite := inputConfig.TestSuite.Get()
	logger.Debug("Executing Report suite action", "suite", testSuite)
	suiteCtx := facade.NewSuiteContext(token, isolation, testSuite, false, model.ReportAction, inputConfig, false)
	ProcessSuiteError(suiteCtx, parseArgsErrors.Return())
	defer suiteCtx.Close()

	def := model.ReportDefinition{
		Token:     token,
		Isolation: isolation,
		TestSuite: testSuite,
		Config:    suiteCtx.Config,
	}
	op := model.ReportOperation(testSuite, true, def) // FIXME should not block if test can be run simultaneously

	// Process report without daemon
	logger.Trace("Forged context", "ctx", suiteCtx)
	Dpl.Quiet(suiteCtx.Config.Quiet.Is(true))

	if suiteCtx.Config.Async.Is(true) && !suiteCtx.Config.Reported.Is(true) {
		//logger.Info("executing report on async display", "suite", testSuite)

		start := time.Now()
		for suiteCtx.Repo.TestCount(testSuite) == 0 {
			if time.Since(start) > model.WaitAsyncReportTestTimeout {
				err := fmt.Errorf("you must perform some test prior to report")
				ProcessSuiteError(suiteCtx, err)
			}
			time.Sleep(time.Millisecond)
		}

		// Delegate report processing to daemon
		logger.Info("executing report async (queueing report)", "suite", testSuite)
		def := model.ReportDefinition{
			Token:     token,
			Isolation: isolation,
			TestSuite: testSuite,
			Config:    suiteCtx.Config,
		}
		op := model.ReportOperation(testSuite, true, def) // FIXME should not block if test can be run simultaneously
		err = suiteCtx.Repo.QueueOperation(&op)
		ProcessSuiteError(suiteCtx, err)

		// Report always launch a daemon, just in case daemon was not running.
		background.RequireDaemonRunning(suiteCtx.Isolation, suiteCtx.Token)

		asyncDpl := asyncdisplay.New(suiteCtx.Repo.BackingFilepath(), false, printz.NewStandardOutputs())

		// always wait
		// if suiteCtx.Config.Wait.Is(true) {
		wait = func() int16 {
			// FIXME: bad timeout
			pt := logger.QualifiedPerfTimer("waiting report done ...", "suite", testSuite)
			defer pt.End()
			var opErr error
			exitCode, opErr, err = suiteCtx.Repo.WaitOperationDone(&op, suiteCtx.Config.SuiteTimeout.Get())
			if err != nil {
				//panic(err)
				Dpl.Errors(err)
			} else if opErr != nil {
				Dpl.Errors(fmt.Errorf("daemon error: %w", opErr))
			}

			err = cliAfterSuiteReport(suiteCtx.Token, suiteCtx.Isolation, testSuite, asyncDpl)
			ProcessSuiteError(suiteCtx, err)

			return exitCode
		}
		// } else {
		// 	exitCode = 0
		// }

		wait = func() int16 {
			err = asyncDpl.TailBlocking(testSuite, suiteCtx.Config.SuiteTimeout.Get())
			ProcessSuiteError(suiteCtx, err)
			logger.Info("finished async TailBlocking")

			err := suiteCtx.Repo.WaitSuiteOperationsDoneBefore(&op, suiteCtx.Config.SuiteTimeout.GetOr(defaultGlobalTimeout))
			if err != nil {
				//panic(err)
				Dpl.Errors(err)
			}
			fmt.Fprintf(os.Stderr, "\n<<>> all op done.\n")
			logger.Info("all op done")

			exitCode, err = ProcessReportDef(def)
			ProcessSuiteError(suiteCtx, err)
			err = cliAfterSuiteReport(suiteCtx.Token, suiteCtx.Isolation, testSuite, Dpl)
			ProcessSuiteError(suiteCtx, err)

			return exitCode
		}

		/*
			go func() {
				err = asyncDpl.TailBlocking(testSuite, suiteCtx.Config.SuiteTimeout.Get())
				ProcessSuiteError(suiteCtx, err)
				logger.Info("finished async TailBlocking")
			}()
		*/

	} else {
		logger.Info("executing report in sync", "suite", testSuite)
		exitCode, err = ProcessReportDef(def)
		ProcessSuiteError(suiteCtx, err)
		suiteCtx.Repo.Done(&op)
		err = cliAfterSuiteReport(suiteCtx.Token, suiteCtx.Isolation, testSuite, Dpl)
		ProcessSuiteError(suiteCtx, err)
	}

	return
}

func testAction(token, isolation string, inputConfig model.Config, parseArgsErrors errorz.Aggregated, signifientArgs []string) (exitCode int16, wait func() int16, err error) {
	exitCode = 1
	wait = func() int16 { return exitCode }
	testSuite := inputConfig.TestSuite.Get()

	ppid := uint32(utils.ReadEnvPpid())
	logger.Debug("Executing Test action", "suite", testSuite)
	testCtx, err := facade.NewTestContext(token, isolation, testSuite, 0, inputConfig, ppid, false)
	if err != nil {
		Dpl.Errors(err)
	}
	defer testCtx.Close()

	if testCtx.Config.IgnoreSuite.GetOr(false) {
		exitCode = 0
		err = nil
		return
	}

	//ProcessTestError(testCtx, err)

	testCtx.IncrementTestCount()
	seq := testCtx.Seq

	testCfg := testCtx.Config
	token = testCtx.Token
	logger.Trace("Forged context", "ctx", testCtx)

	testDef := model.TestDefinition{
		TestSignature: model.TestSignature{
			TestSuite:  testSuite,
			Seq:        seq,
			TestName:   testCfg.TestName.GetOr(""),
			CmdAndArgs: testCfg.CmdAndArgs,
		},
		Ppid:      ppid,
		Token:     token,
		Isolation: isolation,

		Config: testCfg,
		//SuitePrefix: testCtx.Suite.Config.Prefix.Get(),
		CmdArgs: signifientArgs,
	}

	if parseArgsErrors.GotError() {
		exitCode = ProcessMalDefinedTest(testDef, testCtx, parseArgsErrors.Return())
		err = nil
		return
	}

	logger.Debug("Test definition", "token", token, "isolation", isolation, "suite", testSuite, "seq", seq)
	if !testCfg.Async.Is(true) {
		// Process test without daemon
		// enforce wait
		logger.Info("executing test in sync (not queueing test)", "suite", testSuite, "seq", seq)
		exitCode = ProcessTestDef(testDef, false)
	} else {
		// Init suite config in repo if necessary
		//facade.NewSuiteContext(token, isolation, testSuite, true, action, model.Config{Async: utilz.OptionalOf(true)})

		// Delegate test processing to daemon
		logger.Info("executing test async (queueing test)", "suite", testSuite, "seq", seq)
		testOp := model.TestOperation(testSuite, seq, false, testDef) // FIXME should not block if test can be run simultaneously
		err = testCtx.Repo.QueueOperation(&testOp)
		ProcessTestError(testCtx, err)

		if testCfg.Wait.Is(true) {
			wait = func() int16 {
				exitCode, opErr, err := testCtx.Repo.WaitOperationDone(&testOp, testCfg.SuiteTimeout.Get())
				if err != nil {
					//panic(err)
					Dpl.Errors(err)
				} else if opErr != nil {
					Dpl.Errors(fmt.Errorf("daemon error: %w", opErr))
				}
				return exitCode
			}
		} else {
			// Don't wait return exit code 0
			exitCode = 0
		}

		daemonPid, err := testCtx.Repo.GetDaemonPid()
		ProcessTestError(testCtx, err)
		if daemonPid == 0 {
			// For better perf : test launch a daemon only if no PID in DB.
			background.RequireDaemonRunning(token, isolation)
		}
	}
	return
}
