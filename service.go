package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
	"mby.fr/utils/cmdz"
)

func PersistSuiteContext(testSuite, uniqKey string, config Context) {
	contextFilepath := testConfigFilepath(testSuite, uniqKey)
	//stdPrinter.Errf("Built context: %v\n", context)
	content, err := yaml.Marshal(config)
	if err != nil {
		log.Fatal(err)
	}
	//stdPrinter.Errf("Persisting context: %s\n", content)
	err = os.WriteFile(contextFilepath, content, 0600)
	if err != nil {
		log.Fatal(err)
	}
}

func LoadSuiteContext(testSuite, uniqKey string) (config Context, err error) {
	contextFilepath := testConfigFilepath(testSuite, uniqKey)
	content, err2 := os.ReadFile(contextFilepath)
	if err2 != nil {
		//log.Fatal(err)
		err = buildNoTestToReportError(testSuite)
		return
	}
	err2 = yaml.Unmarshal(content, &config)
	if err2 != nil {
		log.Fatal(err2)
	}
	return
}

func InitTestSuite(ctx Context) string {
	testSuite := ctx.TestSuite
	// forge a uniq key
	uniqKey := forgeUniqKey(testSuite)
	// init the tmp directory
	tmpDir := testsuiteDirectoryPath(testSuite, uniqKey)
	err := os.MkdirAll(tmpDir, 0700)
	if err != nil {
		log.Fatalf("Unable to create temp dir: %s ! Error: %s", tmpDir, err)
	}
	ctx.StartTime = time.Now()
	// store config
	PersistSuiteContext(testSuite, uniqKey, ctx)
	initSeq(testSuite, uniqKey)
	// print export the key
	fmt.Printf("export %s%s=%s\n", ContextEnvVarName, strings.ToUpper(sanitizeTestSuiteName(testSuite)), uniqKey)
	if testSuite == "" {
		testSuite = DefaultTestSuiteName
	}
	stdPrinter.ColoredErrf(messageColor, "Initialized new [%s] test suite.\n", testSuite)
	//stdPrinter.Errf("%s\n", tmpDir)
	//stdPrinter.Errf("%s\n", context)
	stdPrinter.Flush()
	return uniqKey
}

func ReportTestSuite(ctx Context) {
	testSuite := ctx.TestSuite
	// load context
	uniqKey, err := loadUniqKey(testSuite)
	if err != nil {
		log.Fatal(err)
	}
	tmpDir := testsuiteDirectoryPath(testSuite, uniqKey)
	defer os.RemoveAll(tmpDir)

	suiteContext, err := LoadSuiteContext(testSuite, uniqKey)
	if err != nil {
		log.Fatal(err)
	}
	ctx = MergeContext(suiteContext, ctx)
	testCount := readSeq(testSuite, uniqKey, TestSequenceFilename)
	ignoredCount := readSeq(testSuite, uniqKey, IgnoredSequenceFilename)
	failureReports := failureReports(testSuite, uniqKey)
	failedCount := len(failureReports)

	if testSuite == "" {
		testSuite = DefaultTestSuiteName
	}
	stdPrinter.ColoredErrf(messageColor, "Reporting [%s] test suite ...\n", testSuite)

	ignoredMessage := ""
	if ignoredCount > 0 {
		ignoredMessage = fmt.Sprintf(" (%d ignored)", ignoredCount)
	}
	if failedCount == 0 {
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
}

func PerformTest(ctx Context, cmdAndArgs []string, assertions []Assertion) (success bool) {
	testSuite := ctx.TestSuite
	testName := ctx.TestName
	success = false

	// load config
	uniqKey, err := loadUniqKey(testSuite)
	if err != nil {
		uniqKey = InitTestSuite(ctx)
	}
	suiteContext, err := LoadSuiteContext(testSuite, uniqKey)
	if err != nil {
		log.Fatal(err)
	}
	ctx = MergeContext(suiteContext, ctx)
	timecode := int(time.Since(ctx.StartTime).Milliseconds())

	cmd := cmdz.Cmd(cmdAndArgs[0])
	if len(cmdAndArgs) == 0 {
		log.Fatalf("no command supplied to test")
	} else if len(cmdAndArgs) > 1 {
		cmd.AddArgs(cmdAndArgs[1:]...)
	}

	if testName == "" {
		testName = fmt.Sprintf("cmd: [%s]", cmd)
	}

	qulifiedName := testName
	if testSuite != "" {
		qulifiedName = fmt.Sprintf("[%s]/%s", testSuite, testName)
	}
	if *ctx.Ignore {
		stdPrinter.ColoredErrf(warningColor, "[%05d] Ignored test: %s\n", timecode, qulifiedName)
		incrementSeq(testSuite, uniqKey, IgnoredSequenceFilename)
		return true
	}
	seq := incrementSeq(testSuite, uniqKey, TestSequenceFilename)
	stdoutLog, stderrLog, reportLog := cmdLogFiles(testSuite, uniqKey, seq)
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
	if ctx.Timeout.Milliseconds() > 0 {
		cmd.Timeout(int(ctx.Timeout.Milliseconds()))
	}

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
	success = false
	if err == nil {
		success = true
		testDuration = cmd.Duration()
		for _, assertion := range assertions {
			var result AssertionResult
			result, err = assertion.Asserter(cmd)
			if err != nil {
				log.Fatal(err)
			}
			if !result.Success {
				failedResults = append(failedResults, result)
			}
			success = success && result.Success
		}
	}

	if *ctx.KeepStdout || *ctx.KeepStderr {
		// NewLine in printer to print test result in a new line
		stdPrinter.Errf("        ")
	}

	if success {
		stdPrinter.ColoredErrf(successColor, "PASSED")
		stdPrinter.Errf(" (in %s)\n", testDuration)
		defer os.Remove(reportLog.Name())
	} else {
		stdPrinter.ColoredErrf(failureColor, "FAILED")
		if err == nil {
			stdPrinter.Errf(" (in %s)\n", testDuration)
		} else {
			stdPrinter.Errf(" (not executed)\n")
		}
		stdPrinter.Errf("Failure calling: [%s]\n", cmd)
		if err != nil {
			stdPrinter.ColoredErrf(errorColor, "error executing command: \n%s\n", err)
			reportLog.WriteString(testTitle + "  =>  not executed")
		} else {
			for _, result := range failedResults {
				//log.Printf("failedResult: %v\n", result)
				assertName := result.Assertion.Name
				assertOp := result.Assertion.Operator

				if assertName == "success" || assertName == "fail" {
					stdPrinter.Errf("Expected %s%s\n", AssertionPrefix, assertName)
				}
				expected := result.Assertion.Expected
				got := result.Value
				if expected != got {
					if assertOp == "=" {
						stdPrinter.Errf("Expected %s%s to be: [%s] but got: [%s]\n", AssertionPrefix, assertName, expected, got)
					} else if assertOp == "~" {
						stdPrinter.Errf("Expected %s%s to contains: [%s] but got: [%s]\n", AssertionPrefix, assertName, expected, got)
					}
				} else {
					stdPrinter.Errf("assertion %s%s%s%s failed\n", AssertionPrefix, assertName, assertOp, expected)
				}
			}
			failedAssertionsReport := ""
			for _, result := range failedResults {
				assertName := result.Assertion.Name
				assertOp := result.Assertion.Operator
				expected := result.Assertion.Expected
				failedAssertionsReport = AssertionPrefix + string(assertName) + string(assertOp) + string(expected)
			}
			reportLog.WriteString(testTitle + "  => " + failedAssertionsReport)
		}
	}

	stdPrinter.Flush()

	if *ctx.StopOnFailure && !success {
		ReportTestSuite(ctx)
	}

	return
}

func DoSomething(allArgs []string) {
	exitCode := 1
	defer func() { os.Exit(exitCode) }()

	if len(allArgs) == 1 {
		usage()
		return
	}

	config, cmdAndArgs, assertions, err := ParseArgs(allArgs[1:])
	if err != nil {
		log.Fatal(err)
	}
	switch config.Action {
	case "init":
		InitTestSuite(config)
	case "test":
		if !PerformTest(config, cmdAndArgs, assertions) {
			return
		}
	case "report":
		ReportTestSuite(config)
	default:
		log.Fatalf("action: %s not known", config.Action)
	}
	exitCode = 0
}
