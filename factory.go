package main

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"mby.fr/utils/cmdz"
)

const (
	NamePattern    = "[a-zA-Z][a-zA-Z0-9-_ .]*[a-zA-Z0-9]"
	AbsNamePattern = "(?:(" + NamePattern + ")/)?" + NamePattern
)

var (
	NameRegexp    = regexp.MustCompile(NamePattern)
	AbsNameRegexp = regexp.MustCompile(AbsNamePattern)
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

func BoolMapper(s string) (v bool, err error) {
	if s == "true" || s == "" {
		v = true
	} else if s == "false" {
		v = false
	} else {
		err = fmt.Errorf("bool rule value must be true or false")
	}

	return
}

func DurationMapper(s string) (v time.Duration, err error) {
	return time.ParseDuration(s)
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

func TestNameValidater(rule, op string, v string) (err error) {
	switch rule {
	case "init", "report":
		if !NameRegexp.MatchString(v) {
			err = fmt.Errorf("name %s does not match expected pattern: %s", v, NamePattern)
		}
	case "test":
		if !AbsNameRegexp.MatchString(v) {
			err = fmt.Errorf("name %s does not match expected pattern: %s", v, AbsNamePattern)
		}
	}
	return
}

func BooleanValidater(rule, op string, v bool) (err error) {
	err = EqualOperatorValidater(rule, op, v)
	return
}

func Validate[T any](rule, operator string, val T, validaters ...Validater[T]) (err error) {
	for _, v := range validaters {
		err = v(rule, operator, val)
		if err != nil {
			return
		}
	}
	return
}

func Translate[T any](rule, operator, value string, m Mapper[T], validaters ...Validater[T]) (val T, err error) {
	val, err = m(value)
	if err != nil {

		return
	}
	err = Validate(rule, operator, val, validaters...)
	return
}

/*
func BuildAction(c *Context, ruleExpr string) (ok bool, a Action, err error) {
	var name, operator, value string
	ok, name, operator, value = SplitRuleExpr(ruleExpr)
	if ok {
		switch name {
		case "init", "report":
			err = Validate[string](name, operator, value, TestNameValidater)
			a = Action(name)
			if value != "" {
				c.TestSuite = value
			}
		case "test":
			err = Validate[string](name, operator, value, TestNameValidater)
			a = Action(name)
			if value != "" {
				matches := AbsNameRegexp.FindStringSubmatch(value)
				if len(matches) == 2 {
					c.TestName = matches[1]
				} else {
					c.TestSuite = matches[1]
					c.TestName = matches[2]
				}
			}
		default:
			ok = false
		}
	}
	return
}
*/

func ApplyConfig(c *Context, ruleExpr string) (ok bool, err error) {
	var name, operator, value string
	ok, name, operator, value = SplitRuleExpr(ruleExpr)
	if ok {
		switch name {
		case "init", "report":
			err = Validate[string](name, operator, value, TestNameValidater)
			c.Action = Action(name)
			if value != "" {
				c.TestSuite = value
			}
		case "test":
			err = Validate[string](name, operator, value, TestNameValidater)
			c.Action = Action(name)
			if value != "" {
				matches := AbsNameRegexp.FindStringSubmatch(value)
				if len(matches) == 2 {
					c.TestName = matches[1]
				} else {
					c.TestSuite = matches[1]
					c.TestName = matches[2]
				}
			}

		case "fork":
			c.ForkCount, err = Translate(name, operator, value, IntMapper, EqualOperatorValidater[int], IntValueValidater(1, 5))
		case "suiteTimeout":
			c.SuiteTimeout, err = Translate(name, operator, value, DurationMapper)
		case "before":
		case "after":

		case "ignore":
			c.Ignore, err = Translate(name, operator, value, BoolMapper, BooleanValidater)
		case "stopOnFailure":
			c.StopOnFailure, err = Translate(name, operator, value, BoolMapper, BooleanValidater)
		case "keepStdout":
			c.KeepStdout, err = Translate(name, operator, value, BoolMapper, BooleanValidater)
		case "keepStderr":
			c.KeepStderr, err = Translate(name, operator, value, BoolMapper, BooleanValidater)
		case "keepOutputs":
			var keepOutputs bool
			keepOutputs, err = Translate(name, operator, value, BoolMapper, BooleanValidater)
			c.KeepStdout = keepOutputs
			c.KeepStderr = keepOutputs
		case "timeout":
			c.Timeout, err = Translate(name, operator, value, DurationMapper)
		case "runCount":
			c.RunCount, err = Translate(name, operator, value, IntMapper, EqualOperatorValidater[int], IntValueValidater(1, 1000))
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
			_, err = Translate(name, operator, value, NoMapper, NoOperatorValidater[any])
			a = func(cmd cmdz.Executer) (res AssertionResult, err error) {
				assert := Assertion{Name: name, Operator: operator, Expected: value}
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

func ParseArgs(args []string) (cfg Context, cmdAndArgs []string, assertions []Asserter, err error) {
	for _, arg := range args {
		var ok bool
		if IsRule(arg) {
			/*
				ok, action, err = BuildAction(&cfg, arg)
				if err != nil {
					return
				}
				if ok {
					continue
				}
			*/
			ok, err = ApplyConfig(&cfg, arg)
			if err != nil {
				return
			}
			if ok {
				continue
			}
			var asserter Asserter
			ok, asserter, err = BuildAssertion(arg)
			if err != nil {
				return
			}
			if ok {
				assertions = append(assertions, asserter)
				continue
			}
			err = fmt.Errorf("rule %s%s does not exists", AssertionPrefix, arg)
		} else {
			cmdAndArgs = append(cmdAndArgs, arg)
		}
	}
	return
}

func ValidateMutualyExclusiveRules(args []string) (err error) {
	MutualyExclusiveRules := [][]string{
		[]string{"init", "test", "report"},
		[]string{"keepOutputs", "keepStdout"},
		[]string{"keepOutputs", "keepStderr"},
		[]string{"test", "suiteTimeout"},
		[]string{"test", "forkCount"},
	}

	// FIXME: init ne supporte aucune assertion
	// FIXME: report ne supporte aucune assertion ni aucune config
	_ = MutualyExclusiveRules
	return
}
