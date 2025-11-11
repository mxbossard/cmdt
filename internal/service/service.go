package service

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"

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

	nothingToReport := false
	if all {
		allSuites, err := ctx.Repo.ListAllSuites()
		if err != nil {
			return exitCode, err
		}
		nothingToReport = len(allSuites) == 0
	} else {
		reportableSuites, err := ctx.Repo.ListReportableSuites()
		if err != nil {
			return exitCode, err
		}

		nothingToReport = len(reportableSuites) == 0
	}

	if nothingToReport {
		err = fmt.Errorf("you must perform some test prior to report all suites")
		return
	}

	reportableModedSuites, err := ctx.Repo.ListReportableSuitesByMode(asyncMode, all) //ToReportTestCountByMode(asyncMode, all)
	if err != nil {
		return exitCode, err
	}

	logger.Info("Global reporting suites", "token", token, "all", all, "asyncMode", asyncMode, "reportableSuites", reportableModedSuites)

	var suiteOutcomes []model.SuiteOutcome
	var suiteContexts []facade.SuiteContext
	goodModeSuite := len(reportableModedSuites) > 0
	reportPassed := true
	for _, testSuite := range reportableModedSuites {
		suiteCtx := facade.NewSuiteContext(token, isolation, testSuite, false, model.ReportAction, model.Config{}, false)
		// if suiteCtx.Config.TestSuite.IsEmpty() {
		// 	fmt.Printf("\n<<>> empty suite name in ctx !!! \nctx: %v ; \ncfg: %v\n", suiteCtx, suiteCtx.Config)
		// }

		suiteContexts = append(suiteContexts, suiteCtx)
		var code int16
		var suiteOutcome model.SuiteOutcome
		suiteOutcome, code, err = reportTestSuite(suiteCtx, true, all)
		if err != nil {
			// FIXME: aggregate errors
			exitCode = 1
			return
		}
		if code != 0 {
			exitCode = code
		}
		suiteOutcomes = append(suiteOutcomes, suiteOutcome)
		if suiteOutcome.Outcome != model.IGNORED && suiteOutcome.Outcome != model.PASSED && suiteOutcome.Outcome != model.EMPTY {
			reportPassed = false
		}
		err = suiteCtx.Close()
		if err != nil {
			return 1, err
		}
		// }
	}

	if len(reportableModedSuites) > 0 && !goodModeSuite {
		// No suites in supplied async mode => Nothing to report.
		return
	}

	Dpl.ReportSuites(suiteOutcomes)

	for _, suiteCtx := range suiteContexts {
		Dpl.CloseSuite(suiteCtx, fmt.Sprintf("following global report"))
	}

	if reportPassed {
		exitCode = 0
	}
	return
}

func ProcessGlobalReportDef(def model.ReportDefinition, asyncMode bool) (exitCode int16, err error) {
	ctx := facade.NewGlobalContext(def.Token, def.Isolation, model.Config{}, false)
	defer ctx.Close()
	exitCode, err = globalReport(ctx, asyncMode)
	return
}

func reportTestSuite(ctx facade.SuiteContext, global, all bool) (suiteOutcome model.SuiteOutcome, exitCode int16, err error) {
	exitCode = 1
	cfg := ctx.Config
	testSuite := cfg.TestSuite.Get()
	testCount := ctx.Repo.TestCount(testSuite)
	suiteIgnored := ctx.Config.IgnoreSuite.GetOr(false)
	logger.Info("Reporting suite", "suite", testSuite, "testCount", testCount)

	if !all && !suiteIgnored && testCount == 0 && !global {
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

	if suiteOutcome.Outcome == model.IGNORED || suiteOutcome.Outcome == model.PASSED || suiteOutcome.Outcome == model.EMPTY {
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
	ctx := facade.NewSuiteContext(def.Token, def.Isolation, def.TestSuite, false, model.ReportAction, def.Config, false) // FIXME ? removing def.Config ?
	defer ctx.Close()

	suiteOutcome, exitCode, err := reportTestSuite(ctx, false, false)
	if err != nil {
		return 1, err
	}

	if suiteOutcome.Duration < 0 {
		// FIXME: report should save an endTime for suite duration to be saved
		suiteOutcome.Duration = time.Since(ctx.Config.SuiteStartTime.Get())
	}

	Dpl.ReportSuite(suiteOutcome)
	Dpl.CloseSuite(ctx, "following report")

	return
}

func performTest(testDef model.TestDefinition, ctx facade.TestContext) (exitCode int16, err error) {
	logger.Debug("Performing test")
	exitCode = 1
	cfg := testDef.Config
	seq := testDef.Seq
	// fmt.Printf("\n<<>> display: opening test\n")

	td := Dpl.OpenTest(ctx)
	defer Dpl.CloseTest(ctx)
	td.Title()
	// fmt.Printf("\n<<>> display: printed test\n")

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

func ProcessTestDef(testDef model.TestDefinition, pooledRepo bool) (exitCode int16) {
	testSuite := testDef.TestSuite
	testCfg := testDef.Config
	testCtx, err := facade.NewTestContext2(testDef, pooledRepo)
	ProcessTestError(testCtx, err)
	defer testCtx.Close()

	// Firstly check if test not already performed (to fix async restart bug which may want to redo a test already performed but not done)
	testOc, err := testCtx.Repo.LoadTestOutcome(testDef.TestSignature)
	ProcessTestError(testCtx, err)
	if testOc != nil {
		// Test already performed
		return testOc.ExitCode
	}

	Dpl.Quiet(testCfg.Quiet.Is(true))

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
		exitCode, err = performTest(testDef, testCtx)

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

func ProcessMalDefinedTest(testDef model.TestDefinition, testCtx facade.TestContext, malDefErr error) (exitCode int16) {
	// testCtx, err := facade.NewTestContext2(testDef)
	// if err != nil {
	// 	// TODO
	// 	panic(err)
	// }
	// defer testCtx.Close()

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

func ProcessArgs(allArgs []string) (wait func() int16) {
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
		envCtx := facade.NewGlobalContext(envToken, isol, defaultCfg, false)
		defer envCtx.Close()
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
		exitCode, err = globalAction(token, isolation, inputConfig, parseArgsErrors)

	case model.InitAction:
		exitCode, err = initAction(token, isolation, inputConfig, parseArgsErrors)

	case model.ReportAction:
		// Report can be async (run by daemon) or not
		// Report can wait (for termination) or not
		// Report must always be delayed until all tests are done

		if inputConfig.GlobalReport.Is(true) {
			exitCode, wait, err = syncReportAllAction(token, isolation, inputConfig, parseArgsErrors)
		} else {
			exitCode, wait, err = reportSuiteAction(token, isolation, inputConfig, parseArgsErrors)
		}

	case model.TestAction:
		exitCode, wait, err = testAction(token, isolation, inputConfig, parseArgsErrors, signifientArgs)

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
