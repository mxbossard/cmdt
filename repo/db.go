package repo

import (
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"mby.fr/cmdtest/dao"
	"mby.fr/cmdtest/model"
)

type dbRepo struct {
	token     string
	isolation string

	suiteDao dao.Suite
	queueDao dao.Queue
	//lastUpdate time.Time
}

func (r dbRepo) BackingFilepath() string {
	path, err := forgeWorkDirectoryPath(r.token, r.isolation)
	if err != nil {
		log.Fatal(err)
	}
	return path
}

func (r dbRepo) MockDirectoryPath(testSuite string, testId uint32) (mockDir string, err error) {
	var path string
	path, err = testSuiteDirectoryPath(testSuite, r.token, r.isolation)
	if err != nil {
		return
	}
	mockDir = filepath.Join(path, fmt.Sprintf("__mock_%d", testId))
	// create a mock dir
	err = os.MkdirAll(mockDir, 0755)
	if err != nil {
		return
	}
	return
}

func (r dbRepo) SaveGlobalConfig(cfg model.Config) (err error) {
	// TODO
}

func (r dbRepo) GetGlobalConfig() (cfg model.Config, err error) {
	// TODO
}

func (r dbRepo) InitSuite(cfg model.Config) (err error) {
	// TODO
}

func (r dbRepo) SaveSuiteConfig(cfg model.Config) (err error) {
	// TODO
}

func (r dbRepo) GetSuiteConfig(testSuite string, initless bool) (cfg model.Config, err error) {
	// TODO
}

func (r dbRepo) ClearTestSuite(testSuite string) (err error) {
	// TODO
}

func (r dbRepo) ListTestSuites() (suites []string, err error) {
	// TODO
}

func (r dbRepo) SaveTestOutcome(outcome model.TestOutcome) (err error) {
	// TODO
}

func (r dbRepo) UpdateLastTestTime(testSuite string) {
	// TODO
}

func (r dbRepo) LoadSuiteOutcome(testSuite string) (outcome model.SuiteOutcome, err error) {
	// TODO
}

func (r dbRepo) TestCount(testSuite string) (n uint32) {
	// TODO
}

func (r dbRepo) PassedCount(testSuite string) (n uint32) {
	// TODO
}

func (r dbRepo) IgnoredCount(testSuite string) (n uint32) {
	// TODO
}

func (r dbRepo) FailedCount(testSuite string) (n uint32) {
	// TODO
}

func (r dbRepo) ErroredCount(testSuite string) (n uint32) {
	// TODO
}

func (r dbRepo) TooMuchCount(testSuite string) (n uint32) {
	// TODO
}

func (r dbRepo) IncrementSuiteSeq(testSuite, name string) (n uint32) {
	// TODO
}

func (r dbRepo) QueueOperation(op model.Operater) (err error) {
	err = r.queueDao.QueueOperater(op)
	//logger.Warn("Queue() added", "testSuite", testSuite, "kind", op.Kind(), "seq", op.Seq())
	return
}

func (r dbRepo) UnqueueOperation() (op model.Operater, err error) {
	// TODO
}

func (r dbRepo) Done(op model.Operater) (err error) {
	if op == nil {
		return
	}

	err = r.queueDao.Done(op)
	//logger.Warn("Unblock() unblocked", "opId", op.Id())
	return
}

func (r dbRepo) WaitOperationDone(op model.Operater, timeout time.Duration) (exitCode int16, err error) {
	exitCode = -1
	start := time.Now()
	for time.Since(start) < timeout {
		var done bool
		done, exitCode, err = r.queueDao.IsOperationsDone(op)
		if done || err != nil {
			// Operater done
			return
		}
		time.Sleep(1 * time.Millisecond)
		//logger.Debug("waiting ...", "op", op)
	}
	err = errors.New("WaitOperationDone() timed out")
	return
}

func (r dbRepo) WaitEmptyQueue(testSuite string, timeout time.Duration) (err error) {
	start := time.Now()
	for time.Since(start) < timeout {
		var count int
		count, err = r.queueDao.QueuedOperationsCountBySuite(testSuite, nil)
		if err != nil {
			return
		}
		//logger.Warn("WaitEmptyQueue()", "testSuite", testSuite, "count", count)
		if count == 0 {
			// Queue is empty
			return
		}
		time.Sleep(1 * time.Millisecond)
	}
	err = errors.New("WaitEmptyQueue() timed out")
	return
}

func (r dbRepo) WaitAllEmpty(timeout time.Duration) (err error) {
	start := time.Now()
	for time.Since(start) < timeout {
		var count int
		count, err = r.queueDao.QueuedOperationsCount()
		if err != nil {
			return
		}
		if count == 0 {
			// No operation queued
			return
		}
		time.Sleep(1 * time.Millisecond)
	}
	err = errors.New("WaitAllEmpty() timed out")
	return
}

func (r dbRepo) unqueue() (ok bool, op model.Operater, err error) {
	queuedOperationsCount, err := r.queueDao.QueuedOperationsCount()
	if err != nil {
		return
	}
	//logger.Warn("unqueue()", "globalOperationsCount", globalOperationsCount)
	if queuedOperationsCount == 0 {
		return
	}

	op, err = r.queueDao.UnqueueOperater()
	if err != nil || op == nil {
		return
	}

	ok = true
	return
}

func (r dbRepo) Unqueue0() (ok bool, op model.Operater, err error) {
	ok, op, err = r.unqueue()
	//	logger.Warn("Unqueue()", "kind", op.Kind(), "opId", op.Id())
	return
}

func newDbRepo(dirpath string) (d dbRepo, err error) {
	db, err := dao.DbOpen(dirpath)
	if err != nil {
		return
	}
	d.queueDao, err = dao.NewQueue(db)
	if err != nil {
		return
	}
	d.suiteDao, err = dao.NewSuite(db)
	return
}
