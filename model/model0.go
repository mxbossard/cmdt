package model

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"mby.fr/utils/ansi"
	"mby.fr/utils/cmdz"
)

type Action string

const (
	GlobalAction = Action("global")
	InitAction   = Action("init")
	TestAction   = Action("test")
	ReportAction = Action("report")
)

type Mapper[T any] func(expr, op string) (T, error)

type Validater[T any] func(rule Rule, value T) error

//type Configurer func(ctx Context) (Context, error)

type Asserter func(cmdz.Executer) (AssertionResult, error)

const (
	ContextTokenEnvVarName        = "__CMDT_TOKEN"
	DefaultTestSuiteName          = "main"
	GlobalConfigTestSuiteName     = "__global"
	DefaultContainerImage         = "busybox"
	DefaultContainerDirtiesPolicy = "beforeSuite"
)

const (
	NamePattern = "[a-zA-Z][^/]*[a-zA-Z0-9]"
)

var (
	TempDirPrefix           = "cmdtest"
	ContextFilename         = "context.yaml"
	TestSequenceFilename    = "test-seq.txt"
	PassedSequenceFilename  = "passed-seq.txt"
	FailedSequenceFilename  = "failed-seq.txt"
	IgnoredSequenceFilename = "ignored-seq.txt"
	ErroredSequenceFilename = "errored-seq.txt"
	TooMuchSequenceFilename = "tooMuch-seq.txt"
	StdoutFilename          = "stdout.log"
	StderrFilename          = "stderr.log"
	ReportFilename          = "report.log"

	MessageColor = ansi.HiPurple
	TestColor    = ansi.HiCyan
	SuccessColor = ansi.BoldGreen
	FailureColor = ansi.BoldRed
	ReportColor  = ansi.Yellow
	WarningColor = ansi.BoldHiYellow
	ErrorColor   = ansi.Red
)

var (
	DefaultTestTimeout, _ = time.ParseDuration("24h")
	AbsNamePattern        = fmt.Sprintf("(%s/)?(%s)?", NamePattern, NamePattern)
	NameRegexp            = regexp.MustCompile("^" + NamePattern + "$")
	AbsNameRegexp         = regexp.MustCompile("^" + AbsNamePattern + "$")
)

var (
	Actions = []RuleDefinition{ruleDef("global", ""), ruleDef("init", "", "="), ruleDef("test", "", "="),
		ruleDef("report", "", "=")}
	// Global config available to global
	GlobalConfigs = []RuleDefinition{ruleDef("fork", "="), ruleDef("suiteTimeout", "="), ruleDef("prefix", "=")}
	// Suite config available to suite
	SuiteConfigs = append(GlobalConfigs, []RuleDefinition{
		ruleDef("exportToken", ""), ruleDef("printToken", "")}...)
	// Test config available at all levels (global, suite and test)
	TestConfigs = []RuleDefinition{ruleDef("before", "="), ruleDef("after", "="), ruleDef("ignore", "", "="),
		ruleDef("stopOnFailure", "", "="), ruleDef("keepStdout", "", "="), ruleDef("keepStderr", "", "="),
		ruleDef("keepOutputs", "", "="), ruleDef("quiet", "", "="), ruleDef("timeout", "="),
		ruleDef("parallel", "="), ruleDef("runCount", "="), ruleDef("mock", "=", ":"),
		ruleDef("container", "", "="), ruleDef("dirtyContainer", "=")}
	// Config of test flow (init -> test -> report)
	FlowConfigs = []RuleDefinition{ruleDef("token", "="), ruleDef("verbose", "", "="),
		ruleDef("debug", "", "="), ruleDef("failuresLimit", "=")}
	Assertions = []RuleDefinition{ruleDef("success", ""), ruleDef("fail", ""), ruleDef("exit", "="),
		ruleDef("cmd", "="), ruleDef("exists", "="),
		ruleDef("stdout", "=", ":", "~", "!=", "!:", "!~", "@=", "@:"),
		ruleDef("stderr", "=", ":", "~", "!=", "!:", "!~", "@=", "@:")}
	Concatenables = []RuleDefinition{
		ruleDef("init", "="), ruleDef("test", "="), ruleDef("report", "="),
		ruleDef("before", "="), ruleDef("after", "="),
		ruleDef("cmd", "="), ruleDef("exists", "="),
		ruleDef("stdout", "=", ":", "~", "!=", "!:", "!~", "@=", "@:"),
		ruleDef("stderr", "=", ":", "~", "!=", "!:", "!~", "@=", "@:"),
	}
)

type Rule struct {
	Prefix   string
	Name     string
	Op       string
	Expected string
}

type RuleKey struct {
	Name, Op string
}

func (r RuleKey) String() string {
	return fmt.Sprintf("%s%s", r.Name, r.Op)
}

type RuleDefinition struct {
	Name string
	Ops  []string
}

func ruleDef(name string, ops ...string) (r RuleDefinition) {
	r.Name = name
	r.Ops = ops
	return
}

func MatchRuleDef(rulePrefix, ruleStatement string, ruleDefs ...RuleDefinition) bool {
	for _, def := range ruleDefs {
		for _, op := range def.Ops {
			if strings.HasPrefix(ruleStatement, rulePrefix+def.Name+op) {
				return true
			}
		}
	}
	return false
}

func IsRuleOfKind(ruleDefs []RuleDefinition, r Rule) (ok bool, err error) {
	ok = false
	var expectedOperators []string
	for _, ruleDef := range ruleDefs {
		if r.Name == ruleDef.Name {
			expectedOperators = append(expectedOperators, ruleDef.Ops...)
			for _, op := range ruleDef.Ops {
				if r.Op == op {
					ok = true
					return
				}
			}
		}
	}

	if len(expectedOperators) > 0 {
		// name matched but not operator
		err = fmt.Errorf("rule %s expect one of operators: %s", r.Name, expectedOperators)
	}
	return
}

func NormalizeDurationInSec(d time.Duration) (duration string) {
	duration = fmt.Sprintf("%.3f s", float32(d.Milliseconds())/1000)
	return
}
