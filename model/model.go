package model

import (
	"fmt"
	"strings"
	"time"

	"mby.fr/utils/cmdz"
)

type Action string

type Mapper[T any] func(expr, op string) (T, error)

type Validater[T any] func(rule Rule, value T) error

type Asserter func(cmdz.Executer) (AssertionResult, error)

type Rule struct {
	Prefix   string
	Name     string
	Op       string
	Expected string
}

type RuleKey struct {
	Name, Op string
}

func (r RuleKey) String() string {
	return fmt.Sprintf("%s%s", r.Name, r.Op)
}

type RuleDefinition struct {
	Name string
	Ops  []string
}

func ruleDef(name string, ops ...string) (r RuleDefinition) {
	r.Name = name
	r.Ops = ops
	return
}

func MatchRuleDef(rulePrefix, ruleStatement string, ruleDefs ...RuleDefinition) bool {
	for _, def := range ruleDefs {
		for _, op := range def.Ops {
			if strings.HasPrefix(ruleStatement, rulePrefix+def.Name+op) {
				return true
			}
		}
	}
	return false
}

func IsRuleOfKind(ruleDefs []RuleDefinition, r Rule) (ok bool, err error) {
	ok = false
	var expectedOperators []string
	for _, ruleDef := range ruleDefs {
		if r.Name == ruleDef.Name {
			expectedOperators = append(expectedOperators, ruleDef.Ops...)
			for _, op := range ruleDef.Ops {
				if r.Op == op {
					ok = true
					return
				}
			}
		}
	}

	if len(expectedOperators) > 0 {
		// name matched but not operator
		err = fmt.Errorf("rule %s expect one of operators: %s", r.Name, expectedOperators)
	}
	return
}

func NormalizeDurationInSec(d time.Duration) (duration string) {
	duration = fmt.Sprintf("%.3f s", float32(d.Milliseconds())/1000)
	return
}
