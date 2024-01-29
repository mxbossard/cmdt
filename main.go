/*
Copyright © 2021 NAME HERE <EMAIL ADDRESS>

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package main

import (
	"fmt"
	"io"
	"io/fs"
	"log"
	"math/rand"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"gopkg.in/yaml.v3"

	"mby.fr/utils/ansi"
	"mby.fr/utils/cmdz"
	"mby.fr/utils/collections"
	"mby.fr/utils/printz"
	"mby.fr/utils/trust"
)

/**
Tester des commandes.

## cmdt @init
Initialize the test suite with the supplied configuration.
Options:
- @stopOnFailure : Stop test suite on first failure and report.
- @logOnSuccess : Always log cmd output.

This should initialize a uniq context shared by all test suite. How ?
- Export vars ?
- Produce a hidden file ?
- Produce a hidden dir ?

This could be ommited => testSuite name will then be empty and all following tests placed in this test suite.


## cmdt @report
Display a test suite report and return a failure if at least one test failed.

## cmdt <testName>
Launch a command test and do some assertion.
Assertions:
- @fail : cmd should fail (rc > 0)
- @success : cmd should succeed (rc = 0)
- @stdout=string : cmd stdout should be exactly message
- @stdout~string : cmd stdout should match message
- @stderr=string : cmd stderr should be exactly message
- @stderr~string : cmd stderr should match message
- @exists=path,perms,owners : a file should be produced at path with perms and owners

## Principes
- eval $( cmdt [testSuite] @init @stopOnFailure @logOnSuccess )
- cmdt <[testSuite/]testName> <myCommand> myArg1 ... myArgN @fail @rc=
- cmdt <[testSuite/]testName> <myCommand> myArg1 ... myArgN @success @stdout="MyOut" @stderr="MyErr" @exists="MyFile,Perms,Owners"
- cmdt [testSuite] @report

## Improvements
- Manage stdin
- Manage stdout and stderr redirects (disable outputs override and report with another following command ?)

*/

/*
TODO:
Bugs:

Features :
- add @timeout=N
- @fork=5 by default instead of @parallel. Fork = 5 increment and decrement a seq
- change @ @directive= ??? attention à la sécurité ça pourrait etre galere
- @exists=
- @runCount=N + @parallel=FORK_COUNT (min, max, median exec time)
- @assert="CMD_WITH_ARGS_THAT_SHOULD_BE_OK_IF_OR_FAIL_TEST"
- Use os temp dir and Clean old temp dir (older than 2 min ?)
- mock des appels de commande
- mock web spawning a web server
- test port opening if daemon ; test sending data on port ???
*/

type RuleType string

type AssertionRule struct {
	Typ      RuleType
	Operator string
	Expected string
	Result   string
}

type Context struct {
	Ignore        bool      `yaml:""`
	StopOnFailure bool      `yaml:""`
	KeepStdout    bool      `yaml:""`
	KeepStderr    bool      `yaml:""`
	ForkCount     int       `yaml:""`
	StartTime     time.Time `yaml:""`
}

var (
	AssertionPrefix      = "@"
	ContextEnvVarName    = "__CMDTEST_CONTEXT_KEY_"
	DefaultTestSuiteName = "_default"

	ActionInit   = RuleType("init")
	ActionReport = RuleType("report")
	ActionTest   = RuleType("test")

	ConfigStopOnFailure = RuleType("stopOnFailure")
	ConfigKeepStdout    = RuleType("keepStdout")
	ConfigKeepStderr    = RuleType("keepStderr")
	ConfigKeepOutputs   = RuleType("keepOutputs")
	ConfigIgnore        = RuleType("ignore")
	ConfigFork          = RuleType("fork")

	RuleFail    = RuleType("fail")
	RuleSuccess = RuleType("success")
	RuleExit    = RuleType("exit")
	RuleStdout  = RuleType("stdout")
	RuleStderr  = RuleType("stderr")
	RuleExists  = RuleType("exists")

	AssertFail    = &AssertionRule{Typ: RuleFail}
	AssertSuccess = &AssertionRule{Typ: RuleSuccess}

	TempDirPrefix           = "cmdtest"
	ContextFilename         = "context.yaml"
	TestSequenceFilename    = "test-seq.txt"
	IgnoredSequenceFilename = "ignored-seq.txt"
	StdoutFilename          = "stdout.log"
	StderrFilename          = "stderr.log"
	ReportFilename          = "report.log"

	messageColor = ansi.HiPurple
	testColor    = ansi.HiCyan
	successColor = ansi.BoldGreen
	failureColor = ansi.BoldRed
	reportColor  = ansi.Yellow
	warningColor = ansi.BoldHiYellow
	errorColor   = ansi.Red
)

var stdPrinter printz.Printer
var assertionRulePattern = regexp.MustCompile("^" + AssertionPrefix + "([a-zA-Z]+)([=~])?(.+)?$")
var testSuiteNameSanitizerPatter = regexp.MustCompile("[^a-zA-Z0-9]")

//var logger = logz.Default("cmdtest", 0)

func usage() {
	cmd := filepath.Base(os.Args[0])
	stdPrinter.Errf("cmdtest tool is usefull to test various scripts cli and command behaviors.\n")
	stdPrinter.Errf("You must initialize a test suite (%[1]s @init) before running tests and then report the test (%[1]s @report).\n", cmd)
	stdPrinter.Errf("usage: \t%s @init[=TEST_SUITE_NAME] [@CONFIG_1] ... [@CONFIG_N] \n", cmd)
	stdPrinter.Errf("usage: \t%s <COMMAND> [ARG_1] ... [ARG_N] [@CONFIG_1] ... [@CONFIG_N] [@ASSERTION_1] ... [@ASSERTION_N]\n", cmd)
	stdPrinter.Errf("usage: \t%s @report[=TEST_SUITE_NAME] \n", cmd)
	stdPrinter.Errf("\tCONFIG available: @ignore @stopOnFailure @keepStdout @keepStderr @keepOutputs @fork=N\n")
	stdPrinter.Errf("\tCOMMAND and ARGs: the command on which to run tests\n")
	stdPrinter.Errf("\tASSERTIONs available: @fail @success @exit=N @stdout= @stdout~ @stderr= @stderr~\n")
}

func buildRule(arg string) (rule *AssertionRule, err error) {
	submatch := assertionRulePattern.FindStringSubmatch(arg)
	if submatch != nil {
		typ := RuleType(submatch[1])
		operator := submatch[2]
		value := submatch[3]

		// check rule existance
		switch typ {
		case ActionTest, ActionInit, ActionReport:
		case ConfigIgnore, ConfigStopOnFailure, ConfigKeepOutputs, ConfigKeepStdout, ConfigKeepStderr, ConfigFork:
		case RuleSuccess, RuleFail, RuleExit, RuleStdout, RuleStderr, RuleExists:
		default:
			err = fmt.Errorf("assertion @%s does not exist", typ)
			return
		}

		// Check rule operator
		switch typ {
		case RuleSuccess, RuleFail:
			if operator != "" || value != "" {
				err = fmt.Errorf("assertion @%s must have no value", typ)
				return
			}
		case ConfigIgnore, ConfigStopOnFailure, ConfigKeepOutputs, ConfigKeepStdout, ConfigKeepStderr:
			if operator == "=" {
				if value != "true" && value != "false" {
					err = fmt.Errorf("config @%s only support 'true' and 'false' values", typ)
				}
			} else if operator == "~" {
				err = fmt.Errorf("config @%s only support '=' operator", typ)
			} else if operator == "" {
				operator = "="
				value = "true"
			}
		case ConfigFork:
			if operator != "=" || value == "" {
				err = fmt.Errorf("assertion @%s must have a value", typ)
				return
			}
			n, err2 := strconv.Atoi(value)
			if err2 != nil || n < 1 || n > 20 {
				err = fmt.Errorf("config @%s only support integer > 0 and <= 20", typ)
			}
		default:
			if operator == "" {
				err = fmt.Errorf("assertion @%s rmust have a value", typ)
				return
			}
		}

		// Check rule value
		switch typ {
		case RuleExit:
			// assert rc rule value is an integer
			var i int
			i, err = strconv.Atoi(value)
			if err != nil || i < 0 || i > 255 {
				err = fmt.Errorf("rc rule value must be an integer >= 0 && <= 255")
				return
			}
		}

		// fix \n not correctly passed by shell
		value = strings.ReplaceAll(value, "\\n", "\n")
		rule = &AssertionRule{typ, operator, value, ""}
	}
	return
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

func buildContext(rules []*AssertionRule) (config Context) {
	config.StartTime = time.Now()
	config.ForkCount = 5
	for _, rule := range rules {
		switch rule.Typ {
		case ConfigStopOnFailure:
			config.StopOnFailure = rule.Expected == "true"
		case ConfigKeepOutputs:
			config.KeepStdout = rule.Expected == "true"
			config.KeepStderr = config.KeepStdout
		case ConfigKeepStdout:
			config.KeepStdout = rule.Expected == "true"
		case ConfigKeepStderr:
			config.KeepStderr = rule.Expected == "true"
		case ConfigFork:
			config.ForkCount, _ = strconv.Atoi(rule.Expected)
		case ConfigIgnore:
			config.Ignore = rule.Expected == "true"
		}
	}
	return
}

func buildConfig(context Context, rules []*AssertionRule) Context {
	config := context
	for _, r := range rules {
		switch r.Typ {
		case ConfigIgnore:
			config.Ignore = r.Expected == "true"
		case ConfigStopOnFailure:
			config.StopOnFailure = r.Expected == "true"
		case ConfigKeepOutputs:
			config.KeepStdout = r.Expected == "true"
			config.KeepStderr = r.Expected == "true"
		case ConfigKeepStdout:
			config.KeepStdout = r.Expected == "true"
		case ConfigKeepStderr:
			config.KeepStderr = r.Expected == "true"
		case ConfigFork:
			config.ForkCount, _ = strconv.Atoi(r.Expected)
		}
	}
	return config
}

func persistContext(testSuite, uniqKey string, configRules []*AssertionRule) Context {
	contextFilepath := testConfigFilepath(testSuite, uniqKey)
	context := buildContext(configRules)
	//stdPrinter.Errf("Built context: %v\n", context)
	content, err := yaml.Marshal(context)
	if err != nil {
		log.Fatal(err)
	}
	//stdPrinter.Errf("Persisting context: %s\n", content)
	err = os.WriteFile(contextFilepath, content, 0600)
	if err != nil {
		log.Fatal(err)
	}
	return context
}

func loadContext(testSuite, uniqKey string) (context Context, err error) {
	contextFilepath := testConfigFilepath(testSuite, uniqKey)
	content, err2 := os.ReadFile(contextFilepath)
	if err2 != nil {
		//log.Fatal(err)
		err = buildNoTestToReportError(testSuite)
		return
	}
	err2 = yaml.Unmarshal(content, &context)
	if err2 != nil {
		log.Fatal(err2)
	}
	return
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
	return testSuiteNameSanitizerPatter.ReplaceAllString(s, "_")
}

func initAction(testSuite string, configs []*AssertionRule) string {
	// forge a uniq key
	uniqKey := forgeUniqKey(testSuite)
	// init the tmp directory
	tmpDir := testsuiteDirectoryPath(testSuite, uniqKey)
	err := os.MkdirAll(tmpDir, 0700)
	if err != nil {
		log.Fatalf("Unable to create temp dir: %s ! Error: %s", tmpDir, err)
	}
	// store config
	context := persistContext(testSuite, uniqKey, configs)
	initSeq(testSuite, uniqKey)
	// print export the key
	fmt.Printf("export %s%s=%s\n", ContextEnvVarName, strings.ToUpper(sanitizeTestSuiteName(testSuite)), uniqKey)
	if testSuite == "" {
		testSuite = DefaultTestSuiteName
	}
	stdPrinter.ColoredErrf(messageColor, "Initialized new [%s] test suite.\n", testSuite)
	//stdPrinter.Errf("%s\n", tmpDir)
	//stdPrinter.Errf("%s\n", context)
	_ = context
	return uniqKey
}

func reportAction(testSuite string, configs []*AssertionRule) {
	// load context
	uniqKey, err := loadUniqKey(testSuite)
	if err != nil {
		log.Fatal(err)
	}
	tmpDir := testsuiteDirectoryPath(testSuite, uniqKey)
	defer os.RemoveAll(tmpDir)

	context, err := loadContext(testSuite, uniqKey)
	if err != nil {
		log.Fatal(err)
	}
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
		stdPrinter.ColoredErrf(successColor, "Successfuly ran %s test suite (%d tests in %s)", testSuite, testCount, time.Since(context.StartTime))
		stdPrinter.ColoredErrf(warningColor, "%s", ignoredMessage)
		stdPrinter.Errf("\n")
	} else {
		successCount := testCount - failedCount
		stdPrinter.ColoredErrf(failureColor, "Failures in %s test suite (%d success, %d failures, %d tests in %s)", testSuite, successCount, failedCount, testCount, time.Since(context.StartTime))
		stdPrinter.ColoredErrf(warningColor, "%s", ignoredMessage)
		stdPrinter.Errf("\n")
		for _, report := range failureReports {
			stdPrinter.ColoredErrf(reportColor, "%s\n", report)
		}
	}
}

func testAction(testSuite, name string, command []string, configs, rules []*AssertionRule) (success bool) {
	success = false

	// load config
	uniqKey, err := loadUniqKey(testSuite)
	if err != nil {
		uniqKey = initAction(testSuite, nil)
	}
	context, err := loadContext(testSuite, uniqKey)
	if err != nil {
		log.Fatal(err)
	}
	config := buildConfig(context, configs)

	timecode := int(time.Since(context.StartTime).Milliseconds())

	cmd := cmdz.Cmd(command[0], command[1:]...)

	if name == "" {
		name = fmt.Sprintf("cmd: [%s]", cmd)
	}

	testName := name
	if testSuite != "" {
		testName = fmt.Sprintf("%s/%s", testSuite, name)
	}

	if config.Ignore {
		stdPrinter.ColoredErrf(warningColor, "[%05d] Ignored test: %s\n", timecode, testName)
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
	if config.KeepStdout {
		stdout = io.MultiWriter(os.Stdout, stdoutLog)
	}
	stderr = stdoutLog
	if config.KeepStderr {
		stderr = io.MultiWriter(os.Stderr, stderrLog)
	}
	cmd.SetOutputs(stdout, stderr)

	stat, _ := os.Stdin.Stat()
	if (stat.Mode() & os.ModeCharDevice) == 0 {
		cmd.SetInput(os.Stdin)
	}

	//fmt.Printf("%s", os.Environ())
	cmd = cmd.AddEnviron(os.Environ())

	testTitle := fmt.Sprintf("[%05d] Test %s #%02d", timecode, testName, seq)
	stdPrinter.ColoredErrf(testColor, "%s... ", testTitle)

	if config.KeepStdout || config.KeepStderr {
		// NewLine because we expect cmd outputs
		stdPrinter.Errf("\n")
	}

	stdPrinter.Flush()

	exitCode, err := cmd.BlockRun()
	var failedRules []*AssertionRule
	var testDuration time.Duration
	if err != nil {
		success = false
	} else {
		testDuration = cmd.Duration()
		for _, rule := range rules {
			switch rule.Typ {
			case RuleSuccess:
				success = exitCode == 0
			case RuleFail:
				success = exitCode > 0
			case RuleExit:
				expectedRc, _ := strconv.Atoi(rule.Expected)
				success = exitCode == expectedRc
				rule.Result = fmt.Sprintf("%d", exitCode)
			case RuleStdout:
				rule.Result = cmd.StdoutRecord()
				if rule.Operator == "=" {
					success = rule.Expected == cmd.StdoutRecord()
				} else if rule.Operator == "~" {
					success = strings.Contains(rule.Result, rule.Expected)
				}
			case RuleStderr:
				rule.Result = cmd.StderrRecord()
				if rule.Operator == "=" {
					success = rule.Expected == cmd.StderrRecord()
				} else if rule.Operator == "~" {
					success = strings.Contains(rule.Result, rule.Expected)

				}
			case RuleExists:
				log.Fatalf("assertion @%s not implemented yet", rule.Typ)
			default:
				log.Fatalf("assertion @%s does not exists", rule.Typ)
			}
			if !success {
				failedRules = append(failedRules, rule)
			}
		}
	}

	if config.KeepStdout || config.KeepStderr {
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
			for _, rule := range failedRules {
				if rule.Typ == RuleSuccess || rule.Typ == RuleFail {
					stdPrinter.Errf("Expected @%s\n", rule.Typ)
				}
				if rule.Expected != rule.Result {
					if rule.Operator == "=" {
						stdPrinter.Errf("Expected @%s [%s] but got: [%s]\n", rule.Typ, rule.Expected, rule.Result)
					} else if rule.Operator == "~" {
						stdPrinter.Errf("Expected @%s to contains [%s] but got: [%s]\n", rule.Typ, rule.Expected, rule.Result)
					}
				}
			}
			failedAssertionsReport := ""
			for _, rule := range failedRules {
				failedAssertionsReport = "@" + string(rule.Typ) + string(rule.Operator) + string(rule.Expected)
			}
			reportLog.WriteString(testTitle + "  => " + failedAssertionsReport)
		}
	}

	stdPrinter.Flush()

	if config.StopOnFailure && !success {
		reportAction(testSuite, nil)
	}

	return
}

func main() {
	exitCode := 1
	defer func() { os.Exit(exitCode) }()

	stdPrinter = printz.NewStandard()
	defer stdPrinter.Flush()

	if len(os.Args) == 1 {
		usage()
		return
	}

	actionInit := false
	actionReport := false
	actionTest := false

	var configs []*AssertionRule
	var rules []*AssertionRule

	name := ""
	var cmdArgs []string

	for _, arg := range os.Args[1:] {
		switch arg {

		case AssertionPrefix + string(ActionInit):
			actionInit = true
		case AssertionPrefix + string(ActionReport):
			actionReport = true
		default:
			rule, err := buildRule(arg)
			if err != nil {
				log.Fatal(err)
			}
			if rule != nil {
				if rule.Typ == ActionTest {
					actionTest = true
					name = rule.Expected
				} else if rule.Typ == ActionInit {
					actionInit = true
					name = rule.Expected
				} else if rule.Typ == ActionReport {
					actionReport = true
					name = rule.Expected
				} else if rule.Typ == ConfigStopOnFailure || rule.Typ == ConfigKeepOutputs || rule.Typ == ConfigKeepStdout || rule.Typ == ConfigKeepStderr || rule.Typ == ConfigIgnore || rule.Typ == ConfigFork {
					configs = append(configs, rule)
				} else {
					rules = append(rules, rule)
				}
			} else {
				cmdArgs = append(cmdArgs, arg)
			}
		}
	}

	if actionInit && actionReport {
		log.Fatalf("You must not declare both @init and @report actions !")
	}

	if actionInit || actionReport {
		if len(rules) > 0 {
			log.Fatalf("You must not declare assertions on @init and @report actions !")
		}
	}

	actionTest = !actionInit && !actionReport
	slashCount := strings.Count(name, "/")
	testSuiteName := ""
	testName := ""

	if actionTest {
		if len(rules) == 0 {
			//log.Fatalf("You must declare at least one assertion on test: [%s]", testName)
			rules = append(rules, AssertSuccess)
		}

		if len(cmdArgs) == 0 {
			log.Fatalf("You must supply a command argument")
		}

		if collections.Contains[*AssertionRule](&rules, AssertFail) && collections.Contains[*AssertionRule](&rules, AssertSuccess) {
			log.Fatalf("You must not declare @fail and @success on same test: [%s]", testName)
		}

		slashCount := strings.Count(name, "/")
		if slashCount > 1 {
			log.Fatalf("Your test name cannot contains more than one slash (/) !")
		} else if slashCount == 0 {
			testName = name
		} else if slashCount == 1 {
			splittedName := strings.Split(name, "/")
			testSuiteName = splittedName[0]
			testName = splittedName[1]
		}

	} else {
		if slashCount > 0 {
			log.Fatalf("Your test suite names cannot contains any slash (/) !")
		}
		testSuiteName = name
	}

	if actionInit {
		initAction(testSuiteName, configs)
		exitCode = 0
	} else if actionReport {
		reportAction(testSuiteName, configs)
		exitCode = 0
	} else if actionTest {
		success := testAction(testSuiteName, testName, cmdArgs, configs, rules)
		if success {
			exitCode = 0
		}
	}

}
