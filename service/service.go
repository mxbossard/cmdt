package service

import (
	"fmt"
	"log"
	"log/slog"
	"os"
	"path/filepath"

	"mby.fr/cmdtest/display"
	"mby.fr/cmdtest/facade"
	"mby.fr/cmdtest/model"
	"mby.fr/cmdtest/utils"
	"mby.fr/utils/cmdz"
	"mby.fr/utils/printz"
	"mby.fr/utils/utilz"
)

var dpl = display.New()

var logger = slog.New(slog.NewTextHandler(os.Stderr, model.DefaultLoggerOpts))

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

/*
func RulePrefix() string {
	return rulePrefix
}

func SetRulePrefix(prefix string) {
	if prefix != "" {
		rulePrefix = prefix
	}
}
*/
/*
func readEnvToken() (token string) {
	// Search uniqKey in env
	for _, env := range os.Environ() {
		if strings.HasPrefix(env, model.ContextTokenEnvVarName+"=") {
			splitted := strings.Split(env, "=")
			token = strings.Join(splitted[1:], "")
		}
	}
	logger.Debug("Found a token in env: " + token)
	return
}
*/

/*
func updateGlobalConfig0(ctx model.Context) (err error) {
	token := ctx.Token
	ctx.TestSuite = model.GlobalConfigTestSuiteName
	ctx.StartTime = time.Time{}
	var prev model.Context
	prev, err = LoadGlobalContext(token)
	if err != nil {
		return
	}
	ctx = model.MergeContext(prev, ctx)
	log.Printf("Updating global with: %s\n", ctx)
	err = PersistSuiteContext(ctx)
	return
}

func initConfig0(ctx model.Context) (ok bool, err error) {
	ok = false
	token := ctx.Token
	testSuite := ctx.TestSuite

	var contextFilepath string
	contextFilepath, err = utils.TestsuiteConfigFilepath(testSuite, token)
	if err != nil {
		return
	}
	_, err = os.Stat(contextFilepath)
	if err == nil {
		// Config already initialized
		return
	} else if !errors.Is(err, os.ErrNotExist) {
		return
	} else {
		err = nil
		ctx.StartTime = time.Now()
		ok = true
	}

	// store config
	PersistSuiteContext(ctx)
	logger.Debug("Initialized new config", "token", token, "suite", testSuite)
	return
}
*/

func GlobalConfig(ctx facade.GlobalContext, update bool) (exitCode int, err error) {
	// Init or Update Global config
	// FIXME: do we need to use update bool ?
	exitCode = 0
	err = ctx.Save()
	return
	/*
		ctx.TestSuite = model.GlobalConfigTestSuiteName
		err = initWorkspace(ctx)
		if err != nil {
			return
		}
		var ok bool
		ok, err = initConfig(ctx)
		if update && err == nil && !ok {
			err = updateGlobalConfig(ctx)
		}
		return
	*/
}

func InitTestSuite(ctx facade.SuiteContext) (exitCode int, err error) {
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

	/*
		exitCode, err = GlobalConfig(model.Context{Token: token, Silent: ctx.Silent}, false)
		if err != nil {
			return
		}

		// Clear the test suite directory
		var tmpDir string
		tmpDir, err = utils.TestsuiteDirectoryPath(testSuite, token)
		if err != nil {
			return
		}
		err = os.RemoveAll(tmpDir)
		if err != nil {
			return
		}
		logger.Debug("Cleared test suite", "token", token, "suite", testSuite, "dir", tmpDir)

		err = initWorkspace(ctx)
		if err != nil {
			return
		}
		_, err = initConfig(ctx)
		if err != nil {
			return
		}
	*/

	dpl.Suite(ctx)
	return
}

func ReportAllTestSuites(ctx facade.GlobalContext) (exitCode int, err error) {
	token := ctx.Token

	var testSuites []string
	testSuites, err = ctx.Repo.ListTestSuites()
	if err != nil {
		return
	}

	logger.Info("Reporting all suites", "token", token, "suites", testSuites)
	if len(testSuites) > 0 {
		exitCode = 0
		logger.Debug("reporting found suites", "suites", testSuites)
		for _, testSuite := range testSuites {
			suiteCtx := facade.NewSuiteContext(token, testSuite, model.ReportAction, ctx.Config)
			code := 0
			code, err = ReportTestSuite(suiteCtx)
			if err != nil {
				return
			}
			if code != 0 {
				exitCode = code
			}
		}
		dpl.ReportAllFooter(ctx)
	}

	return
}

func ReportTestSuite(ctx facade.SuiteContext) (exitCode int, err error) {
	exitCode = 1
	cfg := ctx.Config
	logger.Info("Reporting suite", "ctx", ctx)

	var suiteOutcome model.SuiteOutcome
	suiteOutcome, err = ctx.Repo.LoadSuiteOutcome(cfg.TestSuite.Get())
	if err != nil {
		return
	}
	dpl.ReportSuite(ctx, suiteOutcome)

	if suiteOutcome.FailedCount == 0 && suiteOutcome.ErroredCount == 0 {
		exitCode = 0
	}

	err = ctx.Repo.ClearTestSuite(suiteOutcome.TestSuite)

	return
}

func PerformTest(ctx facade.TestContext, assertions []model.Assertion) (exitCode int, err error) {
	exitCode = 1
	cfg := ctx.Config

	seq := ctx.IncrementTestCount()

	dpl.TestTitle(ctx, seq)

	if cfg.Ignore.Is(true) {
		ctx.IncrementIgnoredCount()
		dpl.TestOutcome(ctx, ctx.IgnoredTestOutcome(seq))
		exitCode = 0
		return
	}

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

	outcome := ctx.AssertCmdExecBlocking(seq, assertions)

	dpl.TestOutcome(ctx, outcome)

	for _, after := range cfg.After {
		cmdAfter := cmdz.Cmd(after...)
		afterExit, afterErr := cmdAfter.BlockRun()
		// FIXME: what to do of before exit code or beforeErr ?
		_ = afterExit
		if afterErr != nil {
			err = fmt.Errorf("error running after cmd: [%s]: %w", cmdAfter.String(), afterErr)
		}
	}

	for _, asseriontResult := range outcome.AssertionResults {
		dpl.AssertionResult(asseriontResult)
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

func ProcessArgs(allArgs []string) (exitCode int) {
	exitCode = 1

	if len(allArgs) == 1 {
		usage()
		return
	}

	// FIXME: if token supplied by ENV should be retrieved FIRST to get token and load Global config

	args := allArgs[1:]
	logger.Debug("Processing cmdt args", "args", args)
	config, assertions, agg := ParseArgs(args)
	if config.Debug.IsPresent() {
		model.DefaultLoggerOpts.Level = slog.Level(config.Debug.Get()*4 - 4)
	}

	logger.Debug("Parsed args", "config", config, "assertions", assertions, "error", agg)
	token := config.Token.GetOr("")
	action := config.Action.Get()

	var err error
	switch action {
	case model.GlobalAction:
		if agg.GotError() {
			log.Fatal(agg)
		}
		globalCtx := facade.NewGlobalContext(token, config)
		logger.Debug("Forged context", "ctx", globalCtx)
		logger.Info("Processing global action", "token", token)
		exitCode, err = GlobalConfig(globalCtx, true)
	case model.InitAction:
		testSuite := config.TestSuite.Get()
		suiteCtx := facade.NewSuiteContext(token, testSuite, action, config)
		logger.Debug("Forged context", "ctx", suiteCtx)
		suiteCtx.NoErrorOrFatal(agg.Return())
		logger.Info("Processing init action", "token", token)
		exitCode, err = InitTestSuite(suiteCtx)
	case model.ReportAction:
		if config.ReportAll.Is(true) {
			if agg.GotError() {
				log.Fatal(agg)
			}
			globalCtx := facade.NewGlobalContext(token, config)
			logger.Debug("Forged context", "ctx", globalCtx)
			logger.Info("Processing report all action", "token", token)
			exitCode, err = ReportAllTestSuites(globalCtx)
		} else {
			testSuite := config.TestSuite.Get()
			suiteCtx := facade.NewSuiteContext(token, testSuite, action, config)
			logger.Debug("Forged context", "ctx", suiteCtx)
			suiteCtx.NoErrorOrFatal(agg.Return())
			logger.Info("Processing report suite action", "token", token)
			exitCode, err = ReportTestSuite(suiteCtx)
		}
	case model.TestAction:
		testSuite := config.TestSuite.Get()
		testCtx := facade.NewTestContext(token, testSuite, config)
		logger.Debug("Forged context", "ctx", testCtx)
		testCtx.NoErrorOrFatal(agg.Return())
		logger.Info("Processing test action", "token", token)

		if config.ContainerDisabled.Is(true) || config.ContainerImage.IsEmpty() {
			//logger.Debug("Performing test outside container", "context", config, "image", config.ContainerImage, "containerDisabled", config.ContainerDisabled)
			exitCode, err = PerformTest(testCtx, assertions)
			testCtx.NoErrorOrFatal(err)
		} else {
			logger.Info("Performing test inside container", "context", config, "image", config.ContainerImage, "containerDisabled", config.ContainerDisabled)
			var ctId string
			ctId, exitCode, err = PerformTestInContainer(testCtx)
			if config.ContainerScope.Is(model.GLOBAL_SCOPE) {
				globalCfg, err2 := testCtx.Repo.LoadGlobalConfig()
				testCtx.NoErrorOrFatal(err2)
				globalCfg.ContainerId = utilz.OptionalOf(ctId)
				err2 = testCtx.Repo.SaveGlobalConfig(globalCfg)
				testCtx.NoErrorOrFatal(err2)
			} else if config.ContainerScope.Is(model.SUITE_SCOPE) {
				suiteCfg, err2 := testCtx.Repo.LoadSuiteConfig(testSuite)
				testCtx.NoErrorOrFatal(err2)
				suiteCfg.ContainerId = utilz.OptionalOf(ctId)
				err2 = testCtx.Repo.SaveSuiteConfig(suiteCfg)
				testCtx.NoErrorOrFatal(err2)
			}
			testCtx.NoErrorOrFatal(err)

		}
	default:
		err = fmt.Errorf("action: [%v] not known", config.Action)
	}

	logger.Info("exiting", "exitCode", exitCode)

	if err != nil {
		log.Fatal(config.TestSuite, config.Token, err)
	}
	return
}
