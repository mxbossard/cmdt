package service

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"cmdt/internal/asyncdisplay"
	"cmdt/internal/display"
	"cmdt/internal/facade"
	"cmdt/internal/model"
	"cmdt/internal/utils"

	"github.com/mxbossard/utilz/cmdz"
	"github.com/mxbossard/utilz/errorz"
	"github.com/mxbossard/utilz/printz"
	"github.com/mxbossard/utilz/zlog"
)

var (
	logger = zlog.New() //slog.New(slog.NewTextHandler(os.Stderr, model.DefaultLoggerOpts))
)

var Dpl display.Displayer

func init() {
	Dpl = display.New()
}

func usage() {
	usagePrinter := printz.NewStandard()
	cmd := filepath.Base(os.Args[0])
	usagePrinter.Errf("cmdtest tool is usefull to test various scripts cli and command behaviors.\n")
	usagePrinter.Errf("You must initialize a test suite (%[1]s @init) before running tests and then report the test (%[1]s @report).\n", cmd)
	usagePrinter.Errf("usage: \t%s @init[=TEST_SUITE_NAME] [@CONFIG_1] ... [@CONFIG_N] \n", cmd)
	usagePrinter.Errf("usage: \t%s <COMMAND> [ARG_1] ... [ARG_N] [@CONFIG_1] ... [@CONFIG_N] [@ASSERTION_1] ... [@ASSERTION_N]\n", cmd)
	usagePrinter.Errf("usage: \t%s @report[=TEST_SUITE_NAME] \n", cmd)
	usagePrinter.Errf("\tCONFIG available: @ignore @stopOnFailure @keepStdout @keepStderr @keepOutputs @timeout=Duration @fork=N\n")
	usagePrinter.Errf("\tCOMMAND and ARGs: the command on which to run tests\n")
	usagePrinter.Errf("\tASSERTIONs available: @fail @success @exit=N @stdout= @stdout~ @stderr= @stderr~ @cmd= @exists=\n")
	usagePrinter.Errf("In complex cases assertions must be correlated by a token. You can generate a token with @init @printToken or @init @exportToken and supply it with @token=\n")
	usagePrinter.Flush()
}

func GlobalConfig(ctx facade.GlobalContext) (exitCode int16, err error) {
	// Init or Update Global config
	exitCode = 0
	err = ctx.Save()
	return
}

func globalReport(ctx facade.GlobalContext, asyncMode bool) (exitCode int16, err error) {
	exitCode = 1
	token := ctx.Token
	isolation := ctx.Isolation
	all := ctx.Config.ReportAll.Get()

	var testSuites []string
	// if all {
	// 	testSuites, err = ctx.Repo.ListAllSuites()
	// } else {
	// 	testSuites, err = ctx.Repo.ListReportableSuites()
	// }
	testSuites, err = ctx.Repo.ListReportableSuitesByMode(asyncMode, all)
	if err != nil {
		return
	}

	logger.Info("Global reporting suites", "token", token, "all", all, "asyncMode", asyncMode, "suites", testSuites)

	nothingToReport := true

	var suiteOutcomes []model.SuiteOutcome
	var suiteContexts []facade.SuiteContext
	goodModeSuite := false
	reportPassed := true
	for _, testSuite := range testSuites {
		suiteCtx := facade.NewSuiteContext(token, isolation, testSuite, false, model.ReportAction, model.Config{})
		suiteAsync := suiteCtx.Config.Async.Get()
		if suiteAsync != asyncMode {
			// Ignore suites in bad async mode
			continue
		}
		// Override suite Keep config for reportAll which is global
		suiteCtx.Config.Keep = ctx.Config.Keep
		suiteIgnored := suiteCtx.Config.IgnoreSuite.GetOr(false)
		goodModeSuite = true
		count := suiteCtx.Repo.ToReportTestCountBySuiteAndMode(testSuite, asyncMode, all)
		if count > 0 || suiteIgnored {
			nothingToReport = false
			suiteContexts = append(suiteContexts, suiteCtx)
			var code int16
			var suiteOutcome model.SuiteOutcome
			suiteOutcome, code, err = reportTestSuite(suiteCtx)
			if err != nil {
				// FIXME: aggregate errors
				exitCode = 1
				return
			}
			if code != 0 {
				exitCode = code
			}
			suiteOutcomes = append(suiteOutcomes, suiteOutcome)
			if suiteOutcome.Outcome != model.IGNORED && suiteOutcome.Outcome != model.PASSED {
				reportPassed = false
			}
		}
	}

	if len(testSuites) > 0 && !goodModeSuite {
		// No suites in supplied async mode => Nothing to report.
		return
	}

	if nothingToReport {
		allSuites, _ := ctx.Repo.ListAllSuites()
		err = fmt.Errorf("you must perform some test prior to report all suites (testSuites: %s ; asyncMode: %v ; allSuites: %s)", testSuites, asyncMode, allSuites)
		return
	}

	Dpl.ReportSuites(suiteOutcomes)

	for _, suiteCtx := range suiteContexts {
		Dpl.CloseSuite(suiteCtx)
	}

	if reportPassed {
		exitCode = 0
	}
	return
}

func ProcessGlobalReportDef(def model.ReportDefinition, asyncMode bool) (exitCode int16, err error) {
	ctx := facade.NewGlobalContext(def.Token, def.Isolation, model.Config{})
	exitCode, err = globalReport(ctx, asyncMode)
	return
}

func reportTestSuite(ctx facade.SuiteContext) (suiteOutcome model.SuiteOutcome, exitCode int16, err error) {
	exitCode = 1
	cfg := ctx.Config
	testSuite := cfg.TestSuite.Get()
	testCount := ctx.Repo.TestCount(testSuite)
	suiteIgnored := ctx.Config.IgnoreSuite.GetOr(false)
	logger.Info("Reporting suite", "suite", testSuite, "testCount", testCount)

	if !suiteIgnored && testCount == 0 {
		err = fmt.Errorf("you must perform some test prior to report: [%s] suite", testSuite)
		ProcessSuiteError(ctx, err)
		exitCode = 1
		return
	} else {
		suiteOutcome, err = ctx.Repo.LoadSuiteOutcome(testSuite)
		if err != nil {
			ProcessSuiteError(ctx, err)
			return
		}
	}

	if suiteOutcome.Outcome == model.IGNORED || suiteOutcome.Outcome == model.PASSED {
		exitCode = 0
	}

	// FIXME: should not clear test suite in report but in suite opening
	// FIXME: but report suite should report only not reported suite and not all existing suites !
	// if !cfg.Keep.Is(true) {
	// 	err = ctx.Repo.ClearTestSuite(suiteOutcome.TestSuite)
	// 	ProcessSuiteError(ctx, err)
	// }

	return
}

func ProcessReportDef(def model.ReportDefinition) (exitCode int16, err error) {
	//logger.Warn("ProcessReportDef()", "def", def)
	ctx := facade.NewSuiteContext(def.Token, def.Isolation, def.TestSuite, false, model.ReportAction, def.Config) // FIXME ? removing def.Config ?
	//ctx := facade.NewSuiteContext(def.Token, def.Isolation, def.TestSuite, false, model.ReportAction, model.Config{}) // FIXME ? removing def.Config ?

	suiteOutcome, exitCode, err := reportTestSuite(ctx)
	if err != nil {
		return 1, err
	}

	if suiteOutcome.Duration < 0 {
		// FIXME: report should save an endTime for suite duration to be saved
		suiteOutcome.Duration = time.Since(ctx.Config.SuiteStartTime.Get())
	}

	Dpl.ReportSuite(suiteOutcome)
	Dpl.CloseSuite(ctx)

	return
}

func performTest(testDef model.TestDefinition) (exitCode int16, err error) {
	logger.Debug("Performing test")
	exitCode = 1
	cfg := testDef.Config
	ctx, err := facade.NewTestContext2(testDef)
	ProcessTestError(ctx, err)
	seq := testDef.Seq

	td := Dpl.OpenTest(ctx)
	//defer td.Close()
	defer Dpl.CloseTest(ctx)
	td.Title()

	if cfg.Ignore.Is(true) {
		ctx.IncrementIgnoredCount()
		oc := ctx.IgnoredTestOutcome()
		ctx.Repo.SaveTestOutcome(oc)
		td.Outcome(oc)
		exitCode = 0
		return
	}

	err = ctx.ConfigMocking()
	ProcessTestError(ctx, err)

	var beforeErrors errorz.Aggregated
	for _, before := range cfg.Before {
		cmdBefore := cmdz.Cmd(before...)
		beforeExit, beforeErr := cmdBefore.BlockRun()
		// FIXME: what to do of before exit code or beforeErr ?
		_ = beforeExit
		if beforeErr != nil {
			err := fmt.Errorf("error running before cmd: [%s]: %w", cmdBefore.String(), beforeErr)
			beforeErrors.Add(err)
		}
	}
	if beforeErrors.GotError() {
		outcome := model.TestOutcome{
			Outcome:       model.ERRORED,
			Duration:      0 * time.Millisecond,
			TestSignature: testDef.TestSignature,
			Err:           beforeErrors.Return(),
		}
		ctx.Repo.SaveTestOutcome(outcome)
		td.Outcome(outcome)
		return 1, nil
	}

	// Build assertions
	_, assertions, agg := ParseArgs(testDef.Config.Prefix.Get(), testDef.CmdArgs)
	if agg.GotError() {
		err = agg.Return()
		return
	}
	outcome, err := ctx.AssertCmdExecBlocking(seq, assertions)
	ProcessTestError(ctx, err)

	td.Outcome(outcome)

	for _, after := range cfg.After {
		cmdAfter := cmdz.Cmd(after...)
		afterExit, afterErr := cmdAfter.BlockRun()
		// FIXME: what to do of after exit code or afterErr ?
		_ = afterExit
		if afterErr != nil {
			err = fmt.Errorf("error running after cmd: [%s]: %w", cmdAfter.String(), afterErr)
		}
	}

	if cfg.StopOnFailure.Is(true) && outcome.Outcome != model.PASSED {
		exitCode = max(outcome.ExitCode, 1)
	} else {
		exitCode = 0
	}

	return
}

func ProcessTestDef(testDef model.TestDefinition) (exitCode int16) {
	testSuite := testDef.TestSuite
	testCfg := testDef.Config
	testCtx, err := facade.NewTestContext2(testDef)

	Dpl.Quiet(testCfg.Quiet.Is(true))

	ProcessTestError(testCtx, err)

	tooMuchFailures := testCtx.ProcessTooMuchFailures()

	if tooMuchFailures == 1 {
		// First time detecting TOO MUCH FAILURES
		Dpl.TooMuchFailures(testCtx.SuiteContext, testSuite)
	}
	if tooMuchFailures > 0 {
		exitCode = 0
		return
	}

	if !utils.IsWithinContainer() && (testCfg.ContainerDisabled.Is(true) || testCfg.ContainerImage.IsEmpty()) && len(testCfg.RootMocks) > 0 {
		err = fmt.Errorf("cannot mock absolute path outside a container")
		ProcessTestError(testCtx, err)
	}

	if testCfg.ContainerDisabled.Is(true) || testCfg.ContainerImage.IsEmpty() {
		logger.Debug("Performing test outside container", "image", testCfg.ContainerImage, "containerDisabled", testCfg.ContainerDisabled, "testConfig", testCfg)
		exitCode, err = performTest(testDef)

		ProcessTestError(testCtx, err)
	} else {
		logger.Info("Performing test inside container", "image", testCfg.ContainerImage, "containeriId", testCfg.ContainerId, "testConfig", testCfg)
		var ctId string
		ctId, exitCode, err = PerformTestInContainer(testCtx)
		ProcessTestError(testCtx, err)
		if testCfg.ContainerScope.Is(model.GLOBAL_SCOPE) {
			globalCfg, err2 := testCtx.Repo.GetGlobalConfig()
			ProcessTestError(testCtx, err2)
			globalCfg.ContainerId.Set(ctId)
			err2 = testCtx.Repo.SaveGlobalConfig(globalCfg)
			ProcessTestError(testCtx, err2)
		} else if testCfg.ContainerScope.Is(model.SUITE_SCOPE) {
			suiteCfg, err2 := testCtx.Repo.GetSuiteConfig(testSuite, true)
			ProcessTestError(testCtx, err2)
			suiteCfg.ContainerId.Set(ctId)
			err2 = testCtx.Repo.SaveSuiteConfig(suiteCfg)
			ProcessTestError(testCtx, err2)
		}
		//testCtx.NoErrorOrFatal(err)
	}
	return
}

func ProcessMalDefinedTest(testDef model.TestDefinition, malDefErr error) (exitCode int16) {
	testCtx, err := facade.NewTestContext2(testDef)
	if err != nil {
		// TODO
		panic(err)
	}

	td := Dpl.OpenTest(testCtx)
	defer td.Close()
	td.Title()

	outcome := model.TestOutcome{
		Outcome:       model.ERRORED,
		Duration:      0 * time.Millisecond,
		TestSignature: testDef.TestSignature,
		Err:           malDefErr,
	}
	testCtx.Repo.SaveTestOutcome(outcome)
	td.Outcome(outcome)

	//ProcessTestError(testCtx, parseArgsErrors.Return())
	return 1
}

func ProcessArgs(allArgs []string) (daemonToken, daemonIsol string, wait func() int16) {
	var exitCode int16
	exitCode = 1
	wait = func() int16 { return exitCode }

	if len(allArgs) == 1 {
		usage()
		return
	}

	// FIXME: if token supplied by ENV should be retrieved FIRST to get token and load Global config
	defaultCfg := model.NewGlobalDefaultConfig()
	envToken := utils.ReadEnvToken()
	if envToken != "" {
		// With token in env catch isolation from args quickly
		isol := utils.IsolationFromArgs(allArgs)
		envCtx := facade.NewGlobalContext(envToken, isol, defaultCfg)
		defaultCfg = envCtx.Config
	}
	rulePrefix := defaultCfg.Prefix.Get()

	signifientArgs := allArgs[1:]
	inputConfig, assertions, parseArgsErrors := ParseArgs(rulePrefix, signifientArgs)

	inputConfig.Token.Default(envToken)
	logger.Debug("Parsed args", "args", signifientArgs, "inputConfig", inputConfig, "assertions", assertions, "error", parseArgsErrors)
	if inputConfig.Debug.IsPresent() {
		model.LoggerLevel.Set(slog.Level(8 - inputConfig.Debug.Get()*4))
	}

	if inputConfig.Action.Is(model.UsageAction) {
		usage()
		return
	}

	token := inputConfig.Token.GetOr("")
	isolation := inputConfig.Isol.GetOr("")
	action := inputConfig.Action.Get()

	var err error
	switch action {
	case model.GlobalAction:
		logger.Debug("Executing Global action")
		if parseArgsErrors.GotError() {
			errorz.Fatal(parseArgsErrors)
		}
		globalCtx := facade.NewGlobalContext(token, isolation, inputConfig)

		ProcessGlobalError(globalCtx, parseArgsErrors.Return())
		Dpl.SetVerbose(globalCtx.Config.Verbose.Get())
		logger.Trace("Forged context", "ctx", globalCtx)
		logger.Info("Processing global action", "token", token)
		Dpl.Quiet(globalCtx.Config.Quiet.Is(true))
		exitCode, err = GlobalConfig(globalCtx)
	case model.InitAction:
		testSuite := inputConfig.TestSuite.Get()
		logger.Debug("Executing Init action", "suite", testSuite)

		// Check if suite exists and it's status
		rep := facade.Repo(token, isolation)
		exists, reported, kept, err := rep.SuiteStatus(testSuite)

		suiteCtx := facade.NewSuiteContext(token, isolation, testSuite, false, action, inputConfig)
		ProcessSuiteError(suiteCtx, err)

		// Store ignore at suite level
		suiteCtx.Config.IgnoreSuite = suiteCtx.Config.Ignore
		if suiteCtx.Config.IgnoreSuite.GetOr(false) {
			// Ignored suite should not be async
			suiteCtx.Config.Async.Set(false)
		}

		logger.Debug("repo suite status", "testSuite", testSuite, "exists", exists, "reported", reported, "kept", kept)

		if exists {
			n := rep.TestCount(testSuite)
			if n > 0 && !reported {
				err = fmt.Errorf("cannot erase test suite: [%s] which contains %d test(s) not reported yet", testSuite, n)
				ProcessSuiteError(suiteCtx, err)
			} else if n > 0 && kept {
				err = fmt.Errorf("cannot erase test suite: [%s] which must be kept", testSuite)
				ProcessSuiteError(suiteCtx, err)
			}
		}

		ProcessSuiteError(suiteCtx, parseArgsErrors.Return())
		Dpl.SetVerbose(suiteCtx.Config.Verbose.Get())
		logger.Trace("Forged context", "ctx", suiteCtx)
		logger.Info("Processing init action", "token", token, "suite", testSuite, "async", suiteCtx.Config.Async.Get())
		Dpl.Quiet(suiteCtx.Config.Quiet.Is(true))

		exitCode, err = cliInitTestSuite(suiteCtx)
		ProcessSuiteError(suiteCtx, err)

	case model.ReportAction:
		// Report can be async (run by daemon) or not
		// Report can wait (for termination) or not
		// Report must always be delayed until all tests are done

		defaultGlobalTimeout := 5 * time.Minute

		// For now enforce async false on report
		//inputConfig.Async.Set(false)

		if inputConfig.GlobalReport.Is(true) {
			// Report all sync suites then all async suites

			logger.Debug("Executing Report all action")
			// Reporting All test suite
			if parseArgsErrors.GotError() {
				errorz.Fatal(parseArgsErrors)
			}
			globalCtx := facade.NewGlobalContext(token, isolation, inputConfig)
			globalCfg := globalCtx.Config
			Dpl.SetVerbose(globalCfg.Verbose.Get())
			rep := facade.Repo(token, isolation)

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

			// 1- Report all sync suites
			toReportSyncTestCount := rep.ToReportTestCountByMode(false, globalCfg.ReportAll.GetOr(model.DefaultReportAll))
			if ignoredSuiteCount+toReportSyncTestCount > 0 {
				syncSuites, err := rep.ListSyncSuites()
				ProcessGlobalError(globalCtx, err)
				exitCode, err = globalReport(globalCtx, false)
				ProcessGlobalError(globalCtx, err)
				for _, suite := range syncSuites {
					err = cliAfterSuiteReport(globalCtx.Token, globalCtx.Isolation, suite, Dpl)
					ProcessGlobalError(globalCtx, err)
				}
			} else {
				exitCode = 0
			}

			// 2- Report all async suites
			toReportAsyncTestCount := rep.ToReportTestCountByMode(true, globalCfg.ReportAll.GetOr(model.DefaultReportAll))
			if toReportAsyncTestCount > 0 {

				asyncSuites, err := rep.ListAsyncSuites()
				ProcessGlobalError(globalCtx, err)
				// fmt.Printf("<<>> ASYNC suites count: %d\n", len(asyncSuites))
				//if globalCtx.Config.Async.Is(true) {
				if len(asyncSuites) > 0 {
					start := time.Now()
					for globalCtx.Repo.NotReportedTestCount() == 0 {
						if time.Since(start) > model.WaitAsyncReportTestTimeout {
							err := fmt.Errorf("you must perform some test prior to report")
							ProcessGlobalError(globalCtx, err)
						}
						time.Sleep(time.Millisecond)
					}

					// Delegate report all processing to daemon
					//logger.Info("executing report all on async display")
					logger.Info("executing report all (queueing report)")
					def := model.ReportDefinition{
						Token:     token,
						Isolation: isolation,
						//TestSuite: "__global",
						Config: globalCtx.Config,
					}
					op := model.ReportAllOperation(true, def) // FIXME should not block if test can be run simultaneously
					err = globalCtx.Repo.QueueOperation(&op)
					if err != nil {
						errorz.Fatal(err)
					}

					asyncDpl := asyncdisplay.New(globalCtx.Repo.BackingFilepath(), false, printz.NewStandardOutputs())
					daemonIsol = globalCtx.Isolation
					daemonToken = globalCtx.Token

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
							err = cliAfterSuiteReport(daemonToken, daemonIsol, suite, asyncDpl)
							ProcessGlobalError(globalCtx, err)
						}

						err = globalCtx.Repo.MarkSuitesReported()
						ProcessGlobalError(globalCtx, err)

						// Clear all reported suite async display
						suites, err := globalCtx.Repo.ListReportedAsyncSuites()
						ProcessGlobalError(globalCtx, err)

						for _, suite := range suites {
							asyncdisplay.ClearSuite(globalCtx.Repo.BackingFilepath(), suite)
						}
						return max(exitCode, asyncExitCode)
					}

					// FIXME: Waiting for zcreen tail but daemon could not be launched !
					err = asyncDpl.TailSuppliedBlocking(asyncSuites, globalCtx.Config.SuiteTimeout.GetOr(model.DefaultSuiteTimeout))
					ProcessGlobalError(globalCtx, err)
					logger.Info("finished async TailAllBlocking", "opId", op.Id())
				}
			}

			if ignoredSuiteCount+toReportSyncTestCount+toReportAsyncTestCount == 0 {
				exitCode = 1
				err := fmt.Errorf("you must perform some test prior to report globaly")
				ProcessGlobalError(globalCtx, err)
			}

			// Display report all footer before wait is called and then before suites are marked reported for accurate timings
			Dpl.ReportAllFooter(globalCtx)

		} else {
			// Reporting One test suite
			testSuite := inputConfig.TestSuite.Get()
			logger.Debug("Executing Report suite action", "suite", testSuite)
			suiteCtx := facade.NewSuiteContext(token, isolation, testSuite, false, action, inputConfig)

			ProcessSuiteError(suiteCtx, parseArgsErrors.Return())

			Dpl.SetVerbose(suiteCtx.Config.Verbose.Get())

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

				asyncDpl := asyncdisplay.New(suiteCtx.Repo.BackingFilepath(), false, printz.NewStandardOutputs())
				daemonIsol = suiteCtx.Isolation
				daemonToken = suiteCtx.Token

				if suiteCtx.Config.Wait.Is(true) {
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

						err = cliAfterSuiteReport(daemonToken, daemonIsol, testSuite, asyncDpl)
						ProcessSuiteError(suiteCtx, err)

						//asyncDpl.ClearSuite(suiteCtx)
						return exitCode
					}
				} else {
					exitCode = 0
				}

				// FIXME: Waiting for zcreen tail but daemon could not be launched !
				err = asyncDpl.TailBlocking(testSuite, suiteCtx.Config.SuiteTimeout.Get())
				ProcessSuiteError(suiteCtx, err)
				logger.Info("finished async TailBlocking")

			} else {
				logger.Info("executing report in sync", "suite", testSuite)
				exitCode, err = ProcessReportDef(def)
				ProcessSuiteError(suiteCtx, err)
				suiteCtx.Repo.Done(&op)
				err = cliAfterSuiteReport(suiteCtx.Token, suiteCtx.Isolation, testSuite, Dpl)
				ProcessSuiteError(suiteCtx, err)
			}

		}
	case model.TestAction:
		testSuite := inputConfig.TestSuite.Get()

		ppid := uint32(utils.ReadEnvPpid())
		logger.Debug("Executing Test action", "suite", testSuite)
		testCtx, err := facade.NewTestContext(token, isolation, testSuite, 0, inputConfig, ppid)

		if err != nil {
			Dpl.Errors(err)
		}

		if testCtx.Config.IgnoreSuite.GetOr(false) {
			exitCode = 0
			return
		}

		//ProcessTestError(testCtx, err)

		testCtx.IncrementTestCount()
		Dpl.SetVerbose(testCtx.Config.Verbose.Get())
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
			exitCode = ProcessMalDefinedTest(testDef, parseArgsErrors.Return())
			return
		}

		logger.Debug("Test definition", "token", token, "isolation", isolation, "suite", testSuite, "seq", seq)
		if !testCfg.Async.Is(true) {
			// Process test without daemon
			// enforce wait
			logger.Info("executing test in sync (not queueing test)", "suite", testSuite, "seq", seq)
			exitCode = ProcessTestDef(testDef)
		} else {
			// Init suite config in repo if necessary
			//facade.NewSuiteContext(token, isolation, testSuite, true, action, model.Config{Async: utilz.OptionalOf(true)})

			// Delegate test processing to daemon
			logger.Info("executing test async (queueing test)", "suite", testSuite, "seq", seq)
			testOp := model.TestOperation(testSuite, seq, true, testDef) // FIXME should not block if test can be run simultaneously
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
			daemonIsol = isolation
			daemonToken = token
		}
	default:
		err = fmt.Errorf("action: [%v] not known", inputConfig.Action)
	}

	logger.Info("exiting", "exitCode", exitCode)

	if err != nil {
		errorz.Fatal(err, "suite:", inputConfig.TestSuite, "token:", inputConfig.Token)
	}
	return
}

func ProcessGlobalError(ctx facade.GlobalContext, err error) {
	if err != nil {
		logger.Debug("Reporting global error", "error", err)
		ctx.Config.TestSuite.IfPresent(func(testSuite string) error {
			ctx.Repo.UpdateLastTestTime(testSuite)
			return nil
		})
		Dpl.GlobalErrors(ctx, err)
	}
}

func ProcessSuiteError(ctx facade.SuiteContext, err error) {
	if err != nil {
		logger.Debug("Reporting suite error", "error", err)
		ctx.Config.TestSuite.IfPresent(func(testSuite string) error {
			ctx.IncrementErroredCount()
			return nil
		})
		Dpl.SuiteErrors(ctx, err)
		ProcessGlobalError(ctx.GlobalContext, err)
	}
}

func ProcessTestError(ctx facade.TestContext, err error) {
	if err != nil {
		logger.Debug("Reporting test error", "error", err)
		outcome := model.NewTestOutcome2(ctx.Config, ctx.Seq)
		outcome.Outcome = model.ERRORED
		outcome.Err = err
		err2 := ctx.Repo.SaveTestOutcome(outcome)
		if err2 != nil {
			logger.Error("unable to save errored test outcome", "error", err2)
		}
		Dpl.TestErrors(ctx, err)
		ProcessSuiteError(ctx.SuiteContext, err)
	}
}
