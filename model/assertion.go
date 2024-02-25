package model

import (
	"time"
)

type Outcome string

const (
	PASSED  = Outcome("PASSED")
	FAILED  = Outcome("FAILED")
	ERRORED = Outcome("ERRORED")
	IGNORED = Outcome("IGNORED")
)

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
	Context          Context
	CmdTitle         string
	Duration         time.Duration
	Err              error
	Stdout           string
	Stderr           string
	AssertionResults []AssertionResult
}

type SuiteOutcome struct {
	Duration time.Duration
	Err      error
}
