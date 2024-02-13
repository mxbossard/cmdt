package main

import (
	"time"

	"mby.fr/utils/cmdz"
	"mby.fr/utils/ptr"
)

type Action string

type Mapper[T any] func(expr string) (T, error)

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

type Rule struct {
	Prefix   string
	Name     string
	Op       string
	Expected string
}

type RuleKey struct {
	Name, Op string
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

	return baseContext
}
