package model

type Outcome string

const (
	PASSED  = Outcome("PASSED")
	FAILED  = Outcome("FAILED")
	ERRORED = Outcome("ERRORED")
	IGNORED = Outcome("IGNORED")
	UNKNOWN = Outcome("UNKNOWN")
	TIMEOUT = Outcome("TIMEOUT")
)

type Assertion struct {
	Rule
	Asserter Asserter
}

type AssertionResult struct {
	Rule
	//Assertion  Assertion
	Success    bool
	Value      any
	ErrMessage string
}

func NewAssertionResult(prefix, name, op, expected string, value any, success bool,
	errorMsg string) AssertionResult {
	rule := Rule{
		Prefix:   prefix,
		Name:     name,
		Op:       op,
		Expected: expected,
	}
	res := AssertionResult{
		Rule:       rule,
		Success:    success,
		Value:      value,
		ErrMessage: errorMsg,
	}

	return res
}
