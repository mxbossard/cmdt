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
	TIMEOUT = Outcome("TIMEOUT")
)

type Assertion struct {
	Rule
	Asserter Asserter
}

type AssertionResult struct {
	Assertion  Assertion
	Success    bool
	Value      any
	ErrMessage string
}

type TestOutcome struct {
	TestSuite         string
	Seq               int
	TestQualifiedName string
	CmdTitle          string
	ExitCode          int
	Err               error
	Duration          time.Duration
	Stdout            string
	Stderr            string
	Outcome           Outcome
	AssertionResults  []AssertionResult
}

type SuiteOutcome struct {
	TestSuite string
	//ExitCode    int
	Duration time.Duration
	//Err         error
	FailureReports []string
	TestCount      int
	PassedCount    int
	FailedCount    int
	ErroredCount   int
	IgnoredCount   int
	TooMuchCount   int
}
