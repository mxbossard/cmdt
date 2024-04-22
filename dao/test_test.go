package dao

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	"mby.fr/cmdtest/model"
	"mby.fr/utils/filez"
)

func initTestDao(t *testing.T, dirpath string) Test {
	db, err := DbOpen(dirpath)
	require.NoError(t, err)

	dao, err := NewTest(db)
	require.NoError(t, err)
	return dao
}

func TestGetSuiteOutcome(t *testing.T) {
	dirpath := filez.MkTempDir("", "")
	defer os.RemoveAll(dirpath)
	dao := initTestDao(t, dirpath)
	_ = dao
	// TODO

	expectedSuite := "foo"
	outcome, err := dao.GetSuiteOutcome(expectedSuite)
	require.NoError(t, err)
	_ = outcome
}

func TestSaveTestOutcome_ThenGet(t *testing.T) {
	dirpath := filez.MkTempDir("", "")
	defer os.RemoveAll(dirpath)
	dao := initTestDao(t, dirpath)
	_ = dao
	// TODO

	expectedSuite := "foo"
	expectedTitle := "bar"
	expectedTestOutcome := model.TestOutcome{}
	err := dao.SaveTestOutcome(expectedSuite, expectedTitle, expectedTestOutcome)
	require.NoError(t, err)
}

func TestClearSuite(t *testing.T) {
	dirpath := filez.MkTempDir("", "")
	defer os.RemoveAll(dirpath)
	dao := initTestDao(t, dirpath)
	_ = dao
	// TODO

	expectedSuite := "foo"
	err := dao.ClearSuite(expectedSuite)
	require.NoError(t, err)
}
