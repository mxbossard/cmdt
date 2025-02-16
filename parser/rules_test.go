package parser

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"mby.fr/utils/errorz"
)

func TestMatch(t *testing.T) {
	repo := ruleRepo{}
	repo.addRuleSet(rsActions)

	var args []string
	expectedPrefix := "@"
	emptyArgs := []string{}
	var n int
	var configurer configurer
	var agg errorz.Aggregated

	// No arg should match nothing
	n, configurer, agg = test.Match(expectedPrefix, emptyArgs)
	require.NotNil(t, agg)
	assert.NoError(t, agg.Return())
	assert.Nil(t, configurer)
	assert.Equal(t, 0, n)

	// Right arg should match
	args = []string{"@test"}
	n, configurer, agg = test.Match(expectedPrefix, args)
	require.NotNil(t, agg)
	assert.NoError(t, agg.Return())
	assert.NotNil(t, configurer)
	assert.Equal(t, 1, n)

	args = []string{"@test", "foo", "bar", "baz"}
	n, configurer, agg = test.Match(expectedPrefix, args)
	require.NotNil(t, agg)
	assert.NoError(t, agg.Return())
	assert.NotNil(t, configurer)
	assert.Equal(t, 1, n)

	// Unknown args should not match
	args = []string{"test"}
	n, configurer, agg = test.Match(expectedPrefix, args)
	require.NotNil(t, agg)
	assert.NoError(t, agg.Return())
	assert.Nil(t, configurer)
	assert.Equal(t, 0, n)

	args = []string{"@test "}
	n, configurer, agg = test.Match(expectedPrefix, args)
	require.NotNil(t, agg)
	assert.NoError(t, agg.Return())
	assert.Nil(t, configurer)
	assert.Equal(t, 0, n)

	args = []string{"@testa"}
	n, configurer, agg = test.Match(expectedPrefix, args)
	require.NotNil(t, agg)
	assert.NoError(t, agg.Return())
	assert.Nil(t, configurer)
	assert.Equal(t, 0, n)

	args = []string{"@test foo bar baz"}
	n, configurer, agg = test.Match(expectedPrefix, args)
	require.NotNil(t, agg)
	assert.NoError(t, agg.Return())
	assert.Nil(t, configurer)
	assert.Equal(t, 0, n)

	// Bad args should raise an error
	args = []string{"@test:"}
	n, configurer, agg = test.Match(expectedPrefix, args)
	require.NotNil(t, agg)
	assert.Error(t, agg.Return())
	assert.ErrorContains(t, agg.Return(), "unknown operator")
	assert.Nil(t, configurer)
	assert.Equal(t, 0, n)

}
