package main

import (
	"fmt"
	"time"

	"mby.fr/utils/cmdz"
	"mby.fr/utils/ptr"
)

type Action string

type Mapper[T any] func(expr, op string) (T, error)

type Validater[T any] func(rule Rule, value T) error

type Configurer func(ctx Context) (Context, error)

type Asserter func(cmdz.Executer) (AssertionResult, error)

type ConfigScope int

const (
	DefaultRulePrefix         = "@"
	ContextTokenEnvVarName    = "__CMDT_TOKEN"
	DefaultTestSuiteName      = "main"
	GlobalConfigTestSuiteName = "__global"
)

const (
	Global ConfigScope = iota // How to use this ?
	Suite                     // can be placed on suite init only
	Test                      // can be placed on test or on suite to configure all tests
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
		ruleDef("keepOutputs", "", "="), ruleDef("silent", "", "="), ruleDef("timeout", "="),
		ruleDef("parallel", "="), ruleDef("runCount", "="), ruleDef("mock", "=", ":")}
	// Config of test flow (init -> test -> report)
	FlowConfigs = []RuleDefinition{ruleDef("token", "=")}
	Assertions  = []RuleDefinition{ruleDef("success", ""), ruleDef("fail", ""), ruleDef("exit", "="),
		ruleDef("cmd", "="), ruleDef("stdout", "=", ":", "~", "!=", "!:", "!~"),
		ruleDef("stderr", "=", ":", "~", "!=", "!:", "!~"), ruleDef("exists", "=")}
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

type Context struct {
	// TestSuite only
	Token        string        `yaml:""`
	Prefix       string        `yaml:""`
	TestSuite    string        `yaml:""`
	TestName     string        `yaml:""`
	Action       Action        `yaml:""`
	StartTime    time.Time     `yaml:""`
	SuiteTimeout time.Duration `yaml:""`
	ForkCount    int           `yaml:""`

	// Test or TestSuite
	PrintToken    bool
	ExportToken   bool
	ReportAll     bool
	Silent        *bool         `yaml:""`
	Ignore        *bool         `yaml:""`
	StopOnFailure *bool         `yaml:""`
	KeepStdout    *bool         `yaml:""`
	KeepStderr    *bool         `yaml:""`
	Timeout       time.Duration `yaml:""`
	RunCount      int           `yaml:""`
	Parallel      int           `yaml:""`
	Mocks         []CmdMock     `yaml:""`
}

type Config struct {
	Name  string
	Scope ConfigScope
	Value string
}

type Assertion struct {
	Rule
	Asserter Asserter
}

type AssertionResult struct {
	Assertion Assertion
	Success   bool
	Value     any
	Message   string
}

type CmdMock struct {
	Op               string
	Cmd              string
	Args             []string
	Stdin            string
	Stdout           string
	Stderr           string
	ExitCode         int
	Delegate         bool
	OnCallCmdAndArgs []string
}

func MergeContext(baseContext, overridingContext Context) Context {
	baseContext.Token = overridingContext.Token
	baseContext.Prefix = overridingContext.Prefix
	baseContext.TestName = overridingContext.TestName
	baseContext.Action = overridingContext.Action
	if overridingContext.StartTime.Nanosecond() != 0 {
		baseContext.StartTime = overridingContext.StartTime
	}

	if baseContext.TestSuite == "" {
		baseContext.TestSuite = overridingContext.TestSuite
	}

	if overridingContext.Ignore != nil {
		baseContext.Ignore = overridingContext.Ignore
	}
	if baseContext.Ignore == nil {
		baseContext.Ignore = ptr.BoolPtr(false)
	}
	if overridingContext.StopOnFailure != nil {
		baseContext.StopOnFailure = overridingContext.StopOnFailure
	}
	if baseContext.StopOnFailure == nil {
		baseContext.StopOnFailure = ptr.BoolPtr(false)
	}
	if overridingContext.KeepStdout != nil {
		baseContext.KeepStdout = overridingContext.KeepStdout
	}
	if baseContext.KeepStdout == nil {
		baseContext.KeepStdout = ptr.BoolPtr(false)
	}
	if overridingContext.KeepStderr != nil {
		baseContext.KeepStderr = overridingContext.KeepStderr
	}
	if baseContext.KeepStderr == nil {
		baseContext.KeepStderr = ptr.BoolPtr(false)
	}
	if overridingContext.Timeout.Nanoseconds() > 0 {
		baseContext.Timeout = overridingContext.Timeout
	}
	if overridingContext.RunCount != 0 {
		baseContext.RunCount = overridingContext.RunCount
	}
	if overridingContext.Parallel != 0 {
		baseContext.Parallel = overridingContext.Parallel
	}
	if overridingContext.Silent != nil {
		baseContext.Silent = overridingContext.Silent
	}
	baseContext.Mocks = append(baseContext.Mocks, overridingContext.Mocks...)

	return baseContext
}

func ruleDef(name string, ops ...string) (r RuleDefinition) {
	r.Name = name
	r.Ops = ops
	return
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
