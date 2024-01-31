package main

import (
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
