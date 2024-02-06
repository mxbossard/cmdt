package main

import (
	"errors"
	"fmt"
	"log"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"mby.fr/utils/cmdz"
	"mby.fr/utils/collections"
	"mby.fr/utils/filez"
)

const (
	NamePattern = "[a-zA-Z][^/]*[a-zA-Z0-9]"
)

var (
	DefaultTestTimeout, _ = time.ParseDuration("1000h")
	AbsNamePattern        = fmt.Sprintf("(%s/)?(%s)?", NamePattern, NamePattern)
	NameRegexp            = regexp.MustCompile("^" + NamePattern + "$")
	AbsNameRegexp         = regexp.MustCompile("^" + AbsNamePattern + "$")
)

func IsRule(s string) bool {
	return strings.HasPrefix(s, RulePrefix())
}

func SplitRuleExpr(ruleExpr string) (ok bool, name string, operator string, value string) {
	ok = false
	assertionRulePattern := regexp.MustCompile("^" + RulePrefix() + "([a-zA-Z]+)([=~])?(.+)?$")
	submatch := assertionRulePattern.FindStringSubmatch(ruleExpr)
	if submatch != nil {
		ok = true
		name = submatch[1]
		operator = submatch[2]
		value = submatch[3]
	}
	return
}

func DummyMapper(s string) (v string, err error) {
	return s, nil
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

func FileContentMapper(s string) (v string, err error) {
	if strings.HasPrefix(s, "@") {
		// treat supplied value as a filepath
		path := s[1:]
		v, err = filez.ReadString(path)
		log.Printf("Reading file: %s => [%s]\n", path, v)
	} else {
		v = strings.ReplaceAll(s, "\\n", "\n")
	}
	return
}

func CmdMapper(s string) (v []string, err error) {
	// FIXME: should leverage simple and double quottes to split args
	if len(s) > 1 {
		separator := " "
		if s[0] == ',' || s[0] == ':' || s[0] == '|' {
			separator = s[0:1]
			s = s[1:]
		}
		v = strings.Split(s, separator)
		//log.Printf("CMD: [%v]", v)
	}
	return
}

func ExistsMapper(s string) (v []string, err error) {
	// @exists=FILEPATH,PERMS,OWNERS
	v = strings.Split(s, ",")
	//log.Printf("EXISTS: [%v]", v)
	return
}

func IntValueValidater(min, max int) Validater[int] {
	return func(rule, op string, n int) (err error) {
		if n < min || n > max {
			err = fmt.Errorf("rule %s%s value must be an integer >= %d and <= %d", RulePrefix(), rule, min, max)
		}
		return
	}
}

func NoOperatorValidater[T any](rule, op string, v T) (err error) {
	if op != "" {
		err = fmt.Errorf("rule %s%s cannot have a value nor an operator", RulePrefix(), rule)
	}
	return
}

func EqualOperatorValidater[T any](rule, op string, v T) (err error) {
	if op != "=" {
		err = fmt.Errorf("rule %s%s operator must be '='", RulePrefix(), rule)
	}
	return
}

func EqualOrTildeOperatorValidater[T any](rule, op string, v T) (err error) {
	if op != "=" && op != "~" {
		err = fmt.Errorf("rule %s%s operator must be '=' or '~'", RulePrefix(), rule)
	}
	return
}

func CmdValidater(rule, op string, v []string) (err error) {
	if len(v) == 0 {
		err = fmt.Errorf("rule %s%s value must be an executable command", RulePrefix(), rule)
	}

	return
}

func ExistsValidater(rule, op string, v []string) (err error) {
	if len(v) == 0 || len(v[0]) == 0 {
		err = fmt.Errorf("rule %s%s value must have a filepath", RulePrefix(), rule)
	}

	return
}

func TestNameValidater(rule, op string, v string) (err error) {
	switch rule {
	case "init", "report":
		if v != "" && !NameRegexp.MatchString(v) {
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
	if op != "" && op != "=" {
		err = fmt.Errorf("rule %s%s operator must be '='", RulePrefix(), rule)
	}
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
		err = fmt.Errorf("cannot map rule %s%s value: [%s] : %w", RulePrefix(), rule, value, err)
		return
	}
	err = Validate(rule, operator, val, validaters...)
	return
}

func ApplyConfig(c *Context, ruleExpr string) (ok bool, name string, err error) {
	var operator, value string
	ok, name, operator, value = SplitRuleExpr(ruleExpr)
	var boolVal bool
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
				//log.Printf("Matching names: %v", matches)
				if len(matches) == 2 {
					name := matches[1]
					if strings.HasSuffix(name, "/") {
						c.TestSuite = name[0 : len(name)-1]
					} else {
						c.TestName = name
					}
				} else if len(matches) == 3 {
					name := matches[1]
					if strings.HasSuffix(name, "/") {
						c.TestSuite = name[0 : len(name)-1]
					}
					c.TestName = matches[2]
				} else {
					err = fmt.Errorf("bad test name: [%s]", value)
				}
			}

		case "fork":
			c.ForkCount, err = Translate(name, operator, value, IntMapper, EqualOperatorValidater[int], IntValueValidater(1, 5))
		case "suiteTimeout":
			c.SuiteTimeout, err = Translate(name, operator, value, DurationMapper)
		case "before":
		case "after":

		case "ignore":
			boolVal, err = Translate(name, operator, value, BoolMapper, BooleanValidater)
			c.Ignore = &boolVal
		case "stopOnFailure":
			boolVal, err = Translate(name, operator, value, BoolMapper, BooleanValidater)
			c.StopOnFailure = &boolVal
		case "keepStdout":
			boolVal, err = Translate(name, operator, value, BoolMapper, BooleanValidater)
			c.KeepStdout = &boolVal
		case "keepStderr":
			boolVal, err = Translate(name, operator, value, BoolMapper, BooleanValidater)
			c.KeepStderr = &boolVal
		case "keepOutputs":
			var keepOutputs bool
			keepOutputs, err = Translate(name, operator, value, BoolMapper, BooleanValidater)
			c.KeepStdout = &keepOutputs
			c.KeepStderr = &keepOutputs
		case "timeout":
			c.Timeout, err = Translate(name, operator, value, DurationMapper)
		case "runCount":
			c.RunCount, err = Translate(name, operator, value, IntMapper, EqualOperatorValidater[int], IntValueValidater(1, 1000))
		case "prefix":
			c.Prefix, err = Translate(name, operator, value, DummyMapper, EqualOperatorValidater[string])
			if err == nil {
				SetRulePrefix(c.Prefix)
			}
		case "parallel":
		default:
			ok = false
		}
	}
	return
}

func BuildAssertion(ruleExpr string) (ok bool, assertion Assertion, err error) {
	var name, operator, value string
	ok, name, operator, value = SplitRuleExpr(ruleExpr)
	assertion = Assertion{Name: name, Operator: operator, Expected: value}
	if ok {
		switch name {
		case "success":
			_, err = Translate(name, operator, value, DummyMapper, NoOperatorValidater[string])
			if err != nil {
				return
			}
			assertion.Asserter = func(cmd cmdz.Executer) (res AssertionResult, err error) {
				res.Value = ""
				res.Success = cmd.ExitCode() == 0
				return
			}
		case "fail":
			_, err = Translate(name, operator, value, DummyMapper, NoOperatorValidater[string])
			if err != nil {
				return
			}
			assertion.Asserter = func(cmd cmdz.Executer) (res AssertionResult, err error) {
				res.Value = ""
				res.Success = cmd.ExitCode() > 0
				return
			}
		case "exit":
			var expectedExitCode int
			expectedExitCode, err = Translate(name, operator, value, IntMapper, IntValueValidater(0, 255), EqualOperatorValidater[int])
			if err != nil {
				return
			}
			assertion.Asserter = func(cmd cmdz.Executer) (res AssertionResult, err error) {
				res.Value = cmd.ExitCode()
				res.Success = cmd.ExitCode() == expectedExitCode
				return
			}
		case "stdout":
			var fileContent string
			fileContent, err = Translate(name, operator, value, FileContentMapper, EqualOrTildeOperatorValidater[string])
			if err != nil {
				return
			}
			assertion.Expected = fileContent
			assertion.Asserter = func(cmd cmdz.Executer) (res AssertionResult, err error) {
				res.Value = cmd.StdoutRecord()
				if operator == "=" {
					res.Success = cmd.StdoutRecord() == fileContent
				} else if operator == "~" {
					res.Success = strings.Contains(cmd.StdoutRecord(), fileContent)
				} else {
					err = fmt.Errorf("rule %s%s must use an operator '=' or '~'", RulePrefix(), name)
				}
				return
			}
		case "stderr":
			var fileContent string
			fileContent, err = Translate(name, operator, value, FileContentMapper, EqualOrTildeOperatorValidater[string])
			if err != nil {
				return
			}
			assertion.Expected = fileContent
			assertion.Asserter = func(cmd cmdz.Executer) (res AssertionResult, err error) {
				res.Value = cmd.StderrRecord()
				if operator == "=" {
					res.Success = cmd.StderrRecord() == fileContent
				} else if operator == "~" {
					res.Success = strings.Contains(cmd.StderrRecord(), fileContent)
				} else {
					err = fmt.Errorf("rule %s%s must use an operator '=' or '~'", RulePrefix(), name)
				}
				return
			}
		case "cmd":
			var cmdAndArgs []string
			cmdAndArgs, err = Translate(name, operator, value, CmdMapper, EqualOperatorValidater[[]string], CmdValidater)
			if err != nil {
				return
			}
			assertionCmd := cmdz.Cmd(cmdAndArgs...).Timeout(10 * time.Second)
			assertion.Asserter = func(cmd cmdz.Executer) (res AssertionResult, err error) {
				res.Value = 0
				exitCode := -1
				exitCode, err = assertionCmd.BlockRun()
				res.Value = exitCode
				res.Success = exitCode == 0
				return
			}
		case "exists":
			var filepathRules []string
			filepathRules, err = Translate(name, operator, value, ExistsMapper, EqualOperatorValidater[[]string], ExistsValidater)
			if err != nil {
				return
			}
			var path, permissions, owners string
			if len(filepathRules) > 0 {
				path = filepathRules[0]
			}
			if len(filepathRules) > 1 {
				permissions = filepathRules[1]
			}
			if len(filepathRules) > 2 {
				owners = filepathRules[2]
			}

			assertion.Asserter = func(cmd cmdz.Executer) (res AssertionResult, err error) {
				var stat os.FileInfo
				stat, err = os.Stat(path)
				if errors.Is(err, os.ErrNotExist) {
					res.Success = false
					res.Message = fmt.Sprintf("file %s does not exists", path)
					err = nil
					return
				} else if err != nil {
					return
				}
				if permissions != "" {
					if permissions != stat.Mode().String() {
						res.Success = false
						res.Value = stat.Mode().String()
						res.Message = fmt.Sprintf("file %s have wrong permissions", path)
						return
					}
				}
				if owners != "" {
					// FIXME: how to checke file owners ?
				}
				res.Success = true
				return
			}

		default:
			ok = false
		}
	}
	return
}

func ParseArgs(args []string) (cfg Context, cmdAndArgs []string, assertions []Assertion, err error) {
	var rules []string
	var rule string
	parseRules := true
	for _, arg := range args {
		var ok bool
		if len(cmdAndArgs) == 0 && arg == "--" {
			// If no command found yet first -- param is interpreted as a rule parsing stopper
			// stop parsing rules
			parseRules = false
			continue
		}
		if parseRules && IsRule(arg) {
			ok, rule, err = ApplyConfig(&cfg, arg)
			if err != nil {
				return
			}
			if ok {
				rules = append(rules, rule)
				continue
			}
			var assertion Assertion
			ok, assertion, err = BuildAssertion(arg)
			if err != nil {
				return
			}
			if ok {
				assertions = append(assertions, assertion)
				rules = append(rules, assertion.Name)
				continue
			}
			err = fmt.Errorf("rule %s%s does not exists", RulePrefix(), arg)
			return
		} else {
			cmdAndArgs = append(cmdAndArgs, arg)
		}
	}

	statusAssertionFound := false
	for _, a := range assertions {
		// If no status assertion found add an implicit success rule
		statusAssertionFound = statusAssertionFound || a.Name == "success" || a.Name == "fail" || a.Name == "exit"
	}
	if !statusAssertionFound {
		_, successAssertion, _ := BuildAssertion(RulePrefix() + "success")
		assertions = append(assertions, successAssertion)
	}

	if cfg.Action == Action("") {
		// If no action supplied add implicit test rule.
		_, rule, err = ApplyConfig(&cfg, RulePrefix()+"test")
		if err != nil {
			return
		}
		rules = append(rules, rule)
	}

	err = ValidateMutualyExclusiveRules(rules)
	//log.Printf("Parsed config: %v", cfg)
	return
}

func buildMutualyExclusiveCouples(rule string, exclusiveRules ...string) (res [][]string) {
	for _, e := range exclusiveRules {
		res = append(res, []string{rule, e})
	}
	return
}

func ValidateMutualyExclusiveRules(args []string) (err error) {
	MutualyExclusiveRules := [][]string{
		{"init", "test", "report"},
		{"fail", "success", "exit"},
		{"keepOutputs", "keepStdout"},
		{"keepOutputs", "keepStderr"},
	}

	exlusiveRules := MutualyExclusiveRules
	// FIXME: init ne supporte aucune assertion
	exlusiveRules = append(exlusiveRules, buildMutualyExclusiveCouples("init", "success", "fail", "exit", "stdout",
		"stderr", "exists")...)
	exlusiveRules = append(exlusiveRules, buildMutualyExclusiveCouples("test", "suiteTimeout", "forkCount")...)
	// FIXME: report ne supporte aucune assertion ni aucune config
	exlusiveRules = append(exlusiveRules, buildMutualyExclusiveCouples("report", "fork", "suiteTimeout", "before",
		"after", "ignore", "stopOnFailure", "keepStdout", "keepStderr", "keepOutputs", "timeout", "runCount", "parallel",
		"success", "fail", "exit", "stdout", "stderr", "exists")...)

	//log.Printf("args: %s\n", args)
	for _, mer := range exlusiveRules {
		matchCount := 0
		for _, arg := range args {
			if collections.Contains[string](&mer, arg) {
				matchCount++
			}
		}
		//log.Printf("%s rules => %d\n", mer, matchCount)
		if matchCount > 1 {
			err = fmt.Errorf("you can't use simultaneously following rules which are mutually exclusives: [%s]", strings.Join(mer, ","))
		}
	}

	return
}
