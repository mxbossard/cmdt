package dao

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
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

func addTest(t *testing.T, dao Test, suite string, seq uint16, title, stdout, stderr string, exitCode int16, outcome model.Outcome) {
	def := model.TestDefinition{TestSuite: suite, TestName: title, Seq: seq}
	to := model.TestOutcome{TestDefinition: def, CmdTitle: title, ExitCode: exitCode, Stdout: stdout, Stderr: stderr, Outcome: outcome}
	err := dao.SaveTestOutcome(to)
	require.NoError(t, err)
}

func TestSaveTestOutcome(t *testing.T) {
	dirpath := filez.MkTempDir("", "")
	defer os.RemoveAll(dirpath)
	dao := initTestDao(t, dirpath)
	suiteDao := initSuiteDao(t, dirpath)

	expectedSuite1 := "suite1"
	expectedStartTime1 := time.Now().Add(-2 * time.Second)
	//expectedEndTime1 := time.Now()
	//expectedDuration1 := expectedEndTime1.Sub(expectedStartTime1).Truncate(time.Microsecond)
	expectedSuite2 := "suite2"
	expectedStartTime2 := time.Now().Add(-1 * time.Second)
	//expectedEndTime2 := time.Now()
	//expectedDuration2 := expectedEndTime2.Sub(expectedStartTime2).Truncate(time.Microsecond)
	addSuiteWithOutcome(t, suiteDao, expectedSuite1, model.ERRORED, expectedStartTime1)
	addSuiteWithOutcome(t, suiteDao, expectedSuite2, model.IGNORED, expectedStartTime2)
	addTest(t, dao, expectedSuite1, 0, "fooTitle", "fooStdout", "fooStderr", 1, model.ERRORED)
	addTest(t, dao, expectedSuite1, 1, "barTitle", "barStdout", "barStderr", 2, model.FAILED)
	addTest(t, dao, expectedSuite1, 2, "bazTitle", "bazStdout", "bazStderr", 3, model.PASSED)
	addTest(t, dao, expectedSuite1, 3, "pifTitle", "pifStdout", "pifStderr", 2, model.FAILED)
	addTest(t, dao, expectedSuite2, 0, "pafTitle", "", "", -1, model.IGNORED)

	row := dao.db.QueryRow("SELECT count(*) FROM tested")
	var count int
	err := row.Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 4, count)
}

func TestGetSuiteOutcome(t *testing.T) {
	dirpath := filez.MkTempDir("", "")
	defer os.RemoveAll(dirpath)
	dao := initTestDao(t, dirpath)
	suiteDao := initSuiteDao(t, dirpath)

	expectedSuite1 := "suite1"
	expectedStartTime1 := time.Now().Add(-2 * time.Second)
	expectedEndTime1 := time.Now()
	expectedDuration1 := expectedEndTime1.Sub(expectedStartTime1).Truncate(time.Microsecond)
	expectedSuite2 := "suite2"
	expectedStartTime2 := time.Now().Add(-1 * time.Second)
	expectedEndTime2 := time.Now()
	expectedDuration2 := expectedEndTime2.Sub(expectedStartTime2).Truncate(time.Microsecond)
	addSuiteWithOutcome(t, suiteDao, expectedSuite1, model.ERRORED, expectedStartTime1)
	addSuiteWithOutcome(t, suiteDao, expectedSuite2, model.IGNORED, expectedStartTime2)
	addTest(t, dao, expectedSuite1, 0, "fooTitle", "fooStdout", "fooStderr", 1, model.ERRORED)
	addTest(t, dao, expectedSuite1, 1, "barTitle", "barStdout", "barStderr", 2, model.FAILED)
	addTest(t, dao, expectedSuite1, 2, "bazTitle", "bazStdout", "bazStderr", 3, model.PASSED)
	addTest(t, dao, expectedSuite1, 3, "pifTitle", "pifStdout", "pifStderr", 2, model.FAILED)
	addTest(t, dao, expectedSuite2, 0, "pafTitle", "", "", -1, model.IGNORED)
	suiteDao.UpdateEndTime(expectedSuite1, expectedEndTime1)
	suiteDao.UpdateEndTime(expectedSuite2, expectedEndTime2)

	outcome, err := dao.GetSuiteOutcome(expectedSuite1)
	require.NoError(t, err)
	assert.Equal(t, expectedSuite1, outcome.TestSuite)
	assert.Equal(t, uint32(1), outcome.ErroredCount)
	assert.Equal(t, uint32(2), outcome.FailedCount)
	assert.Equal(t, uint32(0), outcome.IgnoredCount)
	assert.Equal(t, uint32(1), outcome.PassedCount)
	assert.Equal(t, uint32(4), outcome.TestCount)
	assert.Equal(t, expectedDuration1, outcome.Duration)
	assert.Len(t, outcome.FailureReports, 3)

	outcome, err = dao.GetSuiteOutcome(expectedSuite2)
	require.NoError(t, err)
	assert.Equal(t, expectedSuite2, outcome.TestSuite)
	assert.Equal(t, uint32(0), outcome.ErroredCount)
	assert.Equal(t, uint32(0), outcome.FailedCount)
	assert.Equal(t, uint32(1), outcome.IgnoredCount)
	assert.Equal(t, uint32(0), outcome.PassedCount)
	assert.Equal(t, uint32(1), outcome.TestCount)
	assert.Equal(t, expectedDuration2, outcome.Duration)
	assert.Len(t, outcome.FailureReports, 1)
}

func TestClearSuite(t *testing.T) {
	dirpath := filez.MkTempDir("", "")
	defer os.RemoveAll(dirpath)
	dao := initTestDao(t, dirpath)
	suiteDao := initSuiteDao(t, dirpath)

	expectedSuite1 := "suite1"
	expectedStartTime1 := time.Now().Add(-2 * time.Second)
	//expectedEndTime1 := time.Now()
	//expectedDuration1 := expectedEndTime1.Sub(expectedStartTime1).Truncate(time.Microsecond)
	expectedSuite2 := "suite2"
	expectedStartTime2 := time.Now().Add(-1 * time.Second)
	//expectedEndTime2 := time.Now()
	//expectedDuration2 := expectedEndTime2.Sub(expectedStartTime2).Truncate(time.Microsecond)
	addSuiteWithOutcome(t, suiteDao, expectedSuite1, model.ERRORED, expectedStartTime1)
	addSuiteWithOutcome(t, suiteDao, expectedSuite2, model.IGNORED, expectedStartTime2)
	addTest(t, dao, expectedSuite1, 0, "fooTitle", "fooStdout", "fooStderr", 1, model.ERRORED)
	addTest(t, dao, expectedSuite1, 1, "barTitle", "barStdout", "barStderr", 2, model.FAILED)
	addTest(t, dao, expectedSuite1, 2, "bazTitle", "bazStdout", "bazStderr", 3, model.PASSED)
	addTest(t, dao, expectedSuite2, 0, "pifTitle", "pifStdout", "pifStderr", 2, model.FAILED)
	addTest(t, dao, expectedSuite2, 1, "pafTitle", "", "", -1, model.IGNORED)

	var count int
	row := dao.db.QueryRow("SELECT count(*) FROM tested")
	err := row.Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 5, count)

	err = dao.ClearSuite(expectedSuite2)
	require.NoError(t, err)

	row = dao.db.QueryRow("SELECT count(*) FROM tested")
	err = row.Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 3, count)
}
