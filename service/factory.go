package service

import (
	"errors"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"mby.fr/cmdtest/model"
	"mby.fr/utils/cmdz"
	"mby.fr/utils/collections"
	"mby.fr/utils/errorz"
	"mby.fr/utils/filez"
	"mby.fr/utils/utilz"
)

/*
	func IsRule(s string) bool {
		return strings.HasPrefix(s, RulePrefix())
	}
*/

func SplitRuleExpr(cfg model.Config, ruleExpr string) (ok bool, r model.Rule) {
	return cfg.SplitRuleExpr(ruleExpr)
}

func DummyMapper(s, op string) (v string, err error) {
	return s, nil
}

func Uint16Mapper(s, op string) (v uint16, err error) {
	var i int
	i, err = strconv.Atoi(s)
	v = uint16(i)
	return
}

func Int32Mapper(s, op string) (v int32, err error) {
	var i int
	i, err = strconv.Atoi(s)
	v = int32(i)
	return
}

func Uint32Mapper(s, op string) (v uint32, err error) {
	var i int
	i, err = strconv.Atoi(s)
	v = uint32(i)
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

func DirtiesMapper(s, op string) (v model.DirtyScope, err error) {
	switch s {
	case "beforeSuite":
		v = model.DirtyBeforeSuite
	case "afterSuite":
		v = model.DirtyAfterSuite
	case "beforeTest":
		v = model.DirtyBeforeTest
	case "afterTest":
		v = model.DirtyAfterTest
	case "beforeRun":
		v = model.DirtyBeforeRun
	case "afterRun":
		v = model.DirtyAfterRun
	default:
		err = fmt.Errorf("dirty scope: %s not supported", s)
	}
	return
}

func FileContentMapper(s, op string) (v string, err error) {
	if strings.HasPrefix(op, "@") {
		// treat supplied value as a filepath
		path := s
		v, err = filez.ReadString(path)
		logger.Debug("reading file", "path", path, "content", v)
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

func MockMapper(s, op string) (m model.CmdMock, err error) {
	var splitted, mockedCmdAndArgs []string
	if len(s) > 1 {
		splitted = strings.Split(s, ",")
		// cmd always defined first
	} else {
		splitted = append(splitted, s)
	}

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
			if strings.HasPrefix(rule, "stdin=") {
				value := rule[6:]
				m.Stdin = &value
				m.StdinOp = "="
			} else if strings.HasPrefix(rule, "stdin:") {
				value := rule[6:]
				m.Stdin = &value
				m.StdinOp = ":"
			} else if strings.HasPrefix(rule, "stdin@=") {
				path := rule[7:]
				var value string
				value, err = filez.ReadString(path)
				if err != nil {
					return
				}
				//logger.Warn("mock stdin @=", "path", path, "content", value)
				m.Stdin = &value
				m.StdinOp = "="
			} else if strings.HasPrefix(rule, "stdin@:") {
				path := rule[7:]
				var value string
				value, err = filez.ReadString(path)
				if err != nil {
					return
				}
				m.Stdin = &value
				m.StdinOp = ":"
			} else if strings.HasPrefix(rule, "stdout=") {
				value := rule[7:]
				m.Delegate = false
				m.Stdout = value
			} else if strings.HasPrefix(rule, "stdout@=") {
				path := rule[8:]
				var value string
				value, err = filez.ReadString(path)
				if err != nil {
					return
				}
				m.Delegate = false
				m.Stdout = value
			} else if strings.HasPrefix(rule, "stderr=") {
				value := rule[7:]
				m.Delegate = false
				m.Stderr = value
			} else if strings.HasPrefix(rule, "stderr@=") {
				path := rule[8:]
				var value string
				value, err = filez.ReadString(path)
				if err != nil {
					return
				}
				m.Delegate = false
				m.Stderr = value
			} else if strings.HasPrefix(rule, "exit=") {
				value := rule[5:]
				m.Delegate = false
				m.ExitCode, err = Uint16Mapper(value, "=")
				if err != nil {
					// FIXME: aggregate errors
					return
				}
			} else if strings.HasPrefix(rule, "cmd=") {
				value := rule[4:]
				m.Delegate = false
				m.OnCallCmdAndArgs, err = CmdMapper(value, "=")
				if err != nil {
					// FIXME: aggregate errors
					return
				}
			} else {
				err = fmt.Errorf("mock rule: %s does not exists", rule)
				// FIXME: aggregate errors
				return
			}

			/*
				ruleSplit := strings.Split(rule, "=")
				if len(ruleSplit) < 2 {
					err = fmt.Errorf("bad format for mock rule: expect an = sign")
					return
				}
				key := ruleSplit[0]
				value := strings.Join(ruleSplit[1:], "=")
				switch key {
				case "stdin":
					m.Stdin = &value
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
			*/
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

func Uint16ValueValidater(min, max uint16) model.Validater[uint16] {
	return func(rule model.Rule, n uint16) (err error) {
		if n < min || n > max {
			err = fmt.Errorf("rule %s%s value must be an integer >= %d and <= %d", rule.Prefix, rule.Name, min, max)
		}
		return
	}
}

func Int32ValueValidater(min, max int32) model.Validater[int32] {
	return func(rule model.Rule, n int32) (err error) {
		if n < min || n > max {
			err = fmt.Errorf("rule %s%s value must be an integer >= %d and <= %d", rule.Prefix, rule.Name, min, max)
		}
		return
	}
}

func Uint32ValueValidater(min, max uint32) model.Validater[uint32] {
	return func(rule model.Rule, n uint32) (err error) {
		if n < min || n > max {
			err = fmt.Errorf("rule %s%s value must be an integer >= %d and <= %d", rule.Prefix, rule.Name, min, max)
		}
		return
	}
}

func OperatorValidater[T any](ops ...string) model.Validater[T] {
	return func(rule model.Rule, v T) (err error) {
		if !collections.Contains[string](&ops, rule.Op) {
			err = fmt.Errorf("rule %s%s%s bad operator. Must be one of: [%s]", rule.Prefix, rule.Name, rule.Op, ops)
		}
		return
	}
}

func KeywordsValidater[T any](keywords ...string) model.Validater[T] {
	return func(rule model.Rule, v T) (err error) {
		if !collections.Contains[string](&keywords, rule.Expected) {
			err = fmt.Errorf("rule %s%s%s bad value. Must be one of: [%s]", rule.Prefix, rule.Name, rule.Expected, keywords)
		}
		return
	}
}

func NotEmptyForOpValidater[T any](ops ...string) model.Validater[T] {
	return func(rule model.Rule, v T) (err error) {
		if collections.Contains[string](&ops, rule.Op) && rule.Expected == "" {
			err = fmt.Errorf("rule %s%s%s must have a value", rule.Prefix, rule.Name, rule.Op)
		}
		return
	}
}

func NotEmptyValidater[T any](rule model.Rule, v T) (err error) {
	if fmt.Sprintf("%v", v) == "" {
		err = fmt.Errorf("rule %s%s%s must have a value", rule.Prefix, rule.Name, rule.Op)
	}
	return
}

func CmdValidater(rule model.Rule, v []string) (err error) {
	if len(v) == 0 {
		err = fmt.Errorf("rule %s%s value must be an executable command", rule.Prefix, rule.Name)
	}

	return
}

func ExistsValidater(rule model.Rule, v []string) (err error) {
	if len(v) == 0 || len(v[0]) == 0 {
		err = fmt.Errorf("rule %s%s value must have a filepath", rule.Prefix, rule.Name)
	}

	return
}

func TestNameValidater(rule model.Rule, v string) (err error) {
	switch rule.Name {
	case "init", "report":
		if v != "" && !model.NameRegexp.MatchString(v) {
			err = fmt.Errorf("name %s does not match expected pattern: %s", v, model.NamePattern)
		}
	case "test":
		if !model.AbsNameRegexp.MatchString(v) {
			err = fmt.Errorf("name %s does not match expected pattern: %s", v, model.AbsNamePattern)
		}
	}
	return
}

func StringOrNotingValidater(rule model.Rule, v string) (err error) {
	if rule.Op == "=" && v == "" {
		err = fmt.Errorf("rule %s%s= must have a value", rule.Prefix, rule.Name)
	}
	return
}

func BooleanValidater(rule model.Rule, v bool) (err error) {
	if rule.Op != "" && rule.Op != "=" {
		err = fmt.Errorf("rule %s%s operator must be '='", rule.Prefix, rule.Name)
	}
	return
}

func MockValidater(rule model.Rule, v model.CmdMock) (err error) {
	// if strings.Contains(v.Cmd, "/") {
	// 	err = fmt.Errorf("rule %s%s command does not support slash", rule.Prefix, rule.Name)
	// }
	return
}

func Validate[T any](rule model.Rule, val T, validaters ...model.Validater[T]) (err error) {
	for _, v := range validaters {
		err = v(rule, val)
		if err != nil {
			return
		}
	}
	return
}

func Translate[T any](rule model.Rule, m model.Mapper[T], validaters ...model.Validater[T]) (val T, err error) {
	val, err = m(rule.Expected, rule.Op)
	if err != nil {
		err = fmt.Errorf("cannot map rule %s%s value: [%s] : %w", rule.Prefix, rule.Name, rule.Expected, err)
		return
	}
	err = Validate(rule, val, validaters...)
	return
}

func TranslateOptional[T comparable](rule model.Rule, m model.Mapper[T], validaters ...model.Validater[T]) (opt utilz.Optional[T], err error) {
	var val T
	val, err = m(rule.Expected, rule.Op)
	if err != nil {
		err = fmt.Errorf("cannot map rule %s%s value: [%s] : %w", rule.Prefix, rule.Name, rule.Expected, err)
		return
	}
	err = Validate(rule, val, validaters...)
	opt = utilz.OptionalOf(val)
	return
}

func ApplyConfig(c *model.Config, ruleExpr string) (ok bool, rule model.Rule, err error) {
	c.Prefix.Default(model.DefaultRulePrefix)
	ok, rule = c.SplitRuleExpr(ruleExpr)
	if !ok {
		return
	}

	var isAction, isTestConf, isSuiteConf, isFlowConf bool
	isAction, err = model.IsRuleOfKind(model.Actions, rule)
	if err != nil {
		return
	}
	isTestConf, err = model.IsRuleOfKind(model.TestConfigs, rule)
	if err != nil {
		return
	}
	isSuiteConf, err = model.IsRuleOfKind(model.SuiteConfigs, rule)
	if err != nil {
		return
	}
	isFlowConf, err = model.IsRuleOfKind(model.FlowConfigs, rule)
	if err != nil {
		return
	}
	ok = isAction || isTestConf || isSuiteConf || isFlowConf
	if !ok {
		return
	}

	if ok {
		switch rule.Name {
		case "global":
			c.Action = utilz.OptionalOf(model.Action(rule.Name))
			if rule.Op != "" || rule.Expected != "" {
				err = fmt.Errorf("rule %s%s does not accept an operator nor a value", rule.Prefix, rule.Name)
			}
		case "init":
			suiteName := rule.Expected
			err = Validate[string](rule, suiteName, TestNameValidater)
			c.Action = utilz.OptionalOf(model.Action(rule.Name))
			if suiteName != "" {
				c.TestSuite.Set(suiteName)
			}
			c.TestSuite.Default(model.DefaultTestSuiteName)
			c.Verbose.Default(model.DefaultInitedVerboseLevel)
		case "report":
			suiteName := rule.Expected
			err = Validate[string](rule, suiteName, TestNameValidater)
			c.Action = utilz.OptionalOf(model.Action(rule.Name))
			if suiteName != "" {
				c.TestSuite.Set(suiteName)
			} else {
				c.ReportAll.Set(true)
				//c.TestSuite.Clear()
			}
		case "test":
			c.TestSuite.Default(model.DefaultTestSuiteName)
			testName := rule.Expected
			err = Validate[string](rule, testName, TestNameValidater)
			c.Action = utilz.OptionalOf(model.Action(rule.Name))
			if testName != "" {
				matches := model.AbsNameRegexp.FindStringSubmatch(testName)
				//log.Printf("Matching names: %v", matches)
				if len(matches) == 2 {
					name := matches[1]
					if strings.HasSuffix(name, "/") {
						c.TestSuite = utilz.OptionalOf(name[0 : len(name)-1])
					} else {
						c.TestName = utilz.OptionalOf(name)
					}
				} else if len(matches) == 3 {
					name := matches[1]
					if strings.HasSuffix(name, "/") {
						c.TestSuite = utilz.OptionalOf(name[0 : len(name)-1])
					}
					c.TestName = utilz.OptionalOf(matches[2])
				} else {
					err = fmt.Errorf("bad test name: [%s]", testName)
				}
			}

		case "fork":
			c.ForkCount, err = TranslateOptional(rule, Uint16Mapper, OperatorValidater[uint16]("="), Uint16ValueValidater(1, 5))
		case "suiteTimeout":
			c.SuiteTimeout, err = TranslateOptional(rule, DurationMapper)

		case "async":
			c.Async, err = TranslateOptional(rule, BoolMapper, BooleanValidater)
		case "wait":
			c.Wait, err = TranslateOptional(rule, BoolMapper, BooleanValidater)
		case "ignore":
			c.Ignore, err = TranslateOptional(rule, BoolMapper, BooleanValidater)
		case "stopOnFailure":
			c.StopOnFailure, err = TranslateOptional(rule, BoolMapper, BooleanValidater)
		case "keepStdout":
			c.KeepStdout, err = TranslateOptional(rule, BoolMapper, BooleanValidater)
		case "keepStderr":
			c.KeepStderr, err = TranslateOptional(rule, BoolMapper, BooleanValidater)
		case "keepOutputs":
			var keepOutputs utilz.Optional[bool]
			keepOutputs, err = TranslateOptional(rule, BoolMapper, BooleanValidater)
			c.KeepStdout = keepOutputs
			c.KeepStderr = keepOutputs
		case "timeout":
			c.Timeout, err = TranslateOptional(rule, DurationMapper)
		case "runCount":
			c.RunCount, err = TranslateOptional(rule, Uint16Mapper, OperatorValidater[uint16]("="), Uint16ValueValidater(1, 1000))
		case "prefix":
			c.Prefix, err = TranslateOptional(rule, DummyMapper, OperatorValidater[string]("="))
		case "token":
			c.Token, err = TranslateOptional(rule, DummyMapper, OperatorValidater[string]("="), NotEmptyValidater[string])
		case "printToken":
			c.PrintToken, err = TranslateOptional(rule, BoolMapper, BooleanValidater)
		case "exportToken":
			c.ExportToken, err = TranslateOptional(rule, BoolMapper, BooleanValidater)
		case "keep":
			c.Keep, err = TranslateOptional(rule, BoolMapper, BooleanValidater)
		case "parallel":
		case "quiet":
			c.Quiet, err = TranslateOptional(rule, BoolMapper, BooleanValidater)
		case "mock":
			var mock model.CmdMock
			mock, err = Translate(rule, MockMapper, OperatorValidater[model.CmdMock]("=", ":"), NotEmptyValidater[model.CmdMock], MockValidater)
			if err == nil {
				if strings.HasPrefix(mock.Cmd, string(os.PathSeparator)) {
					c.RootMocks = append(c.RootMocks, mock)
				} else {
					c.Mocks = append(c.Mocks, mock)
				}
			}
		case "before":
			var cmdAndArgs []string
			cmdAndArgs, err = Translate(rule, CmdMapper, OperatorValidater[[]string]("="), CmdValidater)
			if err != nil {
				return
			}
			c.Before = append(c.Before, cmdAndArgs)
		case "after":
			var cmdAndArgs []string
			cmdAndArgs, err = Translate(rule, CmdMapper, OperatorValidater[[]string]("="), CmdValidater)
			if err != nil {
				return
			}
			c.After = append(c.After, cmdAndArgs)
		case "container":
			var image string
			image, err = Translate(rule, DummyMapper, OperatorValidater[string]("", "="))
			c.ContainerDisabled.Set(false)
			// Erase ContainerId because we will want a new Container
			c.ContainerId.Set("")
			if image == "true" {
				c.ContainerImage.Set(model.DefaultContainerImage)
			} else if image == "false" {
				c.ContainerImage.Clear()
				c.ContainerDisabled.Set(true)
			} else if image != "" {
				c.ContainerImage.Set(image)
			} else {
				c.ContainerImage.Set(model.DefaultContainerImage)
			}
		case "dirtyContainer":
			c.ContainerDirties, err = TranslateOptional(rule, DirtiesMapper, OperatorValidater[model.DirtyScope]("="))
		case "verbose":
			if rule.Op == "" {
				rule.Op = "="
				rule.Expected = fmt.Sprintf("%d", model.SHOW_PASSED)
			}
			var level uint16
			level, err = Translate(rule, Uint16Mapper, OperatorValidater[uint16]("="), Uint16ValueValidater(0, uint16(model.SHOW_ALL)))
			c.Verbose.Set(model.VerboseLevel(level))
		case "debug":
			if rule.Op == "" {
				rule.Op = "="
				rule.Expected = fmt.Sprintf("%d", model.DefaultDebugLevel)
			}
			var level uint16
			level, err = Translate(rule, Uint16Mapper, OperatorValidater[uint16]("="), Uint16ValueValidater(0, uint16(model.TRACE)))
			c.Debug.Set(model.DebugLevel(level))
		case "failuresLimit":
			c.TooMuchFailures, err = TranslateOptional(rule, Int32Mapper, OperatorValidater[int32]("="), Int32ValueValidater(-1, 1000))
		default:
			ok = false
		}
	}
	return
}

func BuildAssertion(cfg model.Config, ruleExpr string) (ok bool, assertion model.Assertion, err error) {
	ok, assertion.Rule = cfg.SplitRuleExpr(ruleExpr)
	if !ok {
		return
	}

	ok, err = model.IsRuleOfKind(model.Assertions, assertion.Rule)
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
		assertion.Asserter = func(cmd cmdz.Executer) (res model.AssertionResult, err error) {
			res.Value = ""
			res.Success = cmd.ExitCode() == 0
			return
		}
	case "fail":
		_, err = Translate(assertion.Rule, DummyMapper, OperatorValidater[string](""))
		if err != nil {
			return
		}
		assertion.Asserter = func(cmd cmdz.Executer) (res model.AssertionResult, err error) {
			res.Value = ""
			res.Success = cmd.ExitCode() > 0
			return
		}
	case "exit":
		var expectedExitCode uint16
		expectedExitCode, err = Translate(assertion.Rule, Uint16Mapper, Uint16ValueValidater(0, 255), OperatorValidater[uint16]("="))
		if err != nil {
			return
		}
		assertion.Asserter = func(cmd cmdz.Executer) (res model.AssertionResult, err error) {
			res.Value = cmd.ExitCode()
			res.Success = uint16(cmd.ExitCode()) == expectedExitCode
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
			assertion.Asserter = func(cmd cmdz.Executer) (res model.AssertionResult, err error) {
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
			fileContent, err = Translate(assertion.Rule, FileContentMapper, OperatorValidater[string]("=", ":", "!=", "!:", "@=", "@:"), NotEmptyForOpValidater[string](":", "!:"))
			if err != nil {
				return
			}
			assertion.Expected = fileContent
			assertion.Asserter = func(cmd cmdz.Executer) (res model.AssertionResult, err error) {
				res.Value = cmd.StdoutRecord()
				if assertion.Rule.Op == "=" || assertion.Rule.Op == "@=" {
					res.Success = cmd.StdoutRecord() == fileContent
				} else if assertion.Rule.Op == ":" || assertion.Rule.Op == "@:" {
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
			assertion.Asserter = func(cmd cmdz.Executer) (res model.AssertionResult, err error) {
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
			fileContent, err = Translate(assertion.Rule, FileContentMapper, OperatorValidater[string]("=", ":", "!=", "!:", "@=", "@:"), NotEmptyForOpValidater[string](":", "!:"))
			if err != nil {
				return
			}
			assertion.Expected = fileContent
			assertion.Asserter = func(cmd cmdz.Executer) (res model.AssertionResult, err error) {
				res.Value = cmd.StderrRecord()
				if assertion.Rule.Op == "=" || assertion.Rule.Op == "@=" {
					res.Success = cmd.StderrRecord() == fileContent
				} else if assertion.Rule.Op == ":" || assertion.Rule.Op == "@:" {
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
		assertion.Asserter = func(cmd cmdz.Executer) (res model.AssertionResult, err error) {
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

		assertion.Asserter = func(cmd cmdz.Executer) (res model.AssertionResult, err error) {
			var stat os.FileInfo
			stat, err = os.Stat(path)
			if errors.Is(err, os.ErrNotExist) {
				res.Success = false
				res.ErrMessage = fmt.Sprintf("file %s does not exists", path)
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
					res.ErrMessage = fmt.Sprintf("file %s have wrong permissions. Expected: [%s] but got: [%s] ", path, expectedPerms, actualPerms)
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

func ParseArgs(rulePrefix string, args []string) (cfg model.Config, assertions []model.Assertion, agg errorz.Aggregated) {
	var err error
	cfg.Prefix.Default(rulePrefix)
	ruleParsingStopper := rulePrefix + "--"

	// Check if ruleParsingStopper is present
	ruleParsingStopperPresent := false
	for _, arg := range args {
		if arg == ruleParsingStopper {
			ruleParsingStopperPresent = true
			break
		}
	}

	// Concatenate args with space before ruleParsingStopper
	if ruleParsingStopperPresent {
		var concatenatedArgs []string
		// Check all args before ruleParsingStopper
		var buffer string
		for p, arg := range args {
			if arg == ruleParsingStopper {
				if buffer != "" {
					concatenatedArgs = append(concatenatedArgs, buffer)
				}
				concatenatedArgs = append(concatenatedArgs, args[p:]...)
				break
			} else if model.MatchRuleDef(rulePrefix, arg, model.Concatenables...) {
				// Concatenable rule
				if buffer != "" {
					// flush buffer
					concatenatedArgs = append(concatenatedArgs, buffer)
				}
				// init buffer
				buffer = arg
			} else if strings.HasPrefix(arg, rulePrefix) {
				if buffer != "" {
					// flush buffer
					concatenatedArgs = append(concatenatedArgs, buffer)
				}
				buffer = ""
				concatenatedArgs = append(concatenatedArgs, arg)
			} else if buffer == "" {
				concatenatedArgs = append(concatenatedArgs, arg)
			} else {
				buffer += " " + arg
			}
		}

		logger.Debug("concatenated args", "args", args, "concatenated", concatenatedArgs)
		// replace args by concatenated ones
		args = concatenatedArgs
	}

	var rules []model.Rule
	parseRules := true
	for _, arg := range args {

		var ok bool
		if parseRules && arg == ruleParsingStopper {
			// Reached rule parsing stopper
			if len(cfg.CmdAndArgs) > 0 {
				err = fmt.Errorf("found some cmd: [%s] before rule parsing stopper %s", cfg.CmdAndArgs, ruleParsingStopper)
				agg.Add(err)
				//continue
			}
			// stop parsing rules
			parseRules = false
			continue
		}
		if parseRules && cfg.IsRule(arg) {
			var rule model.Rule
			ok, rule, err = ApplyConfig(&cfg, arg)
			if err != nil {
				agg.Add(err)
			}
			if ok {
				rules = append(rules, rule)
				continue
			}
			var assertion model.Assertion
			ok, assertion, err = BuildAssertion(cfg, arg)
			if err != nil {
				agg.Add(err)
			}
			if ok {
				assertions = append(assertions, assertion)
				rules = append(rules, assertion.Rule)
				continue
			}
			err = fmt.Errorf("rule %s does not exists", arg)
			agg.Add(err)
		} else {
			cfg.CmdAndArgs = append(cfg.CmdAndArgs, arg)
		}
	}

	if cfg.Action.IsEmpty() {
		// If no action supplied add implicit test rule.
		var rule model.Rule
		_, rule, err = ApplyConfig(&cfg, cfg.Prefix.Get()+"test")
		if err != nil {
			agg.Add(err)
		}
		rules = append(rules, rule)
		cfg.Action.Default(model.TestAction)
		cfg.TestSuite.Default(model.DefaultTestSuiteName)
	}

	if cfg.Action.Is(model.TestAction) {
		// If no status assertion found add an implicit success rule
		statusAssertionFound := false
		for _, a := range assertions {
			statusAssertionFound = statusAssertionFound || a.Name == "success" || a.Name == "fail" || a.Name == "exit" // || a.Name == "cmd"
		}
		if !statusAssertionFound {
			_, successAssertion, _ := BuildAssertion(cfg, cfg.Prefix.Get()+"success")
			assertions = append(assertions, successAssertion)
		}
	}

	if (cfg.Action.Is(model.InitAction) || cfg.Action.Is(model.ReportAction)) && len(cfg.CmdAndArgs) > 0 {
		err = fmt.Errorf("you cannot run commands with action %s%s", cfg.Prefix.Get(), cfg.Action.Get())
		agg.Add(err)
	}

	if cfg.Action.Is(model.ReportAction) {
		cfg.Async.Default(true)
		cfg.Wait.Default(true)
	}

	err = ValidateMutualyExclusiveRules(rules...)
	if err != nil {
		agg.Add(err)
	}
	err = ValidateOnceOnlyDefinedRule(rules...)
	if err != nil {
		agg.Add(err)
	}

	var cfgScope model.ConfigScope
	switch cfg.Action.Get() {
	case model.GlobalAction:
		cfgScope = model.GLOBAL_SCOPE
	case model.InitAction:
		cfgScope = model.SUITE_SCOPE
	case model.TestAction:
		cfgScope = model.TEST_SCOPE
	}

	if cfg.ContainerImage.IsPresent() {
		cfg.ContainerScope = utilz.OptionalOf(cfgScope)
	}
	return
}

func buildMutualyExclusiveCouples(rule model.RuleKey, exclusiveRules ...model.RuleKey) (res [][]model.RuleKey) {
	for _, e := range exclusiveRules {
		res = append(res, []model.RuleKey{rule, e})
	}
	return
}

func ruleKey(s ...string) (r model.RuleKey) {
	r.Name = s[0]
	r.Op = "all"
	if len(s) > 1 {
		r.Op = s[1]
	}
	return
}

func ruleKeys(ruleDefs ...[]model.RuleDefinition) (r []model.RuleKey) {
	for _, ruleDef := range ruleDefs {
		for _, def := range ruleDef {
			r = append(r, model.RuleKey{Name: def.Name, Op: "all"})
		}
	}
	return
}

// ValidateOnceOnlyDefinedRule => verify rules which cannot be defined multiple times are not defined twice or more
func ValidateOnceOnlyDefinedRule(rules ...model.Rule) (err error) {
	multiDefinedRules := []model.RuleKey{
		{"stdout", "~"}, {"stderr", "~"}, {"stdout", "!~"}, {"stderr", "!~"},
		{"stdout", "!="}, {"stderr", "!="},
		{"stdout", ":"}, {"stderr", ":"}, {"stdout", "!:"}, {"stderr", "!:"},
		{"stdout", "@:"}, {"stderr", "@:"},
		{"before", "="}, {"after", "="}, {"mock", "="}, {"mock", ":"}, {"verbose", "="},
	}
	matches := map[model.RuleKey][]model.Rule{}
	for _, rule := range rules {
		key := model.RuleKey{rule.Name, rule.Op}
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

func ValidateMutualyExclusiveRules(rules ...model.Rule) (err error) {
	// FIXME: stdout= and stdout~ are ME ; stdout= and stdout= are ME but stdout~ and stdout~ are not ME
	MutualyExclusiveRules := [][]model.RuleKey{
		ruleKeys(model.Actions),
		{{"fail", "all"}, {"success", "all"}, {"exit", "all"}},
		//{{"fail", "all"}, {"success", "all"}, {"cmd", "all"}},
		{{"test", "all"}, {"token", ""}},
		{{"report", "all"}, {"token", ""}},
	}

	exlusiveRules := MutualyExclusiveRules
	exlusiveRules = append(exlusiveRules, buildMutualyExclusiveCouples(model.RuleKey{"global", "all"}, ruleKeys(model.Assertions)...)...)
	exlusiveRules = append(exlusiveRules, buildMutualyExclusiveCouples(model.RuleKey{"init", "all"}, ruleKeys(model.Assertions)...)...)
	exlusiveRules = append(exlusiveRules, buildMutualyExclusiveCouples(model.RuleKey{"report", "all"}, ruleKeys(model.Assertions, model.TestConfigs, model.SuiteConfigs)...)...)
	exlusiveRules = append(exlusiveRules, buildMutualyExclusiveCouples(model.RuleKey{"test", "all"}, model.RuleKey{"suiteTimeout", "all"}, model.RuleKey{"fork", "all"})...)
	exlusiveRules = append(exlusiveRules, buildMutualyExclusiveCouples(ruleKey("keepOutputs"), ruleKey("keepStdout"), ruleKey("keepStderr"))...)
	exlusiveRules = append(exlusiveRules, buildMutualyExclusiveCouples(ruleKey("stdout", "="), ruleKey("stdout", "~"), ruleKey("stdout", "!~"), ruleKey("stdout", "!="), ruleKey("stdout", ":"), ruleKey("stdout", "!:"), ruleKey("stdout", "@:"))...)
	exlusiveRules = append(exlusiveRules, buildMutualyExclusiveCouples(ruleKey("stderr", "="), ruleKey("stderr", "~"), ruleKey("stderr", "!~"), ruleKey("stderr", "!="), ruleKey("stderr", ":"), ruleKey("stderr", "!:"), ruleKey("stderr", "@:"))...)

	// Compter le nombre de match pour chaque key
	// Pour chaque MER compter le nombre de key
	//matches := map[RuleKey][]Rule{}
	matchingMers := [][]model.RuleKey{}
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
