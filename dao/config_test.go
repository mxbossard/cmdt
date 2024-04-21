package dao

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"mby.fr/cmdtest/model"
	"mby.fr/utils/filez"
)

func compareConfig(t *testing.T, expected, actual model.Config) {
	assert.True(t, expected.Action.Equal(actual.Action), "bad Action")
	assert.True(t, expected.Async.Equal(actual.Async), "bad Async")
	assert.True(t, expected.Wait.Equal(actual.Wait), "bad Wait")
	assert.True(t, expected.Debug.Equal(actual.Debug), "bad Debug")
	assert.True(t, expected.Verbose.Equal(actual.Verbose), "bad Verbose")
	assert.True(t, expected.Keep.Equal(actual.Keep), "bad Keep")
	assert.True(t, expected.Prefix.Equal(actual.Prefix), "bad Prefix")

	require.True(t, actual.Async.IsPresent(), "Async empty")
	assert.Equal(t, expected.Async.Get(), actual.Async.Get(), "bad Async 2")
	require.True(t, actual.Wait.IsPresent(), "Wait empty")
	assert.Equal(t, expected.Wait.Get(), actual.Wait.Get(), "bad Wait 2")
}

func TestSerializeConfig_Then_DeserializeConfig(t *testing.T) {
	expected := model.NewGlobalDefaultConfig()
	require.True(t, expected.Async.IsPresent(), "expected async empty")
	assert.Equal(t, model.DefaultAsync, expected.Async.Get(), "bad expected async")

	ser, err := serializeConfig(expected)
	require.NoError(t, err)
	assert.NotEmpty(t, ser)

	var des model.Config
	err = deserializeConfig(ser, &des)
	require.NoError(t, err)
	require.True(t, des.Async.IsPresent(), "expected async empty")
	assert.Equal(t, model.DefaultAsync, des.Async.Get(), "bad expected async")

	compareConfig(t, expected, des)
}

func TestFindGlobalConfig_Empty(t *testing.T) {
	dirpath := filez.MkTempDir("", "")
	defer os.RemoveAll(dirpath)

	db, err := DbOpen(dirpath)
	require.NoError(t, err)

	dao, err := NewSuite(db)
	require.NoError(t, err)

	cfg, err := dao.FindGlobalConfig()
	require.NoError(t, err)
	assert.Nil(t, cfg)
}

func TestFindGlobalConfig_SaveAndGet(t *testing.T) {
	dirpath := filez.MkTempDir("", "")
	defer os.RemoveAll(dirpath)

	db, err := DbOpen(dirpath)
	require.NoError(t, err)

	dao, err := NewSuite(db)
	require.NoError(t, err)

	expected := model.NewGlobalDefaultConfig()
	require.True(t, expected.Async.IsPresent(), "expected async empty")
	assert.Equal(t, model.DefaultAsync, expected.Async.Get(), "bad expected async")
	err = dao.SaveGlobalConfig(expected)
	require.NoError(t, err)

	cfg, err := dao.FindGlobalConfig()
	require.NoError(t, err)
	assert.NotNil(t, cfg)
	require.True(t, cfg.Async.IsPresent(), "found async empty")
	assert.Equal(t, model.DefaultAsync, cfg.Async.Get(), "bad found async")

	compareConfig(t, expected, *cfg)
}

func TestFindSuiteConfig_Empty(t *testing.T) {
	dirpath := filez.MkTempDir("", "")
	defer os.RemoveAll(dirpath)

	db, err := DbOpen(dirpath)
	require.NoError(t, err)

	dao, err := NewSuite(db)
	require.NoError(t, err)

	expectedSuite := "foo"
	cfg, err := dao.FindSuiteConfig(expectedSuite)
	require.NoError(t, err)
	assert.Nil(t, cfg)
}

func TestFindSuiteConfig_SaveAndGet(t *testing.T) {
	dirpath := filez.MkTempDir("", "")
	defer os.RemoveAll(dirpath)

	db, err := DbOpen(dirpath)
	require.NoError(t, err)

	dao, err := NewSuite(db)
	require.NoError(t, err)

	expectedSuite := "bar"
	expected := model.NewSuiteDefaultConfig()
	require.True(t, expected.Async.IsPresent(), "expected async empty")
	assert.Equal(t, model.DefaultAsync, expected.Async.Get(), "bad expected async")
	err = dao.SaveSuiteConfig(expectedSuite, expected)
	require.NoError(t, err)

	cfg, err := dao.FindSuiteConfig(expectedSuite)
	require.NoError(t, err)
	assert.NotNil(t, cfg)
	require.True(t, cfg.Async.IsPresent(), "found async empty")
	assert.Equal(t, model.DefaultAsync, cfg.Async.Get(), "bad found async")

	compareConfig(t, expected, *cfg)
}
