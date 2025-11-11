package service

import (
	"cmdt/internal/model"
	"cmdt/internal/repo"
	"fmt"
	"time"

	"github.com/mxbossard/utilz/collectionz"
	"github.com/mxbossard/utilz/errorz"
)

const WaitingOpDoneSleepPeriodInMs = 50

// FIXME: move wait functions outside of repo
func WaitAllOperationsDoneBefore(r repo.DbRepo, op model.Operater, timeout time.Duration) (err error) {
	start := time.Now()
	for time.Since(start) < timeout {
		var count int
		count, err = r.CountGlobalNotDoneBeforeOp(op)
		if count == 0 || err != nil {
			// All operater done
			return
		}
		time.Sleep(WaitingOpDoneSleepPeriodInMs * time.Millisecond)
		logger.Trace("waiting ...", "op", op)
	}
	err = fmt.Errorf("WaitAllOperationsDoneBefore() for op: %s timed out after %s", op, timeout)
	return
}

// Watch for all operations done adding each suite done on channel.
func WatchAllTestsPerformed(r repo.Repo, reportAll bool, timeout time.Duration) (c chan errorz.ValueOrErr[string], err error) {
	start := time.Now()

	var waitingSuites, completedSuites, erroredSuites []string
	if reportAll {
		waitingSuites, err = r.ListAllSuites()
		if err != nil {
			return
		}
	} else {
		waitingSuites, err = r.ListReportableSuites()
		if err != nil {
			return
		}
	}

	testsCountBySuite := make(map[string]uint)
	// FIXME: LastGlobalSeq does not exists ! seq is not global scoped.
	// For each suite must check if notReportedCount == 0  before time ?
	// Need to watch difference between seq and tested count
	// Need to do that for each suite ?
	// Problem: sync suites does not have a
	// var count, lastSeq int
	// lastSeq, err = r.LastGlobalSeq()
	// if err != nil {
	// 	return
	// }

	c = make(chan errorz.ValueOrErr[string], len(waitingSuites))

	go func() {
		firstLoop := true
		for len(waitingSuites) > 0 && time.Since(start) < timeout {
			for _, suite := range waitingSuites {
				if firstLoop {
					testsCount := r.TestCount(suite)
					testsCountBySuite[suite] = testsCount
				}
				testsCount := testsCountBySuite[suite]
				count, err := r.CountSuiteTestedBeforeId(suite, testsCount+1)
				//_, reported, err := r.SuiteStatus(suite)
				if err != nil {
					c <- errorz.ValOrErr("", err)
					erroredSuites = append(erroredSuites, suite)
					continue
				}
				if uint(count) == testsCount {
					c <- errorz.Val(suite)
					completedSuites = append(completedSuites, suite)
				}
			}
			firstLoop = false
			waitingSuites = collectionz.KeepLeft(&waitingSuites, &completedSuites)
			waitingSuites = collectionz.KeepLeft(&waitingSuites, &erroredSuites)
			time.Sleep(WaitingOpDoneSleepPeriodInMs * time.Millisecond)
		}
		close(c)
	}()

	return
}

func WaitAllOperationsDone(r repo.DbRepo, timeout time.Duration) (err error) {
	start := time.Now()
	for time.Since(start) < timeout {
		var count, lastId int
		lastId, err = r.LastGlobalOperationId()
		if err != nil {
			return err
		}
		count, err = r.CountGlobalNotDoneBeforeId(uint(lastId) + 1)
		if count == 0 || err != nil {
			// All operater done
			return
		}
		time.Sleep(WaitingOpDoneSleepPeriodInMs * time.Millisecond)
		logger.Trace("waiting ...", "ops <=", lastId)
	}
	err = fmt.Errorf("WaitAllOperationsDone() timed out after %s", timeout)
	return
}

func WaitSuiteOperationsDoneBefore(r repo.DbRepo, op model.Operater, timeout time.Duration) (err error) {
	start := time.Now()
	for time.Since(start) < timeout {
		var count int
		count, err = r.CountSuiteNotDoneBeforeOp(op)
		if count == 0 || err != nil {
			// All operater done
			return
		}
		time.Sleep(WaitingOpDoneSleepPeriodInMs * time.Millisecond)
		logger.Trace("waiting ...", "op", op)
	}
	err = fmt.Errorf("WaitSuiteOperationsDoneBefore() for op: %s timed out after %s", op, timeout)
	return
}

func WaitSuiteOperationsDone(r repo.DbRepo, suite string, timeout time.Duration) (err error) {
	start := time.Now()
	for time.Since(start) < timeout {
		var count, lastId int
		lastId, err = r.LastSuiteOperationId(suite)
		if err != nil {
			return err
		}
		count, err = r.CountSuiteNotDoneBeforeId(suite, uint(lastId)+1)
		if count == 0 || err != nil {
			// All operater done
			return
		}
		time.Sleep(WaitingOpDoneSleepPeriodInMs * time.Millisecond)
		logger.Trace("waiting ...", "op <=", lastId)
	}
	err = fmt.Errorf("WaitSuiteOperationsDone() for suite: %s timed out after %s", suite, timeout)
	return
}

func WaitOperationDone(r repo.DbRepo, op model.Operater, timeout time.Duration) (exitCode int16, opErr, err error) {
	exitCode = -1
	start := time.Now()
	for time.Since(start) < timeout {
		var done bool
		done, exitCode, opErr, err = r.IsOperationsDone(op)
		if done || opErr != nil {
			// Operater done
			return
		}
		time.Sleep(WaitingOpDoneSleepPeriodInMs * time.Millisecond)
		logger.Trace("waiting ...", "op", op)
	}
	err = fmt.Errorf("WaitOperationDone() for op: %s timed out after %s", op, timeout)
	return
}

func WaitEmptyQueue(r repo.DbRepo, testSuite string, timeout time.Duration) (err error) {
	start := time.Now()
	for time.Since(start) < timeout {
		var count int
		count, err = r.QueuedOperationsCountBySuite(testSuite)
		if err != nil {
			return
		}
		logger.Trace("WaitEmptyQueue()", "testSuite", testSuite, "count", count)
		if count == 0 {
			// Queue is empty
			return
		}
		time.Sleep(1 * time.Millisecond)
	}
	err = fmt.Errorf("WaitEmptyQueue() for suite: %s timed out after %s", testSuite, timeout)
	return
}

func WaitAllEmpty(r repo.DbRepo, timeout time.Duration) (err error) {
	start := time.Now()
	for time.Since(start) < timeout {
		var count int
		count, err = r.QueuedOperationsCount()
		if err != nil {
			return
		}
		if count == 0 {
			// No operation queued
			return
		}
		time.Sleep(1 * time.Millisecond)
	}
	err = fmt.Errorf("WaitAllEmpty() timed out after %s", timeout)
	return
}
