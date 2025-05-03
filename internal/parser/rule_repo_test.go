package parser

import (
	"testing"

	"github.com/mxbossard/utilz/errorz"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const badContextUseErrorMsg = "can only be used in context"

func TestParseArgs(t *testing.T) {
	repo := ruleRepo{}
	repo.addRuleSet(rsActions)
	repo.addRuleSet(rsVerbosity)

	var args []string
	expectedPrefix := "@"
	emptyArgs := []string{}

	// ## ----- should not error but return no results
	matches, cmdAndArgs, agg := repo.parseArgs(expectedPrefix, emptyArgs)
	require.NotNil(t, agg)
	assert.NoError(t, agg.Return())
	assert.Len(t, matches, 0)
	assert.Len(t, cmdAndArgs, 0)

	// ## ----- should not error but return results
	args = []string{"@test"}
	matches, cmdAndArgs, agg = repo.parseArgs(expectedPrefix, args)
	require.NotNil(t, agg)
	assert.NoError(t, agg.Return())
	assert.Len(t, matches, 1)
	assert.Len(t, cmdAndArgs, 0)

	args = []string{"@suite"}
	matches, cmdAndArgs, agg = repo.parseArgs(expectedPrefix, args)
	require.NotNil(t, agg)
	assert.NoError(t, agg.Return())
	assert.Len(t, matches, 1)
	assert.Len(t, cmdAndArgs, 0)

	args = []string{"@init"} // suite alias
	matches, cmdAndArgs, agg = repo.parseArgs(expectedPrefix, args)
	require.NotNil(t, agg)
	assert.NoError(t, agg.Return())
	assert.Len(t, matches, 1)
	assert.Len(t, cmdAndArgs, 0)

	args = []string{"@test=foo"}
	matches, cmdAndArgs, agg = repo.parseArgs(expectedPrefix, args)
	require.NotNil(t, agg)
	assert.NoError(t, agg.Return())
	assert.Len(t, matches, 1)
	assert.Len(t, cmdAndArgs, 0)

	args = []string{"@test", "@debug"}
	matches, cmdAndArgs, agg = repo.parseArgs(expectedPrefix, args)
	require.NotNil(t, agg)
	assert.NoError(t, agg.Return())
	assert.Len(t, matches, 2)
	assert.Len(t, cmdAndArgs, 0)

	args = []string{"test"}
	matches, cmdAndArgs, agg = repo.parseArgs(expectedPrefix, args)
	require.NotNil(t, agg)
	assert.NoError(t, agg.Return())
	assert.Len(t, matches, 0)
	assert.Len(t, cmdAndArgs, 1)
	assert.Contains(t, cmdAndArgs, "test")

	// Should error because of badly scoped rules
	args = []string{"@debug"}
	matches, cmdAndArgs, agg = repo.parseArgs(expectedPrefix, args)
	require.NotNil(t, agg)
	assert.Error(t, agg.Return())
	assert.ErrorContains(t, agg.Return(), badContextUseErrorMsg)
	assert.Len(t, matches, 1)
	assert.Len(t, cmdAndArgs, 0)

	// after rule stopper @unknown should be a command
	args = []string{"@test", "@--", "unknown", "@--"}
	matches, cmdAndArgs, agg = repo.parseArgs(expectedPrefix, args)
	require.NotNil(t, agg)
	assert.Error(t, agg.Return())
	assert.ErrorContains(t, agg.Return(), "must be uniq")
	assert.Len(t, matches, 1)
	assert.Len(t, cmdAndArgs, 2)
	assert.Contains(t, cmdAndArgs, "unknown")
	assert.Contains(t, cmdAndArgs, "@--")

	args = []string{"@test", "@--", "unknown", "@prefix=%", "%--", "arg"}
	matches, cmdAndArgs, agg = repo.parseArgs(expectedPrefix, args)
	require.NotNil(t, agg)
	assert.Error(t, agg.Return())
	assert.ErrorContains(t, agg.Return(), "must be uniq")
	assert.Len(t, matches, 1)
	assert.Len(t, cmdAndArgs, 4)
	assert.Contains(t, cmdAndArgs, "unknown")
	assert.Contains(t, cmdAndArgs, "@prefix=%")
	assert.Contains(t, cmdAndArgs, "%--")
	assert.Contains(t, cmdAndArgs, "arg")

	args = []string{"@test", "@--", "@unknown", "arg"}
	matches, cmdAndArgs, agg = repo.parseArgs(expectedPrefix, args)
	require.NotNil(t, agg)
	assert.NoError(t, agg.Return())
	assert.Len(t, matches, 1)
	assert.Len(t, cmdAndArgs, 2)
	assert.Contains(t, cmdAndArgs, "@unknown")
	assert.Contains(t, cmdAndArgs, "arg")

	// @test prefix change
	args = []string{"@test", "@prefix=%", "%debug", "@unknown", "arg"}
	matches, cmdAndArgs, agg = repo.parseArgs(expectedPrefix, args)
	require.NotNil(t, agg)
	assert.NoError(t, agg.Return())
	assert.Len(t, matches, 2)
	assert.Len(t, cmdAndArgs, 2)
	assert.Contains(t, cmdAndArgs, "@unknown")
	assert.Contains(t, cmdAndArgs, "arg")

	args = []string{"@test", "@prefix=%", "%--", "%unknown", "arg"}
	matches, cmdAndArgs, agg = repo.parseArgs(expectedPrefix, args)
	require.NotNil(t, agg)
	assert.NoError(t, agg.Return())
	assert.Len(t, matches, 1)
	assert.Len(t, cmdAndArgs, 2)
	assert.Contains(t, cmdAndArgs, "%unknown")
	assert.Contains(t, cmdAndArgs, "arg")

	// ## ----- should error
	// unknown operator
	args = []string{"@test:"}
	matches, cmdAndArgs, agg = repo.parseArgs(expectedPrefix, args)
	require.NotNil(t, agg)
	assert.Error(t, agg.Return())
	assert.ErrorContains(t, agg.Return(), "invalid operator")
	assert.Len(t, matches, 0)
	assert.Len(t, cmdAndArgs, 0)

	// ## ----- unknown rule
	args = []string{"@testa"}
	matches, cmdAndArgs, agg = repo.parseArgs(expectedPrefix, args)
	require.NotNil(t, agg)
	assert.Error(t, agg.Return())
	assert.ErrorContains(t, agg.Return(), "unkown rule")
	assert.Len(t, matches, 0)
	assert.Len(t, cmdAndArgs, 0)

	args = []string{"@test @debug"}
	matches, cmdAndArgs, agg = repo.parseArgs(expectedPrefix, args)
	require.NotNil(t, agg)
	assert.Error(t, agg.Return())
	assert.ErrorContains(t, agg.Return(), "unkown rule")
	assert.Len(t, matches, 0)
	assert.Len(t, cmdAndArgs, 0)

	// @unknown rule is unknown
	args = []string{"@test", "@unknwon", "cmd"}
	matches, cmdAndArgs, agg = repo.parseArgs(expectedPrefix, args)
	require.NotNil(t, agg)
	assert.Error(t, agg.Return())
	assert.ErrorContains(t, agg.Return(), "unkown rule")
	assert.Len(t, matches, 1)
	assert.Len(t, cmdAndArgs, 1)
	assert.Contains(t, cmdAndArgs, "cmd")

	// cmd before after rule stopper should error
	args = []string{"@test", "cmd", "@--", "arg"}
	matches, cmdAndArgs, agg = repo.parseArgs(expectedPrefix, args)
	require.NotNil(t, agg)
	assert.Error(t, agg.Return())
	assert.ErrorContains(t, agg.Return(), "bad placement for command")
	assert.Len(t, matches, 1)
	assert.Len(t, cmdAndArgs, 1)
	//assert.Contains(t, cmdAndArgs, "cmd")
	assert.Contains(t, cmdAndArgs, "arg")

	// ## ----- mutually exclusive rules
	args = []string{"@test", "@suite"}
	matches, cmdAndArgs, agg = repo.parseArgs(expectedPrefix, args)
	require.NotNil(t, agg)
	assert.Error(t, agg.Return())
	assert.ErrorContains(t, agg.Return(), "mutually exclusive")
	assert.Len(t, matches, 2)
	assert.Len(t, cmdAndArgs, 0)

	args = []string{"@test", "@init"}
	matches, cmdAndArgs, agg = repo.parseArgs(expectedPrefix, args)
	require.NotNil(t, agg)
	assert.Error(t, agg.Return())
	assert.ErrorContains(t, agg.Return(), "mutually exclusive")
	assert.Len(t, matches, 2)
	assert.Len(t, cmdAndArgs, 0)

}

func TestParseArgs_Missmatch(t *testing.T) {
	repo := ruleRepo{}
	repo.addRuleSet(rsActions)
	repo.addRuleSet(rsVerbosity)
	repo.addRuleSet(rsSuiteConfig)
	repo.addRuleSet(rsOutcomeAssertions)

	var args []string
	expectedPrefix := "@"

	// ## ----- config only should return an error
	args = []string{"@suiteTimeout=3s"}
	matches, cmdAndArgs, agg := repo.parseArgs(expectedPrefix, args)
	require.NotNil(t, agg)
	assert.Error(t, agg.Return())
	assert.ErrorContains(t, agg.Return(), badContextUseErrorMsg)
	assert.Len(t, matches, 1)
	assert.Len(t, cmdAndArgs, 0)

	// ## ----- bad action's config should return an error
	args = []string{"@test", "@suiteTimeout=3s"}
	matches, cmdAndArgs, agg = repo.parseArgs(expectedPrefix, args)
	require.NotNil(t, agg)
	assert.Error(t, agg.Return())
	assert.ErrorContains(t, agg.Return(), badContextUseErrorMsg)
	assert.Len(t, matches, 2)
	assert.Len(t, cmdAndArgs, 0)

	args = []string{"@suite", "@fail"}
	matches, cmdAndArgs, agg = repo.parseArgs(expectedPrefix, args)
	require.NotNil(t, agg)
	assert.Error(t, agg.Return())
	assert.ErrorContains(t, agg.Return(), badContextUseErrorMsg)
	assert.Len(t, matches, 2)
	assert.Len(t, cmdAndArgs, 0)
}

func TestParseArgs_Validation(t *testing.T) {
	repo := ruleRepo{}
	repo.addRuleSet(rsActions)
	repo.addRuleSet(rsVerbosity)
	repo.addRuleSet(rsTestConfig)

	var args []string
	expectedPrefix := "@"

	// ## ----- should be valid and return no error
	args = []string{"@test", "@timeout=3s"}
	matches, cmdAndArgs, agg := repo.parseArgs(expectedPrefix, args)
	require.NotNil(t, agg)
	assert.NoError(t, agg.Return())
	assert.Len(t, matches, 2)
	assert.Len(t, cmdAndArgs, 0)

	// ## ----- should not be valid and return an error
	args = []string{"@test", "@timeout=3a"}
	matches, cmdAndArgs, agg = repo.parseArgs(expectedPrefix, args)
	require.NotNil(t, agg)
	assert.Error(t, agg.Return())
	assert.ErrorContains(t, agg.Return(), "invalid value for rule")
	assert.Len(t, matches, 1)
	assert.Len(t, cmdAndArgs, 0)

	args = []string{"@test", "@timeout=a"}
	matches, cmdAndArgs, agg = repo.parseArgs(expectedPrefix, args)
	require.NotNil(t, agg)
	assert.Error(t, agg.Return())
	assert.ErrorContains(t, agg.Return(), "invalid value for rule")
	assert.Len(t, matches, 1)
	assert.Len(t, cmdAndArgs, 0)

	args = []string{"@test", "@timeout:3s"}
	matches, cmdAndArgs, agg = repo.parseArgs(expectedPrefix, args)
	require.NotNil(t, agg)
	assert.Error(t, agg.Return())
	assert.ErrorContains(t, agg.Return(), "invalid operator")
	assert.Len(t, matches, 1)
	assert.Len(t, cmdAndArgs, 0)

	args = []string{"@test", "@timeout:3a"}
	matches, cmdAndArgs, agg = repo.parseArgs(expectedPrefix, args)
	require.NotNil(t, agg)
	assert.Error(t, agg.Return())
	assert.ErrorContains(t, agg.Return(), "invalid operator")
	//assert.ErrorContains(t, agg.Return(), "invalid value")
	assert.Len(t, matches, 1)
	assert.Len(t, cmdAndArgs, 0)
}

func TestChilds(t *testing.T) {
	repo := ruleRepo{}
	repo.addRuleSet(rsActions)
	repo.addRuleSet(rsReportConfig)
	repo.addRuleSet(rsVerbosity)

	var args []string
	var ruleDefs []RuleDef
	var warns errorz.Aggregated
	var errors errorz.Aggregated

	// ## ----- calls that should works
	args = []string{}
	ruleDefs, warns, errors = repo.children(args...)
	assert.NoError(t, errors.Return())
	assert.NoError(t, warns.Return())
	assert.NotEmpty(t, ruleDefs)
	assert.Len(t, ruleDefs, len(rsActions.rules))

	args = []string{"@suite"}
	ruleDefs, warns, errors = repo.children(args...)
	assert.NoError(t, errors.Return())
	assert.NoError(t, warns.Return())
	assert.NotEmpty(t, ruleDefs)
	assert.Len(t, ruleDefs, len(rsVerbosity.rules))

	args = []string{"suite"}
	ruleDefs, warns, errors = repo.children(args...)
	assert.NoError(t, errors.Return())
	assert.NoError(t, warns.Return())
	assert.NotEmpty(t, ruleDefs)
	assert.Len(t, ruleDefs, len(rsVerbosity.rules))

	args = []string{"-suite"}
	ruleDefs, warns, errors = repo.children(args...)
	assert.NoError(t, errors.Return())
	assert.NoError(t, warns.Return())
	assert.NotEmpty(t, ruleDefs)
	assert.Len(t, ruleDefs, len(rsVerbosity.rules))

	args = []string{"--suite"}
	ruleDefs, warns, errors = repo.children(args...)
	assert.NoError(t, errors.Return())
	assert.NoError(t, warns.Return())
	assert.NotEmpty(t, ruleDefs)
	assert.Len(t, ruleDefs, len(rsVerbosity.rules))

	args = []string{"s"}
	ruleDefs, warns, errors = repo.children(args...)
	assert.NoError(t, errors.Return())
	assert.NoError(t, warns.Return())
	assert.NotEmpty(t, ruleDefs)
	assert.Len(t, ruleDefs, len(rsVerbosity.rules))

	args = []string{"-s"}
	ruleDefs, warns, errors = repo.children(args...)
	assert.NoError(t, errors.Return())
	assert.NoError(t, warns.Return())
	assert.NotEmpty(t, ruleDefs)
	assert.Len(t, ruleDefs, len(rsVerbosity.rules))

	args = []string{"--s"}
	ruleDefs, warns, errors = repo.children(args...)
	assert.NoError(t, errors.Return())
	assert.NoError(t, warns.Return())
	assert.NotEmpty(t, ruleDefs)
	assert.Len(t, ruleDefs, len(rsVerbosity.rules))

	args = []string{"init"}
	ruleDefs, warns, errors = repo.children(args...)
	assert.NoError(t, errors.Return())
	assert.NoError(t, warns.Return())
	assert.NotEmpty(t, ruleDefs)
	assert.Len(t, ruleDefs, len(rsVerbosity.rules))

	args = []string{"test"}
	ruleDefs, warns, errors = repo.children(args...)
	assert.NoError(t, errors.Return())
	assert.NoError(t, warns.Return())
	assert.NotEmpty(t, ruleDefs)
	assert.Len(t, ruleDefs, len(rsVerbosity.rules))

	args = []string{"suite", "debug"}
	ruleDefs, warns, errors = repo.children(args...)
	assert.NoError(t, errors.Return())
	assert.NoError(t, warns.Return())
	assert.Empty(t, ruleDefs)

	args = []string{"@init", "@verbose"}
	ruleDefs, warns, errors = repo.children(args...)
	assert.NoError(t, errors.Return())
	assert.NoError(t, warns.Return())
	assert.Empty(t, ruleDefs)

	args = []string{"test", "@verbose"}
	ruleDefs, warns, errors = repo.children(args...)
	assert.NoError(t, errors.Return())
	assert.NoError(t, warns.Return())
	assert.Empty(t, ruleDefs)

	args = []string{"test=foo"}
	ruleDefs, warns, errors = repo.children(args...)
	assert.NoError(t, errors.Return())
	assert.NoError(t, warns.Return())
	assert.NotEmpty(t, ruleDefs)
	assert.Len(t, ruleDefs, len(rsVerbosity.rules))

	args = []string{"test=foo", "@verbose"}
	ruleDefs, warns, errors = repo.children(args...)
	assert.NoError(t, errors.Return())
	assert.NoError(t, warns.Return())
	assert.Empty(t, ruleDefs)

	args = []string{"report"}
	ruleDefs, warns, errors = repo.children(args...)
	assert.NoError(t, errors.Return())
	assert.NoError(t, warns.Return())
	assert.Len(t, ruleDefs, len(rsReportConfig.rules)+len(rsVerbosity.rules))

	args = []string{"report", "all"}
	ruleDefs, warns, errors = repo.children(args...)
	assert.NoError(t, errors.Return())
	assert.NoError(t, warns.Return())
	assert.Empty(t, ruleDefs)
	//assert.Len(t, ruleDefs, len(rsVerbosity.rules))

	// ## ----- should be valid to help user but indicate an improper usage
	args = []string{"test:foo"}
	// should return a warning because op : does not exists for test rule
	ruleDefs, warns, errors = repo.children(args...)
	assert.NoError(t, errors.Return())
	assert.Error(t, warns.Return())
	assert.NotEmpty(t, ruleDefs)
	assert.Len(t, ruleDefs, len(rsVerbosity.rules))

	args = []string{"test=foo", "debug=bar"}
	// should return a warning because debug value is invalid
	ruleDefs, warns, errors = repo.children(args...)
	assert.NoError(t, errors.Return())
	assert.Error(t, warns.Return())
	assert.Empty(t, ruleDefs)

	args = []string{"test=foo", "debug:4"}
	// should return a warning because debug op is invalid
	ruleDefs, warns, errors = repo.children(args...)
	assert.NoError(t, errors.Return())
	assert.Error(t, warns.Return())
	assert.Empty(t, ruleDefs)

	// ## ----- calls that should not works
	args = []string{"unknown"}
	// should return an error because unknown rule does not exists
	ruleDefs, warns, errors = repo.children(args...)
	assert.Error(t, errors.Return())
	assert.NoError(t, warns.Return())
	assert.Len(t, ruleDefs, 0)

	args = []string{"suite", "unknown"}
	// should return an error because unknown rule does not exists
	ruleDefs, warns, errors = repo.children(args...)
	assert.Error(t, errors.Return())
	assert.NoError(t, warns.Return())
	assert.Len(t, ruleDefs, 0)

	args = []string{"unknown", "timeout"}
	// should return an error because unknown rule does not exists
	ruleDefs, warns, errors = repo.children(args...)
	assert.Error(t, errors.Return())
	assert.NoError(t, warns.Return())
	assert.Len(t, ruleDefs, 0)

	args = []string{"suite", "ignore"}
	// should return an error because keepReport rule is not a valid child of suite rule
	ruleDefs, warns, errors = repo.children(args...)
	assert.Error(t, errors.Return())
	assert.NoError(t, warns.Return())
	assert.Len(t, ruleDefs, 0)

}
