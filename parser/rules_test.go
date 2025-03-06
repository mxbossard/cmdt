package parser

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"mby.fr/utils/errorz"
)

func TestRuleCheck(t *testing.T) {
	var agg errorz.Aggregated

	matchOutContains := basicRuleMatch[string]{prefix: "@", name: "stdout", op: ":", value: "bar", mappedValue: "bar"}
	matchOutEq := basicRuleMatch[string]{prefix: "@", name: "stdout", op: "=", value: "bar", mappedValue: "bar"}
	matchOutEqAlias := basicRuleMatch[string]{prefix: "@", name: "out", op: "=", value: "bar", mappedValue: "bar"}
	matchErrEq := basicRuleMatch[string]{prefix: "@", name: "stderr", op: "=", value: "bar", mappedValue: "bar"}
	matchInitNoOp := basicRuleMatch[string]{prefix: "@", name: "init", op: "", value: "", mappedValue: ""}
	matchSuiteNoOp := basicRuleMatch[string]{prefix: "@", name: "suite", op: "", value: "", mappedValue: ""}

	// test rule should be valid against out or err match
	agg = test.Check(matchOutContains, matchOutEq, matchErrEq)
	assert.NoError(t, agg.Return())

	agg = stderr.Check(matchOutContains, matchOutEq)
	assert.NoError(t, agg.Return())

	// err rule should be valid against one errEq match
	agg = stderr.Check(matchErrEq)
	assert.NoError(t, agg.Return())

	// out rule should be valid against one outEq match
	agg = stdout.Check(matchOutEq)
	assert.NoError(t, agg.Return())

	// out rule should be valid against one outContains match
	agg = stdout.Check(matchOutContains)
	assert.NoError(t, agg.Return())

	// out rule should be valid against multiple outContains matches
	agg = stdout.Check(matchOutContains, matchOutContains)
	assert.NoError(t, agg.Return())

	// out rule should not be valid against multiple outEq matches
	agg = stdout.Check(matchOutEq, matchOutEq)
	assert.Error(t, agg.Return())
	assert.ErrorContains(t, agg.Return(), "is uniq")

	// alias should be considered same rule
	agg = stdout.Check(matchOutEq, matchOutEqAlias)
	assert.Error(t, agg.Return())
	assert.ErrorContains(t, agg.Return(), "is uniq")

	// out rule should not be valid against outEq match and another op match
	agg = stdout.Check(matchOutEq, matchOutContains)
	assert.Error(t, agg.Return())
	assert.ErrorContains(t, agg.Return(), "is exclusive")

	agg = suite.Check(matchSuiteNoOp)
	assert.NoError(t, agg.Return())

	agg = suite.Check(matchInitNoOp)
	assert.NoError(t, agg.Return())

	agg = suite.Check(matchSuiteNoOp, matchSuiteNoOp)
	assert.Error(t, agg.Return())
	assert.ErrorContains(t, agg.Return(), "is uniq")

	// alias should be considered same rule
	agg = suite.Check(matchSuiteNoOp, matchInitNoOp)
	assert.Error(t, agg.Return())
	assert.ErrorContains(t, agg.Return(), "is uniq")
}

func TestRuleMatch(t *testing.T) {
	var args []string
	expectedPrefix := "@"
	emptyArgs := []string{}
	var n int
	var match ruleMatch
	var agg errorz.Aggregated

	// No arg should match nothing
	n, match, agg = test.Match(expectedPrefix, emptyArgs)
	require.NotNil(t, agg)
	assert.NoError(t, agg.Return())
	assert.Nil(t, match)
	assert.Equal(t, 0, n)

	// Right arg should match
	args = []string{"@test"}
	n, match, agg = test.Match(expectedPrefix, args)
	require.NotNil(t, agg)
	assert.NoError(t, agg.Return())
	assert.NotNil(t, match)
	assert.Equal(t, 1, n)

	args = []string{"test"}
	n, match, agg = test.Match("", args)
	require.NotNil(t, agg)
	assert.NoError(t, agg.Return())
	assert.NotNil(t, match)
	assert.Equal(t, 1, n)

	args = []string{"-test"}
	n, match, agg = test.Match("-", args)
	require.NotNil(t, agg)
	assert.NoError(t, agg.Return())
	assert.NotNil(t, match)
	assert.Equal(t, 1, n)

	args = []string{"--test"}
	n, match, agg = test.Match("--", args)
	require.NotNil(t, agg)
	assert.NoError(t, agg.Return())
	assert.NotNil(t, match)
	assert.Equal(t, 1, n)

	args = []string{"-t"}
	n, match, agg = test.Match("-", args)
	require.NotNil(t, agg)
	assert.NoError(t, agg.Return())
	assert.NotNil(t, match)
	assert.Equal(t, 1, n)

	args = []string{"--t"}
	n, match, agg = test.Match("--", args)
	require.NotNil(t, agg)
	assert.NoError(t, agg.Return())
	assert.NotNil(t, match)
	assert.Equal(t, 1, n)

	args = []string{"@test", "foo", "bar", "baz"}
	n, match, agg = test.Match(expectedPrefix, args)
	require.NotNil(t, agg)
	assert.NoError(t, agg.Return())
	assert.NotNil(t, match)
	assert.Equal(t, 1, n)

	// Unknown args should not match
	args = []string{"test"}
	n, match, agg = test.Match(expectedPrefix, args)
	require.NotNil(t, agg)
	assert.NoError(t, agg.Return())
	assert.Nil(t, match)
	assert.Equal(t, 0, n)

	args = []string{"@testa"}
	n, match, agg = test.Match(expectedPrefix, args)
	require.NotNil(t, agg)
	assert.NoError(t, agg.Return())
	assert.Nil(t, match)
	assert.Equal(t, 0, n)

	// Bad args should raise an error
	args = []string{"@test:"}
	n, match, agg = test.Match(expectedPrefix, args)
	require.NotNil(t, agg)
	assert.Error(t, agg.Return())
	assert.ErrorContains(t, agg.Return(), "invalid operator")
	assert.Nil(t, match)
	assert.Equal(t, 0, n)

	args = []string{"@test "}
	n, match, agg = test.Match(expectedPrefix, args)
	require.NotNil(t, agg)
	assert.Error(t, agg.Return())
	assert.ErrorContains(t, agg.Return(), "invalid operator")
	assert.Nil(t, match)
	assert.Equal(t, 0, n)

	args = []string{"@test foo bar baz"}
	n, match, agg = test.Match(expectedPrefix, args)
	require.NotNil(t, agg)
	assert.Error(t, agg.Return())
	assert.ErrorContains(t, agg.Return(), "invalid operator")
	assert.Nil(t, match)
	assert.Equal(t, 0, n)

	// Alias should match
	args = []string{"@suite"}
	n, match, agg = suite.Match(expectedPrefix, args)
	require.NotNil(t, agg)
	assert.NoError(t, agg.Return())
	assert.NotNil(t, match)
	assert.Equal(t, 1, n)

	args = []string{"@init"}
	n, match, agg = suite.Match(expectedPrefix, args)
	require.NotNil(t, agg)
	assert.NoError(t, agg.Return())
	assert.NotNil(t, match)
	assert.Equal(t, 1, n)

}

func TestRuleSetCheck(t *testing.T) {
	var agg errorz.Aggregated

	matchOutContains := basicRuleMatch[string]{prefix: "@", name: "stdout", op: ":", value: "bar", mappedValue: "bar"}
	matchOutEq := basicRuleMatch[string]{prefix: "@", name: "stdout", op: "=", value: "bar", mappedValue: "bar"}
	matchOutEqAlias := basicRuleMatch[string]{prefix: "@", name: "out", op: "=", value: "bar", mappedValue: "bar"}
	matchErrEq := basicRuleMatch[string]{prefix: "@", name: "stderr", op: "=", value: "bar", mappedValue: "bar"}
	matchTestNoOp := basicRuleMatch[string]{prefix: "@", name: "test", op: "", value: "", mappedValue: ""}
	matchSuiteNoOp := basicRuleMatch[string]{prefix: "@", name: "suite", op: "", value: "", mappedValue: ""}
	matchInitNoOp := basicRuleMatch[string]{prefix: "@", name: "init", op: "", value: "", mappedValue: ""}

	// test rsActions should be valid against out or err match
	agg = rsStackableAssertions.Check(matchOutContains)
	assert.NoError(t, agg.Return())

	agg = rsStackableAssertions.Check(matchOutEq, matchErrEq)
	assert.NoError(t, agg.Return())

	agg = rsStackableAssertions.Check(matchOutEq, matchOutEq)
	assert.Error(t, agg.Return())
	assert.ErrorContains(t, agg.Return(), "is uniq")

	agg = rsStackableAssertions.Check(matchOutEq, matchOutEqAlias)
	assert.Error(t, agg.Return())
	assert.ErrorContains(t, agg.Return(), "is uniq")

	agg = rsStackableAssertions.Check(matchOutContains, matchOutEq, matchErrEq)
	assert.Error(t, agg.Return())
	assert.ErrorContains(t, agg.Return(), "is exclusive")

	agg = rsActions.Check(matchTestNoOp)
	assert.NoError(t, agg.Return())

	agg = rsActions.Check(matchSuiteNoOp)
	assert.NoError(t, agg.Return())

	agg = rsActions.Check(matchTestNoOp, matchSuiteNoOp)
	assert.Error(t, agg.Return())
	assert.ErrorContains(t, agg.Return(), "mutually exclusive")

	agg = rsActions.Check(matchTestNoOp, matchInitNoOp)
	assert.Error(t, agg.Return())
	assert.ErrorContains(t, agg.Return(), "mutually exclusive")
}

func TestRuleSetMatch(t *testing.T) {
	var args []string
	expectedPrefix := "@"
	emptyArgs := []string{}
	var matches []ruleMatch
	var noMatches []string
	var agg errorz.Aggregated

	// No arg should match nothing
	matches, noMatches, agg = rsActions.Match(expectedPrefix, emptyArgs)
	require.NotNil(t, agg)
	assert.NoError(t, agg.Return())
	assert.Len(t, matches, 0)
	assert.Len(t, noMatches, 0)

	// Alias should match
	args = []string{"@suite"}
	matches, noMatches, agg = rsActions.Match(expectedPrefix, args)
	require.NotNil(t, agg)
	assert.NoError(t, agg.Return())
	assert.Len(t, matches, 1)
	assert.Len(t, noMatches, 0)

	args = []string{"@debug"}
	matches, noMatches, agg = rsActions.Match(expectedPrefix, args)
	require.NotNil(t, agg)
	assert.NoError(t, agg.Return())
	assert.Len(t, matches, 0)
	assert.Len(t, noMatches, 1)

	args = []string{"@suite", "@suite"}
	matches, noMatches, agg = rsActions.Match(expectedPrefix, args)
	require.NotNil(t, agg)
	assert.NoError(t, agg.Return())
	assert.Len(t, matches, 2)
	assert.Len(t, noMatches, 0)

	args = []string{"@suite", "@init"}
	matches, noMatches, agg = rsActions.Match(expectedPrefix, args)
	require.NotNil(t, agg)
	assert.NoError(t, agg.Return())
	assert.Len(t, matches, 2)
	assert.Len(t, noMatches, 0)

	args = []string{"@test", "@suite"}
	matches, noMatches, agg = rsActions.Match(expectedPrefix, args)
	require.NotNil(t, agg)
	assert.NoError(t, agg.Return())
	assert.Len(t, matches, 2)
	assert.Len(t, noMatches, 0)

	args = []string{"@test", "@init"} // should work with aliases too
	matches, noMatches, agg = rsActions.Match(expectedPrefix, args)
	require.NotNil(t, agg)
	assert.NoError(t, agg.Return())
	assert.Len(t, matches, 2)
	assert.Len(t, noMatches, 0)

	args = []string{"@test", "cmd", "arg"}
	matches, noMatches, agg = rsActions.Match(expectedPrefix, args)
	require.NotNil(t, agg)
	assert.NoError(t, agg.Return())
	assert.Len(t, matches, 1)
	assert.Len(t, noMatches, 2)
	assert.Contains(t, noMatches, "cmd")
	assert.Contains(t, noMatches, "arg")

	args = []string{"cmd", "@test", "arg"}
	matches, noMatches, agg = rsActions.Match(expectedPrefix, args)
	require.NotNil(t, agg)
	assert.NoError(t, agg.Return())
	assert.Len(t, matches, 1)
	assert.Len(t, noMatches, 2)
	assert.Contains(t, noMatches, "cmd")
	assert.Contains(t, noMatches, "arg")

	args = []string{"@test", "@cmd"}
	matches, noMatches, agg = rsActions.Match(expectedPrefix, args)
	require.NotNil(t, agg)
	assert.NoError(t, agg.Return())
	assert.Len(t, matches, 1)
	assert.Len(t, noMatches, 1)
	assert.Contains(t, noMatches, "@cmd")

	args = []string{"@test", "@--", "@cmd"}
	matches, noMatches, agg = rsActions.Match(expectedPrefix, args)
	require.NotNil(t, agg)
	assert.NoError(t, agg.Return())
	assert.Len(t, matches, 1)
	assert.Len(t, noMatches, 2)
	assert.Contains(t, noMatches, "@--")
	assert.Contains(t, noMatches, "@cmd")

	args = []string{"@test", "@prefix=%", "%init"}
	matches, noMatches, agg = rsActions.Match(expectedPrefix, args)
	require.NotNil(t, agg)
	assert.NoError(t, agg.Return())
	assert.Len(t, matches, 2)
	assert.Len(t, noMatches, 1)
	assert.Contains(t, noMatches, "@prefix=%")
}
