package parser

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"mby.fr/cmdtest/model"
	"mby.fr/utils/filez"
)

var authorizedOperators = []string{"", " ", "=", "!=", ":", "!:", "~", "!~", "@=", "@:"}

var (
	// VALIDATERS
	isUint8         = buildValidater(func(val string) error { return nil })
	isUint16        = buildValidater(func(val string) error { return nil })
	isInt32         = buildValidater(func(val string) error { return nil })
	isUint32        = buildValidater(func(val string) error { return nil })
	isString        = buildValidater(func(val string) error { return nil })
	isStringOrEmpty = buildValidater(func(val string) error { return nil })
	isBoolean       = buildValidater(func(val string) error { return nil })
	isFilepath      = buildValidater(func(val string) error { return nil })
	isSuiteName     = buildValidater(func(val string) error { return nil })
	isTestName      = buildValidater(func(val string) error { return nil })
	isDuration      = buildValidater(func(val string) error { return nil })
	isCmd           = buildValidater(func(val string) error { return nil })
	isMock          = buildValidater(func(val string) error { return nil })
	//isRuleName      = buildValidater(func(val string) error { return nil })

	// OPERATORS

	// strings
	equalOneChar        = buildOp("=", &stringMapper, stringValidater(1, 1)) // TODO: implements equalString(minLength, maxLength)
	equalString         = buildOp("=", &stringMapper, stringValidater(1, 1024))
	equalStringOrEmpty  = buildOp("=", &stringMapper, stringValidater(0, 1024))
	notEqualString      = buildOp("!=", &stringMapper, stringValidater(1, 1024))
	notEqualStringEmpty = buildOp("!=", &stringMapper, stringValidater(0, 1024))
	containsString      = buildOp(":", &stringMapper, stringValidater(1, 1024))
	notContainsString   = buildOp("!:", &stringMapper, stringValidater(0, 1024))
	matchString         = buildOp("~", &stringMapper, stringValidater(1, 1024))
	notMatchString      = buildOp("!~", &stringMapper, stringValidater(1, 1024))
	// files
	equalFileContent    = buildOp("@=", &stringMapper, isFilepath)
	containsFileContent = buildOp("@:", &stringMapper, isFilepath)
	equalFilepath       = buildOp("=", &stringMapper, isFilepath)
	// numbers
	equalInt    = buildOp("=", &intMapper, intValidater(0, 255))                // TODO: implements equalInt(min, max)
	equalInt16  = buildOp("=", &int16Mapper, int16Validater(-32535, 32535))     // TODO: implements equalInt(min, max)
	equalInt32  = buildOp("=", &int32Mapper, int32Validater(-1000000, 1000000)) // TODO: implements equalInt(min, max)
	equalUint8  = buildOp("=", &uint8Mapper, uint8Validater(0, 255))            // TODO: implements equalInt(min, max)
	equalUint16 = buildOp("=", &uint16Mapper, uint16Validater(0, 65535))        // TODO: implements equalInt(min, max)
	// booleans
	equalBoolean = buildOp("=", &boolMapper)
	// durations
	equalDuration = buildOp("=", &durationMapper)
	// patterns
	equalSuiteName = buildOp("=", &stringMapper)
	equalTestName  = buildOp("=", &stringMapper)
	// cmd & mock
	equalCmd        = buildOp("=", &cmdMapper)
	equalMock       = buildOp("=", &mockMapper)
	equalDirtyScope = buildOp("=", &dirtiesMapper)
	//equalRuleName       = buildOp("=", isRuleName)

)

func noOp[T any]() *operator[T] {
	return buildOp[T]("", nil)
}

func noOps() []*operator[any] {
	return ops[any](noOp[any]())
}

func buildOp[T any](op string, mapper *mapper[T], validaters ...*validater[T]) *operator[T] {
	return &operator[T]{
		op:         op,
		mapper:     mapper,
		validaters: validaters,
	}
}

func buildValidater(f func(string) error) *validater[string] {
	var res validater[string] = func(s string) (err error) {
		return f(s)
	}
	return &res
}

var intMapper mapper[int] = func(op, val string) (int, error) { return strconv.Atoi(val) }
var int16Mapper mapper[int16] = func(op, val string) (int16, error) {
	i, err := strconv.Atoi(val)
	if err != nil {
		return 0, err
	}
	return int16(i), nil
}
var int32Mapper mapper[int32] = func(op, val string) (int32, error) {
	i, err := strconv.Atoi(val)
	if err != nil {
		return 0, err
	}
	return int32(i), nil
}
var uint8Mapper mapper[uint8] = func(op, val string) (uint8, error) {
	i, err := strconv.Atoi(val)
	if err != nil {
		return 0, err
	}
	return uint8(i), nil
}
var uint16Mapper mapper[uint16] = func(op, val string) (uint16, error) {
	i, err := strconv.Atoi(val)
	if err != nil {
		return 0, err
	}
	return uint16(i), nil
}
var stringMapper mapper[string] = func(op, val string) (string, error) { return val, nil }
var boolMapper mapper[bool] = func(op, val string) (res bool, err error) {
	if op == "" && val == "" {
		return true, nil
	}
	if val == "true" || val == "True" || val == "TRUE" || val == "1" {
		res = true
	} else if val == "false" || val == "False" || val == "FALSE" || val == "0" {
		res = false
	} else {
		err = fmt.Errorf("bool rule value must be true or false")
	}
	return
}
var durationMapper mapper[time.Duration] = func(op, val string) (time.Duration, error) {
	// TODO
	return 0, nil
}

// TODO: could have a Cmd type wrapping []string ?
var cmdMapper mapper[[]string] = func(op, val string) (v []string, err error) {
	// FIXME: should leverage simple and double quottes to split args
	if len(val) > 1 {
		separator := " "
		if val[0] == ';' || val[0] == ':' || val[0] == '|' {
			separator = val[0:1]
			val = val[1:]
		}
		v = strings.Split(val, separator)
		//log.Printf("CMD: [%v]", v)
	}
	return
}

var mockMapper mapper[model.CmdMock] = func(op, val string) (m model.CmdMock, err error) {
	var splitted, mockedCmdAndArgs []string
	if len(val) > 1 {
		splitted = strings.Split(val, ",")
		// cmd always defined first
	} else {
		splitted = append(splitted, val)
	}

	mockedCmdAndArgs, err = cmdMapper(op, splitted[0])
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
				m.ExitCode, err = uint16Mapper("=", value)
				if err != nil {
					// FIXME: aggregate errors
					return
				}
			} else if strings.HasPrefix(rule, "cmd=") {
				value := rule[4:]
				m.Delegate = false
				m.OnCallCmdAndArgs, err = cmdMapper("=", value)
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

var dirtiesMapper mapper[model.DirtyScope] = func(op, val string) (v model.DirtyScope, err error) {
	switch val {
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
		err = fmt.Errorf("dirty scope: %s not supported", val)
	}
	return
}

func intValidater(min, max int) *validater[int] {
	var f validater[int] = func(val int) error {
		if val < min {
			return fmt.Errorf("must be >= %d", min)
		}
		if val > max {
			return fmt.Errorf("must be <= %d", max)
		}
		return nil
	}
	return &f
}

func int16Validater(min, max int16) *validater[int16] {
	var f validater[int16] = func(val int16) error {
		if val < min {
			return fmt.Errorf("must be >= %d", min)
		}
		if val > max {
			return fmt.Errorf("must be <= %d", max)
		}
		return nil
	}
	return &f
}

func int32Validater(min, max int32) *validater[int32] {
	var f validater[int32] = func(val int32) error {
		if val < min {
			return fmt.Errorf("must be >= %d", min)
		}
		if val > max {
			return fmt.Errorf("must be <= %d", max)
		}
		return nil
	}
	return &f
}

func uint8Validater(min, max uint8) *validater[uint8] {
	var f validater[uint8] = func(val uint8) error {
		if val < min {
			return fmt.Errorf("must be >= %d", min)
		}
		if val > max {
			return fmt.Errorf("must be <= %d", max)
		}
		return nil
	}
	return &f
}

func uint16Validater(min, max uint16) *validater[uint16] {
	var f validater[uint16] = func(val uint16) error {
		if val < min {
			return fmt.Errorf("must be >= %d", min)
		}
		if val > max {
			return fmt.Errorf("must be <= %d", max)
		}
		return nil
	}
	return &f
}

func stringValidater(min, max int) *validater[string] {
	var f validater[string] = func(val string) error {
		if len(val) < min {
			return fmt.Errorf("length must be >= %d", min)
		}
		if len(val) > max {
			return fmt.Errorf("length must be <= %d", max)
		}
		return nil
	}
	return &f
}

var authorizedOperatorsPattern = buildAuthorizedOperatorsPattern()

func buildAuthorizedOperatorsPattern() regexp.Regexp {

	rex := "^" + authorizedOperators[0] + "$"
	for _, op := range authorizedOperators[1:] {
		rex = rex + "|^" + op + "$"
	}
	return *regexp.MustCompile(rex)
}

func isOperatorPrefixed(s string) bool {
	return operatorPrefix(s) != ""
}

func operatorPrefix(s string) string {
	maxLen := min(len(s), 3)
	for p := maxLen; p > 0; p-- {
		m := authorizedOperatorsPattern.FindString(s[:p])
		if m != "" {
			return m
		}
	}
	return ""
}
