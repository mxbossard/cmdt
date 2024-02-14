package main

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSplitRuleExpr(t *testing.T) {
	var ok bool
	var rule Rule

	ok, _ = SplitRuleExpr("foo")
	assert.False(t, ok)

	ok, rule = SplitRuleExpr("@foo")
	assert.True(t, ok)
	assert.Equal(t, "foo", rule.Name)
	assert.Equal(t, "", rule.Op)
	assert.Equal(t, "", rule.Expected)

	ok, rule = SplitRuleExpr("@bar=")
	assert.True(t, ok)
	assert.Equal(t, "bar", rule.Name)
	assert.Equal(t, "=", rule.Op)
	assert.Equal(t, "", rule.Expected)

	ok, rule = SplitRuleExpr("@baz~pif")
	assert.True(t, ok)
	assert.Equal(t, "baz", rule.Name)
	assert.Equal(t, "~", rule.Op)
	assert.Equal(t, "pif", rule.Expected)
}

func TestApplyConfig(t *testing.T) {
	var ok bool
	var cfg Context
	var err error

	ok, _, err = ApplyConfig(&cfg, "foo")
	assert.NoError(t, err)
	assert.False(t, ok)

	ok, _, err = ApplyConfig(&cfg, "@foo")
	assert.NoError(t, err)
	assert.False(t, ok)

	ok, _, err = ApplyConfig(&cfg, "@fork")
	assert.Error(t, err)
	assert.True(t, ok)

	ok, _, err = ApplyConfig(&cfg, "@fork=")
	assert.Error(t, err)
	assert.True(t, ok)

	ok, _, err = ApplyConfig(&cfg, "@fork~")
	assert.Error(t, err)
	assert.True(t, ok)

	expectedFork := 3
	ok, _, err = ApplyConfig(&cfg, fmt.Sprintf("@fork=%d", expectedFork))
	assert.NoError(t, err)
	assert.True(t, ok)
	assert.Equal(t, expectedFork, cfg.ForkCount)

	expectedFork = 7
	ok, _, err = ApplyConfig(&cfg, fmt.Sprintf("@fork=%d", expectedFork))
	assert.Error(t, err)
	assert.True(t, ok)

}

func TestBuildAssertion(t *testing.T) {
	var ok bool
	var assertion Assertion
	var err error

	ok, _, err = BuildAssertion("foo")
	assert.NoError(t, err)
	assert.False(t, ok)

	ok, _, err = BuildAssertion("@foo")
	assert.NoError(t, err)
	assert.False(t, ok)

	ok, assertion, err = BuildAssertion("@fail")
	assert.NoError(t, err)
	assert.True(t, ok)
	assert.Equal(t, "fail", assertion.Name)
	assert.Equal(t, "", assertion.Op)
	assert.Equal(t, "", assertion.Expected)

	ok, assertion, err = BuildAssertion("@stdout=")
	assert.NoError(t, err)
	assert.True(t, ok)
	assert.Equal(t, "stdout", assertion.Name)
	assert.Equal(t, "=", assertion.Op)
	assert.Equal(t, "", assertion.Expected)

	ok, assertion, err = BuildAssertion("@stdout:baz")
	assert.NoError(t, err)
	assert.True(t, ok)
	assert.Equal(t, "stdout", assertion.Name)
	assert.Equal(t, ":", assertion.Op)
	assert.Equal(t, "baz", assertion.Expected)

	ok, assertion, err = BuildAssertion("@stdout~/baz/i")
	assert.NoError(t, err)
	assert.True(t, ok)
	assert.Equal(t, "stdout", assertion.Name)
	assert.Equal(t, "~", assertion.Op)
	assert.Equal(t, "(?i)baz", assertion.Expected)

	_, assertion, err = BuildAssertion("@stdout~/baz")
	assert.Error(t, err)

	_, assertion, err = BuildAssertion("@stdout~")
	assert.Error(t, err)

	ok, assertion, err = BuildAssertion("@stdout+")
	assert.Error(t, err)
	assert.False(t, ok)
}

func TestParseArgs(t *testing.T) {
	var cfg Context
	var cmdAndArgs []string
	var assertions []Assertion
	var err error

	// Parse command and args without config nor assertions
	cfg, cmdAndArgs, assertions, err = ParseArgs([]string{"foo", "bar", "baz"})
	require.NoError(t, err)
	assert.Equal(t, []string{"foo", "bar", "baz"}, cmdAndArgs)
	assert.Equal(t, DefaultTestSuiteName, cfg.TestSuite)
	assert.Equal(t, "", cfg.TestName)
	assert.Len(t, assertions, 1)

	// Parse command and args with a not existing rule
	_, _, _, err = ParseArgs([]string{"foo", "bar", "@foo"})
	assert.Error(t, err)

	// Parse command and args with an existing rule
	cfg, cmdAndArgs, assertions, err = ParseArgs([]string{"foo", "bar", "@fail", "@test=pif"})
	require.NoError(t, err)
	assert.Equal(t, []string{"foo", "bar"}, cmdAndArgs)
	assert.Equal(t, DefaultTestSuiteName, cfg.TestSuite)
	assert.Equal(t, "pif", cfg.TestName)
	assert.Len(t, assertions, 1)

	// Parse command and args with an existing rule
	cfg, cmdAndArgs, assertions, err = ParseArgs([]string{"foo", "bar", "@fail", "@stdout=", "@test=paf/"})
	require.NoError(t, err)
	assert.Equal(t, []string{"foo", "bar"}, cmdAndArgs)
	assert.Equal(t, "paf", cfg.TestSuite)
	assert.Equal(t, "", cfg.TestName)
	assert.Len(t, assertions, 2)

	// Parse command and args with mutualy exclusive rules
	cfg, cmdAndArgs, assertions, err = ParseArgs([]string{"foo", "bar", "@fail", "@success"})
	require.Error(t, err)
	assert.Equal(t, []string{"foo", "bar"}, cmdAndArgs)
	assert.Equal(t, DefaultTestSuiteName, cfg.TestSuite)
	assert.Equal(t, "", cfg.TestName)
	assert.Len(t, assertions, 2)

	// Parse command and args with a test name
	cfg, cmdAndArgs, assertions, err = ParseArgs([]string{"foo", "bar", "@test=foo", "@success"})
	require.NoError(t, err)
	assert.Equal(t, []string{"foo", "bar"}, cmdAndArgs)
	assert.Equal(t, "foo", cfg.TestName)
	assert.Len(t, assertions, 1)

	// Parse command and args with an absolute test name
	cfg, cmdAndArgs, assertions, err = ParseArgs([]string{"foo", "bar", "@test=bar/foo", "@success"})
	require.NoError(t, err)
	assert.Equal(t, []string{"foo", "bar"}, cmdAndArgs)
	assert.Equal(t, "bar", cfg.TestSuite)
	assert.Equal(t, "foo", cfg.TestName)
	assert.Len(t, assertions, 1)

}

func buildRule(name, op string) (r Rule) {
	r.Name = name
	r.Op = op
	return
}

func TestValidateOnceOnlyDefinedRule(t *testing.T) {
	var err error

	err = ValidateOnceOnlyDefinedRule(buildRule("init", "="), buildRule("timeout", "="))
	assert.NoError(t, err)

	err = ValidateOnceOnlyDefinedRule(buildRule("init", "="), buildRule("report", "="))
	assert.NoError(t, err)

	err = ValidateOnceOnlyDefinedRule(buildRule("stdout", "~"), buildRule("stdout", "~"))
	assert.NoError(t, err)

	err = ValidateOnceOnlyDefinedRule(buildRule("stdout", "~"), buildRule("stdout", "!~"))
	assert.NoError(t, err)

	err = ValidateOnceOnlyDefinedRule(buildRule("stdout", "~"), buildRule("stdout", "!="))
	assert.NoError(t, err)

	err = ValidateOnceOnlyDefinedRule(buildRule("stdout", "~"), buildRule("stdout", "!~"), buildRule("stdout", "!="))
	assert.NoError(t, err)

	err = ValidateOnceOnlyDefinedRule(buildRule("stdout", "="), buildRule("stdout", "~"))
	assert.NoError(t, err)

	err = ValidateOnceOnlyDefinedRule(buildRule("stdout", "="), buildRule("stdout", "!~"))
	assert.NoError(t, err)

	err = ValidateOnceOnlyDefinedRule(buildRule("stdout", "="), buildRule("stdout", "!="))
	assert.NoError(t, err)

	err = ValidateOnceOnlyDefinedRule(buildRule("stdout", "="), buildRule("stdout", "="))
	assert.Error(t, err)

}

func TestValidateMutualyExclusiveRules(t *testing.T) {
	var err error

	err = ValidateMutualyExclusiveRules(buildRule("init", ""), buildRule("timeout", "="))
	assert.NoError(t, err)

	err = ValidateMutualyExclusiveRules(buildRule("stdout", "="), buildRule("stderr", "~"))
	assert.NoError(t, err)

	err = ValidateMutualyExclusiveRules(buildRule("init", "="), buildRule("report", ""))
	assert.Error(t, err)

	err = ValidateMutualyExclusiveRules(buildRule("stdout", "!="), buildRule("stdout", "~"))
	assert.NoError(t, err)

	err = ValidateMutualyExclusiveRules(buildRule("stdout", "!="), buildRule("stdout", "!~"))
	assert.NoError(t, err)

	err = ValidateMutualyExclusiveRules(buildRule("stdout", "!="), buildRule("stdout", "~"), buildRule("stdout", "!~"))
	assert.NoError(t, err)

	err = ValidateMutualyExclusiveRules(buildRule("stdout", "="), buildRule("stdout", "~"))
	assert.Error(t, err)

	err = ValidateMutualyExclusiveRules(buildRule("stdout", "="), buildRule("stdout", "!~"))
	assert.Error(t, err)

	err = ValidateMutualyExclusiveRules(buildRule("stdout", "="), buildRule("stdout", "!="))
	assert.Error(t, err)
}

func TestMockMapper(t *testing.T) {
	var m CmdMock
	var err error

	m, err = MockMapper("true")
	require.NoError(t, err)
	assert.Equal(t, "true", m.Cmd)
	assert.Empty(t, m.Args)
	assert.Equal(t, "", m.Stdin)
	assert.Equal(t, "", m.Stdout)
	assert.Equal(t, "", m.Stderr)
	assert.Equal(t, 0, m.ExitCode)
	assert.True(t, m.Delegate)

	m, err = MockMapper("true arg1 arg2")
	require.NoError(t, err)
	assert.Equal(t, "true", m.Cmd)
	assert.Equal(t, []string{"arg1", "arg2"}, m.Args)
	assert.Equal(t, "", m.Stdin)
	assert.Equal(t, "", m.Stdout)
	assert.Equal(t, "", m.Stderr)
	assert.Equal(t, 0, m.ExitCode)
	assert.True(t, m.Delegate)

	m, err = MockMapper("true arg1 arg2;stdin=foo")
	require.NoError(t, err)
	assert.Equal(t, "true", m.Cmd)
	assert.Equal(t, []string{"arg1", "arg2"}, m.Args)
	assert.Equal(t, "foo", m.Stdin)
	assert.Equal(t, "", m.Stdout)
	assert.Equal(t, "", m.Stderr)
	assert.Equal(t, 0, m.ExitCode)
	assert.True(t, m.Delegate)

	m, err = MockMapper("true arg1 arg2;stdin=foo;stdout=bar=;stderr=pif paf;exit=12")
	require.NoError(t, err)
	assert.Equal(t, "true", m.Cmd)
	assert.Equal(t, []string{"arg1", "arg2"}, m.Args)
	assert.Equal(t, "foo", m.Stdin)
	assert.Equal(t, "bar=", m.Stdout)
	assert.Equal(t, "pif paf", m.Stderr)
	assert.Equal(t, 12, m.ExitCode)
	assert.False(t, m.Delegate)

	m, err = MockMapper(",true,arg 1,arg 2")
	require.NoError(t, err)
	assert.Equal(t, "true", m.Cmd)
	assert.Equal(t, []string{"arg 1", "arg 2"}, m.Args)
	assert.Equal(t, "", m.Stdin)
	assert.Equal(t, "", m.Stdout)
	assert.Equal(t, "", m.Stderr)
	assert.Equal(t, 0, m.ExitCode)
	assert.True(t, m.Delegate)

	m, err = MockMapper(".true.arg 1.arg 2")
	require.NoError(t, err)
	assert.Equal(t, "true", m.Cmd)
	assert.Equal(t, []string{"arg 1", "arg 2"}, m.Args)
	assert.Equal(t, "", m.Stdin)
	assert.Equal(t, "", m.Stdout)
	assert.Equal(t, "", m.Stderr)
	assert.Equal(t, 0, m.ExitCode)
	assert.True(t, m.Delegate)

	m, err = MockMapper("|true|arg 1|arg 2;stdin=foo;stdout=bar=;stderr=pif paf;exit=12")
	require.NoError(t, err)
	assert.Equal(t, "true", m.Cmd)
	assert.Equal(t, []string{"arg 1", "arg 2"}, m.Args)
	assert.Equal(t, "foo", m.Stdin)
	assert.Equal(t, "bar=", m.Stdout)
	assert.Equal(t, "pif paf", m.Stderr)
	assert.Equal(t, 12, m.ExitCode)
	assert.False(t, m.Delegate)

}
