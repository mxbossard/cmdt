package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log"
	"math/rand"
	"os"
	"path/filepath"
	"strconv"
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

func forgeUniqKey(name string) string {
	h, err := trust.SignStrings(strings.ToUpper(name), time.Now().String(), fmt.Sprint(rand.Int()))
	if err != nil {
		log.Fatalf("Cannot forge a uniq key ! Error: %s", err)
	}
	return h
}

func buildNoTestToReportError(testSuite string) error {
	if testSuite == "" {
		testSuite = DefaultTestSuiteName
	}
	return fmt.Errorf("cannot found context env var for test suite: [%s]. You must perform some test before reporting", testSuite)
}

func loadUniqKey(testSuite string) (key string, err error) {
	// Search uniqKey in env
	for _, env := range os.Environ() {
		if strings.HasPrefix(env, ContextEnvVarName+strings.ToUpper(sanitizeTestSuiteName(testSuite))+"=") {
			splitted := strings.Split(env, "=")
			key = strings.Join(splitted[1:], "")
			//log.Printf("Laoded uniqKey from env: %s", key)
			return key, nil
		}
	}
	// Search uniqKey in tmp dir
	lastTmpDir := lastTmpDirectoryPath(testSuite)
	if lastTmpDir != "" {
		key = filepath.Base(lastTmpDir)
		//log.Printf("Laoded uniqKey from tmp dir: %s", key)
		return key, nil
	}

	//err = fmt.Errorf("cannot found context env var for test suite: [%s]. You must export init action like this : eval $( cmdt @init )", testSuite)
	err = buildNoTestToReportError(testSuite)
	return
}

func tmpDirectoryPath(testSuite string) string {
	tempDirName := fmt.Sprintf("%s.%d.%s", TempDirPrefix, os.Getppid(), sanitizeTestSuiteName(testSuite))
	tempDirPath := filepath.Join(os.TempDir(), tempDirName)
	os.MkdirAll(tempDirPath, 0700)
	return tempDirPath
}

func testsuiteDirectoryPath(testSuite, uniqKey string) string {
	path := filepath.Join(tmpDirectoryPath(testSuite), uniqKey)
	return path
}

func lastTmpDirectoryPath(testSuite string) string {
	var matchingDirs []fs.DirEntry
	var lastMatchingDir *fs.DirEntry
	rootPath := tmpDirectoryPath(testSuite)
	err := filepath.WalkDir(rootPath, func(path string, d fs.DirEntry, err error) error {
		matcher := rootPath + string(filepath.Separator)
		//log.Printf("%s / %s", path, matcher)
		if d.IsDir() && strings.HasPrefix(path, matcher) {
			matchingDirs = append(matchingDirs, d)
			return filepath.SkipDir
		}
		return nil
	})
	if err != nil {
		log.Fatal(err)
	}
	var lastModTime time.Time
	for _, d := range matchingDirs {
		info, err2 := d.Info()
		if err2 != nil {
			log.Fatal(err2)
		}
		if lastMatchingDir == nil || info.ModTime().After(lastModTime) {
			lastMatchingDir = &d
		}

	}
	if lastMatchingDir != nil {
		dirName := filepath.Base((*lastMatchingDir).Name())
		return dirName
	}
	return ""
}

func testConfigFilepath(testSuite, uniqKey string) string {
	return filepath.Join(testsuiteDirectoryPath(testSuite, uniqKey), ContextFilename)
}

func cmdLogFiles(testSuite, uniqKey string, seq int) (*os.File, *os.File, *os.File) {
	tmpDir := testsuiteDirectoryPath(testSuite, uniqKey)
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

func initSeq(testSuite, uniqKey string) {
	tmpDir := testsuiteDirectoryPath(testSuite, uniqKey)
	seqFilepath := filepath.Join(tmpDir, TestSequenceFilename)
	err := os.WriteFile(seqFilepath, []byte("0"), 0600)
	if err != nil {
		log.Fatalf("Cannot initialize seq file ! Error: %s", err)
	}
	seqFilepath = filepath.Join(tmpDir, IgnoredSequenceFilename)
	err = os.WriteFile(seqFilepath, []byte("0"), 0600)
	if err != nil {
		log.Fatalf("Cannot initialize seq file ! Error: %s", err)
	}
}

func incrementSeq(testSuite, uniqKey, filename string) (seq int) {
	// return an increment for test indexing
	tmpDir := testsuiteDirectoryPath(testSuite, uniqKey)
	seqFilepath := filepath.Join(tmpDir, filename)

	file, err := os.OpenFile(seqFilepath, os.O_RDWR|os.O_CREATE, 0600)
	if err != nil {
		log.Fatalf("Cannot open seq file ! Error: %s", err)
	}
	defer file.Close()
	var strSeq string
	_, err = fmt.Fscanln(file, &strSeq)
	if err != nil {
		log.Fatalf("cannot read seq file as an integer ! Error: %s", err)
	}
	seq, err = strconv.Atoi(strSeq)
	if err != nil {
		log.Fatalf("cannot read seq file as an integer ! Error: %s", err)
	}

	newSec := seq + 1
	_, err = file.WriteAt([]byte(fmt.Sprint(newSec)), 0)
	if err != nil {
		log.Fatalf("Cannot write seq file ! Error: %s", err)
	}

	//fmt.Printf("Incremented seq: %d => %d\n", seq, newSec)
	return newSec
}

func readSeq(testSuite, uniqKey, filename string) (c int) {
	// return the count of run test
	tmpDir := testsuiteDirectoryPath(testSuite, uniqKey)
	seqFilepath := filepath.Join(tmpDir, filename)

	file, err := os.OpenFile(seqFilepath, os.O_RDWR|os.O_CREATE, 0600)
	if err != nil {
		log.Fatalf("Cannot open seq file ! Error: %s", err)
	}
	defer file.Close()
	var strSeq string
	_, err = fmt.Fscanln(file, &strSeq)
	if err != nil {
		log.Fatalf("cannot read seq file as an integer ! Error: %s", err)
	}
	c, err = strconv.Atoi(strSeq)
	if err != nil {
		log.Fatalf("cannot read seq file as an integer ! Error: %s", err)
	}
	return
}

func failureReports(testSuite, uniqKey string) (reports []string) {
	tmpDir := testsuiteDirectoryPath(testSuite, uniqKey)
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
		log.Fatal(err2)
		//err = buildNoTestToReportError(testSuite)
		return
	}
	err2 = yaml.Unmarshal(content, &config)
	if err2 != nil {
		log.Fatal(err2)
	}

	SetRulePrefix(config.Prefix)
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

func ReportTestSuite(ctx Context) (exitCode int) {
	exitCode = 1
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
			result.Assertion = assertion
			if err != nil {
				//log.Fatal(err)
				//stdPrinter.ColoredErrf(errorColor, "FAILED (error: %s) ", err)
				result.Message = fmt.Sprintf("%s", err)
				result.Success = false
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
				assertOp := result.Assertion.Operator
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
					} else if assertOp == "~" {
						stdPrinter.Errf("Expected %s%s to contains: [%s] but got: [%v]\n", assertPrefix, assertName, expected, got)
					}
				} else {
					stdPrinter.Errf("assertion %s%s%s%s failed\n", assertPrefix, assertName, assertOp, expected)
				}
			}
			failedAssertionsReport := ""
			for _, result := range failedResults {
				assertName := result.Assertion.Name
				assertOp := result.Assertion.Operator
				expected := result.Assertion.Expected
				failedAssertionsReport = RulePrefix() + string(assertName) + string(assertOp) + string(expected)
			}
			reportLog.WriteString(testTitle + "  => " + failedAssertionsReport)
		}
	}

	stdPrinter.Flush()

	if ctx.StopOnFailure == nil || *ctx.StopOnFailure && !success {
		ReportTestSuite(ctx)
	}
	
	if ctx.StopOnFailure == nil || !*ctx.StopOnFailure {
		// FIXME: Do not return a success to let test continue
		success=true
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
	switch config.Action {
	case "init":
		InitTestSuite(config)
		exitCode = 0
	case "test":
		if !PerformTest(config, cmdAndArgs, assertions) {
			return
		}
		exitCode = 0
	case "report":
		exitCode = ReportTestSuite(config)
	default:
		log.Fatalf("action: [%s] not known", config.Action)
	}
}
