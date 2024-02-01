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
	"io/fs"
	"log"
	"math/rand"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"mby.fr/utils/ansi"
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
- @timeout=N
- @exists=FILEPATH,PERMS,OWNERS
- @prefix= change @ char ??? attention à la sécurité ça pourrait etre galere
- @runCount=N + @parallel=FORK_COUNT (min, max, median exec time)
- @cmd="CMD_WITH_ARGS_THAT_SHOULD_BE_OK_IF_OR_FAIL_TEST"
- @fork=5 global config only by default instead of @parallel. Fork = 5 increment and decrement a seq file
- Clean old temp dir (older than 2 min ?)
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

var (
	AssertionPrefix      = "@"
	ContextEnvVarName    = "__CMDTEST_CONTEXT_KEY_"
	DefaultTestSuiteName = "_default"

	ActionInit   = RuleType("init")
	ActionReport = RuleType("report")
	ActionTest   = RuleType("test")

	GlobalFork   = RuleType("fork")
	GlobalPrefix = RuleType("prefix")

	ConfigStopOnFailure = RuleType("stopOnFailure")
	ConfigKeepStdout    = RuleType("keepStdout")
	ConfigKeepStderr    = RuleType("keepStderr")
	ConfigKeepOutputs   = RuleType("keepOutputs")
	ConfigIgnore        = RuleType("ignore")
	ConfigRunCount      = RuleType("runCount")
	ConfigParallel      = RuleType("parallel")

	RuleFail    = RuleType("fail")
	RuleSuccess = RuleType("success")
	RuleExit    = RuleType("exit")
	RuleStdout  = RuleType("stdout")
	RuleStderr  = RuleType("stderr")
	RuleExists  = RuleType("exists")
	RuleTimeout = RuleType("timeout")
	RuleCmd     = RuleType("cmd")

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
var testSuiteNameSanitizerPattern = regexp.MustCompile("[^a-zA-Z0-9]")

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

func main() {
	stdPrinter = printz.NewStandard()
	defer stdPrinter.Flush()

	DoSomething(os.Args)
}
