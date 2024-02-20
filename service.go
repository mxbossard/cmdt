package main

import (
	"bytes"
	"context"
	cryptorand "crypto/rand"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"io/ioutil"
	"log"
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

func Fatal(testSuite, token string, v ...any) {
	tmpDir, err := testsuiteDirectoryPath(testSuite, token)
	if err != nil {
		log.Fatal(err)
	}
	IncrementSeq(tmpDir, ErrorSequenceFilename)
	log.Fatal(v...)
}

func Fatalf(testSuite, token, format string, v ...any) {
	log.Fatal(testSuite, token, fmt.Sprintf(format, v...))
}

func SuiteError(testSuite, token string, v ...any) error {
	return SuiteError(testSuite, token, fmt.Sprint(v...))
}

func SuiteErrorf(testSuite, token, format string, v ...any) error {
	IncrementSeq(testSuite, token, ErrorSequenceFilename)
	return fmt.Errorf(format, v...)
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
		if strings.HasPrefix(env, ContextTokenEnvVarName+"=") {
			splitted := strings.Split(env, "=")
			token = strings.Join(splitted[1:], "")
		}
	}
	//log.Printf("Found a token in env: %s", token)
	return
}

func getProcessStartTime(pid int) (int64, error) {
	// Index of the starttime field. See proc(5).
	const StartTimeIndex = 21

	fname := filepath.Join("/proc", strconv.Itoa(pid), "stat")
	data, err := ioutil.ReadFile(fname)
	if err != nil {
		return 0, err
	}

	fields := bytes.Fields(data)
	if len(fields) < StartTimeIndex+1 {
		return 0, fmt.Errorf("invalid /proc/[pid]/stat format: too few fields: %d", len(fields))
	}

	s := string(fields[StartTimeIndex])
	starttime, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid starttime: %w", err)
	}

	return starttime, nil
}

func forgeContextualToken() (string, error) {
	// If no token supplied use Workspace dir + ppid to forge tmp directory path
	workDirPath, err := os.Getwd()
	if err != nil {
		//log.Fatalf("cannot find workspace dir: %s", err)
		return "", fmt.Errorf("cannot find workspace dir: %w", err)
	}
	ppid := os.Getppid()
	ppidStr := fmt.Sprintf("%d", ppid)
	ppidStartTime, err := getProcessStartTime(ppid)
	if err != nil {
		//log.Fatalf("cannot find parent process start time: %s", err)
		return "", fmt.Errorf("cannot find parent process start time: %w", err)
	}
	ppidStartTimeStr := fmt.Sprintf("%d", ppidStartTime)
	token, err := trust.SignStrings(workDirPath, "--", ppidStr, "--", ppidStartTimeStr)
	if err != nil {
		err = fmt.Errorf("cannot hash workspace dir: %w", err)
	}
	//log.Printf("contextual token: %s base on workDirPath: %s and ppid: %s\n", token, workDirPath, ppid)
	return token, err
}

func forgeTmpDirectoryPath(token string) (tempDirPath string, err error) {
	if token == "" {
		token, err = forgeContextualToken()
	}
	if err != nil {
		return
	}
	tempDirName := fmt.Sprintf("%s-%s", TempDirPrefix, token)
	tempDirPath = filepath.Join(os.TempDir(), tempDirName)
	err = os.MkdirAll(tempDirPath, 0700)
	return
}

func testsuiteDirectoryPath(testSuite, token string) (path string, err error) {
	var tmpDir string
	tmpDir, err = forgeTmpDirectoryPath(token)
	if err != nil {
		return
	}
	suiteDir := sanitizeTestSuiteName(testSuite)
	path = filepath.Join(tmpDir, suiteDir)
	//log.Printf("testsuiteDir: %s\n", path)
	err = os.MkdirAll(path, 0700)
	return
}

func testDirectoryPath(testSuite, token string, seq int) (testDir string, err error) {
	var tmpDir string
	tmpDir, err = testsuiteDirectoryPath(testSuite, token)
	if err != nil {
		return
	}
	testDir = filepath.Join(tmpDir, "test-"+fmt.Sprintf("%06d", seq))
	return
}

func listTestSuites(token string) (suites []string, err error) {
	var tmpDir string
	tmpDir, err = forgeTmpDirectoryPath(token)
	if err != nil {
		return
	}
	_, err = os.Stat(tmpDir)
	if os.IsNotExist(err) {
		err = nil
		return
	}

	matches, err := filepath.Glob(tmpDir + "/*")
	if err != nil {
		err = fmt.Errorf("cannot list test suites: %w", err)
		return
	}
	for _, m := range matches {
		testSuite := filepath.Base(m)
		if testSuite != GlobalConfigTestSuiteName {
			suites = append(suites, testSuite)
		}
	}
	return
}

func testsuiteConfigFilepath(testSuite, token string) (path string, err error) {
	var testDir string
	testDir, err = testsuiteDirectoryPath(testSuite, token)
	if err != nil {
		return
	}
	path = filepath.Join(testDir, ContextFilename)
	return
}

func cmdLogFiles(testSuite, token string, seq int) (stdoutFile, stderrFile, reportFile *os.File, err error) {
	var testDir string
	testDir, err = testDirectoryPath(testSuite, token, seq)
	if err != nil {
		return
	}
	stdoutFilepath := filepath.Join(testDir, StdoutFilename)
	stderrFilepath := filepath.Join(testDir, StderrFilename)
	reportFilepath := filepath.Join(testDir, ReportFilename)

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

func failureReports(testSuite, token string) (reports []string, err error) {
	var tmpDir string
	tmpDir, err = testsuiteDirectoryPath(testSuite, token)
	if err != nil {
		return
	}
	err = filepath.Walk(tmpDir, func(path string, info fs.FileInfo, err error) error {
		if ReportFilename == info.Name() {
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

func sanitizeTestSuiteName(s string) string {
	return testSuiteNameSanitizerPattern.ReplaceAllString(s, "_")
}

func PersistSuiteContext(testSuite, token string, config Context) (err error) {
	var contextFilepath string
	contextFilepath, err = testsuiteConfigFilepath(testSuite, token)
	if err != nil {
		return
	}
	//stdPrinter.Errf("Built context: %v\n", context)
	content, err := yaml.Marshal(config)
	if err != nil {
		return
	}
	//stdPrinter.Errf("Persisting context: %s\n", content)
	err = os.WriteFile(contextFilepath, content, 0600)
	if err != nil {
		err = fmt.Errorf("cannot persist context: %w", err)
		return
	}
	//log.Printf("Persisted context file: %s\n", contextFilepath)
	return
}

func readSuiteContext(testSuite, token string) (config Context, err error) {
	var contextFilepath string
	contextFilepath, err = testsuiteConfigFilepath(testSuite, token)
	if err != nil {
		return
	}
	var content []byte
	content, err = os.ReadFile(contextFilepath)
	if err != nil {
		return
	}
	err = yaml.Unmarshal(content, &config)
	return
}

func LoadSuiteContext(testSuite, token string) (config Context, err error) {
	var globalCtx, suiteCtx Context
	globalCtx, err = LoadGlobalContext(token)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		//log.Printf("readSuiteContext err: %s for: %s\n", err, GlobalConfigTestSuiteName)
		return
	}
	suiteCtx, err = readSuiteContext(testSuite, token)
	if err != nil {
		//log.Printf("readSuiteContext err: %s for: %s\n", err, testSuite)
		return
	}
	config = MergeContext(globalCtx, suiteCtx)
	SetRulePrefix(config.Prefix)
	return
}

func LoadGlobalContext(token string) (config Context, err error) {
	config, err = readSuiteContext(GlobalConfigTestSuiteName, token)
	config.TestSuite = ""
	return
}

func UniqToken() (uuid string, err error) {
	b := make([]byte, 16)
	_, err = cryptorand.Read(b)
	if err != nil {
		return
	}
	uuid = fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:])
	return

	/*
		h, err := trust.SignStrings(uuid)
		if err != nil {
			log.Fatalf("cannot forge a uniq token: %s", err)
		}
		return h
	*/
}

func initWorkspace(ctx Context) (err error) {
	token := ctx.Token
	testSuite := ctx.TestSuite

	// init the tmp directory
	var tmpDir string
	tmpDir, err = testsuiteDirectoryPath(testSuite, token)
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

	if ctx.Silent == nil || !*ctx.Silent {
		stdPrinter.ColoredErrf(messageColor, "Initialized new [%s] workspace.\n", token)
	}

	stdPrinter.Flush()
	return
}

func initConfig(ctx Context) (err error) {
	token := ctx.Token
	testSuite := ctx.TestSuite

	var contextFilepath string
	contextFilepath, err = testsuiteConfigFilepath(testSuite, token)
	if err != nil {
		return
	}
	_, err = os.Stat(contextFilepath)
	if err == nil {
		// Workspace already initialized
		return
	} else if !errors.Is(err, os.ErrNotExist) {
		return
	} else {
		err = nil
	}

	ctx.StartTime = time.Now()
	// store config
	PersistSuiteContext(testSuite, token, ctx)
	stdPrinter.ColoredErrf(messageColor, "Initialized new config [%s].\n", testSuite)
	return
}

func GlobalConfig(ctx Context) (exitCode int, err error) {
	defer stdPrinter.Flush()
	exitCode = 0
	ctx.TestSuite = GlobalConfigTestSuiteName
	err = initWorkspace(ctx)
	if err != nil {
		return
	}
	err = initConfig(ctx)
	return
}

func InitTestSuite(ctx Context) (exitCode int, err error) {
	defer stdPrinter.Flush()
	exitCode = 0
	token := ctx.Token
	testSuite := ctx.TestSuite

	if ctx.Action == "init" && ctx.PrintToken {
		token, err = UniqToken()
		if err != nil {
			return
		}
		fmt.Printf("%s\n", token)
		ctx.Token = token
	} else if ctx.Action == "init" && ctx.ExportToken {
		token, err = UniqToken()
		if err != nil {
			return
		}
		fmt.Printf("export %s=%s\n", ContextTokenEnvVarName, token)
		ctx.Token = token
	}

	exitCode, err = GlobalConfig(Context{Token: token})
	if err != nil {
		return
	}

	// Clear the test suite directory
	var tmpDir string
	tmpDir, err = testsuiteDirectoryPath(testSuite, token)
	if err != nil {
		return
	}
	err = os.RemoveAll(tmpDir)
	if err != nil {
		return
	}
	//log.Printf("Cleared test suite dir: %s\n", tmpDir)

	//log.Printf("init context: %s Silent: %v\n", testSuite, ctx.Silent)
	err = initWorkspace(ctx)
	if err != nil {
		return
	}
	err = initConfig(ctx)
	if err != nil {
		return
	}
	//log.Printf("init context: %s Silent: %v\n", testSuite, ctx.Silent)

	if ctx.Silent == nil || !*ctx.Silent {
		var tokenMsg = ""
		if token != "" {
			tokenMsg = fmt.Sprintf(" (token: %s)", token)
		}
		stdPrinter.ColoredErrf(messageColor, "Initialized new [%s] test suite%s.\n", testSuite, tokenMsg)
	}
	return
}

func ReportTestSuite(ctx Context) (exitCode int, err error) {
	exitCode = 1
	token := ctx.Token
	testSuite := ctx.TestSuite
	defer stdPrinter.Flush()

	if ctx.ReportAll {
		// Report all test suites
		var testSuites []string
		testSuites, err = listTestSuites(token)
		if err != nil {
			return
		}
		if testSuites != nil {
			exitCode = 0
			//log.Printf("reporting found suites: %s\n", testSuites)
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
			var global Context
			global, err = LoadGlobalContext(token)
			if err != nil {
				err = fmt.Errorf("cannot load global context: %s", err)
				return
			}
			globalDuration := time.Since(global.StartTime)
			stdPrinter.ColoredErrf(reportColor, "Global duration time: %s\n", globalDuration)
			return
		}
	}

	var tmpDir string
	tmpDir, err = testsuiteDirectoryPath(testSuite, token)
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
	ctx = MergeContext(suiteContext, ctx)
	testCount := ReadSeq(tmpDir, TestSequenceFilename)
	ignoredCount := ReadSeq(tmpDir, IgnoredSequenceFilename)
	errorCount := ReadSeq(tmpDir, ErrorSequenceFilename)
	var failedReports []string
	failedReports, err = failureReports(testSuite, token)
	if err != nil {
		return
	}
	failedCount := len(failedReports)

	if ctx.Silent == nil || !*ctx.Silent {
		stdPrinter.ColoredErrf(messageColor, "Reporting [%s] test suite (%s) ...\n", testSuite, tmpDir)
	}

	ignoredMessage := ""
	if ignoredCount > 0 {
		ignoredMessage = fmt.Sprintf(" (%d ignored)", ignoredCount)
	}
	duration := suiteContext.LastTestTime.Sub(suiteContext.StartTime)
	if failedCount == 0 && errorCount == 0 {
		exitCode = 0
		stdPrinter.ColoredErrf(successColor, "Successfuly ran [%s] test suite (%d tests in %s)", testSuite, testCount, duration)
		stdPrinter.ColoredErrf(warningColor, "%s", ignoredMessage)
		stdPrinter.Errf("\n")
	} else {
		successCount := testCount - failedCount
		stdPrinter.ColoredErrf(failureColor, "Failures in [%s] test suite (%d success, %d failures, %d errors on %d tests in %s)", testSuite, successCount, failedCount, errorCount, testCount, duration)
		stdPrinter.ColoredErrf(warningColor, "%s", ignoredMessage)
		stdPrinter.Errf("\n")
		for _, report := range failedReports {
			stdPrinter.ColoredErrf(reportColor, "%s\n", report)
		}
	}
	return
}

func updateLastTestTime(suiteCtx Context) {
	token := suiteCtx.Token
	testSuite := suiteCtx.TestSuite
	suiteCtx.LastTestTime = time.Now()
	PersistSuiteContext(testSuite, token, suiteCtx)
}

func PerformTest(ctx Context, cmdAndArgs []string, assertions []Assertion) (exitCode int, err error) {
	token := ctx.Token
	testSuite := ctx.TestSuite
	testName := ctx.TestName
	exitCode = 1
	defer stdPrinter.Flush()

	suiteContext, err := LoadSuiteContext(testSuite, token)
	if err != nil {
		if os.IsNotExist(err) {
			// test suite does not exists yet
			suiteCtx := Context{Token: token, TestSuite: testSuite}
			exitCode, err = InitTestSuite(suiteCtx)
			if err != nil {
				return
			}
			if exitCode > 0 {
				return
			}
			// Recursive call once test suite initialized
			return PerformTest(ctx, cmdAndArgs, assertions)
		} else {
			err = fmt.Errorf("cannot load context: %w", err)
			return
		}
	}
	defer updateLastTestTime(suiteContext)

	ctx = MergeContext(suiteContext, ctx)
	timecode := int(time.Since(ctx.StartTime).Milliseconds())

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

	var tmpDir string
	tmpDir, err = testsuiteDirectoryPath(testSuite, token)
	if err != nil {
		return
	}

	if ctx.Ignore != nil && *ctx.Ignore {
		if ctx.Silent == nil || !*ctx.Silent {
			stdPrinter.ColoredErrf(warningColor, "[%05d] Ignored test: %s\n", timecode, qulifiedName)
		}
		IncrementSeq(tmpDir, IgnoredSequenceFilename)
		exitCode = 0
		return
	}
	seq := IncrementSeq(tmpDir, TestSequenceFilename)
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

	testTitle := fmt.Sprintf("[%05d] Test %s #%02d", timecode, qulifiedName, seq)
	if ctx.Silent == nil || !*ctx.Silent {
		stdPrinter.ColoredErrf(testColor, "%s... ", testTitle)
		if *ctx.KeepStdout || *ctx.KeepStderr {
			// NewLine because we expect cmd outputs
			stdPrinter.Errf("\n")
		}
	}

	stdPrinter.Flush()

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
				//Fatal(testSuite, token, err)
				//stdPrinter.ColoredErrf(errorColor, "FAILED (error: %s) ", err)
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

	if (ctx.Silent == nil || !*ctx.Silent) && (*ctx.KeepStdout || *ctx.KeepStderr) {
		// NewLine in printer to print test result in a new line
		stdPrinter.Errf("        ")
		stdPrinter.Flush()
	}

	if exitCode == 0 {
		if ctx.Silent == nil || !*ctx.Silent {
			stdPrinter.ColoredErrf(successColor, "PASSED")
			stdPrinter.Errf(" (in %s)\n", testDuration)
		}
		defer os.Remove(reportLog.Name())
	} else {
		if ctx.Silent == nil || *ctx.Silent {
			stdPrinter.ColoredErrf(testColor, "%s... ", testTitle)
		}
		if err == nil {
			stdPrinter.ColoredErrf(failureColor, "FAILED")
			stdPrinter.Errf(" (in %s)\n", testDuration)
		} else {
			if errors.Is(err, context.DeadlineExceeded) {
				err = nil
				stdPrinter.ColoredErrf(failureColor, "FAILED")
				stdPrinter.Errf(" (timed out after %s)\n", ctx.Timeout)
				reportLog.WriteString(testTitle + "  =>  timed out")
			} else {
				err = nil
				IncrementSeq(tmpDir, ErrorSequenceFilename)
				stdPrinter.ColoredErrf(warningColor, "ERROR")
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
					if cmd.StderrRecord() != "" {
						stdPrinter.Errf("sdterr> %s\n", cmd.StderrRecord())
					}
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
					} else if assertOp == "!:" {
						stdPrinter.Errf("Expected %s%s not to contains: [%s] but got: [%v]\n", assertPrefix, assertName, expected, got)
					} else if assertOp == "~" {
						stdPrinter.Errf("Expected %s%s to match: [%s] but got: [%v]\n", assertPrefix, assertName, expected, got)
					} else if assertOp == "!~" {
						stdPrinter.Errf("Expected %s%s not to match: [%s] but got: [%v]\n", assertPrefix, assertName, expected, got)
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
				failedAssertionsReport += RulePrefix() + string(assertName) + string(assertOp) + string(expected) + " "
			}
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
	if config.TestSuite == "" {
		config.TestSuite = DefaultTestSuiteName
	}
	if config.Token == "" {
		config.Token = readEnvToken()
	}

	if err != nil {
		suiteContext, err2 := LoadSuiteContext(config.TestSuite, config.Token)
		if err2 == nil {
			updateLastTestTime(suiteContext)
		}
		Fatal(config.TestSuite, config.Token, err)
	}

	switch config.Action {
	case "global":
		exitCode, err = GlobalConfig(config)
	case "init":
		exitCode, err = InitTestSuite(config)
	case "test":
		exitCode, err = PerformTest(config, cmdAndArgs, assertions)
	case "report":
		exitCode, err = ReportTestSuite(config)
	default:
		err = fmt.Errorf("action: [%s] not known", config.Action)
	}

	if err != nil {
		Fatal(config.TestSuite, config.Token, err)
	}
}

func mockDirectoryPath(testSuite, token string, seq int) (mockDir string, err error) {
	var testDir string
	testDir, err = testDirectoryPath(testSuite, token, seq)
	mockDir = filepath.Join(testDir, "mock")
	return
}

func ProcessMocking(testSuite, token string, seq int, mocks []CmdMock) (mockDir string, err error) {
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

func writeMockWrapperScript(wrapperFilepath string, mocks []CmdMock) (err error) {
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
