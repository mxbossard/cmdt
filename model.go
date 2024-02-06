package main

import (
	"time"

	"mby.fr/utils/cmdz"
	"mby.fr/utils/ptr"
)

type Action string

type Mapper[T any] func(expr string) (T, error)

type Validater[T any] func(rule, operator string, value T) error

type Configurer func(ctx Context) (Context, error)

type Asserter func(cmdz.Executer) (AssertionResult, error)

type ConfigScope int

const (
	Global ConfigScope = iota // How to use this ?
	Suite                     // can be placed on suite init only
	Test                      // can be placed on test or on suite to configure all tests
)

type Context0 struct {
	// TestSuite only
	TestSuite    string        `yaml:""`
	TestName     string        `yaml:""`
	Action       Action        `yaml:""`
	StartTime    time.Time     `yaml:""`
	SuiteTimeout time.Duration `yaml:""`
	ForkCount    int           `yaml:""`

	// Test or TestSuite
	Ignore        bool          `yaml:""`
	StopOnFailure bool          `yaml:""`
	KeepStdout    bool          `yaml:""`
	KeepStderr    bool          `yaml:""`
	Timeout       time.Duration `yaml:""`
	RunCount      int           `yaml:""`
	Parallel      int           `yaml:""`
}

type Context struct {
	// TestSuite only
	TestSuite    string        `yaml:""`
	TestName     string        `yaml:""`
	Action       Action        `yaml:""`
	StartTime    time.Time     `yaml:""`
	SuiteTimeout time.Duration `yaml:""`
	ForkCount    int           `yaml:""`

	// Test or TestSuite
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
	Name     string
	Operator string
	Expected string
	Asserter Asserter
}

type AssertionResult struct {
	Assertion Assertion
	Success   bool
	Value     any
	Message   string
}

func MergeContext(suiteContext, testContext Context) Context {
	suiteContext.TestName = testContext.TestName
	suiteContext.Action = testContext.Action
	if suiteContext.TestSuite == "" {
		suiteContext.TestSuite = testContext.TestSuite
	}

	if testContext.Ignore != nil {
		suiteContext.Ignore = testContext.Ignore
	}
	if suiteContext.Ignore == nil {
		suiteContext.Ignore = ptr.BoolPtr(false)
	}
	if testContext.StopOnFailure != nil {
		suiteContext.StopOnFailure = testContext.StopOnFailure
	}
	if suiteContext.StopOnFailure == nil {
		suiteContext.StopOnFailure = ptr.BoolPtr(false)
	}
	if testContext.KeepStdout != nil {
		suiteContext.KeepStdout = testContext.KeepStdout
	}
	if suiteContext.KeepStdout == nil {
		suiteContext.KeepStdout = ptr.BoolPtr(false)
	}
	if testContext.KeepStderr != nil {
		suiteContext.KeepStderr = testContext.KeepStderr
	}
	if suiteContext.KeepStderr == nil {
		suiteContext.KeepStderr = ptr.BoolPtr(false)
	}
	if testContext.Timeout.Nanoseconds() > 0 {
		suiteContext.Timeout = testContext.Timeout
	}
	if testContext.RunCount != 0 {
		suiteContext.RunCount = testContext.RunCount
	}
	if testContext.Parallel != 0 {
		suiteContext.Parallel = testContext.Parallel
	}

	return suiteContext
}
