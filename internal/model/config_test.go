package model

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSplitRuleExpr(t *testing.T) {
	var ok bool
	var rule Rule
	cfg := NewGlobalDefaultConfig()

	ok, _ = cfg.SplitRuleExpr("foo")
	assert.False(t, ok)

	ok, rule = cfg.SplitRuleExpr("@foo")
	assert.True(t, ok)
	assert.Equal(t, "foo", rule.Name)
	assert.Equal(t, "", rule.Op)
	assert.Equal(t, "", rule.Expected)

	ok, rule = cfg.SplitRuleExpr("@bar=")
	assert.True(t, ok)
	assert.Equal(t, "bar", rule.Name)
	assert.Equal(t, "=", rule.Op)
	assert.Equal(t, "", rule.Expected)

	ok, rule = cfg.SplitRuleExpr("@baz~pif")
	assert.True(t, ok)
	assert.Equal(t, "baz", rule.Name)
	assert.Equal(t, "~", rule.Op)
	assert.Equal(t, "pif", rule.Expected)
}
