package service

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"mby.fr/cmdtest/model"
	"mby.fr/utils/errorz"
)

func TestApplyConfig(t *testing.T) {
	var ok bool
	var cfg model.Config
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
	require.True(t, cfg.ForkCount.IsPresent())
	assert.Equal(t, expectedFork, cfg.ForkCount.Get())

	expectedFork = 7
	ok, _, err = ApplyConfig(&cfg, fmt.Sprintf("@fork=%d", expectedFork))
	assert.Error(t, err)
	assert.True(t, ok)
	require.True(t, cfg.ForkCount.IsPresent())
	assert.Equal(t, expectedFork, cfg.ForkCount.Get())
}

func TestBuildAssertion(t *testing.T) {
	var ok bool
	var assertion model.Assertion
	var err error
	var cfg model.Config

	cfg = model.NewGlobalDefaultConfig()
	ok, _, err = BuildAssertion(cfg, "foo")
	assert.NoError(t, err)
	assert.False(t, ok)

	ok, _, err = BuildAssertion(cfg, "@foo")
	assert.NoError(t, err)
	assert.False(t, ok)

	ok, assertion, err = BuildAssertion(cfg, "@fail")
	assert.NoError(t, err)
	assert.True(t, ok)
	assert.Equal(t, "fail", assertion.Name)
	assert.Equal(t, "", assertion.Op)
	assert.Equal(t, "", assertion.Expected)

	ok, assertion, err = BuildAssertion(cfg, "@stdout=")
	assert.NoError(t, err)
	assert.True(t, ok)
	assert.Equal(t, "stdout", assertion.Name)
	assert.Equal(t, "=", assertion.Op)
	assert.Equal(t, "", assertion.Expected)

	ok, assertion, err = BuildAssertion(cfg, "@stdout:baz")
	assert.NoError(t, err)
	assert.True(t, ok)
	assert.Equal(t, "stdout", assertion.Name)
	assert.Equal(t, ":", assertion.Op)
	assert.Equal(t, "baz", assertion.Expected)

	ok, assertion, err = BuildAssertion(cfg, "@stdout~/baz/i")
	assert.NoError(t, err)
	assert.True(t, ok)
	assert.Equal(t, "stdout", assertion.Name)
	assert.Equal(t, "~", assertion.Op)
	assert.Equal(t, "(?i)baz", assertion.Expected)

	_, assertion, err = BuildAssertion(cfg, "@stdout~/baz")
	assert.Error(t, err)

	_, assertion, err = BuildAssertion(cfg, "@stdout~")
	assert.Error(t, err)

	ok, assertion, err = BuildAssertion(cfg, "@stdout+")
	assert.Error(t, err)
	assert.False(t, ok)
}

func TestParseArgs(t *testing.T) {
	var cfg model.Config
	var assertions []model.Assertion
	var agg errorz.Aggregated

	// Parse command and args without config nor assertions
	cfg, assertions, agg = ParseArgs("@", []string{"foo", "bar", "baz"})
	require.NoError(t, agg.Return())
	assert.Equal(t, []string{"foo", "bar", "baz"}, cfg.CmdAndArgs)
	assert.Equal(t, model.DefaultTestSuiteName, cfg.TestSuite.Get())
	require.False(t, cfg.TestName.IsPresent())
	assert.Len(t, assertions, 1)

	// Parse command and args with a not existing rule
	_, _, agg = ParseArgs("@", []string{"foo", "bar", "@foo"})
	assert.Error(t, agg.Return())

	// Parse command and args with an existing rule
	cfg, assertions, agg = ParseArgs("@", []string{"foo", "bar", "@fail", "@test=pif"})
	require.NoError(t, agg.Return())
	assert.Equal(t, []string{"foo", "bar"}, cfg.CmdAndArgs)
	assert.Equal(t, model.DefaultTestSuiteName, cfg.TestSuite.Get())
	require.True(t, cfg.TestName.IsPresent())
	assert.Equal(t, "pif", cfg.TestName.Get())
	assert.Len(t, assertions, 1)

	// Parse command and args with an existing rule
	cfg, assertions, agg = ParseArgs("@", []string{"foo", "bar", "@fail", "@stdout=", "@test=paf/"})
	require.NoError(t, agg.Return())
	assert.Equal(t, []string{"foo", "bar"}, cfg.CmdAndArgs)
	assert.Equal(t, "paf", cfg.TestSuite.Get())
	require.True(t, cfg.TestName.IsPresent())
	assert.Equal(t, "", cfg.TestName.Get())
	assert.Len(t, assertions, 2)

	// Parse command and args with mutualy exclusive rules
	cfg, assertions, agg = ParseArgs("@", []string{"foo", "bar", "@fail", "@success"})
	require.Error(t, agg.Return())
	assert.Equal(t, []string{"foo", "bar"}, cfg.CmdAndArgs)
	assert.Equal(t, model.DefaultTestSuiteName, cfg.TestSuite.Get())
	require.False(t, cfg.TestName.IsPresent())
	assert.Len(t, assertions, 2)

	// Parse command and args with a test name
	cfg, assertions, agg = ParseArgs("@", []string{"foo", "bar", "@test=foo", "@success"})
	require.NoError(t, agg.Return())
	assert.Equal(t, []string{"foo", "bar"}, cfg.CmdAndArgs)
	require.True(t, cfg.TestName.IsPresent())
	assert.Equal(t, "foo", cfg.TestName.Get())
	assert.Len(t, assertions, 1)

	// Parse command and args with an absolute test name
	cfg, assertions, agg = ParseArgs("@", []string{"foo", "bar", "@test=bar/foo", "@success"})
	require.NoError(t, agg.Return())
	assert.Equal(t, []string{"foo", "bar"}, cfg.CmdAndArgs)
	assert.Equal(t, "bar", cfg.TestSuite.Get())
	require.True(t, cfg.TestName.IsPresent())
	assert.Equal(t, "foo", cfg.TestName.Get())
	assert.Len(t, assertions, 1)

}

func buildRule(name, op string) (r model.Rule) {
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
	var m model.CmdMock
	var err error

	m, err = MockMapper("true", "=")
	require.NoError(t, err)
	assert.Equal(t, "true", m.Cmd)
	assert.Empty(t, m.Args)
	assert.Nil(t, m.Stdin)
	assert.Equal(t, "", m.Stdout)
	assert.Equal(t, "", m.Stderr)
	assert.Equal(t, 0, m.ExitCode)
	assert.True(t, m.Delegate)

	m, err = MockMapper("true arg1 arg2", "=")
	require.NoError(t, err)
	assert.Equal(t, "true", m.Cmd)
	assert.Equal(t, []string{"arg1", "arg2"}, m.Args)
	assert.Nil(t, m.Stdin)
	assert.Equal(t, "", m.Stdout)
	assert.Equal(t, "", m.Stderr)
	assert.Equal(t, 0, m.ExitCode)
	assert.True(t, m.Delegate)

	expectedStdin := "foo"
	m, err = MockMapper("true arg1 arg2,stdin=foo", "=")
	require.NoError(t, err)
	assert.Equal(t, "true", m.Cmd)
	assert.Equal(t, []string{"arg1", "arg2"}, m.Args)
	assert.Equal(t, &expectedStdin, m.Stdin)
	assert.Equal(t, "", m.Stdout)
	assert.Equal(t, "", m.Stderr)
	assert.Equal(t, 0, m.ExitCode)
	assert.True(t, m.Delegate)

	m, err = MockMapper("true arg1 arg2,stdin=foo,stdout=bar=,stderr=pif paf,exit=12", "=")
	require.NoError(t, err)
	assert.Equal(t, "true", m.Cmd)
	assert.Equal(t, []string{"arg1", "arg2"}, m.Args)
	assert.Equal(t, &expectedStdin, m.Stdin)
	assert.Equal(t, "bar=", m.Stdout)
	assert.Equal(t, "pif paf", m.Stderr)
	assert.Equal(t, 12, m.ExitCode)
	assert.False(t, m.Delegate)

	m, err = MockMapper(";true;arg 1;arg 2", "=")
	require.NoError(t, err)
	assert.Equal(t, "true", m.Cmd)
	assert.Equal(t, []string{"arg 1", "arg 2"}, m.Args)
	assert.Nil(t, m.Stdin)
	assert.Equal(t, "", m.Stdout)
	assert.Equal(t, "", m.Stderr)
	assert.Equal(t, 0, m.ExitCode)
	assert.True(t, m.Delegate)

	m, err = MockMapper(":true:arg 1:arg 2", "=")
	require.NoError(t, err)
	assert.Equal(t, "true", m.Cmd)
	assert.Equal(t, []string{"arg 1", "arg 2"}, m.Args)
	assert.Nil(t, m.Stdin)
	assert.Equal(t, "", m.Stdout)
	assert.Equal(t, "", m.Stderr)
	assert.Equal(t, 0, m.ExitCode)
	assert.True(t, m.Delegate)

	m, err = MockMapper("|true|arg 1|arg 2,stdin=foo,stdout=bar=,stderr=pif paf,exit=12", "=")
	require.NoError(t, err)
	assert.Equal(t, "true", m.Cmd)
	assert.Equal(t, []string{"arg 1", "arg 2"}, m.Args)
	assert.Equal(t, &expectedStdin, m.Stdin)
	assert.Equal(t, "bar=", m.Stdout)
	assert.Equal(t, "pif paf", m.Stderr)
	assert.Equal(t, 12, m.ExitCode)
	assert.False(t, m.Delegate)

}
