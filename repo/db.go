package repo

import (
	"errors"
	"time"

	"mby.fr/cmdtest/dao"
	"mby.fr/cmdtest/model"
)

type dbRepo struct {
	SuiteDao dao.Suite
	queueDao dao.Queue
	//lastUpdate time.Time
}

func (r dbRepo) Queue(op model.Operater) (err error) {
	err = r.queueDao.QueueOperater(op)
	//logger.Warn("Queue() added", "testSuite", testSuite, "kind", op.Kind(), "seq", op.Seq())
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

func (r dbRepo) Unqueue() (ok bool, op model.Operater, err error) {
	ok, op, err = r.unqueue()
	//	logger.Warn("Unqueue()", "kind", op.Kind(), "opId", op.Id())
	return
}

func (r dbRepo) Done(op model.Operater) (err error) {
	if op == nil {
		return
	}

	err = r.queueDao.Done(op)
	//logger.Warn("Unblock() unblocked", "opId", op.Id())
	return
}

func (r dbRepo) WaitOperaterDone(op model.Operater, timeout time.Duration) (exitCode int16, err error) {
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
	err = errors.New("WaitOperaterDone() timed out")
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

func newDbRepo(dirpath string) (d dbRepo, err error) {
	db, err := dao.DbOpen(dirpath)
	if err != nil {
		return
	}
	d.queueDao, err = dao.NewQueue(db)
	if err != nil {
		return
	}
	d.SuiteDao, err = dao.NewSuite(db)
	return
}
