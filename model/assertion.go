package model

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
