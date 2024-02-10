package main

import (
	"context"
	cryptorand "crypto/rand"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
	"mby.fr/utils/cmdz"
	"mby.fr/utils/printz"
	"mby.fr/utils/trust"
)

var rulePrefix = DefaultRulePrefix

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
		if strings.HasPrefix(env, ContextTokenEnvVarName+"=") {
			splitted := strings.Split(env, "=")
			token = strings.Join(splitted[1:], "")
		}
	}
	//log.Printf("Found a token in env: %s", token)
	return
}

func forgeContextualToken() string {
	// If no token supplied use Workspace dir + ppid to forge tmp directory path
	workDirPath, err := os.Getwd()
	if err != nil {
		log.Fatalf("cannot find workspace dir: %s", err)
	}
	ppid := fmt.Sprintf("%d", os.Getppid())
	token, err := trust.SignStrings(workDirPath, "--", ppid)
	if err != nil {
		log.Fatalf("cannot hash workspace dir: %s", err)
	}
	return token
}

func forgeTmpDirectoryPath(token string) (tempDirName string) {
	if token == "" {
		token = forgeContextualToken()
	}
	tempDirName = fmt.Sprintf("%s-%s", TempDirPrefix, token)
	tempDirPath := filepath.Join(os.TempDir(), tempDirName)
	err := os.MkdirAll(tempDirPath, 0700)
	if err != nil {
		log.Fatal(err)
	}
	return tempDirPath
}

func testsuiteDirectoryPath(testSuite, token string) string {
	tmpDir := forgeTmpDirectoryPath(token)
	suiteDir := sanitizeTestSuiteName(testSuite)
	path := filepath.Join(tmpDir, suiteDir)
	//log.Printf("testsuiteDir: %s\n", path)
	err := os.MkdirAll(path, 0700)
	if err != nil {
		log.Fatal(err)
	}
	return path
}

func listTestSuites(token string) (suites []string) {
	tmpDir := forgeTmpDirectoryPath(token)
	_, err := os.Stat(tmpDir)
	if os.IsNotExist(err) {
		return
	}

	matches, err := filepath.Glob(tmpDir + "/*")
	if err != nil {
		log.Fatalf("cannot list test suites: %s", err)
	}
	for _, m := range matches {
		suites = append(suites, filepath.Base(m))
	}
	return
}

func testConfigFilepath(testSuite, token string) string {
	return filepath.Join(testsuiteDirectoryPath(testSuite, token), ContextFilename)
}

func cmdLogFiles(testSuite, token string, seq int) (*os.File, *os.File, *os.File) {
	tmpDir := testsuiteDirectoryPath(testSuite, token)
	testDir := filepath.Join(tmpDir, "test-"+fmt.Sprintf("%06d", seq))
	stdoutFilepath := filepath.Join(testDir, StdoutFilename)
	stderrFilepath := filepath.Join(testDir, StderrFilename)
	reportFilepath := filepath.Join(testDir, ReportFilename)

	err := os.MkdirAll(testDir, 0700)
	if err != nil {
		log.Fatalf("cannot create work dir: %s ! Error: %s", testDir, err)
	}

	stdoutFile, err := os.OpenFile(stdoutFilepath, os.O_RDWR|os.O_CREATE, 0600)
	if err != nil {
		log.Fatalf("Cannot open file: %s ! Error: %s", stdoutFilepath, err)
	}
	stderrFile, err := os.OpenFile(stderrFilepath, os.O_RDWR|os.O_CREATE, 0600)
	if err != nil {
		log.Fatalf("Cannot open file: %s ! Error: %s", stderrFilepath, err)
	}
	reportFile, err := os.OpenFile(reportFilepath, os.O_RDWR|os.O_CREATE, 0600)
	if err != nil {
		log.Fatalf("Cannot open file: %s ! Error: %s", reportFilepath, err)
	}

	return stdoutFile, stderrFile, reportFile
}

func failureReports(testSuite, token string) (reports []string) {
	tmpDir := testsuiteDirectoryPath(testSuite, token)
	err := filepath.Walk(tmpDir, func(path string, info fs.FileInfo, err error) error {
		if ReportFilename == info.Name() {
			content, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			reports = append(reports, string(content))
		}
		return nil
	})
	if err != nil {
		log.Fatal(err)
	}

	return
}

func sanitizeTestSuiteName(s string) string {
	return testSuiteNameSanitizerPattern.ReplaceAllString(s, "_")
}

func PersistSuiteContext(testSuite, token string, config Context) {
	contextFilepath := testConfigFilepath(testSuite, token)
	//stdPrinter.Errf("Built context: %v\n", context)
	content, err := yaml.Marshal(config)
	if err != nil {
		log.Fatal(err)
	}
	//stdPrinter.Errf("Persisting context: %s\n", content)
	err = os.WriteFile(contextFilepath, content, 0600)
	if err != nil {
		log.Fatalf("cannot persist context: %s", err)
	}
}

func LoadSuiteContext(testSuite, token string) (config Context, err error) {
	contextFilepath := testConfigFilepath(testSuite, token)
	var content []byte
	content, err = os.ReadFile(contextFilepath)
	if err != nil {
		return
	}
	err = yaml.Unmarshal(content, &config)
	if err != nil {
		log.Fatal(err)
	}

	SetRulePrefix(config.Prefix)
	return
}

func UniqToken() string {
	b := make([]byte, 16)
	_, err := cryptorand.Read(b)
	if err != nil {
		log.Fatal(err)
	}
	uuid := fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:])
	return uuid

	/*
		h, err := trust.SignStrings(uuid)
		if err != nil {
			log.Fatalf("cannot forge a uniq token: %s", err)
		}
		return h
	*/
}

func InitWorkspace0(ctx Context) {
	token := ctx.Token
	testSuite := ctx.TestSuite

	// init the tmp directory
	tmpDir := testsuiteDirectoryPath(testSuite, token)
	err := os.MkdirAll(tmpDir, 0700)
	if err != nil {
		log.Fatalf("Unable to create temp dir: %s ! Error: %s", tmpDir, err)
	}

	stdPrinter.ColoredErrf(messageColor, "Initialized new [%s] workspace.\n", token)
	stdPrinter.Flush()
}

func GlobalConfig(ctx Context) (exitCode int) {
	// TODO: persist global config outside a test suite
	// TDOD: merge global config on testsuite loadingg config
	return
}

func InitTestSuite(ctx Context) (exitCode int) {
	exitCode = 0
	token := ctx.Token
	testSuite := ctx.TestSuite

	if ctx.Action == "init" && ctx.PrintToken {
		token = UniqToken()
		fmt.Printf("%s\n", token)
		ctx.Token = token
	} else if ctx.Action == "init" && ctx.ExportToken {
		token = UniqToken()
		fmt.Printf("export %s=%s\n", ContextTokenEnvVarName, token)
		ctx.Token = token
	}

	// init the tmp directory
	tmpDir := testsuiteDirectoryPath(testSuite, token)
	err := os.RemoveAll(tmpDir)
	if err != nil {
		log.Fatalf("Unable to erase temp dir (%s): %s", tmpDir, err)
	}
	err = os.MkdirAll(tmpDir, 0700)
	if err != nil {
		log.Fatalf("unable to create temp dir (%s): %s", tmpDir, err)
	}
	ctx.StartTime = time.Now()
	// store config
	PersistSuiteContext(testSuite, token, ctx)
	InitSeq(testSuite, token)
	// print export the key

	var tokenMsg = ""
	if token != "" {
		tokenMsg = fmt.Sprintf(" (token: %s)", token)
	}
	stdPrinter.ColoredErrf(messageColor, "Initialized new [%s] test suite%s.\n", testSuite, tokenMsg)
	//stdPrinter.Errf("%s\n", tmpDir)
	//stdPrinter.Errf("%s\n", context)
	stdPrinter.Flush()
	return
}

func ReportTestSuite(ctx Context) (exitCode int) {
	exitCode = 1
	token := ctx.Token
	testSuite := ctx.TestSuite

	if ctx.ReportAll {
		// Report all test suites
		testSuites := listTestSuites(token)
		if testSuites != nil {
			log.Printf("reporting found suites: %s", testSuites)
			for _, suite := range testSuites {
				ctx, err := LoadSuiteContext(suite, token)
				if err != nil {
					log.Fatalf("cannot load context: %s", err)
				}
				exitCode = ReportTestSuite(ctx)
				if exitCode > 0 {
					return
				}
			}
			return
		}
	}

	tmpDir := testsuiteDirectoryPath(testSuite, token)
	defer os.RemoveAll(tmpDir)

	suiteContext, err := LoadSuiteContext(testSuite, token)
	if err != nil {
		if os.IsNotExist(err) {
			log.Fatalf("you must perform some test prior to report: [%s] test suite", testSuite)
		} else {
			log.Fatalf("cannot load context: %s", err)
		}
	}
	ctx = MergeContext(suiteContext, ctx)
	testCount := ReadSeq(testSuite, token, TestSequenceFilename)
	ignoredCount := ReadSeq(testSuite, token, IgnoredSequenceFilename)
	failureReports := failureReports(testSuite, token)
	failedCount := len(failureReports)

	stdPrinter.ColoredErrf(messageColor, "Reporting [%s] test suite ...\n", testSuite)

	ignoredMessage := ""
	if ignoredCount > 0 {
		ignoredMessage = fmt.Sprintf(" (%d ignored)", ignoredCount)
	}
	if failedCount == 0 {
		exitCode = 0
		stdPrinter.ColoredErrf(successColor, "Successfuly ran %s test suite (%d tests in %s)", testSuite, testCount, time.Since(ctx.StartTime))
		stdPrinter.ColoredErrf(warningColor, "%s", ignoredMessage)
		stdPrinter.Errf("\n")
	} else {
		successCount := testCount - failedCount
		stdPrinter.ColoredErrf(failureColor, "Failures in %s test suite (%d success, %d failures, %d tests in %s)", testSuite, successCount, failedCount, testCount, time.Since(ctx.StartTime))
		stdPrinter.ColoredErrf(warningColor, "%s", ignoredMessage)
		stdPrinter.Errf("\n")
		for _, report := range failureReports {
			stdPrinter.ColoredErrf(reportColor, "%s\n", report)
		}
	}
	stdPrinter.Flush()
	return
}

func PerformTest(ctx Context, cmdAndArgs []string, assertions []Assertion) (exitCode int) {
	token := ctx.Token
	testSuite := ctx.TestSuite
	testName := ctx.TestName
	exitCode = 1

	suiteContext, err := LoadSuiteContext(testSuite, token)
	if err != nil {
		if os.IsNotExist(err) {
			// test suite does not exists yet
			exitCode = InitTestSuite(ctx)
			if exitCode > 0 {
				return
			}
			// Recursive call once test suite initialized
			return PerformTest(ctx, cmdAndArgs, assertions)
		} else {
			log.Fatalf("cannot load context: %s", err)
		}
	}

	ctx = MergeContext(suiteContext, ctx)
	timecode := int(time.Since(ctx.StartTime).Milliseconds())

	if len(cmdAndArgs) == 0 {
		log.Fatalf("no command supplied to test")
	}
	cmd := cmdz.Cmd(cmdAndArgs[0])
	if len(cmdAndArgs) > 1 {
		cmd.AddArgs(cmdAndArgs[1:]...)
	}

	if ctx.Timeout.Milliseconds() > 0 {
		cmd.Timeout(ctx.Timeout)
	}

	if testName == "" {
		cmdNameParts := strings.Split(cmd.String(), " ")
		shortenedCmd := filepath.Base(cmdNameParts[0])
		shortenCmdNameParts := cmdNameParts
		shortenCmdNameParts[0] = shortenedCmd
		cmdName := strings.Join(shortenCmdNameParts, " ")
		//testName = fmt.Sprintf("cmd: <|%s|>", cmdName)
		testName = fmt.Sprintf("<|%s|>", cmdName)
	}

	qulifiedName := testName
	if testSuite != "" {
		qulifiedName = fmt.Sprintf("[%s]/%s", testSuite, testName)
	}
	if ctx.Ignore != nil && *ctx.Ignore {
		stdPrinter.ColoredErrf(warningColor, "[%05d] Ignored test: %s\n", timecode, qulifiedName)
		IncrementSeq(testSuite, token, IgnoredSequenceFilename)
		return 0
	}
	seq := IncrementSeq(testSuite, token, TestSequenceFilename)
	stdoutLog, stderrLog, reportLog := cmdLogFiles(testSuite, token, seq)
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

	//fmt.Printf("%s", os.Environ())
	cmd.AddEnviron(os.Environ())

	testTitle := fmt.Sprintf("[%05d] Test %s #%02d", timecode, qulifiedName, seq)
	stdPrinter.ColoredErrf(testColor, "%s... ", testTitle)

	if *ctx.KeepStdout || *ctx.KeepStderr {
		// NewLine because we expect cmd outputs
		stdPrinter.Errf("\n")
	}

	stdPrinter.Flush()

	_, err = cmd.BlockRun()
	var failedResults []AssertionResult
	var testDuration time.Duration
	exitCode = 1

	if err == nil {
		exitCode = 0
		testDuration = cmd.Duration()
		for _, assertion := range assertions {
			var result AssertionResult
			result, err = assertion.Asserter(cmd)
			result.Assertion = assertion
			if err != nil {
				//log.Fatal(err)
				//stdPrinter.ColoredErrf(errorColor, "FAILED (error: %s) ", err)
				result.Message = fmt.Sprintf("%s", err)
				result.Success = false
			}
			if !result.Success {
				failedResults = append(failedResults, result)
				exitCode = 1
			}
		}
	}

	if *ctx.KeepStdout || *ctx.KeepStderr {
		// NewLine in printer to print test result in a new line
		stdPrinter.Errf("        ")
	}

	if exitCode == 0 {
		stdPrinter.ColoredErrf(successColor, "PASSED")
		stdPrinter.Errf(" (in %s)\n", testDuration)
		defer os.Remove(reportLog.Name())
	} else {
		stdPrinter.ColoredErrf(failureColor, "FAILED")
		if err == nil {
			stdPrinter.Errf(" (in %s)\n", testDuration)
		} else {
			if errors.Is(err, context.DeadlineExceeded) {
				stdPrinter.Errf(" (timed out after %s)\n", ctx.Timeout)
				reportLog.WriteString(testTitle + "  =>  timed out")
			} else {
				stdPrinter.Errf(" (not executed)\n")
				reportLog.WriteString(testTitle + "  =>  not executed")
			}
		}
		stdPrinter.Errf("Failure calling cmd: <|%s|>\n", cmd)
		if err != nil {
			stdPrinter.ColoredErrf(errorColor, "%s\n", err)
		} else {
			for _, result := range failedResults {
				//log.Printf("failedResult: %v\n", result)
				assertPrefix := result.Assertion.Prefix
				assertName := result.Assertion.Name
				assertOp := result.Assertion.Op
				expected := result.Assertion.Expected
				got := result.Value

				if result.Message != "" {
					stdPrinter.ColoredErrf(errorColor, result.Message+"\n")
				}

				if assertName == "success" || assertName == "fail" {
					stdPrinter.Errf("Expected %s%s\n", assertPrefix, assertName)
					continue
				} else if assertName == "cmd" {
					stdPrinter.Errf("Expected %s%s=%s to succeed\n", assertPrefix, assertName, expected)
					continue
				} else if assertName == "exists" {
					stdPrinter.Errf("Expected file %s%s=%s file to exists\n", assertPrefix, assertName, expected)
					continue
				}

				if expected != got {
					if assertOp == "=" {
						stdPrinter.Errf("Expected %s%s to be: [%s] but got: [%v]\n", assertPrefix, assertName, expected, got)
					} else if assertOp == ":" {
						stdPrinter.Errf("Expected %s%s to contains: [%s] but got: [%v]\n", assertPrefix, assertName, expected, got)
					} else if assertOp == "~" {
						stdPrinter.Errf("Expected %s%s to match: [%s] but got: [%v]\n", assertPrefix, assertName, expected, got)
					}
				} else {
					stdPrinter.Errf("assertion %s%s%s%s failed\n", assertPrefix, assertName, assertOp, expected)
				}
			}
			failedAssertionsReport := ""
			for _, result := range failedResults {
				assertName := result.Assertion.Name
				assertOp := result.Assertion.Op
				expected := result.Assertion.Expected
				failedAssertionsReport = RulePrefix() + string(assertName) + string(assertOp) + string(expected)
			}
			reportLog.WriteString(testTitle + "  => " + failedAssertionsReport)
		}
	}

	stdPrinter.Flush()

	if ctx.StopOnFailure == nil || *ctx.StopOnFailure && exitCode > 0 {
		ReportTestSuite(ctx)
	}

	if ctx.StopOnFailure == nil || !*ctx.StopOnFailure {
		// FIXME: Do not return a success to let test continue
		exitCode = 0
	}
	return
}

func ProcessArgs(allArgs []string) {
	exitCode := 1
	defer func() { os.Exit(exitCode) }()

	stdPrinter = printz.NewStandard()
	defer stdPrinter.Flush()

	if len(allArgs) == 1 {
		usage()
		return
	}

	config, cmdAndArgs, assertions, err := ParseArgs(allArgs[1:])
	if err != nil {
		log.Fatal(err)
	}

	if config.Token == "" {
		config.Token = readEnvToken()
	}

	switch config.Action {
	case "global":
		exitCode = GlobalConfig(config)
	case "init":
		exitCode = InitTestSuite(config)
	case "test":
		exitCode = PerformTest(config, cmdAndArgs, assertions)
	case "report":
		exitCode = ReportTestSuite(config)
	default:
		log.Fatalf("action: [%s] not known", config.Action)
	}
}
