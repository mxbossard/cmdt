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
	"log"
	"math/rand"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
	"strconv"

	"mby.fr/utils/cmdz"
	"mby.fr/utils/collections"
	"mby.fr/utils/trust"
	"mby.fr/utils/printz"
	"mby.fr/utils/ansi"
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

type RuleType string

type AssertionRule struct {
	Typ      RuleType
	Operator string
	Value    string
}

var (
	AssertionPrefix   = "@"
	ContextEnvVarName = "__CMDTEST_CONTEXT_KEY_"

	ActionInit   = "init"
	ActionReport = "report"

	ConfigStopOnFailure = "stopOnFailure"
	ConfigLogOnSuccess  = "logOnSuccess"
	ConfigParallel      = "parallel"

	RuleFail    = RuleType("fail")
	RuleSuccess = RuleType("success")
	RuleRc      = RuleType("rc")
	RuleStdout  = RuleType("stdout")
	RuleStderr  = RuleType("stderr")
	RuleExists  = RuleType("exists")

	AssertFail    = &AssertionRule{Typ: RuleFail}
	AssertSuccess = &AssertionRule{Typ: RuleSuccess}

	ConfigFilename   = "config.yaml"
	SequenceFilename = "seq.txt"
	StdoutFilename   = "stdout.log"
	StderrFilename   = "stderr.log"
	ReportFilename   = "report.log"

	messageColor 	= ansi.Cyan
	testColor 		= ansi.Yellow
	successColor 	= ansi.Green
	failureColor 	= ansi.Red
	errorColor 		= ansi.Red
)

var stdPrinter printz.Printer

// var assertionPattern = regexp.MustCompile("^@([a-zA-Z]+)$")
var assertionRulePattern = regexp.MustCompile("^" + AssertionPrefix + "([a-zA-Z]+)(?:([=~])(.+))?$")

/*
func buildAssertion(arg string) (assert Assertion) {
	submatch := assertionPattern.FindStringSubmatch(arg)
	if submatch != nil {
		name := submatch[1]
		assert = Assertion(name)
	}
	return
}
*/

func buildRule(arg string) (rule *AssertionRule, err error) {
	submatch := assertionRulePattern.FindStringSubmatch(arg)
	if submatch != nil {
		typ := RuleType(submatch[1])
		operator := submatch[2]
		value := submatch[3]

		switch typ {
		case RuleRc:
			// assert rc rule value is an integer
			var i int
			i, err = strconv.Atoi(value)
			if err != nil || i < 0 || i > 255 {
				err = fmt.Errorf("rc rule value must be an integer >= 0 && <= 255")
				return
			}
		}
		rule = &AssertionRule{typ, operator, value}
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

func loadUniqKey(testSuite string) string {
	for _, env := range os.Environ() {
		if strings.HasPrefix(env, ContextEnvVarName+strings.ToUpper(testSuite)+"=") {
			splitted := strings.Split(env, "=")
			key := strings.Join(splitted[1:], "")
			//fmt.Printf("Found key: %s", key)
			return key
		}
	}
	log.Fatalf("Cannot found context env var. You must export init action like this : eval $( cmdt @init )")
	return ""
}

func tmpDirectoryPath(uniqKey string) string {
	path := filepath.Join(os.TempDir(), "cmdtest."+uniqKey)
	return path
}

func storeConfig(uniqKey string, configs []*AssertionRule) {
	tmpDir := tmpDirectoryPath(uniqKey)
	_ = tmpDir
}

func loadConfig(uniqKey string) (configs []*AssertionRule) {
	tmpDir := tmpDirectoryPath(uniqKey)
	_ = tmpDir
	return
}

func cmdLogFilesPathes(uniqKey string, seq int) (string, string, string) {
	tmpDir := tmpDirectoryPath(uniqKey)
	testDir := filepath.Join(tmpDir, "test-"+fmt.Sprint(seq))
	stdoutFilepath := filepath.Join(testDir, StdoutFilename)
	stderrFilepath := filepath.Join(testDir, StderrFilename)
	reportFilepath := filepath.Join(testDir, ReportFilename)
	return stdoutFilepath, stderrFilepath, reportFilepath
}

func initSeq(uniqKey string) {
	tmpDir := tmpDirectoryPath(uniqKey)
	seqFilepath := filepath.Join(tmpDir, SequenceFilename)
	err := os.WriteFile(seqFilepath, []byte("0"), 0600)
	if err != nil {
		log.Fatalf("Cannot initialize seq file ! Error: %s", err)
	}
}

func incrementSeq(uniqKey string) (seq int64) {
	// return an increment for test indexing
	tmpDir := tmpDirectoryPath(uniqKey)
	seqFilepath := filepath.Join(tmpDir, SequenceFilename)

	file, err := os.OpenFile(seqFilepath, os.O_RDWR|os.O_CREATE, 0600)
	if err != nil {
		log.Fatalf("Cannot open seq file ! Error: %s", err)
	}
	defer file.Close()
	/*
		buf := make([]byte, 32)
		n, err := file.Read(buf)
		if err != nil {
			log.Fatalf("Cannot read seq file ! Error: %s", err)
		}
		seq, err = strconv.ParseInt(string(buf[n]), 10, 64)
	*/
	// FIXME: bad seq reading or writing !
	_, err = fmt.Fscanln(file, &seq)
	if err != nil {
		log.Fatalf("Cannot read seq file as an integer ! Error: %s", err)
	}

	seq++
	_, err = file.WriteString(fmt.Sprint(seq))
	if err != nil {
		log.Fatalf("Cannot write seq file ! Error: %s", err)
	}

	fmt.Printf("Incremented seq: %d\n", seq)
	return seq
}

func testCount(uniqKey string) int {
	// TODO: return the count of test run
	return -1
}

func initAction(testSuite string, configs []*AssertionRule) {
	// forge a uniq key
	uniqKey := forgeUniqKey(testSuite)
	// init the tmp directory
	tmpDir := tmpDirectoryPath(uniqKey)
	err := os.MkdirAll(tmpDir, 0700)
	if err != nil {
		log.Fatalf("Unable to create temp dir: %s ! Error: %s", tmpDir, err)
	}
	// store config
	storeConfig(uniqKey, configs)
	initSeq(uniqKey)
	// print export the key
	fmt.Printf("export %s%s=%s\n", ContextEnvVarName, strings.ToUpper(testSuite), uniqKey)
	if testSuite == "" {
		testSuite = "default"
	}
	stdPrinter.ColoredErrf(messageColor, "Initialized new [%s] test suite.\n", testSuite)
}

func reportAction(testSuite string, configs []*AssertionRule) {
	// load context
	uniqKey := loadUniqKey(testSuite)
	context := loadConfig(uniqKey)
	// TODO: print report
	if testSuite == "" {
		testSuite = "default"
	}
	stdPrinter.ColoredErrf(messageColor, "Reporting [%s] test suite ...\n", testSuite)
	fmt.Printf("context: %v\n", context)
	// TODO: clear context (tmp dir)
}

func testAction(testSuite, name string, command []string, configs, rules []*AssertionRule) (success bool) {
	success = false

	// load context
	uniqKey := loadUniqKey(testSuite)
	context := loadConfig(uniqKey)
	seq := incrementSeq(uniqKey)

	testName := name
	if testSuite != "" {
		testName = fmt.Sprintf("%s/%s", testSuite, name)
	}

	fmt.Printf("context: %v\n", context)
	fmt.Printf("will execute cmd (%d): %s\n", seq, command)
	stdPrinter.ColoredErrf(testColor, "Testing: [%s]... ", testName)
	stdPrinter.Flush()

	cmd := cmdz.Cmd(command[0], command[1:]...)
	exitCode, err := cmd.BlockRun()

	// TODO: merge configs
	// TODO: launch command with config
	// TODO: perform all assertions
	for _, rule := range rules {
		switch rule.Typ {
		case RuleSuccess:
			success = exitCode == 0
		case RuleFail:
			success = exitCode > 0
		case RuleRc:
			expectedRc, _ := strconv.Atoi(rule.Value)
			success = exitCode == expectedRc
		case RuleStdout:
			success = rule.Value == cmd.StdoutRecord()
		case RuleStderr:
			success = rule.Value == cmd.StderrRecord()
		case RuleExists:
			// TODO
		}
	}

	if success {
		stdPrinter.ColoredErrf(successColor, "ok\n")
	} else {
		stdPrinter.ColoredErrf(failureColor, "ko\n")
	}
	stdPrinter.Flush()

	if err != nil {
		fmt.Printf("Error: %s\n", err)
	}
	
	fmt.Printf("ExitCode=%d\n", exitCode)
	return
}

func main() {
	exitCode := 1
	defer func() { os.Exit(exitCode) }()

	stdPrinter = printz.NewStandard()
	defer stdPrinter.Flush()
	actionInit := false
	actionReport := false

	var configs []*AssertionRule
	var rules []*AssertionRule

	name := ""
	var cmdArgs []string

	for pos, arg := range os.Args[1:] {
		switch arg {
		case AssertionPrefix + ActionInit:
			actionInit = true
		case AssertionPrefix + ActionReport:
			actionReport = true
		case AssertionPrefix + ConfigStopOnFailure:
			rule, err := buildRule(arg)
			if err != nil {
				log.Fatal(err)
			}
			configs = append(configs, rule)
		case AssertionPrefix + ConfigStopOnFailure:
			rule, err := buildRule(arg)
			if err != nil {
				log.Fatal(err)
			}
			configs = append(configs, rule)
		default:
			rule, err := buildRule(arg)
			if err != nil {
				log.Fatal(err)
			}
			if rule != nil {
				rules = append(rules, rule)

			} else if pos == 0 {
				name = arg
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

	if collections.Contains[*AssertionRule](&rules, AssertFail) && collections.Contains[*AssertionRule](&rules, AssertSuccess) {
		log.Fatalf("You must not declare @fail and @success on same test !")
	}

	actionTest := !actionInit && !actionReport
	slashCount := strings.Count(name, "/")
	testSuiteName := ""
	testName := ""

	if actionTest {
		slashCount := strings.Count(name, "/")
		if name == "" {
			log.Fatalf("You must supply a test name as first arg !")
		} else if slashCount > 1 {
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
	} else if actionReport {
		reportAction(testSuiteName, configs)
	} else if actionTest {
		success := testAction(testSuiteName, testName, cmdArgs, configs, rules)
		if success {
			exitCode = 0
		}
	}

}
