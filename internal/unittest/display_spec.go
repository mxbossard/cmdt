package unittest

import (
	"cmdt/internal/model"
	"fmt"
	"regexp"
	"testing"

	"github.com/mxbossard/utilz/anzi"
	"github.com/stretchr/testify/assert"
)

// ------- Spec for data displayed

func globalSpecPattern(v model.VerboseLevel) *regexp.Regexp {
	panic("not implemented yet")
}

func suiteTitleSpecPattern(v model.VerboseLevel) *regexp.Regexp {
	panic("not implemented yet")
}

func testTitleSpecPattern(v model.VerboseLevel, suite, name, ct, cmd, seq string) *regexp.Regexp {
	if ct == "" {
		ct = "on host"
	}
	if name == "" {
		name = cmd
	}
	return regexp.MustCompile(fmt.Sprintf(`^\[\d+\] Test \[%s\]\(%s\)>%s #%s...\s*.*`, suite, ct, name, seq))
}

func testOutcomeSpecPattern(v model.VerboseLevel, cmd, out, err string, oc model.Outcome) *regexp.Regexp {
	outputInfo := ""
	if out == "" && err == "" {
		outputInfo = "<empty outputs>"
	}
	return regexp.MustCompile(fmt.Sprintf(`.*\s+%s\n\s*Executing cmd:\s*\[%s\]\s*%s\s*$`, oc, cmd, outputInfo))
}

func testOutputSpecPattern(v model.VerboseLevel) *regexp.Regexp {
	panic("not implemented yet")
}

func suiteReportSpecPattern(v model.VerboseLevel) *regexp.Regexp {
	panic("not implemented yet")
}

func gobalReportSpecPattern(v model.VerboseLevel) *regexp.Regexp {
	panic("not implemented yet")
}

func gobalReportAllSpecPattern(v model.VerboseLevel) *regexp.Regexp {
	panic("not implemented yet")
}

func MatchGlobalSpec(t *testing.T, v model.VerboseLevel, result string) {
	panic("not implemented yet")
}

func MatchSuiteTitleSpec(t *testing.T, v model.VerboseLevel, suite string, result string) {
	panic("not implemented yet")
}

func MatchTestTitleSpec(t *testing.T, v model.VerboseLevel, suite, name, ct, cmd, seq string, oc model.Outcome, result string) {
	assert.Regexp(t, testTitleSpecPattern(v, suite, name, ct, cmd, seq), anzi.Unformat(result))
}

func MatchTestOutputSpec(t *testing.T, v model.VerboseLevel, cmd, out, err string, oc model.Outcome, result string) {
	assert.Regexp(t, testOutcomeSpecPattern(v, cmd, out, err, oc), anzi.Unformat(result))
}

func MatchSuiteReportSpec(t *testing.T, v model.VerboseLevel, suite string, succeed, failed, errored, timeouted, ignored int, oc model.Outcome, result string) {
	panic("not implemented yet")
}

func MatchGlobalReportSpec(t *testing.T, v model.VerboseLevel, suite string, succeed, failed, errored, timeouted, ignored int, oc model.Outcome, result string) {
	panic("not implemented yet")
}

func MatchGlobalReportAllSpec(t *testing.T, v model.VerboseLevel, suite string, succeed, failed, errored, timeouted, ignored int, oc model.Outcome, result string) {
	panic("not implemented yet")
}
