package main

import (
	"fmt"
	"strconv"
	"strings"

	"mby.fr/utils/cmdz"
)

func IsRule(s string) bool {
	return strings.HasPrefix(s, AssertionPrefix)
}

func SplitRuleExpr(ruleExpr string) (ok bool, name string, operator string, value string) {
	ok = false
	//if IsRule(ruleExpr) {
	submatch := assertionRulePattern.FindStringSubmatch(ruleExpr)
	if submatch != nil {
		ok = true
		name = submatch[1]
		operator = submatch[2]
		value = submatch[3]
	}
	//}
	return
}

func NoMapper(s string) (v any, err error) {
	return
}

func IntMapper(s string) (v int, err error) {
	v, err = strconv.Atoi(s)
	return
}

func Translate[T any](rule, operator, value string, m Mapper[T], validaters ...Validater[T]) (val T, err error) {
	val, err = m(value)
	if err != nil {
		return
	}
	for _, v := range validaters {
		err = v(rule, operator, val)
		if err != nil {
			return
		}
	}

	return
}

func IntValueValidater(min, max int) Validater[int] {
	return func(rule, op string, n int) (err error) {
		if n < min || n > max {
			err = fmt.Errorf("rule %s%s value must be an integer >= %d and <= %d", AssertionPrefix, rule, min, max)
		}
		return
	}
}

func NoOperatorValidater[T any](rule, op string, v T) (err error) {
	if op != "" {
		err = fmt.Errorf("rule %s%s cannot have a value nor an operator", AssertionPrefix, rule)
	}
	return
}

func EqualOperatorValidater[T any](rule, op string, v T) (err error) {
	if op != "=" {
		err = fmt.Errorf("rule %s%s operator must be '='", AssertionPrefix, rule)
	}
	return
}

/*
func BuildAction(ruleExpr string) (ok bool, c Configurer, err error) {
	var name, operator, value string
	ok, name, operator, value = SplitRuleExpr(ruleExpr)
	if ok {
		switch name {
		case "init":

		case "report":
		case "test":

		default:
			ok = false
		}
	}
	return
}
*/

func BuildConfig(ruleExpr string) (ok bool, c Configurer, err error) {
	var name, operator, value string
	ok, name, operator, value = SplitRuleExpr(ruleExpr)
	if ok {
		switch name {
		case "init":
		case "report":
		case "test":

		case "fork":
			c = func(ctx Context) (ctxOut Context, err error) {
				var i int
				ctxOut = ctx
				i, err = Translate(name, operator, value, IntMapper, EqualOperatorValidater[int], IntValueValidater(1, 5))
				ctxOut.ForkCount = i
				return
			}
		case "suiteTimeout":
		case "before":
		case "after":

		case "ignore":
		case "stopOnFailure":
		case "keepStdout":
		case "keepStderr":
		case "keepOutputs":
		case "runCount":
		case "parallel":
		default:
			ok = false
		}
	}
	return
}

func BuildAssertion(ruleExpr string) (ok bool, a Asserter, err error) {
	var name, operator, value string
	ok, name, operator, value = SplitRuleExpr(ruleExpr)
	if ok {
		switch name {
		case "success":
			a = func(cmd cmdz.Executer) (res AssertionResult, err error) {
				assert := Assertion{Name: name, Operator: operator, Expected: value}
				_, err = Translate(name, operator, value, NoMapper, NoOperatorValidater[any])
				res = AssertionResult{Assertion: assert, Success: cmd.ExitCode() == 0}
				return
			}
		case "fail":
		case "exit":
		case "stdout":
		case "stderr":
		case "timeout":
		case "exists":

		default:
			ok = false
		}
	}
	return
}
