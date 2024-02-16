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

func SplitRuleExpr(ruleExpr string) (ok bool, r Rule) {
	ok = false
	assertionRulePattern := regexp.MustCompile("^" + RulePrefix() + "([a-zA-Z]+)([=~:!]{1,2})?(.+)?$")
	submatch := assertionRulePattern.FindStringSubmatch(ruleExpr)
	if submatch != nil {
		ok = true
		r.Prefix = RulePrefix()
		r.Name = submatch[1]
		r.Op = submatch[2]
		r.Expected = submatch[3]
	}
	return
}

func DummyMapper(s, op string) (v string, err error) {
	return s, nil
}

func IntMapper(s, op string) (v int, err error) {
	v, err = strconv.Atoi(s)
	return
}

func BoolMapper(s, op string) (v bool, err error) {
	if s == "true" || s == "" {
		v = true
	} else if s == "false" {
		v = false
	} else {
		err = fmt.Errorf("bool rule value must be true or false")
	}
	//log.Printf("boolMapper: [%s] => %v", s, v)
	return
}

func DurationMapper(s, op string) (v time.Duration, err error) {
	return time.ParseDuration(s)
}

func FileContentMapper(s, op string) (v string, err error) {
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

func CmdMapper(s, op string) (v []string, err error) {
	// FIXME: should leverage simple and double quottes to split args
	if len(s) > 1 {
		separator := " "
		if s[0] == ';' || s[0] == ':' || s[0] == '|' {
			separator = s[0:1]
			s = s[1:]
		}
		v = strings.Split(s, separator)
		//log.Printf("CMD: [%v]", v)
	}
	return
}

func MockMapper(s, op string) (m CmdMock, err error) {
	if len(s) > 1 {
		splitted := strings.Split(s, ",")
		// cmd always defined first
		var mockedCmdAndArgs []string
		mockedCmdAndArgs, err = CmdMapper(splitted[0], op)
		if err != nil {
			return
		}
		m.Op = op
		m.Cmd = mockedCmdAndArgs[0]
		if len(mockedCmdAndArgs) > 1 {
			m.Args = mockedCmdAndArgs[1:]
		}
		m.Delegate = true
		if len(splitted) > 1 {
			for _, rule := range splitted[1:] {
				ruleSplit := strings.Split(rule, "=")
				if len(ruleSplit) < 2 {
					err = fmt.Errorf("bad format for mock rule: expect an = sign")
					return
				}
				key := ruleSplit[0]
				value := strings.Join(ruleSplit[1:], "=")
				switch key {
				case "stdin":
					m.Stdin = value
				case "stdout":
					m.Delegate = false
					m.Stdout = value
				case "stderr":
					m.Delegate = false
					m.Stderr = value
				case "exit":
					m.Delegate = false
					m.ExitCode, err = IntMapper(value, "=")
					if err != nil {
						// FIXME: aggregate errors
						return
					}
				case "cmd":
					m.Delegate = false
					m.OnCallCmdAndArgs, err = CmdMapper(value, "=")
					if err != nil {
						// FIXME: aggregate errors
						return
					}
				default:
					err = fmt.Errorf("mock rule: %s does not exists", key)
					// FIXME: aggregate errors
					return
				}
			}
		}
	}
	return
}

func ExistsMapper(s, op string) (v []string, err error) {
	// @exists=FILEPATH,PERMS,OWNERS
	v = strings.Split(s, ",")
	//log.Printf("EXISTS: [%v]", v)
	return
}

func RegexpPatternMapper(s, op string) (c *regexp.Regexp, err error) {
	if len(s) < 2 {
		err = fmt.Errorf("regexp pattern must be of form /PATTERN/FLAGS")
		return
	}
	separator := s[0:1] // First char is the spearator
	sepCount := strings.Count(s, separator)
	if sepCount != 2 {
		err = fmt.Errorf("regexp pattern must be of form /PATTERN/FLAGS")
		return
	}
	splitted := strings.Split(s, separator)
	pattern := splitted[1]
	flags := splitted[2]
	for _, flag := range flags {
		switch flag {
		case 'i', 'm', 's', 'u':
		default:
			err = fmt.Errorf("flag: %v is not supported. valid flags are: i m s u", flag)
			return
		}
	}
	if len(flags) > 0 {
		flags = "(?" + flags + ")"
	}
	c, err = regexp.Compile(flags + pattern)
	if err != nil {
		return
	}
	return
}

func IntValueValidater(min, max int) Validater[int] {
	return func(rule Rule, n int) (err error) {
		if n < min || n > max {
			err = fmt.Errorf("rule %s%s value must be an integer >= %d and <= %d", rule.Prefix, rule.Name, min, max)
		}
		return
	}
}

func OperatorValidater[T any](ops ...string) Validater[T] {
	return func(rule Rule, v T) (err error) {
		if !collections.Contains[string](&ops, rule.Op) {
			err = fmt.Errorf("rule %s%s%s bad operator. Must be one of: [%s]", rule.Prefix, rule.Name, rule.Op, ops)
		}
		return
	}
}

func NotEmptyForOpValidater[T any](ops ...string) Validater[T] {
	return func(rule Rule, v T) (err error) {
		if collections.Contains[string](&ops, rule.Op) && rule.Expected == "" {
			err = fmt.Errorf("rule %s%s%s must have a value", rule.Prefix, rule.Name, rule.Op)
		}
		return
	}
}

func NotEmptyValidater[T any](rule Rule, v T) (err error) {
	if fmt.Sprintf("%v", v) == "" {
		err = fmt.Errorf("rule %s%s%s must have a value", rule.Prefix, rule.Name, rule.Op)
	}
	return
}

func CmdValidater(rule Rule, v []string) (err error) {
	if len(v) == 0 {
		err = fmt.Errorf("rule %s%s value must be an executable command", rule.Prefix, rule.Name)
	}

	return
}

func ExistsValidater(rule Rule, v []string) (err error) {
	if len(v) == 0 || len(v[0]) == 0 {
		err = fmt.Errorf("rule %s%s value must have a filepath", rule.Prefix, rule.Name)
	}

	return
}

func TestNameValidater(rule Rule, v string) (err error) {
	switch rule.Name {
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

func StringOrNotingValidater(rule Rule, v string) (err error) {
	if rule.Op == "=" && v == "" {
		err = fmt.Errorf("rule %s%s= must have a value", rule.Prefix, rule.Name)
	}
	return
}

func BooleanValidater(rule Rule, v bool) (err error) {
	if rule.Op != "" && rule.Op != "=" {
		err = fmt.Errorf("rule %s%s operator must be '='", rule.Prefix, rule.Name)
	}
	return
}

func MockValidater(rule Rule, v CmdMock) (err error) {
	if strings.Contains(v.Cmd, "/") {
		err = fmt.Errorf("rule %s%s command does not support slash", rule.Prefix, rule.Name)
	}
	return
}

func Validate[T any](rule Rule, val T, validaters ...Validater[T]) (err error) {
	for _, v := range validaters {
		err = v(rule, val)
		if err != nil {
			return
		}
	}
	return
}

func Translate[T any](rule Rule, m Mapper[T], validaters ...Validater[T]) (val T, err error) {
	val, err = m(rule.Expected, rule.Op)
	if err != nil {
		err = fmt.Errorf("cannot map rule %s%s value: [%s] : %w", rule.Prefix, rule.Name, rule.Expected, err)
		return
	}
	err = Validate(rule, val, validaters...)
	return
}

func ApplyConfig(c *Context, ruleExpr string) (ok bool, rule Rule, err error) {
	ok, rule = SplitRuleExpr(ruleExpr)
	if !ok {
		return
	}

	var isAction, isTestConf, isSuiteConf, isFlowConf bool
	isAction, err = IsRuleOfKind(Actions, rule)
	if err != nil {
		return
	}
	isTestConf, err = IsRuleOfKind(TestConfigs, rule)
	if err != nil {
		return
	}
	isSuiteConf, err = IsRuleOfKind(SuiteConfigs, rule)
	if err != nil {
		return
	}
	isFlowConf, err = IsRuleOfKind(FlowConfigs, rule)
	if err != nil {
		return
	}
	ok = isAction || isTestConf || isSuiteConf || isFlowConf
	if !ok {
		return
	}

	var boolVal bool
	if ok {
		switch rule.Name {
		case "global":
			c.Action = Action(rule.Name)
			if rule.Op != "" || rule.Expected != "" {
				err = fmt.Errorf("rule %s%s does not accept an operator nor a value", rule.Prefix, rule.Name)
			}
		case "init", "report":
			suiteName := rule.Expected
			err = Validate[string](rule, suiteName, TestNameValidater)
			c.Action = Action(rule.Name)
			if suiteName != "" {
				c.TestSuite = suiteName
			}
			if rule.Name == "report" && rule.Expected == "" {
				c.ReportAll = true
			}
		case "test":
			testName := rule.Expected
			err = Validate[string](rule, testName, TestNameValidater)
			c.Action = Action(rule.Name)
			if testName != "" {
				matches := AbsNameRegexp.FindStringSubmatch(testName)
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
					err = fmt.Errorf("bad test name: [%s]", testName)
				}
			}

		case "fork":
			c.ForkCount, err = Translate(rule, IntMapper, OperatorValidater[int]("="), IntValueValidater(1, 5))
		case "suiteTimeout":
			c.SuiteTimeout, err = Translate(rule, DurationMapper)
		case "before":
		case "after":

		case "ignore":
			boolVal, err = Translate(rule, BoolMapper, BooleanValidater)
			c.Ignore = &boolVal
		case "stopOnFailure":
			boolVal, err = Translate(rule, BoolMapper, BooleanValidater)
			c.StopOnFailure = &boolVal
		case "keepStdout":
			boolVal, err = Translate(rule, BoolMapper, BooleanValidater)
			c.KeepStdout = &boolVal
		case "keepStderr":
			boolVal, err = Translate(rule, BoolMapper, BooleanValidater)
			c.KeepStderr = &boolVal
		case "keepOutputs":
			var keepOutputs bool
			keepOutputs, err = Translate(rule, BoolMapper, BooleanValidater)
			c.KeepStdout = &keepOutputs
			c.KeepStderr = &keepOutputs
		case "timeout":
			c.Timeout, err = Translate(rule, DurationMapper)
		case "runCount":
			c.RunCount, err = Translate(rule, IntMapper, OperatorValidater[int]("="), IntValueValidater(1, 1000))
		case "prefix":
			c.Prefix, err = Translate(rule, DummyMapper, OperatorValidater[string]("="))
			if err == nil {
				SetRulePrefix(c.Prefix)
			}
		case "token":
			c.Token, err = Translate(rule, DummyMapper, OperatorValidater[string]("="), NotEmptyValidater[string])
		case "printToken":
			boolVal, err = Translate(rule, BoolMapper, BooleanValidater)
			c.PrintToken = boolVal
		case "exportToken":
			boolVal, err = Translate(rule, BoolMapper, BooleanValidater)
			c.ExportToken = boolVal
		case "parallel":
		case "silent":
			boolVal, err = Translate(rule, BoolMapper, BooleanValidater)
			c.Silent = &boolVal
		case "mock":
			var mock CmdMock
			mock, err = Translate(rule, MockMapper, OperatorValidater[CmdMock]("=", ":"), NotEmptyValidater[CmdMock], MockValidater)
			c.Mocks = append(c.Mocks, mock)
		default:
			ok = false
		}
	}
	return
}

func BuildAssertion(ruleExpr string) (ok bool, assertion Assertion, err error) {
	ok, assertion.Rule = SplitRuleExpr(ruleExpr)
	if !ok {
		return
	}

	ok, err = IsRuleOfKind(Assertions, assertion.Rule)
	if err != nil {
		return
	}
	if !ok {
		return
	}

	switch assertion.Rule.Name {
	case "success":
		_, err = Translate(assertion.Rule, DummyMapper, OperatorValidater[string](""))
		if err != nil {
			return
		}
		assertion.Asserter = func(cmd cmdz.Executer) (res AssertionResult, err error) {
			res.Value = ""
			res.Success = cmd.ExitCode() == 0
			return
		}
	case "fail":
		_, err = Translate(assertion.Rule, DummyMapper, OperatorValidater[string](""))
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
		expectedExitCode, err = Translate(assertion.Rule, IntMapper, IntValueValidater(0, 255), OperatorValidater[int]("="))
		if err != nil {
			return
		}
		assertion.Asserter = func(cmd cmdz.Executer) (res AssertionResult, err error) {
			res.Value = cmd.ExitCode()
			res.Success = cmd.ExitCode() == expectedExitCode
			return
		}
	case "stdout":
		if assertion.Op == "~" || assertion.Op == "!~" {
			var regexpPattern *regexp.Regexp
			regexpPattern, err = Translate(assertion.Rule, RegexpPatternMapper, OperatorValidater[*regexp.Regexp]("~", "!~"), NotEmptyValidater[*regexp.Regexp])
			if err != nil {
				return
			}
			assertion.Expected = regexpPattern.String()
			assertion.Asserter = func(cmd cmdz.Executer) (res AssertionResult, err error) {
				res.Value = cmd.StdoutRecord()
				if assertion.Rule.Op == "~" {
					res.Success = regexpPattern.MatchString(cmd.StdoutRecord())
				} else if assertion.Rule.Op == "!~" {
					res.Success = !regexpPattern.MatchString(cmd.StdoutRecord())
				} else {
					err = fmt.Errorf("rule %s%s must use an operator '~' or '!~'", assertion.Rule.Prefix, assertion.Rule.Name)
				}
				return
			}
		} else {
			var fileContent string
			fileContent, err = Translate(assertion.Rule, FileContentMapper, OperatorValidater[string]("=", ":", "!=", "!:"), NotEmptyForOpValidater[string](":", "!:"))
			if err != nil {
				return
			}
			assertion.Expected = fileContent
			assertion.Asserter = func(cmd cmdz.Executer) (res AssertionResult, err error) {
				res.Value = cmd.StdoutRecord()
				if assertion.Rule.Op == "=" {
					res.Success = cmd.StdoutRecord() == fileContent
				} else if assertion.Rule.Op == ":" {
					res.Success = strings.Contains(cmd.StdoutRecord(), fileContent)
				} else if assertion.Rule.Op == "!=" {
					res.Success = cmd.StdoutRecord() != fileContent
				} else if assertion.Rule.Op == "!:" {
					res.Success = !strings.Contains(cmd.StdoutRecord(), fileContent)
				} else {
					err = fmt.Errorf("rule %s%s must use an operator '=' or ':'", assertion.Rule.Prefix, assertion.Rule.Name)
				}
				return
			}
		}
	case "stderr":
		if assertion.Op == "~" || assertion.Op == "!~" {
			var regexpPattern *regexp.Regexp
			regexpPattern, err = Translate(assertion.Rule, RegexpPatternMapper, OperatorValidater[*regexp.Regexp]("~", "!~"), NotEmptyValidater[*regexp.Regexp])
			if err != nil {
				return
			}
			assertion.Expected = regexpPattern.String()
			assertion.Asserter = func(cmd cmdz.Executer) (res AssertionResult, err error) {
				res.Value = cmd.StderrRecord()
				if assertion.Rule.Op == "~" {
					res.Success = regexpPattern.MatchString(cmd.StderrRecord())
				} else if assertion.Rule.Op == "!~" {
					res.Success = !regexpPattern.MatchString(cmd.StderrRecord())
				} else {
					err = fmt.Errorf("rule %s%s must use an operator '~' or '!~'", assertion.Rule.Prefix, assertion.Rule.Name)
				}
				return
			}
		} else {
			var fileContent string
			fileContent, err = Translate(assertion.Rule, FileContentMapper, OperatorValidater[string]("=", ":", "!=", "!:"), NotEmptyForOpValidater[string](":", "!:"))
			if err != nil {
				return
			}
			assertion.Expected = fileContent
			assertion.Asserter = func(cmd cmdz.Executer) (res AssertionResult, err error) {
				res.Value = cmd.StderrRecord()
				if assertion.Rule.Op == "=" {
					res.Success = cmd.StderrRecord() == fileContent
				} else if assertion.Rule.Op == ":" {
					res.Success = strings.Contains(cmd.StderrRecord(), fileContent)
				} else if assertion.Rule.Op == "!=" {
					res.Success = cmd.StderrRecord() != fileContent
				} else if assertion.Rule.Op == "!:" {
					res.Success = !strings.Contains(cmd.StderrRecord(), fileContent)
				} else {
					err = fmt.Errorf("rule %s%s must use an operator '=' or '~'", assertion.Rule.Prefix, assertion.Rule.Name)
				}
				return
			}
		}
	case "cmd":
		var cmdAndArgs []string
		cmdAndArgs, err = Translate(assertion.Rule, CmdMapper, OperatorValidater[[]string]("="), CmdValidater)
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
		filepathRules, err = Translate(assertion.Rule, ExistsMapper, OperatorValidater[[]string]("="), ExistsValidater)
		if err != nil {
			return
		}
		var path, expectedPerms, owners string
		if len(filepathRules) > 0 {
			path = filepathRules[0]
		}
		if len(filepathRules) > 1 {
			expectedPerms = filepathRules[1]
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
			if expectedPerms != "" {
				actualPerms := stat.Mode().String()
				if expectedPerms != actualPerms {
					res.Success = false
					res.Value = stat.Mode().String()
					res.Message = fmt.Sprintf("file %s have wrong permissions. Expected: [%s] but got: [%s] ", path, expectedPerms, actualPerms)
					return
				}
			}
			if owners != "" {
				// FIXME: how to checke file owners ?
				err = fmt.Errorf("exists assertion owner part not implemented yet")
				return
			}
			res.Success = true
			return
		}

	default:
		ok = false
	}
	return
}

func ParseArgs(args []string) (cfg Context, cmdAndArgs []string, assertions []Assertion, err error) {
	cfg.Silent = nil
	var rules []Rule
	var rule Rule
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
				rules = append(rules, assertion.Rule)
				continue
			}
			err = fmt.Errorf("rule %s does not exists", arg)
			return
		} else {
			cmdAndArgs = append(cmdAndArgs, arg)
		}
	}

	statusAssertionFound := false
	for _, a := range assertions {
		// If no status assertion found add an implicit success rule
		statusAssertionFound = statusAssertionFound || a.Name == "success" || a.Name == "fail" || a.Name == "exit" // || a.Name == "cmd"
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

	if cfg.TestSuite == "" {
		cfg.TestSuite = DefaultTestSuiteName
	}

	if (cfg.Action == "init" || cfg.Action == "report") && len(cmdAndArgs) > 0 {
		err = fmt.Errorf("you cannot run commands with action %s%s", cfg.Prefix, cfg.Action)
		return
	}

	err = ValidateMutualyExclusiveRules(rules...)
	if err != nil {
		return
	}
	err = ValidateOnceOnlyDefinedRule(rules...)

	//log.Printf("build context: %s Silent: %v\n", args, cfg.Silent)
	return
}

func buildMutualyExclusiveCouples(rule RuleKey, exclusiveRules ...RuleKey) (res [][]RuleKey) {
	for _, e := range exclusiveRules {
		res = append(res, []RuleKey{rule, e})
	}
	return
}

func ruleKey(s ...string) (r RuleKey) {
	r.Name = s[0]
	r.Op = "all"
	if len(s) > 1 {
		r.Op = s[1]
	}
	return
}

func ruleKeys(ruleDefs ...[]RuleDefinition) (r []RuleKey) {
	for _, ruleDef := range ruleDefs {
		for _, def := range ruleDef {
			r = append(r, RuleKey{Name: def.Name, Op: "all"})
		}
	}
	return
}

// ValidateOnceOnlyDefinedRule => verify rules which cannot be defined multiple times are not defined twice or more
func ValidateOnceOnlyDefinedRule(rules ...Rule) (err error) {
	multiDefinedRules := []RuleKey{
		{"stdout", "~"}, {"stderr", "~"}, {"stdout", "!~"}, {"stderr", "!~"},
		{"stdout", "!="}, {"stderr", "!="},
		{"stdout", ":"}, {"stderr", ":"}, {"stdout", "!:"}, {"stderr", "!:"},
	}
	matches := map[RuleKey][]Rule{}
	for _, rule := range rules {
		key := RuleKey{rule.Name, rule.Op}
		matches[key] = append(matches[key], rule)
	}

	for key, matchedRules := range matches {
		if len(matchedRules) > 1 && !collections.Contains(&multiDefinedRules, key) {
			// This rule is defined more than once and shouldnt
			err = fmt.Errorf("rule: %s is defined more than once", key.Name)
		}
	}
	return
}

func ValidateMutualyExclusiveRules(rules ...Rule) (err error) {
	// FIXME: stdout= and stdout~ are ME ; stdout= and stdout= are ME but stdout~ and stdout~ are not ME
	MutualyExclusiveRules := [][]RuleKey{
		ruleKeys(Actions),
		{{"fail", "all"}, {"success", "all"}, {"exit", "all"}},
		//{{"fail", "all"}, {"success", "all"}, {"cmd", "all"}},
		{{"test", "all"}, {"token", ""}},
		{{"report", "all"}, {"token", ""}},
	}

	exlusiveRules := MutualyExclusiveRules
	exlusiveRules = append(exlusiveRules, buildMutualyExclusiveCouples(RuleKey{"global", "all"}, ruleKeys(Assertions)...)...)
	exlusiveRules = append(exlusiveRules, buildMutualyExclusiveCouples(RuleKey{"init", "all"}, ruleKeys(Assertions)...)...)
	exlusiveRules = append(exlusiveRules, buildMutualyExclusiveCouples(RuleKey{"report", "all"}, ruleKeys(Assertions, TestConfigs, SuiteConfigs)...)...)
	exlusiveRules = append(exlusiveRules, buildMutualyExclusiveCouples(RuleKey{"test", "all"}, RuleKey{"suiteTimeout", "all"}, RuleKey{"fork", "all"})...)
	exlusiveRules = append(exlusiveRules, buildMutualyExclusiveCouples(ruleKey("keepOutputs"), ruleKey("keepStdout"), ruleKey("keepStderr"))...)
	exlusiveRules = append(exlusiveRules, buildMutualyExclusiveCouples(ruleKey("stdout", "="), ruleKey("stdout", "~"), ruleKey("stdout", "!~"), ruleKey("stdout", "!="), ruleKey("stdout", ":"), ruleKey("stdout", "!:"))...)
	exlusiveRules = append(exlusiveRules, buildMutualyExclusiveCouples(ruleKey("stderr", "="), ruleKey("stderr", "~"), ruleKey("stderr", "!~"), ruleKey("stderr", "!="), ruleKey("stderr", ":"), ruleKey("stderr", "!:"))...)

	// Compter le nombre de match pour chaque key
	// Pour chaque MER compter le nombre de key
	//matches := map[RuleKey][]Rule{}
	matchingMers := [][]RuleKey{}
	for i, rule1 := range rules {
		ruleKey1 := ruleKey(rule1.Name, rule1.Op)
		for j, rule2 := range rules {
			if i == j {
				continue
			}
			ruleKey2 := ruleKey(rule2.Name, rule2.Op)
			// check all rule couples against MER
			for _, mer := range exlusiveRules {
				for _, k := range mer {
					for _, l := range mer {
						if k == l {
							continue
						}

						if (ruleKey1 == k || k.Op == "all" && ruleKey1.Name == k.Name) &&
							(ruleKey2 == l || l.Op == "all" && ruleKey2.Name == l.Name) {
							matchingMers = append(matchingMers, mer)
						}

					}
				}
			}
		}
	}

	//log.Printf("matchingMers: %v\n", matchingMers)
	for _, mer := range matchingMers {
		err = fmt.Errorf("you can't use simultaneously following rules which are mutually exclusives: [%s]", mer)
		return
	}
	return
}
