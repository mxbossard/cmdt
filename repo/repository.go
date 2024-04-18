package repo

import (
	"errors"
	"fmt"
	"io/fs"
	"log"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"gopkg.in/yaml.v2"
	"mby.fr/cmdtest/model"
	"mby.fr/cmdtest/utils"
	"mby.fr/utils/format"
	"mby.fr/utils/utilz"
)

const (
	TEMP_DIR_PREFIX = "cmdtest"
)

var (
	logger                        = slog.New(slog.NewTextHandler(os.Stderr, model.DefaultLoggerOpts))
	testSuiteNameSanitizerPattern = regexp.MustCompile("[^a-zA-Z0-9]")
)

func New(token string) (repo FileRepo) {
	repo.token = token
	path, err := forgeWorkDirectoryPath(token)
	if err != nil {
		log.Fatal(err)
	}
	//queuesBackingFilepath := filepath.Join(path, "operationQueues.yaml")
	//queuesRepo, err := LoadOperationQueueRepo(queuesBackingFilepath)
	//if err != nil {
	//	log.Fatal(err)
	//}
	//repo.queuesRepo = queuesRepo

	//stateBackingFilepath := filepath.Join(path, "state.yaml")
	//repo.State = FileState{backingFilepath: stateBackingFilepath}

	repo.dbRepo, err = newDbRepo(path)
	if err != nil {
		log.Fatal(err)
	}

	return
}

type FileRepo struct {
	token string
	//queuesRepo OperationQueueRepo

	//State  FileState
	dbRepo dbRepo
}

func (r FileRepo) BackingFilepath() string {
	path, err := forgeWorkDirectoryPath(r.token)
	if err != nil {
		log.Fatal(err)
	}
	return path
}

func (r FileRepo) MockDirectoryPath(testSuite string, testId uint32) (mockDir string, err error) {
	var path string
	path, err = testSuiteDirectoryPath(testSuite, r.token)
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

func (r FileRepo) InitSuite(cfg model.Config) (err error) {
	err = r.ClearTestSuite(cfg.TestSuite.Get())
	if err != nil {
		return
	}
	err = persistSuiteConfig(r.token, cfg)
	if err != nil {
		err = fmt.Errorf("unable to init suite: %w", err)
	}
	return
}

func (r FileRepo) SaveGlobalConfig(cfg model.Config) (err error) {
	cfg.TestSuite.Set(model.GlobalConfigTestSuiteName)
	err = persistSuiteConfig(r.token, cfg)
	if err != nil {
		err = fmt.Errorf("unable to save global config: %w", err)
	}
	return
}

func (r FileRepo) LoadGlobalConfig() (cfg model.Config, err error) {
	cfg, err = loadGlobalConfig(r.token)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			// global config does not exists yet
			// create a new default one
			cfg = model.NewGlobalDefaultConfig()
			cfg.Token.Set(r.token)
			cfg.GlobalStartTime.Set(time.Now())
			err = r.SaveGlobalConfig(cfg)
		}
	}
	return
}

func (r FileRepo) SaveSuiteConfig(cfg model.Config) (err error) {
	err = persistSuiteConfig(r.token, cfg)
	if err != nil {
		err = fmt.Errorf("unable to save suite config: %w", err)
	}
	return
}

func (r FileRepo) LoadSuiteConfig(testSuite string, initless bool) (cfg model.Config, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("cannot load suite config: %w", err)
		}
	}()

	cfg, err = loadSuiteConfig(testSuite, r.token)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			// suite config does not exists yet
			// create a new default one
			if initless {
				//logger.Warn("Saving new initless config", "testSuite", testSuite)
				cfg = model.NewInitlessSuiteDefaultConfig()
			} else {
				//logger.Warn("Saving new inited config", "testSuite", testSuite)
				cfg = model.NewSuiteDefaultConfig()
			}
			cfg.TestSuite.Set(testSuite)
			cfg.SuiteStartTime.Set(time.Now())
			err = r.SaveSuiteConfig(cfg)
		}
	}
	return
}

func (r FileRepo) readSuiteSeq(testSuite, name string) (n uint32) {
	//logger.Warn("readSuiteSeq()", "testSuite", testSuite, "name", name)
	suiteDir, err := testSuiteDirectoryPath(testSuite, r.token)
	if err != nil {
		log.Fatal(err)
	}
	n = utils.ReadSeq(suiteDir, name)
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

func (r FileRepo) IncrementSuiteSeq(testSuite, name string) (n uint32) {
	suiteDir, err := testSuiteDirectoryPath(testSuite, r.token)
	if err != nil {
		log.Fatal(err)
	}
	n = utils.IncrementSeq(suiteDir, name)
	logger.Debug("Incrementing seq", "testSuite", testSuite, "name", name, "next", n)
	return
}

func (r FileRepo) SaveTestOutcome(outcome model.TestOutcome) (err error) {
	var stdoutLog, stderrLog, reportLog *os.File
	stdoutLog, stderrLog, reportLog, err = cmdLogFiles(outcome.TestSuite, r.token, outcome.Seq)
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
			assertPrefix := result.Assertion.Prefix
			assertName := result.Assertion.Name
			assertOp := result.Assertion.Op
			expected := result.Assertion.Expected
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

func (r FileRepo) LoadSuiteOutcome(testSuite string) (outcome model.SuiteOutcome, err error) {
	var suiteCfg model.Config
	suiteCfg, err = r.LoadSuiteConfig(testSuite, false)
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
	failureReports, err := failureReports(testSuite, r.token)
	if err != nil {
		return
	}
	outcome.FailureReports = failureReports
	logger.Debug("Loaded suite outcome", "testSuite", testSuite, "outcome", outcome)
	return
}

func (r FileRepo) UpdateLastTestTime(testSuite string) {
	cfg, err := loadSuiteConfig(testSuite, r.token)
	if err != nil {
		log.Fatal(err)
	}
	cfg.LastTestTime = utilz.OptionalOf(time.Now())
	err = persistSuiteConfig(r.token, cfg)
	if err != nil {
		err = fmt.Errorf("unable to update last test time: %w", err)
		log.Fatal(err)
	}
}

func (r FileRepo) ClearTestSuite(testSuite string) (err error) {
	err = clearSuiteWorkspace(r.token, testSuite)
	return
}

func (r FileRepo) ListTestSuites() (suites []string, err error) {
	var tmpDir string
	tmpDir, err = forgeWorkDirectoryPath(r.token)
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

func (r FileRepo) QueueOperation(op Operater) (err error) {
	// r.queuesRepo.Queue(op)
	// err = r.queuesRepo.Persist()

	err = r.dbRepo.Queue(op)

	return
}

func (r FileRepo) UnqueueOperation() (op Operater, err error) {
	// var ok bool
	// ok, op = r.queuesRepo.Unqueue()
	// if ok {
	// 	err = r.queuesRepo.Persist()
	// } else {
	// 	op = nil
	// }

	_, op, err = r.dbRepo.Unqueue()

	return
}

func (r FileRepo) Done(op Operater) (err error) {
	// r.queuesRepo.Unblock(op)
	// err = r.queuesRepo.Persist()

	err = r.dbRepo.Done(op)

	return
}

func (r FileRepo) WaitOperationDone(op Operater, timeout time.Duration) (exitCode int16, err error) {
	return r.dbRepo.WaitOperaterDone(op, timeout)
}

func (r FileRepo) WaitEmptyQueue(testSuite string, timeout time.Duration) {
	// r.queuesRepo.WaitEmptyQueue(testSuite, timeout)

	r.dbRepo.WaitEmptyQueue(testSuite, timeout)
}

func (r FileRepo) WaitAllEmpty(timeout time.Duration) {
	// r.queuesRepo.WaitAllEmpty(timeout)
}

func forgeWorkDirectoryPath(token string) (tempDirPath string, err error) {
	if token == "" {
		token, err = utils.ForgeContextualToken()
	}
	if err != nil {
		return
	}
	tempDirName := fmt.Sprintf("%s-%s", TEMP_DIR_PREFIX, token)
	tempDirPath = filepath.Join(os.TempDir(), tempDirName)
	err = os.MkdirAll(tempDirPath, 0700)
	//logger.Warn("forgeWorkDirectoryPath", "workDir", tempDirPath)
	return
}

func testSuiteDirectoryPath(testSuite, token string) (path string, err error) {
	var tmpDir string
	tmpDir, err = forgeWorkDirectoryPath(token)
	if err != nil {
		return
	}
	suiteDir := sanitizeTestSuiteName(testSuite)
	path = filepath.Join(tmpDir, suiteDir)
	//log.Printf("testsuiteDir: %s\n", path)
	err = os.MkdirAll(path, 0700)
	return
}

func testDirectoryPath(testSuite, token string, seq uint16) (testDir string, err error) {
	var tmpDir string
	tmpDir, err = testSuiteDirectoryPath(testSuite, token)
	if err != nil {
		return
	}
	testDir = filepath.Join(tmpDir, "test-"+fmt.Sprintf("%06d", seq))
	return
}

func testsuiteConfigFilepath(testSuite, token string) (path string, err error) {
	var testDir string
	testDir, err = testSuiteDirectoryPath(testSuite, token)
	if err != nil {
		return
	}
	path = filepath.Join(testDir, model.ContextFilename)
	return
}

func initWorkspaceIfNot(token, testSuite string) (err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("cannot init workspace: %w", err)
		}
	}()

	// init the tmp directory
	var tmpDir string
	tmpDir, err = testSuiteDirectoryPath(testSuite, token)
	if err != nil {
		return
	}
	_, err = os.Stat(tmpDir)
	if err == nil {
		// Workspace already initialized
		return
	} else if !errors.Is(err, os.ErrNotExist) {
		return
	}

	err = os.MkdirAll(tmpDir, 0700)
	if err != nil {
		err = fmt.Errorf("unable to create temp dir: %s ! Error: %w", tmpDir, err)
		return
	}

	return
}

func clearSuiteWorkspace(token, testSuite string) (err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("cannot clear suite workspace: %w", err)
		}
	}()

	var workDir string
	workDir, err = testSuiteDirectoryPath(testSuite, token)
	if err != nil {
		return
	}
	logger.Info("Clearing suite workspace", "suite", testSuite, "workDir", workDir)
	err = os.RemoveAll(workDir)
	return
}

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

func cmdLogFiles(testSuite, token string, seq uint16) (stdoutFile, stderrFile, reportFile *os.File, err error) {
	var testDir string
	testDir, err = testDirectoryPath(testSuite, token, seq)
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

func failureReports(testSuite, token string) (reports []string, err error) {
	var tmpDir string
	tmpDir, err = testSuiteDirectoryPath(testSuite, token)
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

func sanitizeTestSuiteName(s string) string {
	return testSuiteNameSanitizerPattern.ReplaceAllString(s, "_")
}
