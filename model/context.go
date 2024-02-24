package model

import "mby.fr/utils/utilz"

type Context2 struct {
	Token  string
	Action string

	Config       Config
	TestOutcome  utilz.Optional[TestOutcome]
	SuiteOutcome utilz.Optional[SuiteOutcome]
}

func (c Context) TestCount() (n int) {
	// TODO
	return
}

func (c Context) IgnoreCount() (n int) {
	// TODO
	return
}

func (c Context) FailureCount() (n int) {
	// TODO
	return
}

func (c Context) ErrorCount() (n int) {
	// TODO
	return
}

func (c Context) IncrementTestCount() (err error) {
	// TODO
	return
}

func (c Context) IncrementIgnoreCount() (err error) {
	// TODO
	return
}

func (c Context) IncrementFailureCount() (err error) {
	// TODO
	return
}

func (c Context) IncrementErrorCount() (err error) {
	// TODO
	return
}

func (c Context) GlobalWorkDir() (err error) {
	// TODO
	return
}

func (c Context) SuiteWorkDir() (err error) {
	// TODO
	return
}

func (c Context) TestWorkDir() (err error) {
	// TODO
	return
}

func NewContext(token, action string) (err error) {
	// TODO
	return
}
