package parser

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseArgs(t *testing.T) {
	repo := ruleRepo{}
	repo.addRuleSet(rsActions)

	var args []string
	expectedPrefix := "@"
	emptyArgs := []string{}

	configurers, agg := repo.parseArgs(expectedPrefix, emptyArgs)
	require.NotNil(t, agg)
	assert.NoError(t, agg.Return())
	//require.NotNil(t, configurers)
	assert.Len(t, configurers, 0)

	args = []string{"@test"}
	configurers, agg = repo.parseArgs(expectedPrefix, args)
	require.NotNil(t, agg)
	assert.NoError(t, agg.Return())
	//require.NotNil(t, configurers)
	assert.Len(t, configurers, 1)

	args = []string{"test"}
	configurers, agg = repo.parseArgs(expectedPrefix, args)
	require.NotNil(t, agg)
	assert.Error(t, agg.Return())
	assert.ErrorContains(t, agg.Return(), "unable to parse")
	//require.NotNil(t, configurers)
	assert.Len(t, configurers, 0)
}
