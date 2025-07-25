package repo

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"cmdt/internal/dao"
	"cmdt/internal/model"

	"github.com/mxbossard/utilz/errorz"
	"github.com/mxbossard/utilz/poolz"
	"github.com/mxbossard/utilz/zqlite"
)

const WaitingOpDoneSleepPeriodInMs = 50

type DbRepo struct {
	*poolz.PoolCloser

	dirpath   string
	token     string
	isolation string
	db        *zqlite.SynchronizedDB
	suiteDao  dao.Suite
	queueDao  dao.Queue
	testDao   dao.Test
	globalDao dao.Global
	//lastUpdate time.Time
}

func (r *DbRepo) wrap(err error) error {
	if err != nil {
		return fmt.Errorf("DB repo [token: %s ; isol: %s ; file: %s] error: %w", r.token, r.isolation, r.BackingFilepath(), err)
	}
	return nil
}

func (r *DbRepo) Open() (err error) {
	db, err := dao.DbOpen(r.dirpath)
	if err != nil {
		return r.wrap(err)
	}
	if r.db == nil {
		r.db = db
	} else {
		*r.db = *db
	}
	return
}

func (r *DbRepo) Close() error {
	logger.Info("Closing DB", "file", r.db.FileLockPath())
	err := r.db.Close()
	return r.wrap(err)
}

func (r *DbRepo) PoolClose() error {
	if r.PoolCloser != nil {
		return r.PoolCloser.PoolClose()
	}
	return nil
}

func (r *DbRepo) SetPoolCloser(pc poolz.PoolCloser) {
	r.PoolCloser = &pc
}

func (r DbRepo) BackingFilepath() string {
	path, err := forgeWorkDirectoryPath(r.token, r.isolation)
	if err != nil {
		err = r.wrap(err)
		errorz.Fatal(err)
	}
	return path
}

func (r DbRepo) MockDirectoryPath(testSuite string, testId uint) (mockDir string, err error) {
	var path string
	path, err = testSuiteDirectoryPath(testSuite, r.token, r.isolation)
	if err != nil {
		return
	}
	mockDir = filepath.Join(path, fmt.Sprintf("__mock_%d", testId))
	// create a mock dir
	err = os.MkdirAll(mockDir, 0755)
	if err != nil {
		err = r.wrap(err)
		return
	}
	return
}

func (r DbRepo) SaveGlobalConfig(cfg model.Config) (err error) {
	err = r.suiteDao.SaveGlobalConfig(cfg)
	err = r.wrap(err)
	return
}

func (r DbRepo) GetGlobalConfig() (cfg model.Config, err error) {
	found, err := r.suiteDao.FindGlobalConfig()
	if err != nil {
		err = r.wrap(err)
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
		err = r.wrap(err)
	}
	return
}

func (r DbRepo) InitSuite(cfg model.Config) (err error) {
	suite := cfg.TestSuite.Get()

	n := r.TestCount(suite)
	if n > 0 {
		err = r.ClearSuite(suite)
		if err != nil {
			err = r.wrap(err)
			return
		}
		cfg.SuiteStartTime.Set(time.Now())
		fmt.Fprintf(os.Stderr, "Cleared suite: [%s] (contained %d tests)\n", suite, n)
	}

	err = r.SaveSuiteConfig(cfg)
	if err != nil {
		err = fmt.Errorf("unable to init suite: %w", err)
		err = r.wrap(err)
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

func (r DbRepo) SaveSuiteConfig(cfg model.Config) (err error) {
	err = r.suiteDao.SaveSuiteConfig(cfg.TestSuite.Get(), cfg)
	if err != nil {
		err = r.wrap(err)
		return
	}
	logger.Debug("Saved suite config", "suite", cfg.TestSuite.Get(), "async", cfg.Async.Get())
	return
}

func (r DbRepo) GetSuiteConfig(testSuite string, initless bool) (cfg model.Config, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("cannot load suite config: %w", err)
			err = r.wrap(err)
		}
	}()

	found, err := r.suiteDao.FindSuiteConfig(testSuite)
	if err != nil {
		return
	}
	if found != nil {
		cfg = *found
		cfg.TestSuite.Set(testSuite)
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

func (r DbRepo) ClearSuite(testSuite string) (err error) {
	err = r.testDao.DeleteTestsOfSuite(testSuite)
	if err != nil {
		err = r.wrap(err)
		return
	}

	// FIXME: why is that commented out ? => timing out if uncommented
	err = r.queueDao.DeleteQueuesOfSuite(testSuite)
	if err != nil {
		return
	}

	err = r.suiteDao.DeleteSuite(testSuite)
	if err != nil {
		err = r.wrap(err)
		return
	}

	logger.Info("Cleared suite from repo", "suite", testSuite)
	return
}

func (r DbRepo) ListReportableSuites() (suites []string, err error) {
	suites, err = r.suiteDao.ListReportableOrdered()
	err = r.wrap(err)
	return
}

func (r DbRepo) ListReportableSuitesByMode(asyncMode, all bool) (suites []string, err error) {
	suites, err = r.suiteDao.ListReportableOrderedByMode(asyncMode, all)
	err = r.wrap(err)
	return
}

func (r DbRepo) ListAllSuites() (suites []string, err error) {
	suites, err = r.suiteDao.ListOrdered()
	err = r.wrap(err)
	return
}

func (r DbRepo) ListReportedAsyncSuites() (suites []string, err error) {
	suites, err = r.suiteDao.ListReportedAsync()
	err = r.wrap(err)
	return
}

func (r DbRepo) IgnoredSuiteCount(reportAll bool) (n uint) {
	n, err := r.suiteDao.IgnoredSuiteCount(reportAll)
	if err != nil {
		err = r.wrap(err)
		errorz.Fatal(err)
	}
	return
}

func (r DbRepo) EmptySuiteCount(reportAll bool) (n uint) {
	n, err := r.suiteDao.EmptySuiteCount(reportAll)
	if err != nil {
		err = r.wrap(err)
		errorz.Fatal(err)
	}
	return
}

func (r DbRepo) LoadTestOutcome(test model.TestSignature) (outcome *model.TestOutcome, err error) {
	outcome, err = r.testDao.LoadTestOutcome(test)
	if err != nil {
		err = r.wrap(err)
		return
	}
	return
}

func (r DbRepo) SaveTestOutcome(outcome model.TestOutcome) (err error) {
	err = r.testDao.SaveTestOutcome(outcome)
	if err != nil {
		err = r.wrap(err)
		return
	}
	//if outcome.Outcome == model.FAILED || outcome.Outcome == model.ERRORED || outcome.Outcome == model.TIMEOUT {
	if outcome.Outcome != model.IGNORED && outcome.Outcome != model.TIMEOUT {
		// FIXME: do we need to update the suite outcome on each test outcome ?
		// FIXME: which outcome to keep ?
		// An ignored test does not imply an ignored suite
		err = r.suiteDao.UpdateSuiteOutcome(outcome.TestSuite, outcome.Outcome)
		if err != nil {
			err = r.wrap(err)
			return
		}
	}

	//err = r.suiteDao.MarkSuiteReported(outcome.TestSuite, false, false)
	return
}

func (r DbRepo) SaveSuiteOutcome(outcome model.SuiteOutcome) (err error) {
	err = r.suiteDao.UpdateSuiteOutcome(outcome.TestSuite, outcome.Outcome)
	err = r.wrap(err)
	return
}

func (r DbRepo) UpdateLastTestTime(testSuite string) {
	err := r.suiteDao.UpdateSuiteEndTime(testSuite, time.Now())
	if err != nil {
		err = r.wrap(err)
		errorz.Fatal(err)
	}
}

func (r DbRepo) MarkSuiteReported(suite string) (err error) {
	err = r.suiteDao.MarkSuiteReported(suite)
	err = r.wrap(err)
	logger.Infof("Suite: [%s] was marked reported", suite)
	return
}

func (r DbRepo) MarkSuitesReported() (err error) {
	err = r.suiteDao.MarkSuitesReported()
	err = r.wrap(err)
	logger.Infof("All suites were marked reported")
	return
}

func (r DbRepo) SuiteStatus(suite string) (exists, reported bool, err error) {
	exists, reported, err = r.suiteDao.IsSuiteReported(suite)
	err = r.wrap(err)
	return
}

func (r DbRepo) LoadSuiteOutcome(testSuite string) (outcome model.SuiteOutcome, err error) {
	outcome, err = r.testDao.GetSuiteOutcome(testSuite)
	err = r.wrap(err)
	return
}

func (r DbRepo) IncrementSuiteSeq(testSuite, name string) (n uint) {
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
		err = r.wrap(err)
		errorz.Fatal(err)
	}
	logger.Debug("Incremented suite seq", "testSuite", testSuite, "name", name, "n", n)
	return
}

func (r DbRepo) NotReportedTestCount() (n uint) {
	n, err := r.suiteDao.NotReportedTestCount()
	if err != nil {
		err = r.wrap(err)
		errorz.Fatal(err)
	}
	logger.Debugf("Not reported global test count: %d", n)
	return
}

func (r DbRepo) TestCount(testSuite string) (n uint) {
	n, err := r.suiteDao.TestCount(testSuite)
	if err != nil {
		err = r.wrap(err)
		errorz.Fatal(err)
	}
	logger.Debugf("Not reported test count: %d", n)
	return
}

func (r DbRepo) ToReportTestCountByMode(asyncMode, all bool) (n uint) {
	n, err := r.suiteDao.ToReportTestCountByMode(asyncMode, all)
	if err != nil {
		err = r.wrap(err)
		errorz.Fatal(err)
	}
	return
}

func (r DbRepo) ToReportTestCountBySuiteAndMode(testSuite string, asyncMode, all bool) (n uint) {
	n, err := r.suiteDao.ToReportTestCountBySuiteAndMode(testSuite, asyncMode, all)
	if err != nil {
		err = r.wrap(err)
		errorz.Fatal(err)
	}
	return
}

func (r DbRepo) PassedCount(testSuite string) (n uint) {
	n, err := r.testDao.PassedCount(testSuite)
	if err != nil {
		err = r.wrap(err)
		errorz.Fatal(err)
	}
	return
}

func (r DbRepo) IgnoredCount(testSuite string) (n uint) {
	n, err := r.testDao.IgnoredCount(testSuite)
	if err != nil {
		err = r.wrap(err)
		errorz.Fatal(err)
	}
	return
}

func (r DbRepo) FailedCount(testSuite string) (n uint) {
	n, err := r.testDao.FailedCount(testSuite)
	if err != nil {
		err = r.wrap(err)
		errorz.Fatal(err)
	}
	return
}

func (r DbRepo) ErroredCount(testSuite string) (n uint) {
	n, err := r.testDao.ErroredCount(testSuite)
	if err != nil {
		err = r.wrap(err)
		errorz.Fatal(err)
	}
	return
}

func (r DbRepo) TooMuchCount(testSuite string) (n uint) {
	n, err := r.suiteDao.TooMuchCount(testSuite)
	if err != nil {
		err = r.wrap(err)
		errorz.Fatal(err)
	}
	return
}

func (r DbRepo) QueueOperation(op model.Operater) (err error) {
	err = r.queueDao.QueueOperater(op)
	if err == nil {
		logger.Info("Queued operation", "testSuite", op.Suite(), "kind", op.Kind(), "seq", op.Seq())
	}
	err = r.wrap(err)
	return
}

func (r DbRepo) UnqueueOperation() (op model.Operater, err error) {
	op, err = r.queueDao.UnqueueOperater()
	if op != nil {
		logger.Info("Unqueued operation", "testSuite", op.Suite(), "kind", op.Kind(), "seq", op.Seq(), "err", err)
	}
	err = r.wrap(err)
	return
}

func (r DbRepo) NotDone(op model.Operater) (err error) {
	if op == nil {
		panic(fmt.Errorf("nil operation"))
	}

	err = r.queueDao.NotDone(op)
	err = r.wrap(err)
	return
}

func (r DbRepo) Done(op model.Operater) (err error) {
	if op == nil {
		return
	}

	err = r.queueDao.Done(op)
	err = r.wrap(err)
	//logger.Warn("Unblock() unblocked", "opId", op.Id())
	return
}

func (r DbRepo) CountGlobalNotDoneBefore(op model.Operater) (count int, err error) {
	if op == nil {
		panic(fmt.Errorf("nil operation"))
	}

	count, err = r.queueDao.CountGlobalNotDoneBefore(op)
	err = r.wrap(err)
	return
}

func (r DbRepo) CountSuiteNotDoneBefore(op model.Operater) (count int, err error) {
	if op == nil {
		panic(fmt.Errorf("nil operation"))
	}

	count, err = r.queueDao.CountSuiteNotDoneBefore(op)
	err = r.wrap(err)
	return
}

// FIXME: move wait functions outside of repo
func (r DbRepo) WaitAllOperationsDoneBefore(op model.Operater, timeout time.Duration) (err error) {
	start := time.Now()
	for time.Since(start) < timeout {
		var count int
		count, err = r.queueDao.CountGlobalNotDoneBefore(op)
		if count == 0 || err != nil {
			// All operater done
			return
		}
		time.Sleep(WaitingOpDoneSleepPeriodInMs * time.Millisecond)
		logger.Trace("waiting ...", "op", op)
	}
	err = fmt.Errorf("WaitAllOperationsDoneBefore() for op: %s timed out after %s", op, timeout)
	err = r.wrap(err)
	return
}

func (r DbRepo) WaitSuiteOperationsDoneBefore(op model.Operater, timeout time.Duration) (err error) {
	start := time.Now()
	for time.Since(start) < timeout {
		var count int
		count, err = r.queueDao.CountSuiteNotDoneBefore(op)
		if count == 0 || err != nil {
			// All operater done
			return
		}
		time.Sleep(WaitingOpDoneSleepPeriodInMs * time.Millisecond)
		logger.Trace("waiting ...", "op", op)
	}
	err = fmt.Errorf("WaitSuiteOperationsDoneBefore() for op: %s timed out after %s", op, timeout)
	err = r.wrap(err)
	return
}

func (r DbRepo) WaitOperationDone(op model.Operater, timeout time.Duration) (exitCode int16, opErr, err error) {
	exitCode = -1
	start := time.Now()
	for time.Since(start) < timeout {
		var done bool
		done, exitCode, opErr, err = r.queueDao.IsOperationsDone(op)
		if done || opErr != nil {
			// Operater done
			return
		}
		time.Sleep(WaitingOpDoneSleepPeriodInMs * time.Millisecond)
		logger.Trace("waiting ...", "op", op)
	}
	err = fmt.Errorf("WaitOperationDone() for op: %s timed out after %s", op, timeout)
	err = r.wrap(err)
	return
}

func (r DbRepo) WaitEmptyQueue(testSuite string, timeout time.Duration) (err error) {
	start := time.Now()
	for time.Since(start) < timeout {
		var count int
		count, err = r.queueDao.QueuedOperationsCountBySuite(testSuite, nil)
		if err != nil {
			err = r.wrap(err)
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
	err = r.wrap(err)
	return
}

func (r DbRepo) WaitAllEmpty(timeout time.Duration) (err error) {
	start := time.Now()
	for time.Since(start) < timeout {
		var count int
		count, err = r.queueDao.QueuedOperationsCount()
		if err != nil {
			err = r.wrap(err)
			return
		}
		if count == 0 {
			// No operation queued
			return
		}
		time.Sleep(1 * time.Millisecond)
	}
	err = fmt.Errorf("WaitAllEmpty() timed out after %s", timeout)
	err = r.wrap(err)
	return
}

func (r DbRepo) unqueue() (ok bool, op model.Operater, err error) {
	queuedOperationsCount, err := r.queueDao.QueuedOperationsCount()
	if err != nil {
		err = r.wrap(err)
		return
	}
	//logger.Warn("unqueue()", "globalOperationsCount", globalOperationsCount)
	if queuedOperationsCount == 0 {
		return
	}

	op, err = r.queueDao.UnqueueOperater()
	if err != nil || op == nil {
		err = r.wrap(err)
		return
	}

	ok = true
	return
}

func (r DbRepo) SaveDaemonPid(pid int) (err error) {
	return r.globalDao.SaveDaemonPid(pid)
}

func (r DbRepo) ClearDaemonPid(pid int) (err error) {
	return r.globalDao.ClearDaemonPid(pid)
}

func (r DbRepo) GetDaemonPid() (int, error) {
	pid, err := r.globalDao.GetDaemonPid()
	fmt.Printf("\n<<>> found Daemon [%d](%s/%s) PID in DB: %d\n", os.Getpid(), r.token, r.isolation, pid)
	return pid, err
}

func (r DbRepo) ReportActivity() (err error) {
	return r.globalDao.ReportActivity()
}

func (r DbRepo) InactivityDuration() (time.Duration, error) {
	return r.globalDao.InactivityDuration()
}

func newDbRepo(dirpath, isolation, token string) (r DbRepo, err error) {
	r.dirpath = dirpath
	r.token = token
	r.isolation = isolation

	err = r.Open()
	if err != nil {
		err = r.wrap(err)
		return
	}

	db := r.db
	inited, err := dao.IsInitialized(db)
	if err != nil {
		err = r.wrap(err)
		return r, err
	}

	r.queueDao, err = dao.NewQueue(db, !inited)
	if err != nil {
		err = r.wrap(err)
		return
	}
	r.suiteDao, err = dao.NewSuite(db, !inited)
	if err != nil {
		err = r.wrap(err)
		return
	}
	r.testDao, err = dao.NewTest(db, !inited)
	if err != nil {
		err = r.wrap(err)
		return
	}
	r.globalDao, err = dao.NewGlobal(db, !inited)
	if err != nil {
		err = r.wrap(err)
		return
	}

	logger.Info("New db repo", "dirpath", dirpath, "token", token, "isolation", isolation)
	return
}
