package repo

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"mby.fr/cmdtest/model"
)

func checkConfig(t *testing.T, eIsol, eToken, eSuite string, before time.Time, actual model.Config) {
	assert.Equal(t, eIsol, actual.Isol.GetOr(""), "bad Isol")
	assert.Equal(t, eToken, actual.Token.GetOr(""), "bad Token")
	assert.Equal(t, eSuite, actual.TestSuite.GetOr(""), "bad TestSuite")
	assert.Greater(t, actual.GlobalStartTime.Get(), before, "StartTime to low")
	assert.Less(t, actual.GlobalStartTime.Get(), time.Now(), "StartTime to high")
}

func compareConfig(t *testing.T, expected, actual model.Config) {
	assert.Equal(t, expected.Async.Get(), actual.Async.Get(), "bad Async 0")

	assert.True(t, expected.Action.Equal(actual.Action), "bad Action")
	assert.True(t, expected.Async.Equal(actual.Async), "bad Async")
	assert.True(t, expected.Wait.Equal(actual.Wait), "bad Wait")
	assert.True(t, expected.Debug.Equal(actual.Debug), "bad Debug")
	assert.True(t, expected.Verbose.Equal(actual.Verbose), "bad Verbose")
	assert.True(t, expected.Keep.Equal(actual.Keep), "bad Keep")
	assert.True(t, expected.Prefix.Equal(actual.Prefix), "bad Prefix")
}

func TestGetGlobalConfig_Default(t *testing.T) {
	expectedToken := "foo"
	expectedIsol := "bar"
	path, _ := forgeWorkDirectoryPath(expectedToken, expectedIsol)
	os.RemoveAll(path)
	repo := New(expectedToken, expectedIsol)
	defer os.RemoveAll(repo.BackingFilepath())

	before := time.Now()
	cfg, err := repo.GetGlobalConfig()

	require.NoError(t, err)
	assert.Equal(t, model.DefaultAsync, cfg.Async.Get())
	assert.Equal(t, model.DefaultWait, cfg.Wait.Get())
	checkConfig(t, "", expectedToken, "", before, cfg)
	compareConfig(t, model.NewGlobalDefaultConfig(), cfg)
}

func TestGetSuiteConfig_Initless_Default(t *testing.T) {
	expectedToken := "foo"
	expectedIsol := "bar"
	path, _ := forgeWorkDirectoryPath(expectedToken, expectedIsol)
	os.RemoveAll(path)
	repo := New(expectedToken, expectedIsol)
	defer os.RemoveAll(repo.BackingFilepath())

	expectedSuite := "bar"
	before := time.Now()
	cfg, err := repo.GetSuiteConfig(expectedSuite, true)

	require.NoError(t, err)
	assert.Equal(t, model.DefaultInitlessAsync, cfg.Async.Get())
	assert.Equal(t, model.DefaultInitlessWait, cfg.Wait.Get())
	checkConfig(t, "", expectedToken, expectedSuite, before, cfg)
	expectedCfg := model.NewGlobalDefaultConfig()
	expectedCfg.Merge(model.NewInitlessSuiteDefaultConfig())
	compareConfig(t, expectedCfg, cfg)
}

func TestGetSuiteConfig_Inited_Default(t *testing.T) {
	expectedToken := "foo"
	expectedIsol := "bar"
	path, _ := forgeWorkDirectoryPath(expectedToken, expectedIsol)
	os.RemoveAll(path)
	repo := New(expectedToken, expectedIsol)
	defer os.RemoveAll(repo.BackingFilepath())

	expectedSuite := "bar"
	before := time.Now()
	cfg, err := repo.GetSuiteConfig(expectedSuite, false)

	require.NoError(t, err)
	assert.Equal(t, model.DefaultInitedAsync, cfg.Async.Get())
	assert.Equal(t, model.DefaultInitedWait, cfg.Wait.Get())
	checkConfig(t, "", expectedToken, expectedSuite, before, cfg)
	expectedCfg := model.NewGlobalDefaultConfig()
	expectedCfg.Merge(model.NewSuiteDefaultConfig())
	compareConfig(t, expectedCfg, cfg)
}

func TestSaveAndGetGlobalConfig(t *testing.T) {
	expectedToken := "foo"
	expectedIsol := "bar"
	path, _ := forgeWorkDirectoryPath(expectedToken, expectedIsol)
	os.RemoveAll(path)
	repo := New(expectedToken, expectedIsol)
	defer os.RemoveAll(repo.BackingFilepath())

	before := time.Now()
	cfg := model.NewGlobalDefaultConfig()
	cfg.Isol.Set(expectedIsol)
	cfg.Token.Set(expectedToken)
	err := repo.SaveGlobalConfig(cfg)

	require.NoError(t, err)

	loaded, err := repo.GetGlobalConfig()
	require.NoError(t, err)
	checkConfig(t, expectedIsol, expectedToken, "", before, loaded)
	compareConfig(t, cfg, loaded)
}

func TestSaveAndGetSuiteConfig(t *testing.T) {
	expectedToken := "foo"
	expectedIsol := "bar"
	path, _ := forgeWorkDirectoryPath(expectedToken, expectedIsol)
	os.RemoveAll(path)
	repo := New(expectedToken, expectedIsol)
	defer os.RemoveAll(repo.BackingFilepath())

	expectedSuite := "bar"
	before := time.Now()
	cfg := model.NewGlobalDefaultConfig()
	cfg.Merge(model.NewSuiteDefaultConfig())
	cfg.Isol.Set(expectedIsol)
	cfg.Token.Set(expectedToken)
	cfg.TestSuite.Set(expectedSuite)
	err := repo.SaveSuiteConfig(cfg)

	require.NoError(t, err)

	loaded, err := repo.GetSuiteConfig(expectedSuite, false)
	require.NoError(t, err)
	checkConfig(t, expectedIsol, expectedToken, expectedSuite, before, loaded)
	compareConfig(t, cfg, loaded)
}
