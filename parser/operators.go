package parser

import (
	"fmt"
	"strconv"
	"time"
)

var (

	// VALIDATERS
	isUint8         = buildValidater(func(val string) bool { return true })
	isUint16        = buildValidater(func(val string) bool { return true })
	isInt32         = buildValidater(func(val string) bool { return true })
	isUint32        = buildValidater(func(val string) bool { return true })
	isString        = buildValidater(func(val string) bool { return true })
	isStringOrEmpty = buildValidater(func(val string) bool { return true })
	isBoolean       = buildValidater(func(val string) bool { return true })
	isFilepath      = buildValidater(func(val string) bool { return true })
	isSuiteName     = buildValidater(func(val string) bool { return true })
	isTestName      = buildValidater(func(val string) bool { return true })
	isDuration      = buildValidater(func(val string) bool { return true })
	isCmd           = buildValidater(func(val string) bool { return true })
	isMock          = buildValidater(func(val string) bool { return true })
	//isRuleName      = buildValidater(func(val string) bool { return true })

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
	equalUint8 = buildOp("=", &intMapper, intValidater(0, 255)) // TODO: implements equalInt(min, max)
	// booleans
	equalBoolean = buildOp("=", &boolMapper)
	// durations
	equalDuration = buildOp("=", &durationMapper)
	// patterns
	equalSuiteName = buildOp("=", &stringMapper)
	equalTestName  = buildOp("=", &stringMapper)
	// cmd & mock
	equalCmd  = buildOp("=", &stringMapper)
	equalMock = buildOp("=", &stringMapper)
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

func buildValidater(f func(string) bool) *validater[string] {
	var res validater[string] = func(s string) (err error) {
		//TODO
		return nil
	}
	return &res
}

var intMapper mapper[int] = func(s string) (int, error) { return strconv.Atoi(s) }
var stringMapper mapper[string] = func(s string) (string, error) { return s, nil }
var boolMapper mapper[bool] = func(s string) (bool, error) {
	// TODO
	return false, nil
}
var durationMapper mapper[time.Duration] = func(s string) (time.Duration, error) {
	// TODO
	return 0, nil
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
