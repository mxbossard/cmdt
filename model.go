package main

import (
	"time"

	"mby.fr/utils/cmdz"
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

type Context struct {
	TestSuite string `yaml:""`
	TestName  string `yaml:""`
	Action    Action `yaml:""`

	StartTime     time.Time     `yaml:""`
	SuiteTimeout  time.Duration `yaml:""`
	Ignore        bool          `yaml:""`
	StopOnFailure bool          `yaml:""`
	KeepStdout    bool          `yaml:""`
	KeepStderr    bool          `yaml:""`
	ForkCount     int           `yaml:""`
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
}

type AssertionResult struct {
	Assertion Assertion
	Success   bool
}
