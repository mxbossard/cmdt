package parser

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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

	args = []string{"@debug"}
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

	var args []string
	expectedPrefix := "@"

	// ## ----- config only should return an error
	args = []string{"@suiteTimeout=3s"}
	matches, cmdAndArgs, agg := repo.parseArgs(expectedPrefix, args)
	require.NotNil(t, agg)
	assert.Error(t, agg.Return())
	assert.ErrorContains(t, agg.Return(), "missing action")
	assert.Len(t, matches, 1)
	assert.Len(t, cmdAndArgs, 0)

	// ## ----- bad action's config should return an error
	args = []string{"@test", "@suiteTimeout=3s"}
	matches, cmdAndArgs, agg = repo.parseArgs(expectedPrefix, args)
	require.NotNil(t, agg)
	assert.Error(t, agg.Return())
	assert.ErrorContains(t, agg.Return(), "bad config rule")
	assert.Len(t, matches, 2)
	assert.Len(t, cmdAndArgs, 0)

	args = []string{"@suite", "@fail"}
	matches, cmdAndArgs, agg = repo.parseArgs(expectedPrefix, args)
	require.NotNil(t, agg)
	assert.Error(t, agg.Return())
	assert.ErrorContains(t, agg.Return(), "assertion rule can only be used in @test context")
	assert.Len(t, matches, 2)
	assert.Len(t, cmdAndArgs, 0)
}

func TestParseArgs_Validation(t *testing.T) {
	repo := ruleRepo{}
	repo.addRuleSet(rsActions)
	repo.addRuleSet(rsVerbosity)

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
	args = []string{"@test", "@timeout=3"}
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
	assert.ErrorContains(t, agg.Return(), "invalid operator for rule")
	assert.Len(t, matches, 1)
	assert.Len(t, cmdAndArgs, 0)

	args = []string{"@test", "@timeout:3"}
	matches, cmdAndArgs, agg = repo.parseArgs(expectedPrefix, args)
	require.NotNil(t, agg)
	assert.Error(t, agg.Return())
	assert.ErrorContains(t, agg.Return(), "invalid operator for rule")
	assert.ErrorContains(t, agg.Return(), "invalid value for rule")
	assert.Len(t, matches, 1)
	assert.Len(t, cmdAndArgs, 0)
}
