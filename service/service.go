package service

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"mby.fr/cmdtest/display"
	"mby.fr/cmdtest/facade"
	"mby.fr/cmdtest/model"
	"mby.fr/cmdtest/utils"
	"mby.fr/utils/cmdz"
	"mby.fr/utils/errorz"
	"mby.fr/utils/printz"
	"mby.fr/utils/utilz"
	"mby.fr/utils/zlog"
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

func InitTestSuite(ctx facade.SuiteContext) (exitCode int16, err error) {
	// Clear and Init new test suite
	exitCode = 0
	cfg := ctx.Config

	if cfg.Async.Is(true) {
		asyncDpl := display.NewAsync(cfg.Token.Get(), cfg.Isol.Get())
		asyncDpl.Clear(cfg.TestSuite.Get())
	}

	var token string
	if cfg.PrintToken.Is(true) {
		token, err = utils.ForgeUuid()
		if err != nil {
			return
		}
		fmt.Printf("%s\n", token)
		cfg.Token = utilz.OptionalOf(token)
	} else if cfg.ExportToken.Is(true) {
		token, err = utils.ForgeUuid()
		if err != nil {
			return
		}
		fmt.Printf("export %s=%s\n", model.ContextTokenEnvVarName, token)
		cfg.Token = utilz.OptionalOf(token)
	}

	err = ctx.InitSuite()
	ProcessSuiteError(ctx, err)

	Dpl.Suite(ctx)
	return
}

func ReportAllTestSuites(ctx facade.GlobalContext) (exitCode int16, err error) {
	exitCode = 1
	token := ctx.Token
	isolation := ctx.Isolation

	var testSuites []string
	testSuites, err = ctx.Repo.ListTestSuites()
	if err != nil {
		return
	}

	logger.Info("Reporting all suites", "token", token, "suites", testSuites)

	var testCount uint16
	exitCode = 0

	var suiteOutcomes []model.SuiteOutcome

	for _, testSuite := range testSuites {
		suiteCtx := facade.NewSuiteContext(token, isolation, testSuite, false, model.ReportAction, ctx.Config)
		count := suiteCtx.Repo.TestCount(testSuite)
		if count > 0 {
			testCount += count
			var code int16
			var suiteOutcome model.SuiteOutcome
			suiteOutcome, code, err = reportTestSuite(suiteCtx)
			if err != nil {
				// FIXME: aggregate errors
				return
			}
			if code != 0 {
				exitCode = code
			}
			suiteOutcomes = append(suiteOutcomes, suiteOutcome)
		}
	}

	if testCount == 0 {
		err = fmt.Errorf("you must perform some test prior to report all suites")
		return
	}

	Dpl.ReportSuites(suiteOutcomes)
	Dpl.ReportAllFooter(ctx)

	return
}

func ProcessReportAllDef(def model.ReportDefinition) (exitCode int16) {
	var err error
	ctx := facade.NewGlobalContext(def.Token, def.Isolation, model.Config{})
	exitCode, err = ReportAllTestSuites(ctx)
	if err != nil {
		errorz.Fatal(err)
	}
	return
}

func reportTestSuite(ctx facade.SuiteContext) (suiteOutcome model.SuiteOutcome, exitCode int16, err error) {
	exitCode = 1
	cfg := ctx.Config
	testSuite := cfg.TestSuite.Get()
	testCount := ctx.Repo.TestCount(testSuite)
	logger.Info("Reporting suite", "suite", testSuite, "testCount", testCount)

	if testCount == 0 {
		err = fmt.Errorf("you must perform some test prior to report: [%s] suite", testSuite)
		return
	}

	suiteOutcome, err = ctx.Repo.LoadSuiteOutcome(testSuite)
	if err != nil {
		return
	}

	if suiteOutcome.FailedCount == 0 && suiteOutcome.ErroredCount == 0 {
		exitCode = 0
	}

	if !cfg.Keep.Is(true) {
		err = ctx.Repo.ClearTestSuite(suiteOutcome.TestSuite)
		logger.Debug("Cleared suite", "suite", suiteOutcome.TestSuite)
	}

	return
}

func ReportTestSuite(ctx facade.SuiteContext) (exitCode int16, err error) {
	suiteOutcome, exitCode, err := reportTestSuite(ctx)
	Dpl.ReportSuite(suiteOutcome)
	return
}

func ProcessReportDef(def model.ReportDefinition) (exitCode int16, err error) {
	//logger.Warn("ProcessReportDef()", "def", def)
	//var err error
	ctx := facade.NewSuiteContext(def.Token, def.Isolation, def.TestSuite, false, model.ReportAction, def.Config)
	exitCode, err = ReportTestSuite(ctx)
	//ctx.NoErrorOrFatal(err)
	return
}

func PerformTest(testDef model.TestDefinition) (exitCode int16, err error) {
	exitCode = 1
	cfg := testDef.Config
	ctx, err := facade.NewTestContext2(testDef)
	ProcessTestError(ctx, err)
	seq := testDef.Seq

	Dpl.TestTitle(ctx)

	if cfg.Ignore.Is(true) {
		ctx.IncrementIgnoredCount()
		oc := ctx.IgnoredTestOutcome()
		ctx.Repo.SaveTestOutcome(oc)
		Dpl.TestOutcome(ctx, oc)
		exitCode = 0
		return
	}

	err = ctx.ConfigMocking()
	ProcessTestError(ctx, err)

	for _, before := range cfg.Before {
		cmdBefore := cmdz.Cmd(before...)
		beforeExit, beforeErr := cmdBefore.BlockRun()
		// FIXME: what to do of before exit code or beforeErr ?
		_ = beforeExit
		if beforeErr != nil {
			err = fmt.Errorf("error running before cmd: [%s]: %w", cmdBefore.String(), beforeErr)
			return
		}
	}

	// Build assertions
	_, assertions, agg := ParseArgs(testDef.Config.Prefix.Get(), testDef.CmdArgs)
	if agg.GotError() {
		err = agg.Return()
		return
	}
	outcome, err := ctx.AssertCmdExecBlocking(seq, assertions)
	ProcessTestError(ctx, err)

	Dpl.TestOutcome(ctx, outcome)

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
		//ReportTestSuite(ctx)
		// FIXME do we need to call ReportTestSuite ?
	} else {
		exitCode = 0
	}

	return
}

func ProcessTestDef(testDef model.TestDefinition) (exitCode int16) {
	//token := testDef.Token
	testSuite := testDef.TestSuite
	testCfg := testDef.Config
	testCtx, err := facade.NewTestContext2(testDef)

	Dpl.OpenTest(testCtx)
	defer Dpl.CloseTest(testCtx)

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

	Dpl.Quiet(testCfg.Quiet.Is(true))
	if testCfg.ContainerDisabled.Is(true) || testCfg.ContainerImage.IsEmpty() {
		logger.Debug("Performing test outside container", "image", testCfg.ContainerImage, "containerDisabled", testCfg.ContainerDisabled, "testConfig", testCfg)
		exitCode, err = PerformTest(testDef)
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
		envCtx := facade.NewGlobalContext(envToken, "", defaultCfg)
		defaultCfg = envCtx.Config
	}
	rulePrefix := defaultCfg.Prefix.Get()

	signifientArgs := allArgs[1:]
	inputConfig, assertions, agg := ParseArgs(rulePrefix, signifientArgs)

	inputConfig.Token.Default(envToken)
	logger.Debug("Parsed args", "args", signifientArgs, "inputConfig", inputConfig, "assertions", assertions, "error", agg)
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
		// if agg.GotError() {
		// 	log.Fatal(agg.Error())
		// }
		if agg.GotError() {
			errorz.Fatal(agg)
		}
		globalCtx := facade.NewGlobalContext(token, isolation, inputConfig)
		ProcessGlobalError(globalCtx, agg.Return())
		Dpl.SetVerbose(globalCtx.Config.Verbose.Get())
		logger.Trace("Forged context", "ctx", globalCtx)
		logger.Info("Processing global action", "token", token)
		Dpl.Quiet(globalCtx.Config.Quiet.Is(true))
		exitCode, err = GlobalConfig(globalCtx)
	case model.InitAction:
		testSuite := inputConfig.TestSuite.Get()
		suiteCtx := facade.NewSuiteContext(token, isolation, testSuite, false, action, inputConfig)
		ProcessSuiteError(suiteCtx, agg.Return())
		Dpl.SetVerbose(suiteCtx.Config.Verbose.Get())
		logger.Trace("Forged context", "ctx", suiteCtx)
		logger.Info("Processing init action", "token", token)
		Dpl.Quiet(suiteCtx.Config.Quiet.Is(true))

		exitCode, err = InitTestSuite(suiteCtx)

	case model.ReportAction:
		// Report can be async (run by daemon) or not
		// Report can wait (for termination) or not
		// Report must always be delayed until all tests are done

		defaultGlobalTimeout := 5 * time.Minute

		// For now enforce async false on report
		//inputConfig.Async.Set(false)

		if inputConfig.ReportAll.Is(true) {
			// Reporting All test suite
			if agg.GotError() {
				errorz.Fatal(agg)
			}
			globalCtx := facade.NewGlobalContext(token, isolation, inputConfig)
			Dpl.SetVerbose(globalCtx.Config.Verbose.Get())

			//asyncDpl := display.NewAsync(token, isolation)
			// if globalCtx.Config.Async.Is(true) {
			// 	// Switch display to async one
			// 	Dpl = asyncDpl
			// }

			// if globalCtx.Config.Async.Is(false) {
			// Process report all without daemon
			logger.Trace("Forged context", "ctx", globalCtx)
			// logger.Info("executing report all in sync (not queueing report)")
			Dpl.Quiet(globalCtx.Config.Quiet.Is(true))

			// }

			if globalCtx.Config.Async.Is(true) {
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

				asyncDpl := display.NewAsync(token, isolation)
				//asyncDpl.StartDisplayAllRecorded(globalCtx.Config.SuiteTimeout.Get())

				asyncDpl.BlockTailAll(globalCtx.Config.SuiteTimeout.Get())

				// // Daemon must be off or No test remaining in suite queue
				// globalCtx.Repo.WaitAllEmpty(globalCtx.Config.SuiteTimeout.GetOr(defaultGlobalTimeout)) // FIXME: bad timeout
				// asyncDpl.WaitDisplayRecorded()

				// always wait
				// if globalCtx.Config.Wait.Is(true) {
				wait = func() int16 {
					// FIXME: bad timeout
					exitCode, err := globalCtx.Repo.WaitOperationDone(&op, globalCtx.Config.SuiteTimeout.GetOr(defaultGlobalTimeout))
					if err != nil {
						panic(err)
					}
					//asyncDpl.StopDisplayAllRecorded()
					return exitCode
				}
				// } else {
				// 	exitCode = 0
				// }

				daemonIsol = globalCtx.Isolation
				daemonToken = globalCtx.Token
			} else {
				exitCode, err = ReportAllTestSuites(globalCtx)
			}
		} else {
			// Reporting One test suite
			testSuite := inputConfig.TestSuite.Get()
			suiteCtx := facade.NewSuiteContext(token, isolation, testSuite, false, action, inputConfig)
			ProcessSuiteError(suiteCtx, agg.Return())

			Dpl.SetVerbose(suiteCtx.Config.Verbose.Get())

			def := model.ReportDefinition{
				Token:     token,
				Isolation: isolation,
				TestSuite: testSuite,
				Config:    suiteCtx.Config,
			}
			op := model.ReportOperation(testSuite, true, def) // FIXME should not block if test can be run simultaneously

			//asyncDpl := display.NewAsync(token, isolation)
			// if suiteCtx.Config.Async.Is(true) {
			// 	// Switch display to async one
			// 	Dpl = asyncDpl
			// }

			// if suiteCtx.Config.Async.Is(false) {
			// Process report without daemon
			logger.Trace("Forged context", "ctx", suiteCtx)
			Dpl.Quiet(suiteCtx.Config.Quiet.Is(true))
			// }

			if suiteCtx.Config.Async.Is(true) {
				//logger.Info("executing report on async display", "suite", testSuite)

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

				if suiteCtx.Config.Wait.Is(true) {
					wait = func() int16 {
						// FIXME: bad timeout
						p := logger.QualifiedPerfTimer("waiting report done ...", "suite", testSuite)
						exitCode, err = suiteCtx.Repo.WaitOperationDone(&op, suiteCtx.Config.SuiteTimeout.Get())
						if err != nil {
							panic(err)
						}
						p.End()
						return exitCode
					}
				} else {
					exitCode = 0
				}

				asyncDpl := display.NewAsync(token, isolation)
				//asyncDpl.StartDisplayRecorded(testSuite, suiteCtx.Config.SuiteTimeout.Get())
				asyncDpl.BlockTail(testSuite, suiteCtx.Config.SuiteTimeout.Get())
				//suiteCtx.Repo.WaitEmptyQueue(testSuite, suiteCtx.Config.SuiteTimeout.Get())
				//asyncDpl.WaitDisplayRecorded()

				daemonIsol = suiteCtx.Isolation
				daemonToken = suiteCtx.Token
			} else {
				//exitCode, err = ReportTestSuite(suiteCtx)
				exitCode, err = ProcessReportDef(def)
				ProcessSuiteError(suiteCtx, err)
				suiteCtx.Repo.Done(&op)
			}

		}
	case model.TestAction:
		testSuite := inputConfig.TestSuite.Get()
		ppid := uint32(utils.ReadEnvPpid())
		testCtx, err := facade.NewTestContext(token, isolation, testSuite, 0, inputConfig, ppid)
		ProcessTestError(testCtx, err)
		testCtx.IncrementTestCount()
		Dpl.SetVerbose(testCtx.Config.Verbose.Get())
		seq := testCtx.Seq
		ProcessTestError(testCtx, agg.Return())
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

		if !testCfg.Async.Is(true) {
			// Process test without daemon
			// enforce wait
			logger.Info("executing test in sync (not queueing test)", "suite", testSuite, "seq", seq)
			exitCode = ProcessTestDef(testDef)
		} else {
			// Delegate test processing to daemon
			logger.Info("executing test async (queueing test)", "suite", testSuite, "seq", seq)
			testOp := model.TestOperation(testSuite, seq, true, testDef) // FIXME should not block if test can be run simultaneously
			err = testCtx.Repo.QueueOperation(&testOp)
			ProcessTestError(testCtx, err)

			if testCfg.Wait.Is(true) {
				wait = func() int16 {
					exitCode, err := testCtx.Repo.WaitOperationDone(&testOp, testCfg.SuiteTimeout.Get())
					if err != nil {
						panic(err)
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
		errorz.Fatal(inputConfig.TestSuite, inputConfig.Token, err)
	}
	return
}

func ProcessGlobalError(ctx facade.GlobalContext, err error) {
	if err != nil {
		ctx.Config.TestSuite.IfPresent(func(testSuite string) error {
			ctx.Repo.UpdateLastTestTime(testSuite)
			//Dpl.Error(err)
			return nil
		})
		//Dpl.Error(err)
	}
}

func ProcessSuiteError(ctx facade.SuiteContext, err error) {
	if err != nil {
		ctx.Config.TestSuite.IfPresent(func(testSuite string) error {
			ctx.IncrementErroredCount()
			return nil
		})
	}
	ProcessGlobalError(ctx.GlobalContext, err)
}

func ProcessTestError(ctx facade.TestContext, err error) {
	if err != nil {
		outcome := model.NewTestOutcome2(ctx.Config, ctx.Seq)
		outcome.Outcome = model.ERRORED
		outcome.Err = err
		err2 := ctx.Repo.SaveTestOutcome(outcome)
		if err2 != nil {
			logger.Error("unable to save errored test outcome", "error", err2)
		}
	}
	ProcessSuiteError(ctx.SuiteContext, err)
	Dpl.TestErrors(ctx, err)
}
