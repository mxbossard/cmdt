package model

import (
	"time"

	"mby.fr/utils/cmdz"
)

type Rule struct {
	Prefix   string
	Name     string
	Op       string
	Expected string
}

type Asserter func(cmdz.Executer) (AssertionResult, error)

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

type TestOutcome struct {
	Cmd              cmdz.Executer
	Duration         time.Duration
	Err              error
	AssertionResults []AssertionResult
}

type SuiteOutcome struct {
	Duration time.Duration
	Err      error
}
