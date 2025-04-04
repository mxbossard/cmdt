package repo

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"cmdt/internal/dao"
	"cmdt/internal/model"

	"github.com/mxbossard/utilz/errorz"
	"github.com/mxbossard/utilz/zql"
)

const WaitingOpDoneSleepPeriodInMs = 50

type dbRepo struct {
	dirpath   string
	token     string
	isolation string
	db        *zql.SynchronizedDB
	suiteDao  dao.Suite
	queueDao  dao.Queue
	testDao   dao.Test
	//lastUpdate time.Time
}

func (r *dbRepo) Init() (err error) {
	/*
		if r.db != nil {
			err = r.db.Close()
			if err != nil {
				return
			}
		}
	*/

	db, err := dao.DbOpen(r.dirpath)
	if err != nil {
		return
	}
	if r.db == nil {
		r.db = db
	} else {
		*r.db = *db
	}
	return
}

func (r *dbRepo) Close() error {
	logger.Info("Closing DB", "file", r.db.FileLockPath())
	return r.db.Close()
	//return nil
}

func (r dbRepo) BackingFilepath() string {
	path, err := forgeWorkDirectoryPath(r.token, r.isolation)
	if err != nil {
		errorz.Fatal(err)
	}
	return path
}

func (r dbRepo) MockDirectoryPath(testSuite string, testId uint16) (mockDir string, err error) {
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
	err = r.suiteDao.SaveGlobalConfig(cfg)
	return
}

func (r dbRepo) GetGlobalConfig() (cfg model.Config, err error) {
	found, err := r.suiteDao.FindGlobalConfig()
	if err != nil {
		return
	}
	if found != nil {
		cfg = *found
	} else {
		// global config does not exists yet
		// create a new default one
		cfg = model.NewGlobalDefaultConfig()
		cfg.Token.Set(r.token)
		now := time.Now()
		cfg.GlobalStartTime.Set(now)
		cfg.LastReportTime.Set(now)
		err = r.SaveGlobalConfig(cfg)
	}
	return
}

func (r dbRepo) NotReportedTestCount() (n uint16) {
	n, err := r.suiteDao.NotReportedTestCount()
	if err != nil {
		errorz.Fatal(err)
	}
	return
}

func (r dbRepo) InitSuite(cfg model.Config) (err error) {
	suite := cfg.TestSuite.Get()

	n := r.TestCount(suite)
	if n > 0 {
		err = r.ClearSuite(suite)
		if err != nil {
			return
		}
		fmt.Fprintf(os.Stderr, "Cleared suite: [%s] (contained %d tests)\n", suite, n)
	}

	err = r.SaveSuiteConfig(cfg)
	if err != nil {
		err = fmt.Errorf("unable to init suite: %w", err)
	}
	logger.Info("Initialized suite in repo", "suite", suite)

	if cfg.IgnoreSuite.GetOr(false) {
		// Ignored suite can save it's outcome
		outcome := model.SuiteOutcome{}
		outcome.TestSuite = suite
		outcome.Outcome = model.IGNORED
		r.SaveSuiteOutcome(outcome)
	}

	return
}

func (r dbRepo) SaveSuiteConfig(cfg model.Config) (err error) {
	err = r.suiteDao.SaveSuiteConfig(cfg.TestSuite.Get(), cfg)
	if err != nil {
		return
	}
	logger.Debug("Saved suite config", "suite", cfg.TestSuite.Get(), "async", cfg.Async.Get())
	return
}

func (r dbRepo) GetSuiteConfig(testSuite string, initless bool) (cfg model.Config, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("cannot load suite config: %w", err)
		}
	}()

	found, err := r.suiteDao.FindSuiteConfig(testSuite)
	if err != nil {
		return
	}
	if found != nil {
		cfg = *found
		logger.Debug("Loaded suite config from DB.", "suite", testSuite, "async", cfg.Async.Get())
	} else {
		// suite config does not exists yet
		// create a new default one
		cfg, err = r.GetGlobalConfig()
		if err != nil {
			return
		}
		var suiteCfg model.Config
		if initless {
			//logger.Warn("Saving new initless config", "testSuite", testSuite)
			suiteCfg = model.NewInitlessSuiteDefaultConfig()
			logger.Debug("Built initless default suite config.", "suite", testSuite)
		} else {
			//logger.Warn("Saving new inited config", "testSuite", testSuite)
			suiteCfg = model.NewSuiteDefaultConfig()
			logger.Debug("Built default suite config.", "suite", testSuite)
		}
		suiteCfg.TestSuite.Set(testSuite)
		suiteCfg.SuiteStartTime.Set(time.Now())
		cfg.Merge(suiteCfg)
		err = r.SaveSuiteConfig(cfg)
	}
	return
}

func (r dbRepo) ClearSuite(testSuite string) (err error) {
	err = r.testDao.DeleteTestsOfSuite(testSuite)
	if err != nil {
		return
	}
	// FIXME: why is that commented out ? => timing out if uncommented
	/*
		err = r.queueDao.DeleteQueuesOfSuite(testSuite)
		if err != nil {
			return
		}
	*/
	err = r.suiteDao.DeleteSuite(testSuite)
	if err != nil {
		return
	}

	logger.Info("Cleared suite from repo", "suite", testSuite)
	return
}

func (r dbRepo) ListReportableSuites() (suites []string, err error) {
	suites, err = r.suiteDao.ListReportablePassedFailedErrored()
	return
}

func (r dbRepo) ListAllSuites() (suites []string, err error) {
	suites, err = r.suiteDao.ListPassedFailedErrored()
	return
}

func (r dbRepo) ListSyncSuites() (suites []string, err error) {
	suites, err = r.suiteDao.ListSync()
	return
}

func (r dbRepo) ListAsyncSuites() (suites []string, err error) {
	suites, err = r.suiteDao.ListAsync()
	return
}

func (r dbRepo) ListReportedAsyncSuites() (suites []string, err error) {
	suites, err = r.suiteDao.ListReportedAsync()
	return
}

func (r dbRepo) SaveTestOutcome(outcome model.TestOutcome) (err error) {
	err = r.testDao.SaveTestOutcome(outcome)
	if err != nil {
		return
	}
	//if outcome.Outcome == model.FAILED || outcome.Outcome == model.ERRORED || outcome.Outcome == model.TIMEOUT {
	if outcome.Outcome != model.IGNORED && outcome.Outcome != model.TIMEOUT {
		// FIXME: do we need to update the suite outcome on each test outcome ?
		// FIXME: which outcome to keep ?
		// An ignored test does not imply an ignored suite
		err = r.suiteDao.UpdateSuiteOutcome(outcome.TestSuite, outcome.Outcome)
		if err != nil {
			return
		}
	}

	err = r.suiteDao.MarkSuiteReported(outcome.TestSuite, false, false)
	return
}

func (r dbRepo) SaveSuiteOutcome(outcome model.SuiteOutcome) (err error) {
	err = r.suiteDao.UpdateSuiteOutcome(outcome.TestSuite, outcome.Outcome)
	return
}

func (r dbRepo) UpdateLastTestTime(testSuite string) {
	err := r.suiteDao.UpdateSuiteEndTime(testSuite, time.Now())
	if err != nil {
		errorz.Fatal(err)
	}
}

func (r dbRepo) MarkSuiteReported(suite string, kept bool) (err error) {
	return r.suiteDao.MarkSuiteReported(suite, true, kept)
}

func (r dbRepo) MarkReportedAll() (err error) {
	return r.suiteDao.MarkSuiteReported("", true, false)
}

func (r dbRepo) SuiteStatus(suite string) (exists, reported, kept bool, err error) {
	return r.suiteDao.IsSuiteReported(suite)
}

func (r dbRepo) LoadSuiteOutcome(testSuite string) (outcome model.SuiteOutcome, err error) {
	outcome, err = r.testDao.GetSuiteOutcome(testSuite)
	return
}

func (r dbRepo) IncrementSuiteSeq(testSuite, name string) (n uint16) {
	// FIXME should this be used ?

	var err error
	if name == model.TestSequenceFilename {
		n, err = r.suiteDao.NextSeq(testSuite)
	} else if name == model.TooMuchSequenceFilename {
		n, err = r.suiteDao.IncrementTooMuchCount(testSuite)
	} else {
		// Other seq increment are supported by counter in DB
		return 9999
	}
	if err != nil {
		errorz.Fatal(err)
	}
	logger.Debug("Incremented suite seq", "testSuite", testSuite, "name", name, "n", n)
	return
}

func (r dbRepo) TestCount(testSuite string) (n uint16) {
	n, err := r.suiteDao.TestCount(testSuite)
	if err != nil {
		errorz.Fatal(err)
	}
	return
}

func (r dbRepo) PassedCount(testSuite string) (n uint16) {
	n, err := r.testDao.PassedCount(testSuite)
	if err != nil {
		errorz.Fatal(err)
	}
	return
}

func (r dbRepo) IgnoredCount(testSuite string) (n uint16) {
	n, err := r.testDao.IgnoredCount(testSuite)
	if err != nil {
		errorz.Fatal(err)
	}
	return
}

func (r dbRepo) FailedCount(testSuite string) (n uint16) {
	n, err := r.testDao.FailedCount(testSuite)
	if err != nil {
		errorz.Fatal(err)
	}
	return
}

func (r dbRepo) ErroredCount(testSuite string) (n uint16) {
	n, err := r.testDao.ErroredCount(testSuite)
	if err != nil {
		errorz.Fatal(err)
	}
	return
}

func (r dbRepo) TooMuchCount(testSuite string) (n uint16) {
	n, err := r.suiteDao.TooMuchCount(testSuite)
	if err != nil {
		errorz.Fatal(err)
	}
	return
}

func (r dbRepo) QueueOperation(op model.Operater) (err error) {
	err = r.queueDao.QueueOperater(op)
	if err == nil {
		logger.Info("Queued operation", "testSuite", op.Suite(), "kind", op.Kind(), "seq", op.Seq())
	}
	return
}

func (r dbRepo) UnqueueOperation() (op model.Operater, err error) {
	op, err = r.queueDao.UnqueueOperater()
	if op != nil {
		logger.Info("Unqueued operation", "testSuite", op.Suite(), "kind", op.Kind(), "seq", op.Seq(), "err", err)
	}
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
		time.Sleep(WaitingOpDoneSleepPeriodInMs * time.Millisecond)
		logger.Trace("waiting ...", "op", op)
	}
	err = fmt.Errorf("WaitOperationDone() for op: %s timed out after %s", op, timeout)
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
	err = fmt.Errorf("WaitAllEmpty() timed out after %s", timeout)
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

func newDbRepo(dirpath, isolation, token string) (r dbRepo, err error) {
	r.dirpath = dirpath
	r.token = token
	r.isolation = isolation

	err = r.Init()
	if err != nil {
		return
	}

	db := r.db
	inited, err := dao.IsInitialized(db)
	if err != nil {
		return r, err
	}

	// if !inited {
	// 	fmt.Printf(">>> DB not initialized yet. pid: %d; dirpath: %s; token: %s; isol: %s\n", os.Getpid(), dirpath, token, isolation)
	// }

	r.queueDao, err = dao.NewQueue(db, !inited)
	if err != nil {
		return
	}
	r.suiteDao, err = dao.NewSuite(db, !inited)
	if err != nil {
		return
	}
	r.testDao, err = dao.NewTest(db, !inited)
	if err != nil {
		return
	}

	logger.Info("New db repo", "dirpath", dirpath, "token", token, "isolation", isolation)

	//err = r.Close()
	return
}
