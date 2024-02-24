package model

import (
	"time"

	"mby.fr/utils/cmdz"
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
