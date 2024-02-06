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
	"os"
	"path/filepath"
	"regexp"

	"mby.fr/utils/ansi"
	"mby.fr/utils/printz"
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

*/

/*
Done:
- @timeout=N
- @cmd="CMD_WITH_ARGS_THAT_SHOULD_BE_OK_IF_OR_FAIL_TEST"
- @exists=FILEPATH,PERMS,OWNERS
- Improve cmd log delimiting args:
        no spaces: CMD ARG_1 ARG_2 ARG_N
        some spaces: <[ CMD AVEC ESPACE, ARG_1, ARG AVEC ESPACE, ARG_N ]>
        autre idée ??? CMD ARG_1 'ARG 2' ... <|ARG N|>  or 'CMD WITH SPACE' ARG_1 'ARG 2' ... ARG_N Possible separators: simple quotte ; <| |> ;
- pass file content by filepath to @stdout @stderr : @stdout=@FILEPATH


TODO:
Bugs:

Features :
- possibilité de passer un scénario ligne à ligne dans le stdin de cmdtest
	- cmdt cmd arg1 argN @scenario=filepath
	- pour chaque ligne du scenario concat la ligne du scenario avec les arguments fournit en paramétre de cmdt
- tester cmdt avec cmdt ?
	- cmdt @prefix=% cmdt true %success => should success
	- cmdt @prefix=% cmdt false %fail => should success
	- cmdt @prefix=% cmdt false @fail %success => should success
- @prefix= change @ char ??? attention à la sécurité ça pourrait etre galere

- @mock des appels de commande
- @before=TEST_SUITE CMD ARG_1 ARG_2 ... ARG_N => execute CMD before each test
- @after=TEST_SUITE CMD ARG_1 ARG_2 ... ARG_N => execute CMD after each test

- @runCount=N + @parallel=FORK_COUNT (min, max, median exec time) run in separate context or in same context (before/after) ?

- @fork=5 global config only by default instead of @parallel. Fork = 5 increment and decrement a seq file
- Clean old temp dir (older than 2 min ?)
- Run en option les tests dans un conteneur => plus facile de mocké, meilleur reproductibilité des tests

- may chroot be interesting for tests ?
- mock web spawning a web server ?
- test port opening if daemon ; test sending data on port ???

## Idées pour la commande de mock
    insérer dans le $PATH un repertoire temp binMocked
    Pour chaque commande mockée on place un lien symbolique de la commande mocké vers le mock dans le repertoire binMocked
    il nous faut un programme mock qui consomme tous les args, compte les appels avec les args et répond de la manière dont il est configuré
    mock config :
        Exact call @mock="CMD,ARG_1,ARG_2,ARG_N;stdin=baz;exit=0;stdout=foo;stderr=bar" Must receive exactly stdin and specified args in order not more
        Exact incomplete call @mock="CMD,ARG_1,ARG_2,ARG_N,*;stdin=baz;exit=0;stdout=foo;stderr=bar" Must receive exactly stdin and specified args in order then more args
        Contains call @mock~"CMD,ARG_1,ARG_2,ARG_N;stdin=baz;exit=0;stdout=foo;stderr=bar" Must receive exactly stdin and specified args in various order not more
        Default exit code @mock="CMD,*;exit=1"
*/

type RuleType string

type AssertionRule struct {
	Typ      RuleType
	Operator string
	Expected string
	Result   string
}

var (
	//AssertionPrefix      = "@"
	ContextEnvVarName    = "__CMDTEST_CONTEXT_KEY_"
	DefaultTestSuiteName = "_default"

	/*
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
	*/

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

// var assertionRulePattern = regexp.MustCompile("^" + AssertionPrefix + "([a-zA-Z]+)([=~])?(.+)?$")
var testSuiteNameSanitizerPattern = regexp.MustCompile("[^a-zA-Z0-9]")

//var logger = logz.Default("cmdtest", 0)

func usage() {
	cmd := filepath.Base(os.Args[0])
	stdPrinter.Errf("cmdtest tool is usefull to test various scripts cli and command behaviors.\n")
	stdPrinter.Errf("You must initialize a test suite (%[1]s @init) before running tests and then report the test (%[1]s @report).\n", cmd)
	stdPrinter.Errf("usage: \t%s @init[=TEST_SUITE_NAME] [@CONFIG_1] ... [@CONFIG_N] \n", cmd)
	stdPrinter.Errf("usage: \t%s <COMMAND> [ARG_1] ... [ARG_N] [@CONFIG_1] ... [@CONFIG_N] [@ASSERTION_1] ... [@ASSERTION_N]\n", cmd)
	stdPrinter.Errf("usage: \t%s @report[=TEST_SUITE_NAME] \n", cmd)
	stdPrinter.Errf("\tCONFIG available: @ignore @stopOnFailure @keepStdout @keepStderr @keepOutputs @timeout=Duration @fork=N\n")
	stdPrinter.Errf("\tCOMMAND and ARGs: the command on which to run tests\n")
	stdPrinter.Errf("\tASSERTIONs available: @fail @success @exit=N @stdout= @stdout~ @stderr= @stderr~ @cmd= @exists=\n")
}

func main() {
	ProcessArgs(os.Args)
}
