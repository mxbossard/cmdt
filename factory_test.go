package main

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSplitRuleExpr(t *testing.T) {
	var ok bool
	var name, operator, value string

	ok, _, _, _ = SplitRuleExpr("foo")
	assert.False(t, ok)

	ok, name, operator, value = SplitRuleExpr("@foo")
	assert.True(t, ok)
	assert.Equal(t, "foo", name)
	assert.Equal(t, "", operator)
	assert.Equal(t, "", value)

	ok, name, operator, value = SplitRuleExpr("@bar=")
	assert.True(t, ok)
	assert.Equal(t, "bar", name)
	assert.Equal(t, "=", operator)
	assert.Equal(t, "", value)

	ok, name, operator, value = SplitRuleExpr("@baz~pif")
	assert.True(t, ok)
	assert.Equal(t, "baz", name)
	assert.Equal(t, "~", operator)
	assert.Equal(t, "pif", value)
}

func TestApplyConfig(t *testing.T) {
	var ok bool
	var cfg Context
	var err error

	ok, err = ApplyConfig(&cfg, "foo")
	assert.NoError(t, err)
	assert.False(t, ok)

	ok, err = ApplyConfig(&cfg, "@foo")
	assert.NoError(t, err)
	assert.False(t, ok)

	ok, err = ApplyConfig(&cfg, "@fork")
	assert.Error(t, err)
	assert.True(t, ok)

	ok, err = ApplyConfig(&cfg, "@fork=")
	assert.Error(t, err)
	assert.True(t, ok)

	ok, err = ApplyConfig(&cfg, "@fork~")
	assert.Error(t, err)
	assert.True(t, ok)

	expectedFork := 3
	ok, err = ApplyConfig(&cfg, fmt.Sprintf("@fork=%d", expectedFork))
	assert.NoError(t, err)
	assert.True(t, ok)
	assert.Equal(t, expectedFork, cfg.ForkCount)

	expectedFork = 7
	ok, err = ApplyConfig(&cfg, fmt.Sprintf("@fork=%d", expectedFork))
	assert.Error(t, err)
	assert.True(t, ok)

}

func TestBuildAssertion(t *testing.T) {
	var ok bool
	//var asrtr Asserter
	var err error

	ok, _, err = BuildAssertion("foo")
	assert.NoError(t, err)
	assert.False(t, ok)

	ok, _, err = BuildAssertion("@foo")
	assert.NoError(t, err)
	assert.False(t, ok)

	ok, _, err = BuildAssertion("@fail")
	assert.NoError(t, err)
	assert.True(t, ok)

	ok, _, err = BuildAssertion("@stdout=")
	assert.NoError(t, err)
	assert.True(t, ok)

	ok, _, err = BuildAssertion("@stdout~")
	assert.NoError(t, err)
	assert.True(t, ok)

	ok, _, err = BuildAssertion("@stdout+")
	assert.Error(t, err)
	assert.True(t, ok)
}

func TestParseArgs(t *testing.T) {
	var cfg Context
	var cmdAndArgs []string
	var assertions []Asserter
	var err error

	// Parse command and args without config nor assertions
	cfg, cmdAndArgs, assertions, err = ParseArgs([]string{"foo", "bar", "baz"})
	require.NoError(t, err)
	assert.Equal(t, []string{"foo", "bar", "baz"}, cmdAndArgs)
	assert.Equal(t, Context{}, cfg)
	assert.Len(t, assertions, 0)

	// Parse command and args with a not existing rule
	_, _, _, err = ParseArgs([]string{"foo", "bar", "@foo"})
	assert.Error(t, err)

	// Parse command and args with an existing rule
	cfg, cmdAndArgs, assertions, err = ParseArgs([]string{"foo", "bar", "@fail"})
	require.NoError(t, err)
	assert.Equal(t, []string{"foo", "bar"}, cmdAndArgs)
	assert.Equal(t, Context{}, cfg)
	assert.Len(t, assertions, 1)

}
