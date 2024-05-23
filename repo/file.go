package repo

import (
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"mby.fr/cmdtest/model"
	"mby.fr/cmdtest/utils"
	"mby.fr/utils/format"
	"mby.fr/utils/utilz"
)

type FileRepo struct {
	token     string
	isolation string

	dbRepo dbRepo
}

func (r FileRepo) BackingFilepath() string {
	path, err := forgeWorkDirectoryPath(r.token, "")
	if err != nil {
		log.Fatal(err)
	}
	return path
}

func (r FileRepo) MockDirectoryPath(testSuite string, testId uint16) (mockDir string, err error) {
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

func (r FileRepo) SaveGlobalConfig(cfg model.Config) (err error) {
	cfg.TestSuite.Set(model.GlobalConfigTestSuiteName)
	//err = persistSuiteConfig(r.token, cfg)
	err = r.persistGlobalConfig(cfg)
	if err != nil {
		err = fmt.Errorf("unable to save global config: %w", err)
	}
	return
}

func (r FileRepo) GetGlobalConfig() (cfg model.Config, err error) {
	//cfg, err = loadGlobalConfig(r.token)
	loaded, err := r.loadGlobalConfig()
	if err != nil {
		return
	}
	if loaded != nil {
		cfg = *loaded
	} else {
		// global config does not exists yet
		// create a new default one
		cfg = model.NewGlobalDefaultConfig()
		cfg.Token.Set(r.token)
		cfg.GlobalStartTime.Set(time.Now())
		err = r.SaveGlobalConfig(cfg)
	}
	return
}

func (r FileRepo) InitSuite(cfg model.Config) (err error) {
	err = r.ClearTestSuite(cfg.TestSuite.Get())
	if err != nil {
		return
	}
	//err = persistSuiteConfig(r.token, cfg)
	err = r.persistSuiteConfig(cfg)
	if err != nil {
		err = fmt.Errorf("unable to init suite: %w", err)
	}
	return
}

func (r FileRepo) SaveSuiteConfig(cfg model.Config) (err error) {
	//err = persistSuiteConfig(r.token, cfg)
	err = r.persistSuiteConfig(cfg)
	if err != nil {
		err = fmt.Errorf("unable to save suite config: %w", err)
	}
	return
}

func (r FileRepo) GetSuiteConfig(testSuite string, initless bool) (cfg model.Config, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("cannot load suite config: %w", err)
		}
	}()

	//cfg, err = loadSuiteConfig(testSuite, r.token)
	loaded, err := r.loadSuiteConfig(testSuite)
	if err != nil {
		return
	}
	if loaded != nil {
		cfg = *loaded
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
		} else {
			//logger.Warn("Saving new inited config", "testSuite", testSuite)
			suiteCfg = model.NewSuiteDefaultConfig()
		}
		suiteCfg.TestSuite.Set(testSuite)
		suiteCfg.SuiteStartTime.Set(time.Now())
		cfg.Merge(suiteCfg)
		err = r.SaveSuiteConfig(cfg)
	}
	return
}

func (r FileRepo) ClearTestSuite(testSuite string) (err error) {
	err = clearSuiteWorkspace(testSuite, r.token, r.isolation)
	return
}

func (r FileRepo) ListTestSuites() (suites []string, err error) {
	var tmpDir string
	tmpDir, err = forgeWorkDirectoryPath(r.token, "")
	if err != nil {
		return
	}
	_, err = os.Stat(tmpDir)
	if os.IsNotExist(err) {
		err = nil
		return
	}

	matches, err := filepath.Glob(tmpDir + "/*")
	if err != nil {
		err = fmt.Errorf("cannot list test suites: %w", err)
		return
	}

	// keep only dirs
	var dirs []string
	for _, m := range matches {
		f, _ := os.Stat(m)
		if f.IsDir() {
			dirs = append(dirs, m)
		}
	}

	// Add success
	for _, m := range dirs {
		testSuite := filepath.Base(m)
		if !strings.HasPrefix(testSuite, "_") {
			failedCount := utils.ReadSeq(tmpDir, testSuite, model.FailedSequenceFilename)
			errorCount := utils.ReadSeq(tmpDir, testSuite, model.ErroredSequenceFilename)
			if failedCount == 0 && errorCount == 0 {
				suites = append(suites, testSuite)
			}
		}
	}
	// Add failures
	for _, m := range dirs {
		testSuite := filepath.Base(m)
		if !strings.HasPrefix(testSuite, "_") {
			failedCount := utils.ReadSeq(tmpDir, testSuite, model.FailedSequenceFilename)
			errorCount := utils.ReadSeq(tmpDir, testSuite, model.ErroredSequenceFilename)
			if failedCount > 0 && errorCount == 0 {
				suites = append(suites, testSuite)
			}
		}
	}
	// Add errors
	for _, m := range dirs {
		testSuite := filepath.Base(m)
		if !strings.HasPrefix(testSuite, "_") {
			errorCount := utils.ReadSeq(tmpDir, testSuite, model.ErroredSequenceFilename)
			if errorCount > 0 {
				suites = append(suites, testSuite)
			}
		}
	}
	return
}

func (r FileRepo) SaveTestOutcome(outcome model.TestOutcome) (err error) {
	var stdoutLog, stderrLog, reportLog *os.File
	stdoutLog, stderrLog, reportLog, err = cmdLogFiles(outcome.TestSuite, r.token, r.isolation, outcome.Seq)
	if err != nil {
		return
	}
	defer stdoutLog.Close()
	defer stderrLog.Close()
	defer reportLog.Close()

	_, err = stdoutLog.WriteString(outcome.Stdout)
	if err != nil {
		return
	}
	_, err = stderrLog.WriteString(outcome.Stderr)
	if err != nil {
		return
	}

	qualifiedName := fmt.Sprintf("[%s]> %s", outcome.TestSuite, outcome.TestName)
	testTitle := format.PadRight(qualifiedName, 70)
	switch outcome.Outcome {
	case model.PASSED:
		// Nothing to do
	case model.FAILED:
		failedAssertionsReport := ""
		for _, result := range outcome.AssertionResults {
			assertPrefix := result.Rule.Prefix
			assertName := result.Rule.Name
			assertOp := result.Rule.Op
			expected := result.Rule.Expected
			failedAssertionsReport += assertPrefix + assertName + assertOp + expected + " "
		}
		_, err = reportLog.WriteString(testTitle + "  => " + failedAssertionsReport)
	case model.IGNORED:
		// Nothing to do
	case model.ERRORED:
		_, err = reportLog.WriteString(testTitle + "  => not executed")
	case model.TIMEOUT:
		_, err = reportLog.WriteString(testTitle + "  => timed out")
	default:
		err = fmt.Errorf("outcome %s not supported", outcome.Outcome)
	}
	return
}

func (r FileRepo) UpdateLastTestTime(testSuite string) {
	//cfg, err := loadSuiteConfig(testSuite, r.token)
	cfg, err := r.loadSuiteConfig(testSuite)
	if err != nil {
		log.Fatal(err)
	}
	cfg.LastTestTime = utilz.OptionalOf(time.Now())
	err = r.persistSuiteConfig(*cfg)
	if err != nil {
		err = fmt.Errorf("unable to update last test time: %w", err)
		log.Fatal(err)
	}
}

func (r FileRepo) LoadSuiteOutcome(testSuite string) (outcome model.SuiteOutcome, err error) {
	var suiteCfg model.Config
	suiteCfg, err = r.GetSuiteConfig(testSuite, false)
	if err != nil {
		return
	}

	outcome.TestSuite = testSuite
	outcome.TestCount = r.TestCount(testSuite)
	outcome.PassedCount = r.PassedCount(testSuite)
	outcome.FailedCount = r.FailedCount(testSuite)
	outcome.ErroredCount = r.ErroredCount(testSuite)
	outcome.IgnoredCount = r.IgnoredCount(testSuite)
	outcome.TooMuchCount = r.TooMuchCount(testSuite)
	startTime := suiteCfg.SuiteStartTime.Get()
	endTime := suiteCfg.LastTestTime.GetOr(time.Now())
	outcome.Duration = endTime.Sub(startTime)
	failureReports, err := failureReports(testSuite, r.token, r.isolation)
	if err != nil {
		return
	}
	outcome.FailureReports = failureReports
	logger.Debug("Loaded suite outcome", "testSuite", testSuite, "outcome", outcome)
	return
}

func (r FileRepo) IncrementSuiteSeq(testSuite, name string) (n uint32) {
	suiteDir, err := testSuiteDirectoryPath(testSuite, r.token, r.isolation)
	if err != nil {
		log.Fatal(err)
	}
	n = utils.IncrementSeq(suiteDir, name)
	logger.Debug("Incrementing seq", "testSuite", testSuite, "name", name, "next", n)
	return
}

func (r FileRepo) TestCount(testSuite string) (n uint32) {
	return r.readSuiteSeq(testSuite, model.TestSequenceFilename)
}

func (r FileRepo) PassedCount(testSuite string) (n uint32) {
	return r.readSuiteSeq(testSuite, model.PassedSequenceFilename)
}

func (r FileRepo) IgnoredCount(testSuite string) (n uint32) {
	return r.readSuiteSeq(testSuite, model.IgnoredSequenceFilename)
}

func (r FileRepo) FailedCount(testSuite string) (n uint32) {
	//logger.Warn("failedCount()", "testSuite", testSuite)
	return r.readSuiteSeq(testSuite, model.FailedSequenceFilename)
}

func (r FileRepo) ErroredCount(testSuite string) (n uint32) {
	return r.readSuiteSeq(testSuite, model.ErroredSequenceFilename)
}

func (r FileRepo) TooMuchCount(testSuite string) (n uint32) {
	return r.readSuiteSeq(testSuite, model.TooMuchSequenceFilename)
}

func (r FileRepo) QueueOperation(op model.Operater) (err error) {
	err = r.dbRepo.QueueOperation(op)

	return
}

func (r FileRepo) UnqueueOperation() (op model.Operater, err error) {
	_, op, err = r.dbRepo.unqueue()

	return
}

func (r FileRepo) Done(op model.Operater) (err error) {
	err = r.dbRepo.Done(op)

	return
}

func (r FileRepo) WaitOperationDone(op model.Operater, timeout time.Duration) (exitCode int16, err error) {
	return r.dbRepo.WaitOperationDone(op, timeout)
}

func (r FileRepo) WaitEmptyQueue(testSuite string, timeout time.Duration) {
	r.dbRepo.WaitEmptyQueue(testSuite, timeout)
}

func (r FileRepo) WaitAllEmpty(timeout time.Duration) {
	r.dbRepo.WaitAllEmpty(timeout)
}

func (r FileRepo) readSuiteSeq(testSuite, name string) (n uint32) {
	//logger.Warn("readSuiteSeq()", "testSuite", testSuite, "name", name)
	suiteDir, err := testSuiteDirectoryPath(testSuite, r.token, r.isolation)
	if err != nil {
		log.Fatal(err)
	}
	n = utils.ReadSeq(suiteDir, name)
	return
}

func (r FileRepo) loadGlobalConfig() (cfg *model.Config, err error) {
	err = initWorkspaceIfNot(model.GlobalConfigTestSuiteName, r.token, r.isolation)
	if err != nil {
		return
	}

	cfg, err = r.dbRepo.suiteDao.FindGlobalConfig()
	if err != nil {
		return
	}
	if cfg != nil {
		cfg.TestSuite.Clear()
	}
	return
}

func (r FileRepo) loadSuiteConfig(testSuite string) (cfg *model.Config, err error) {
	err = initWorkspaceIfNot(testSuite, r.token, r.isolation)
	if err != nil {
		return
	}

	cfg, err = r.dbRepo.suiteDao.FindSuiteConfig(testSuite)
	return
}

func (r FileRepo) persistGlobalConfig(cfg model.Config) (err error) {
	err = initWorkspaceIfNot(model.GlobalConfigTestSuiteName, r.token, r.isolation)
	if err != nil {
		return
	}
	err = r.dbRepo.suiteDao.SaveGlobalConfig(cfg)
	return
}

func (r FileRepo) persistSuiteConfig(cfg model.Config) (err error) {
	testSuite := cfg.TestSuite.GetOr(model.DefaultTestSuiteName)
	err = initWorkspaceIfNot(testSuite, r.token, r.isolation)
	if err != nil {
		return
	}
	err = r.dbRepo.suiteDao.SaveSuiteConfig(testSuite, cfg)
	return
}

func testSuiteDirectoryPath(testSuite, token, isol string) (path string, err error) {
	var tmpDir string
	tmpDir, err = forgeWorkDirectoryPath(token, isol)
	if err != nil {
		return
	}
	suiteDir := sanitizeTestSuiteName(testSuite)
	path = filepath.Join(tmpDir, suiteDir)
	//log.Printf("testsuiteDir: %s\n", path)
	err = os.MkdirAll(path, 0700)
	return
}

func testDirectoryPath(testSuite, token, isol string, seq uint16) (testDir string, err error) {
	var tmpDir string
	tmpDir, err = testSuiteDirectoryPath(testSuite, token, isol)
	if err != nil {
		return
	}
	testDir = filepath.Join(tmpDir, "test-"+fmt.Sprintf("%06d", seq))
	return
}

func testsuiteConfigFilepath(testSuite, token, isol string) (path string, err error) {
	var testDir string
	testDir, err = testSuiteDirectoryPath(testSuite, token, isol)
	if err != nil {
		return
	}
	path = filepath.Join(testDir, model.ContextFilename)
	return
}

func clearSuiteWorkspace(testSuite, token, isol string) (err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("cannot clear suite workspace: %w", err)
		}
	}()

	var workDir string
	workDir, err = testSuiteDirectoryPath(testSuite, token, isol)
	if err != nil {
		return
	}
	logger.Info("Clearing suite workspace", "suite", testSuite, "workDir", workDir)
	err = os.RemoveAll(workDir)
	return
}

/*
	func persistSuiteConfig(token string, cfg model.Config) (err error) {
		defer func() {
			if err != nil {
				err = fmt.Errorf("cannot persist config: %w", err)
			}
		}()

		testSuite := cfg.TestSuite.Get()
		err = initWorkspaceIfNot(token, testSuite)
		if err != nil {
			return
		}
		contextFilepath, err2 := testsuiteConfigFilepath(testSuite, token)
		if err2 != nil {
			return err2
		}
		content, err2 := yaml.Marshal(&cfg)
		if err2 != nil {
			return err2
		}
		logger.Debug("Persisting config", "suite", testSuite, "file", contextFilepath, "content", content)
		err2 = os.WriteFile(contextFilepath, content, 0600)
		if err2 != nil {
			return err2
		}
		//logger.Warn("Persisted config", "suite", testSuite, "file", contextFilepath, "content", content)
		return nil
		//})
		//return
	}

	func readConfig(name, token string) (config model.Config, err error) {
		defer func() {
			if err != nil {
				err = fmt.Errorf("cannot read config: %w", err)
			}
		}()

		err = initWorkspaceIfNot(token, name)
		if err != nil {
			return
		}
		var contextFilepath string
		contextFilepath, err = testsuiteConfigFilepath(name, token)
		if err != nil {
			return
		}
		var content []byte
		content, err = os.ReadFile(contextFilepath)
		if err != nil {
			return
		}
		err = yaml.Unmarshal(content, &config)
		//log.Printf("Read context from %s\n", contextFilepath)
		return
	}

	func loadGlobalConfig(token string) (config model.Config, err error) {
		defer func() {
			if err != nil {
				err = fmt.Errorf("cannot load global config: %w", err)
			}
		}()

		config = model.NewGlobalDefaultConfig()
		var loaded model.Config
		loaded, err = readConfig(model.GlobalConfigTestSuiteName, token)
		if err != nil {
			return
		}
		loaded.TestSuite.Clear()
		config.Merge(loaded)
		return
	}

	func loadSuiteConfig(testSuite, token string) (config model.Config, err error) {
		defer func() {
			if err != nil {
				err = fmt.Errorf("cannot load suite config: %w", err)
			}
		}()

		var globalCfg, suiteCfg model.Config
		globalCfg, err = loadGlobalConfig(token)
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			return
		}
		suiteCfg, err = readConfig(testSuite, token)
		if err != nil {
			return
		}
		logger.Debug("Loaded context", "global", globalCfg, "suite", suiteCfg)
		config = globalCfg
		config.Merge(suiteCfg)
		//SetRulePrefix(config.Prefix)
		logger.Debug("Merges context", "merged", config)
		return
	}
*/
func cmdLogFiles(testSuite, token, isol string, seq uint16) (stdoutFile, stderrFile, reportFile *os.File, err error) {
	var testDir string
	testDir, err = testDirectoryPath(testSuite, token, isol, seq)
	if err != nil {
		return
	}
	stdoutFilepath := filepath.Join(testDir, model.StdoutFilename)
	stderrFilepath := filepath.Join(testDir, model.StderrFilename)
	reportFilepath := filepath.Join(testDir, model.ReportFilename)

	err = os.MkdirAll(testDir, 0700)
	if err != nil {
		err = fmt.Errorf("cannot create work dir %s : %w", testDir, err)
		return
	}
	stdoutFile, err = os.OpenFile(stdoutFilepath, os.O_APPEND|os.O_RDWR|os.O_CREATE, 0600)
	if err != nil {
		err = fmt.Errorf("cannot open file %s : %w", stdoutFilepath, err)
		return
	}
	stderrFile, err = os.OpenFile(stderrFilepath, os.O_APPEND|os.O_RDWR|os.O_CREATE, 0600)
	if err != nil {
		err = fmt.Errorf("cannot open file %s : %w", stderrFilepath, err)
		return
	}
	reportFile, err = os.OpenFile(reportFilepath, os.O_APPEND|os.O_RDWR|os.O_CREATE, 0600)
	if err != nil {
		err = fmt.Errorf("cannot open file %s : %w", reportFilepath, err)
		return
	}
	return
}

func failureReports(testSuite, token, isol string) (reports []string, err error) {
	var tmpDir string
	tmpDir, err = testSuiteDirectoryPath(testSuite, token, isol)
	if err != nil {
		return
	}
	err = filepath.Walk(tmpDir, func(path string, info fs.FileInfo, err error) error {
		if model.ReportFilename == info.Name() {
			content, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			reports = append(reports, string(content))
		}
		return nil
	})
	return
}
