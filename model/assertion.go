package model

import (
	"time"

	"mby.fr/utils/cmdz"
)

type Outcome string

const (
	PASSED  = Outcome("PASSED")
	FAILED  = Outcome("FAILED")
	ERRORED = Outcome("ERRORED")
	IGNORED = Outcome("IGNORED")
)

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
