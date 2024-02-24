package service

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
	"mby.fr/cmdtest/display"
	"mby.fr/cmdtest/model"
	"mby.fr/cmdtest/utils"
	"mby.fr/utils/cmdz"
	"mby.fr/utils/printz"
)

var rulePrefix = model.DefaultRulePrefix

var dpl = display.New()

var logger = slog.New(slog.NewTextHandler(os.Stderr, nil))

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

func RulePrefix() string {
	return rulePrefix
}

func SetRulePrefix(prefix string) {
	if prefix != "" {
		rulePrefix = prefix
	}
}

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

func cmdLogFiles(testSuite, token string, seq int) (stdoutFile, stderrFile, reportFile *os.File, err error) {
	var testDir string
	testDir, err = utils.TestDirectoryPath(testSuite, token, seq)
	if err != nil {
		return
	}
	stdoutFilepath := filepath.Join(testDir, model.StdoutFilename)
	stderrFilepath := filepath.Join(testDir, model.StderrFilename)
	reportFilepath := filepath.Join(testDir, model.ReportFilename)

	err = os.MkdirAll(testDir, 0700)
	if err != nil {
		err = fmt.Errorf("cannot create work dir %s : %w", testDir, err)
		return
	}
	stdoutFile, err = os.OpenFile(stdoutFilepath, os.O_APPEND|os.O_RDWR|os.O_CREATE, 0600)
	if err != nil {
		err = fmt.Errorf("cannot open file %s : %w", stdoutFilepath, err)
		return
	}
	stderrFile, err = os.OpenFile(stderrFilepath, os.O_APPEND|os.O_RDWR|os.O_CREATE, 0600)
	if err != nil {
		err = fmt.Errorf("cannot open file %s : %w", stderrFilepath, err)
		return
	}
	reportFile, err = os.OpenFile(reportFilepath, os.O_APPEND|os.O_RDWR|os.O_CREATE, 0600)
	if err != nil {
		err = fmt.Errorf("cannot open file %s : %w", reportFilepath, err)
		return
	}
	return
}

func FailureReports(testSuite, token string) (reports []string, err error) {
	var tmpDir string
	tmpDir, err = utils.TestsuiteDirectoryPath(testSuite, token)
	if err != nil {
		return
	}
	err = filepath.Walk(tmpDir, func(path string, info fs.FileInfo, err error) error {
		if model.ReportFilename == info.Name() {
			content, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			reports = append(reports, string(content))
		}
		return nil
	})
	return
}

func PersistSuiteContext0(testSuite, token string, config model.Context) (err error) {
	var contextFilepath string
	contextFilepath, err = utils.TestsuiteConfigFilepath(testSuite, token)
	if err != nil {
		return
	}
	content, err := yaml.Marshal(config)
	if err != nil {
		return
	}
	logger.Debug("Persisting context", "context", content, "file", contextFilepath)
	err = os.WriteFile(contextFilepath, content, 0600)
	if err != nil {
		err = fmt.Errorf("cannot persist context: %w", err)
		return
	}
	return
}

func PersistSuiteContext(config model.Context) (err error) {
	testSuite := config.TestSuite
	token := config.Token
	var contextFilepath string
	contextFilepath, err = utils.TestsuiteConfigFilepath(testSuite, token)
	if err != nil {
		return
	}
	content, err := yaml.Marshal(config)
	if err != nil {
		return
	}
	logger.Debug("Persisting context", "context", content, "file", contextFilepath)
	err = os.WriteFile(contextFilepath, content, 0600)
	if err != nil {
		err = fmt.Errorf("cannot persist context: %w", err)
		return
	}
	return
}

func updateLastTestTime(testSuite, token string) {
	ctx, err := LoadSuiteContext(testSuite, token)
	if err != nil {
		log.Fatal(err)
	}
	ctx.LastTestTime = time.Now()
	err = PersistSuiteContext(ctx)
	if err != nil {
		log.Fatal(err)
	}
}

func initWorkspace(ctx model.Context) (err error) {
	token := ctx.Token
	testSuite := ctx.TestSuite

	// init the tmp directory
	var tmpDir string
	tmpDir, err = utils.TestsuiteDirectoryPath(testSuite, token)
	if err != nil {
		return
	}
	_, err = os.Stat(tmpDir)
	if err == nil {
		// Workspace already initialized
		return
	} else if !errors.Is(err, os.ErrNotExist) {
		return
	}

	err = os.MkdirAll(tmpDir, 0700)
	if err != nil {
		err = fmt.Errorf("unable to create temp dir: %s ! Error: %w", tmpDir, err)
		return
	}

	return
}

func LoadSuiteContext(testSuite, token string) (config model.Context, err error) {
	var globalCtx, suiteCtx model.Context
	globalCtx, err = LoadGlobalContext(token)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return
	}
	suiteCtx, err = utils.ReadSuiteContext(testSuite, token)
	if err != nil {
		return
	}
	logger.Debug("Loaded context", "global", globalCtx, "suite", suiteCtx)
	config = model.MergeContext(globalCtx, suiteCtx)
	SetRulePrefix(config.Prefix)
	logger.Debug("Merges context", "merged", config)
	return
}

func LoadGlobalContext(token string) (config model.Context, err error) {
	config, err = utils.ReadSuiteContext(model.GlobalConfigTestSuiteName, token)
	config.TestSuite = ""
	return
}

func updateGlobalConfig(ctx model.Context) (err error) {
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

func initConfig(ctx model.Context) (ok bool, err error) {
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

func GlobalConfig(ctx model.Context, update bool) (exitCode int, err error) {
	exitCode = 0
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
}

func InitTestSuite(ctx model.Context) (exitCode int, err error) {
	exitCode = 0
	token := ctx.Token
	testSuite := ctx.TestSuite

	if ctx.Action == "init" && ctx.PrintToken {
		token, err = utils.ForgeUuid()
		if err != nil {
			return
		}
		fmt.Printf("%s\n", token)
		ctx.Token = token
	} else if ctx.Action == "init" && ctx.ExportToken {
		token, err = utils.ForgeUuid()
		if err != nil {
			return
		}
		fmt.Printf("export %s=%s\n", model.ContextTokenEnvVarName, token)
		ctx.Token = token
	}

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

	dpl.Suite(ctx)
	return
}

func ReportTestSuite(ctx model.Context) (exitCode int, err error) {
	exitCode = 1
	token := ctx.Token
	testSuite := ctx.TestSuite

	if ctx.ReportAll {
		// Report all test suites
		var testSuites []string
		testSuites, err = utils.ListTestSuites(token)
		if err != nil {
			return
		}
		if len(testSuites) > 0 {
			exitCode = 0
			logger.Debug("reporting found suites", "suites", testSuites)
			for _, suite := range testSuites {
				ctx, err = LoadSuiteContext(suite, token)
				if err != nil {
					err = fmt.Errorf("cannot load context: %w", err)
					return
				}
				var code int
				code, err = ReportTestSuite(ctx)
				if err != nil {
					return
				}
				if code != 0 {
					exitCode = code
				}
			}
			var global model.Context
			global, err = LoadGlobalContext(token)
			if err != nil {
				err = fmt.Errorf("cannot load global context: %s", err)
				return
			}
			dpl.ReportAllFooter(global)
			return
		}
		// Continue to return errors
	}

	var tmpDir string
	tmpDir, err = utils.TestsuiteDirectoryPath(testSuite, token)
	if err != nil {
		return
	}
	defer os.RemoveAll(tmpDir)

	suiteContext, err := LoadSuiteContext(testSuite, token)
	if err != nil {
		if os.IsNotExist(err) {
			err = fmt.Errorf("you must perform some test prior to report: [%s] test suite", testSuite)
			return
		} else {
			err = fmt.Errorf("cannot load context: %w", err)
			return
		}
	}
	ctx = model.MergeContext(suiteContext, ctx)
	var failedReports []string
	failedReports, err = FailureReports(testSuite, token)
	if err != nil {
		return
	}
	dpl.ReportSuite(ctx, tmpDir, failedReports)

	failedCount := utils.ReadSeq(tmpDir, model.FailureSequenceFilename)
	errorCount := utils.ReadSeq(tmpDir, model.ErrorSequenceFilename)
	if failedCount == 0 && errorCount == 0 {
		exitCode = 0
	}

	return
}

func PerformTest(ctx model.Context, cmdAndArgs []string, assertions []model.Assertion) (exitCode int, err error) {
	token := ctx.Token
	testSuite := ctx.TestSuite
	testName := ctx.TestName
	exitCode = 1

	if len(cmdAndArgs) == 0 {
		err = fmt.Errorf("no command supplied to test")
		return
	}
	cmd := cmdz.Cmd(cmdAndArgs[0])
	if len(cmdAndArgs) > 1 {
		cmd.AddArgs(cmdAndArgs[1:]...)
	}

	if ctx.Timeout.Milliseconds() > 0 {
		cmd.Timeout(ctx.Timeout)
	}

	if testName == "" {
		// FIXME: move this in ctx ?
		cmdNameParts := strings.Split(cmd.String(), " ")
		shortenedCmd := filepath.Base(cmdNameParts[0])
		shortenCmdNameParts := cmdNameParts
		shortenCmdNameParts[0] = shortenedCmd
		cmdName := strings.Join(shortenCmdNameParts, " ")
		//testName = fmt.Sprintf("cmd: <|%s|>", cmdName)
		testName = fmt.Sprintf("<|%s|>", cmdName)
		ctx.TestName = testName
	}

	var tmpDir string
	tmpDir, err = utils.TestsuiteDirectoryPath(testSuite, token)
	if err != nil {
		return
	}

	seq := utils.IncrementSeq(tmpDir, model.TestSequenceFilename)

	dpl.TestTitle(ctx, seq)

	if ctx.Ignore != nil && *ctx.Ignore {
		utils.IncrementSeq(tmpDir, model.IgnoredSequenceFilename)
		dpl.TestOutcome(ctx, seq, "IGNORED", nil, 0, nil)
		exitCode = 0
		return
	}

	var stdoutLog, stderrLog, reportLog *os.File
	stdoutLog, stderrLog, reportLog, err = cmdLogFiles(testSuite, token, seq)
	if err != nil {
		return
	}
	defer stdoutLog.Close()
	defer stderrLog.Close()
	defer reportLog.Close()

	var stdout, stderr io.Writer
	stdout = stdoutLog
	if *ctx.KeepStdout {
		stdout = io.MultiWriter(os.Stdout, stdoutLog)
	}
	stderr = stdoutLog
	if *ctx.KeepStderr {
		stderr = io.MultiWriter(os.Stderr, stderrLog)
	}
	cmd.SetOutputs(stdout, stderr)

	stat, _ := os.Stdin.Stat()
	if (stat.Mode() & os.ModeCharDevice) == 0 {
		cmd.SetInput(os.Stdin)
	}

	for _, environ := range os.Environ() {
		if !strings.HasPrefix(environ, "PATH=") {
			cmd.AddEnviron(environ)
		}
	}
	currentPath := os.Getenv("PATH")
	if len(ctx.Mocks) > 0 {
		// Put mockDir in PATH
		var mockDir string
		mockDir, err = ProcessMocking(testSuite, token, seq, ctx.Mocks)
		if err != nil {
			return
		}
		cmd.AddEnv("ORIGINAL_PATH", currentPath)
		newPath := fmt.Sprintf("%s:%s", mockDir, currentPath)
		cmd.AddEnv("PATH", newPath)
		err = os.Setenv("PATH", newPath)
		if err != nil {
			return
		}
	} else {
		cmd.AddEnv("PATH", currentPath)
	}

	qualifiedName := testName
	if testSuite != "" {
		qualifiedName = fmt.Sprintf("[%s]/%s", testSuite, testName)
	}
	testTitle := fmt.Sprintf("Test %s #%02d", qualifiedName, seq)

	for _, before := range ctx.Before {
		cmdBefore := cmdz.Cmd(before...)
		beforeExit, beforeErr := cmdBefore.BlockRun()
		// FIXME: what to do of before exit code or beforeErr ?
		_ = beforeExit
		if beforeErr != nil {
			err = fmt.Errorf("error running before cmd: [%s]: %w", cmdBefore.String(), beforeErr)
			return
		}
	}

	_, err = cmd.BlockRun()
	var failedResults []model.AssertionResult
	var testDuration time.Duration
	exitCode = 1

	if err == nil {
		exitCode = 0
		testDuration = cmd.Duration()
		for _, assertion := range assertions {
			var result model.AssertionResult
			result, err = assertion.Asserter(cmd)
			result.Assertion = assertion
			if err != nil {
				// FIXME: aggregate errors
				result.Message += fmt.Sprintf("%s ", err)
				result.Success = false
			}
			if !result.Success {
				failedResults = append(failedResults, result)
				exitCode = 1
			}
		}
	}

	if exitCode == 0 {
		dpl.TestOutcome(ctx, seq, model.PASSED, cmd, testDuration, err)
		defer os.Remove(reportLog.Name())
	} else {
		if err == nil {
			dpl.TestOutcome(ctx, seq, model.FAILED, cmd, testDuration, err)
		} else {
			if errors.Is(err, context.DeadlineExceeded) {
				// Swallow error
				err = nil
				//IncrementSeq(tmpDir, FailureSequenceFilename)
				dpl.TestOutcome(ctx, seq, model.FAILED, cmd, testDuration, err)
				reportLog.WriteString(testTitle + "  =>  timed out")
			} else {
				err = nil
				// Swallow error
				utils.IncrementSeq(tmpDir, model.ErrorSequenceFilename)
				dpl.TestOutcome(ctx, seq, model.ERRORED, cmd, testDuration, err)
				reportLog.WriteString(testTitle + "  =>  not executed")
			}
		}
		if err == nil {
			for _, result := range failedResults {
				dpl.AssertionResult(cmd, result)
			}
			failedAssertionsReport := ""
			for _, result := range failedResults {
				assertName := result.Assertion.Name
				assertOp := result.Assertion.Op
				expected := result.Assertion.Expected
				failedAssertionsReport += RulePrefix() + string(assertName) + string(assertOp) + string(expected) + " "
			}
			utils.IncrementSeq(tmpDir, model.FailureSequenceFilename)
			reportLog.WriteString(testTitle + "  => " + failedAssertionsReport)
		}
	}

	for _, after := range ctx.After {
		cmdAfter := cmdz.Cmd(after...)
		afterExit, afterErr := cmdAfter.BlockRun()
		// FIXME: what to do of before exit code or beforeErr ?
		_ = afterExit
		if afterErr != nil {
			err = fmt.Errorf("error running after cmd: [%s]: %w", cmdAfter.String(), afterErr)
		}
	}

	if ctx.StopOnFailure == nil || *ctx.StopOnFailure && exitCode > 0 {
		ReportTestSuite(ctx)
	}

	if ctx.StopOnFailure == nil || !*ctx.StopOnFailure {
		// FIXME: Do not return a success to let test continue
		exitCode = 0
	}
	return
}

func NoErrorOrFatal(ctx model.Context, err error) {
	if err != nil {
		suiteContext, err2 := LoadSuiteContext(ctx.TestSuite, ctx.Token)
		if err2 == nil {
			updateLastTestTime(suiteContext.TestSuite, suiteContext.Token)
		}
		utils.Fatal(ctx.TestSuite, ctx.Token, err)
	}
}

func ProcessArgs(allArgs []string) {
	exitCode := 1
	defer func() { os.Exit(exitCode) }()

	if len(allArgs) == 1 {
		usage()
		return
	}

	config, cmdAndArgs, assertions, err := ParseArgs(allArgs[1:])
	if config.TestSuite == "" {
		config.TestSuite = model.DefaultTestSuiteName
	}
	if config.Token == "" {
		config.Token = readEnvToken()
	}
	if config.Token == "" {
		var err2 error
		config.Token, err2 = utils.ForgeContextualToken()
		NoErrorOrFatal(config, err2)
	}

	NoErrorOrFatal(config, err)

	switch config.Action {
	case "global":
		exitCode, err = GlobalConfig(config, true)
	case "init":
		exitCode, err = InitTestSuite(config)
	case "test":
		testSuite := config.TestSuite
		token := config.Token
		suiteContext, err := LoadSuiteContext(testSuite, token)
		if err != nil {
			if os.IsNotExist(err) {
				// test suite does not exists yet
				suiteCtx := model.Context{Token: token, TestSuite: testSuite}
				exitCode, err = InitTestSuite(suiteCtx)
				if err != nil {
					return
				}
				if exitCode > 0 {
					return
				}
			} else {
				err = fmt.Errorf("cannot load context: %w", err)
				NoErrorOrFatal(config, err)
			}
		}
		defer updateLastTestTime(testSuite, token)

		config = model.MergeContext(suiteContext, config)

		if (config.ContainerDisabled != nil && *config.ContainerDisabled) || config.ContainerImage == "" {
			//logger.Debug("Performing test outside container", "context", config, "image", config.ContainerImage, "containerDisabled", config.ContainerDisabled)
			exitCode, err = PerformTest(config, cmdAndArgs, assertions)
			NoErrorOrFatal(config, err)
		} else {
			logger.Debug("Performing test inside container", "context", config, "image", config.ContainerImage, "containerDisabled", config.ContainerDisabled)
			var ctId string
			ctId, exitCode, err = PerformTestInContainer(config)
			if config.ContainerScope != nil && *config.ContainerScope == model.Global {
				globalCtx, err2 := LoadGlobalContext(config.Token)
				NoErrorOrFatal(config, err2)
				globalCtx.ContainerId = &ctId
				err2 = PersistSuiteContext(globalCtx)
				NoErrorOrFatal(config, err2)
			} else if config.ContainerScope != nil && *config.ContainerScope == model.Suite {
				suiteCtx, err2 := LoadSuiteContext(config.TestSuite, config.Token)
				NoErrorOrFatal(config, err2)
				suiteCtx.ContainerId = &ctId
				err2 = PersistSuiteContext(suiteCtx)
				NoErrorOrFatal(config, err2)
			}
			NoErrorOrFatal(config, err)

		}
	case "report":
		exitCode, err = ReportTestSuite(config)
	default:
		err = fmt.Errorf("action: [%s] not known", config.Action)
	}

	if err != nil {
		utils.Fatal(config.TestSuite, config.Token, err)
	}
}

func mockDirectoryPath(testSuite, token string, seq int) (mockDir string, err error) {
	var testDir string
	testDir, err = utils.TestDirectoryPath(testSuite, token, seq)
	mockDir = filepath.Join(testDir, "mock")
	return
}

func ProcessMocking(testSuite, token string, seq int, mocks []model.CmdMock) (mockDir string, err error) {
	// get test dir
	// create a mock dir
	mockDir, err = mockDirectoryPath(testSuite, token, seq)
	if err != nil {
		return
	}
	err = os.MkdirAll(mockDir, 0755)
	if err != nil {
		return
	}
	wrapperFilepath := filepath.Join(mockDir, "mockWrapper.sh")
	// add mockdir to PATH TODO in perform test

	// write the mocke wrapper
	err = writeMockWrapperScript(wrapperFilepath, mocks)
	if err != nil {
		return
	}
	// for each cmd mocked add link to the mock wrapper
	for _, mock := range mocks {
		linkName := filepath.Join(mockDir, mock.Cmd)
		linkSource := wrapperFilepath
		err = os.Symlink(linkSource, linkName)
		if err != nil {
			return
		}
	}

	log.Printf("mock wrapper: %s\n", wrapperFilepath)
	return
}

func writeMockWrapperScript(wrapperFilepath string, mocks []model.CmdMock) (err error) {
	// By default run all args
	//wrapper.sh CMD ARG_1 ARG_2 ... ARG_N
	// Pour chaque CmdMock
	// if "$@" match CmdMock
	wrapperScript := "#! /bin/sh\nset -e\n"
	wrapperScript += `export PATH="$ORIGINAL_PATH"` + "\n"
	//wrapperScript += ">&2 echo PATH:$PATH\n"
	wrapperScript += `cmd=$( basename "$0" )` + "\n"

	for _, mock := range mocks {
		wrapperScript += fmt.Sprintf(`if [ "$cmd" = "%s" ]`, mock.Cmd)
		wildcard := false
		if mock.Op == "=" {
			// args must exactly match mock config
			for pos, arg := range mock.Args {
				if arg != "*" {
					wrapperScript += fmt.Sprintf(` && [ "$%d" = "%s" ] `, pos+1, arg)
				} else {
					wildcard = true
					break
				}
			}
		} else if mock.Op == ":" {
			// args must contains mock config disorderd
			// all mock args must be in $@
			// if multiple same mock args must all be present in $@
			mockArgsCount := make(map[string]int, 8)
			for _, arg := range mock.Args {
				if arg != "*" {
					mockArgsCount[arg]++
				} else {
					wildcard = true
				}
			}
			for arg, count := range mockArgsCount {
				wrapperScript += fmt.Sprintf(` && [ %d -eq $( echo "$@" | grep -c "%s" ) ]`, count, arg)
			}
		}
		if !wildcard {
			wrapperScript += fmt.Sprintf(` && [ "$#" -eq %d ] `, len(mock.Args))
		}

		wrapperScript += `; then` + "\n"
		if mock.Stdin != nil {
			wrapperScript += fmt.Sprintf("\t" + `stdin="$( cat )"` + "\n")
			wrapperScript += fmt.Sprintf("\t"+`if [ "$stdin" = "%s" ]; then`+"\n", *mock.Stdin)
		}

		// FIXME: add stdin management
		if mock.Stdout != "" {
			wrapperScript += fmt.Sprintf("\t"+`echo -n "%s"`+"\n", mock.Stdout)
		}
		if mock.Stderr != "" {
			wrapperScript += fmt.Sprintf("\t"+` >&2 echo -n "%s"`+"\n", mock.Stderr)
		}
		if len(mock.OnCallCmdAndArgs) > 0 {
			wrapperScript += fmt.Sprintf("\t"+`%s`+"\n", strings.Join(mock.OnCallCmdAndArgs, " "))
		}
		if !mock.Delegate {
			wrapperScript += fmt.Sprintf("\t"+`exit %d`+"\n", mock.ExitCode)
		}
		if mock.Stdin != nil {
			wrapperScript += fmt.Sprintf("\t" + `fi` + "\n")
		}
		wrapperScript += fmt.Sprintf(`fi` + "\n")
	}
	wrapperScript += `"$cmd" "$@"` + "\n"

	err = os.WriteFile(wrapperFilepath, []byte(wrapperScript), 0755)
	return
}
