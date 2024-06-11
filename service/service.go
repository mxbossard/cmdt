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
	dpl    = display.New()
	logger = zlog.New() //slog.New(slog.NewTextHandler(os.Stderr, model.DefaultLoggerOpts))
)

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
	if err != nil {
		ctx.NoErrorOrFatal(err)
	}

	dpl.Suite(ctx)
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

	if len(testSuites) == 0 {
		err = fmt.Errorf("you must perform some test prior to report all suites")
		return
	}

	exitCode = 0
	for _, testSuite := range testSuites {
		suiteCtx := facade.NewSuiteContext(token, isolation, testSuite, false, model.ReportAction, ctx.Config)
		if suiteCtx.Repo.TestCount(testSuite) > 0 {
			var code int16
			code, err = ReportTestSuite(suiteCtx)
			if code != 0 {
				exitCode = code
			}
			if err != nil {
				// FIXME: aggregate errors
				return
			}
		}
	}
	dpl.ReportAllFooter(ctx)

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

func ReportTestSuite(ctx facade.SuiteContext) (exitCode int16, err error) {
	exitCode = 1
	cfg := ctx.Config
	testSuite := cfg.TestSuite.Get()
	testCount := ctx.Repo.TestCount(testSuite)
	logger.Debug("reporting suite", "suite", testSuite, "testCount", testCount)

	if testCount == 0 {
		err = fmt.Errorf("you must perform some test prior to report: [%s] suite", testSuite)
		return
	}

	//logger.Info("Reporting suite", "testCount", testCount, "ctx", ctx)

	var suiteOutcome model.SuiteOutcome
	suiteOutcome, err = ctx.Repo.LoadSuiteOutcome(testSuite)
	if err != nil {
		return
	}
	dpl.ReportSuite(ctx, suiteOutcome, 16)

	if suiteOutcome.FailedCount == 0 && suiteOutcome.ErroredCount == 0 {
		exitCode = 0
	}

	if !cfg.Keep.Is(true) {
		err = ctx.Repo.ClearTestSuite(suiteOutcome.TestSuite)
		logger.Debug("Cleared suite", "suite", suiteOutcome.TestSuite)
	}

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
	ctx := facade.NewTestContext2(testDef)
	seq := testDef.Seq

	dpl.TestTitle(ctx, seq)

	if cfg.Ignore.Is(true) {
		ctx.IncrementIgnoredCount()
		oc := ctx.IgnoredTestOutcome(seq)
		ctx.Repo.SaveTestOutcome(oc)
		dpl.TestOutcome(ctx, oc)
		exitCode = 0
		return
	}

	err = ctx.ConfigMocking()
	ctx.NoErrorOrFatal(err)

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
	outcome := ctx.AssertCmdExecBlocking(seq, assertions)

	dpl.TestOutcome(ctx, outcome)

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
	testCtx := facade.NewTestContext2(testDef)

	tooMuchFailures := testCtx.ProcessTooMuchFailures()

	if tooMuchFailures == 1 {
		// First time detecting TOO MUCH FAILURES
		dpl.TooMuchFailures(testCtx.SuiteContext, testSuite)
	}
	if tooMuchFailures > 0 {
		exitCode = 0
		return
	}

	var err error
	if !utils.IsWithinContainer() && (testCfg.ContainerDisabled.Is(true) || testCfg.ContainerImage.IsEmpty()) && len(testCfg.RootMocks) > 0 {
		err = fmt.Errorf("cannot mock absolute path outside a container")
		testCtx.NoErrorOrFatal(err)
	}

	dpl.Quiet(testCfg.Quiet.Is(true))
	if testCfg.ContainerDisabled.Is(true) || testCfg.ContainerImage.IsEmpty() {
		logger.Debug("Performing test outside container", "image", testCfg.ContainerImage, "containerDisabled", testCfg.ContainerDisabled, "testConfig", testCfg)
		exitCode, err = PerformTest(testDef)
		testCtx.NoErrorOrFatal(err)
	} else {
		logger.Info("Performing test inside container", "image", testCfg.ContainerImage, "containeriId", testCfg.ContainerId, "testConfig", testCfg)
		var ctId string
		ctId, exitCode, err = PerformTestInContainer(testCtx)
		testCtx.NoErrorOrFatal(err)
		if testCfg.ContainerScope.Is(model.GLOBAL_SCOPE) {
			globalCfg, err2 := testCtx.Repo.GetGlobalConfig()
			testCtx.NoErrorOrFatal(err2)
			globalCfg.ContainerId.Set(ctId)
			err2 = testCtx.Repo.SaveGlobalConfig(globalCfg)
			testCtx.NoErrorOrFatal(err2)
		} else if testCfg.ContainerScope.Is(model.SUITE_SCOPE) {
			suiteCfg, err2 := testCtx.Repo.GetSuiteConfig(testSuite, true)
			testCtx.NoErrorOrFatal(err2)
			suiteCfg.ContainerId.Set(ctId)
			err2 = testCtx.Repo.SaveSuiteConfig(suiteCfg)
			testCtx.NoErrorOrFatal(err2)
		}
		//testCtx.NoErrorOrFatal(err)
	}
	return
}

func ProcessArgs(allArgs []string) (daemonToken string, wait func() int16) {
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
		globalCtx.NoErrorOrFatal(agg.Return())
		logger.Trace("Forged context", "ctx", globalCtx)
		logger.Info("Processing global action", "token", token)
		dpl.Quiet(globalCtx.Config.Quiet.Is(true))
		exitCode, err = GlobalConfig(globalCtx)
	case model.InitAction:
		testSuite := inputConfig.TestSuite.Get()
		suiteCtx := facade.NewSuiteContext(token, isolation, testSuite, false, action, inputConfig)
		suiteCtx.NoErrorOrFatal(agg.Return())
		logger.Trace("Forged context", "ctx", suiteCtx)
		logger.Info("Processing init action", "token", token)
		dpl.Quiet(suiteCtx.Config.Quiet.Is(true))
		exitCode, err = InitTestSuite(suiteCtx)
	case model.ReportAction:
		// Report can be async (run by daemon) or not
		// Report can wait (for termination) or not
		// Report must always be delayed until all tests are done

		defaultGlobalTimeout := 5 * time.Minute

		//inputConfig.Async.Set(false) // For now enforce async false on report
		if inputConfig.ReportAll.Is(true) {
			// Reporting All test suite
			if agg.GotError() {
				errorz.Fatal(agg)
			}
			globalCtx := facade.NewGlobalContext(token, isolation, inputConfig)
			if globalCtx.Config.Async.Is(false) {
				// Process report all without daemon
				logger.Trace("Forged context", "ctx", globalCtx)
				logger.Info("executing report all in sync (not queueing report)")
				dpl.Quiet(globalCtx.Config.Quiet.Is(true))
				// TODO: wait all tests run
				// Daemon must be off or No test remaining in suite queue
				globalCtx.Repo.WaitAllEmpty(globalCtx.Config.SuiteTimeout.GetOr(defaultGlobalTimeout)) // FIXME: bad timeout
				exitCode, err = ReportAllTestSuites(globalCtx)
			} else {
				// Delegate report all processing to daemon
				logger.Info("executing report all async (queueing report)")
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

				if globalCtx.Config.Wait.Is(true) {
					wait = func() int16 {
						// FIXME: bad timeout
						exitCode, err := globalCtx.Repo.WaitOperationDone(&op, globalCtx.Config.SuiteTimeout.GetOr(defaultGlobalTimeout))
						if err != nil {
							panic(err)
						}
						return exitCode
					}
				} else {
					exitCode = 0
				}
				daemonToken = globalCtx.Token
			}
		} else {
			// Reporting One test suite
			testSuite := inputConfig.TestSuite.Get()
			suiteCtx := facade.NewSuiteContext(token, isolation, testSuite, false, action, inputConfig)
			suiteCtx.NoErrorOrFatal(agg.Return())

			def := model.ReportDefinition{
				Token:     token,
				Isolation: isolation,
				TestSuite: testSuite,
				Config:    suiteCtx.Config,
			}
			op := model.ReportOperation(testSuite, true, def) // FIXME should not block if test can be run simultaneously

			if suiteCtx.Config.Async.Is(false) {
				// Process report without daemon
				logger.Trace("Forged context", "ctx", suiteCtx)
				logger.Info("executing report in sync (not queueing report)", "suite", testSuite)
				dpl.Quiet(suiteCtx.Config.Quiet.Is(true))
				suiteCtx.Repo.WaitEmptyQueue(testSuite, suiteCtx.Config.SuiteTimeout.Get())
				//exitCode, err = ReportTestSuite(suiteCtx)
				exitCode, err = ProcessReportDef(def)
				suiteCtx.NoErrorOrFatal(err)
				suiteCtx.Repo.Done(&op)
			} else {
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
				suiteCtx.NoErrorOrFatal(err)

				if suiteCtx.Config.Wait.Is(true) {
					wait = func() int16 {
						// FIXME: bad timeout
						p := logger.QualifiedPerfTimer("waiting report done ...", "suite", testSuite)
						exitCode, err := suiteCtx.Repo.WaitOperationDone(&op, suiteCtx.Config.SuiteTimeout.Get())
						if err != nil {
							panic(err)
						}
						p.End()
						return exitCode
					}
				} else {
					exitCode = 0
				}
				daemonToken = suiteCtx.Token
			}
		}
	case model.TestAction:
		testSuite := inputConfig.TestSuite.Get()
		ppid := uint32(utils.ReadEnvPpid())
		testCtx := facade.NewTestContext(token, isolation, testSuite, 0, inputConfig, ppid)
		testCtx.IncrementTestCount()
		seq := testCtx.Seq
		testCtx.NoErrorOrFatal(agg.Return())
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

		if testCfg.Async.Is(false) {
			// Process test without daemon
			// enforce wait
			logger.Info("executing test in sync (not queueing test)", "suite", testSuite, "seq", seq)
			exitCode = ProcessTestDef(testDef)
		} else {
			// Delegate test processing to daemon
			logger.Info("executing test async (queueing test)", "suite", testSuite, "seq", seq)
			testOp := model.TestOperation(testSuite, seq, true, testDef) // FIXME should not block if test can be run simultaneously
			err = testCtx.Repo.QueueOperation(&testOp)
			testCtx.NoErrorOrFatal(err)

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
