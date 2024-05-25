package dao

import (
	"database/sql"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"mby.fr/cmdtest/model"
	"mby.fr/utils/filez"
)

func initSuiteDao(t *testing.T, dirpath string) Suite {
	db, err := DbOpen(dirpath)
	require.NoError(t, err)

	dao, err := NewSuite(db)
	require.NoError(t, err)
	return dao
}

func addSuite(t *testing.T, dao Suite, name string) {
	err := dao.SaveSuiteConfig(name, model.Config{})
	require.NoError(t, err)
}

func addSuiteWithOutcome(t *testing.T, dao Suite, name string, outcome model.Outcome, startTime time.Time) {
	err := dao.SaveSuiteConfig(name, model.Config{})
	require.NoError(t, err)

	err = dao.UpdateSuiteOutcome(name, outcome)
	require.NoError(t, err)

	err = dao.UpdateSuiteStartTime(name, startTime)
	require.NoError(t, err)
}

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
	dao := initSuiteDao(t, dirpath)

	cfg, err := dao.FindGlobalConfig()
	require.NoError(t, err)
	assert.Nil(t, cfg)
}

func TestFindGlobalConfig_SaveAndGet(t *testing.T) {
	dirpath := filez.MkTempDir("", "")
	defer os.RemoveAll(dirpath)
	dao := initSuiteDao(t, dirpath)

	expected := model.NewGlobalDefaultConfig()
	require.True(t, expected.Async.IsPresent(), "expected async empty")
	assert.Equal(t, model.DefaultAsync, expected.Async.Get(), "bad expected async")
	err := dao.SaveGlobalConfig(expected)
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
	dao := initSuiteDao(t, dirpath)

	expectedSuite := "foo"
	cfg, err := dao.FindSuiteConfig(expectedSuite)
	require.NoError(t, err)
	assert.Nil(t, cfg)
}

func TestFindSuiteConfig_SaveAndGet(t *testing.T) {
	dirpath := filez.MkTempDir("", "")
	defer os.RemoveAll(dirpath)
	dao := initSuiteDao(t, dirpath)

	expectedSuite := "bar"
	expected := model.NewSuiteDefaultConfig()
	require.True(t, expected.Async.IsPresent(), "expected async empty")
	assert.Equal(t, model.DefaultAsync, expected.Async.Get(), "bad expected async")
	err := dao.SaveSuiteConfig(expectedSuite, expected)
	require.NoError(t, err)

	cfg, err := dao.FindSuiteConfig(expectedSuite)
	require.NoError(t, err)
	assert.NotNil(t, cfg)
	require.True(t, cfg.Async.IsPresent(), "found async empty")
	assert.Equal(t, model.DefaultAsync, cfg.Async.Get(), "bad found async")

	compareConfig(t, expected, *cfg)
}

func TestNextSeq(t *testing.T) {
	dirpath := filez.MkTempDir("", "")
	defer os.RemoveAll(dirpath)
	dao := initSuiteDao(t, dirpath)

	addSuite(t, dao, "foo")
	addSuite(t, dao, "bar")
	addSuite(t, dao, "baz")

	var s uint16
	var err error

	s, err = dao.NextSeq("foo")
	require.NoError(t, err)
	assert.Equal(t, uint32(0), s)

	s, err = dao.NextSeq("foo")
	require.NoError(t, err)
	assert.Equal(t, uint32(1), s)

	s, err = dao.NextSeq("foo")
	require.NoError(t, err)
	assert.Equal(t, uint32(2), s)

	s, err = dao.NextSeq("bar")
	require.NoError(t, err)
	assert.Equal(t, uint32(0), s)

	s, err = dao.NextSeq("foo")
	require.NoError(t, err)
	assert.Equal(t, uint32(3), s)

	s, err = dao.NextSeq("bar")
	require.NoError(t, err)
	assert.Equal(t, uint32(1), s)

	s, err = dao.NextSeq("baz")
	require.NoError(t, err)
	assert.Equal(t, uint32(0), s)
}

func TestUpdateEndTime(t *testing.T) {
	dirpath := filez.MkTempDir("", "")
	defer os.RemoveAll(dirpath)
	dao := initSuiteDao(t, dirpath)

	verifyQuery := "SELECT endTime from suite where name = ? AND endTime > 0"
	expectedSuite := "foo"
	addSuite(t, dao, expectedSuite)

	row := dao.db.QueryRow(verifyQuery, expectedSuite)
	var endTime int64
	var err error
	err = row.Scan(&endTime)
	require.Error(t, err)
	assert.ErrorIs(t, err, sql.ErrNoRows)

	expectedTime := time.Now()
	err = dao.UpdateSuiteEndTime(expectedSuite, expectedTime)
	require.NoError(t, err)

	row = dao.db.QueryRow(verifyQuery, expectedSuite)
	err = row.Scan(&endTime)
	require.NoError(t, err)
	assert.Equal(t, expectedTime.UnixMicro(), endTime)
}

func TestUpdateOutcome(t *testing.T) {
	dirpath := filez.MkTempDir("", "")
	defer os.RemoveAll(dirpath)
	dao := initSuiteDao(t, dirpath)

	verifyQuery := "SELECT outcome from suite where name = ? AND outcome <> ''"
	expectedSuite := "foo"
	addSuite(t, dao, expectedSuite)

	row := dao.db.QueryRow(verifyQuery, expectedSuite)
	var outcome string
	var err error
	err = row.Scan(&outcome)
	require.Error(t, err)
	assert.ErrorIs(t, err, sql.ErrNoRows)

	expectedOutcome := model.FAILED
	err = dao.UpdateSuiteOutcome(expectedSuite, expectedOutcome)
	require.NoError(t, err)

	row = dao.db.QueryRow(verifyQuery, expectedSuite)
	err = row.Scan(&outcome)
	require.NoError(t, err)
	assert.Equal(t, string(expectedOutcome), outcome)
}

func TestDelete(t *testing.T) {
	dirpath := filez.MkTempDir("", "")
	defer os.RemoveAll(dirpath)
	dao := initSuiteDao(t, dirpath)

	addSuite(t, dao, "foo")
	addSuite(t, dao, "bar")
	addSuite(t, dao, "baz")

	verifyQuery := "SELECT count(*) from suite"
	row := dao.db.QueryRow(verifyQuery)
	var count int
	err := row.Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 3, count)

	err = dao.DeleteSuite("bar")
	require.NoError(t, err)

	row = dao.db.QueryRow(verifyQuery)
	err = row.Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 2, count)
}

func TestListPassedFailedErrored(t *testing.T) {
	dirpath := filez.MkTempDir("", "")
	defer os.RemoveAll(dirpath)
	dao := initSuiteDao(t, dirpath)

	addSuiteWithOutcome(t, dao, "p1", "PASSED", time.Now())
	addSuiteWithOutcome(t, dao, "f1", "FAILED", time.Now())
	addSuiteWithOutcome(t, dao, "e1", "ERRORED", time.Now())
	addSuiteWithOutcome(t, dao, "t1", "TIMEOUTED", time.Now())
	addSuiteWithOutcome(t, dao, "e2", "ERRORED", time.Now())
	addSuiteWithOutcome(t, dao, "f2", "FAILED", time.Now())
	addSuiteWithOutcome(t, dao, "p2", "PASSED", time.Now())
	addSuiteWithOutcome(t, dao, "t2", "TIMEOUTED", time.Now())

	suites, err := dao.ListPassedFailedErrored()
	require.NoError(t, err)
	assert.Equal(t, []string{"p1", "p2", "f1", "f2", "e1", "e2"}, suites)
}
