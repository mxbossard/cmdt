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
	"time"

	"gopkg.in/yaml.v2"
	"mby.fr/cmdtest/model"
	"mby.fr/cmdtest/utils"
	"mby.fr/utils/format"
	"mby.fr/utils/utilz"
)

const (
	tempDirPrefix = "cmdtest"
)

var (
	logger                        = slog.New(slog.NewTextHandler(os.Stderr, model.DefaultLoggerOpts))
	testSuiteNameSanitizerPattern = regexp.MustCompile("[^a-zA-Z0-9]")
)

func New(token string) FileRepo {
	return FileRepo{token: token}
}

type FileRepo struct {
	token string
}

func (r FileRepo) BackingFilepath() string {
	path, err := forgeWorkDirectoryPath(r.token)
	if err != nil {
		log.Fatal(err)
	}
	return path
}

func (r FileRepo) InitSuite(cfg model.Config) (err error) {
	err = r.ClearTestSuite(cfg.TestSuite.Get())
	if err != nil {
		return
	}
	err = persistSuiteConfig(r.token, cfg)
	return
}

func (r FileRepo) SaveGlobalConfig(cfg model.Config) (err error) {
	cfg.TestSuite.Set(model.GlobalConfigTestSuiteName)
	return persistSuiteConfig(r.token, cfg)
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
	return persistSuiteConfig(r.token, cfg)
}

func (r FileRepo) LoadSuiteConfig(testSuite string) (cfg model.Config, err error) {
	cfg, err = loadSuiteConfig(testSuite, r.token)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			// suite config does not exists yet
			// create a new default one
			cfg = model.NewSuiteDefaultConfig()
			cfg.TestSuite.Set(testSuite)
			cfg.SuiteStartTime.Set(time.Now())
			err = r.SaveSuiteConfig(cfg)
		}
	}
	return
}

func (r FileRepo) readSuiteSeq(testSuite, name string) (n int) {
	suiteDir, err := testSuiteDirectoryPath(testSuite, r.token)
	if err != nil {
		log.Fatal(err)
	}
	n = utils.ReadSeq(suiteDir, name)
	return
}

func (r FileRepo) TestCount(testSuite string) (n int) {
	return r.readSuiteSeq(testSuite, model.TestSequenceFilename)
}

func (r FileRepo) PassedCount(testSuite string) (n int) {
	return r.readSuiteSeq(testSuite, model.PassedSequenceFilename)
}

func (r FileRepo) IgnoredCount(testSuite string) (n int) {
	return r.readSuiteSeq(testSuite, model.IgnoredSequenceFilename)
}

func (r FileRepo) FailedCount(testSuite string) (n int) {
	return r.readSuiteSeq(testSuite, model.FailedSequenceFilename)
}

func (r FileRepo) ErroredCount(testSuite string) (n int) {
	return r.readSuiteSeq(testSuite, model.ErroredSequenceFilename)
}

func (r FileRepo) IncrementSuiteSeq(testSuite, name string) (n int) {
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

	testTitle := format.PadRight(outcome.TestQualifiedName, 60)
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
		_, err = reportLog.WriteString(testTitle + "  =>  not executed")
	case model.TIMEOUT:
		_, err = reportLog.WriteString(testTitle + "  =>  timed out")
	default:
		err = fmt.Errorf("outcome %s not supported", outcome.Outcome)
	}
	return
}

func (r FileRepo) LoadSuiteOutcome(testSuite string) (outcome model.SuiteOutcome, err error) {
	var suiteCfg model.Config
	suiteCfg, err = r.LoadSuiteConfig(testSuite)
	if err != nil {
		return
	}

	outcome.TestSuite = testSuite
	outcome.TestCount = r.TestCount(testSuite)
	outcome.PassedCount = r.PassedCount(testSuite)
	outcome.FailedCount = r.FailedCount(testSuite)
	outcome.ErroredCount = r.ErroredCount(testSuite)
	outcome.IgnoredCount = r.IgnoredCount(testSuite)
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
	// Add success
	for _, m := range matches {
		testSuite := filepath.Base(m)
		if testSuite != model.GlobalConfigTestSuiteName {
			failedCount := utils.ReadSeq(tmpDir, testSuite, model.FailedSequenceFilename)
			errorCount := utils.ReadSeq(tmpDir, testSuite, model.ErroredSequenceFilename)
			if failedCount == 0 && errorCount == 0 {
				suites = append(suites, testSuite)
			}
		}
	}
	// Add failures
	for _, m := range matches {
		testSuite := filepath.Base(m)
		if testSuite != model.GlobalConfigTestSuiteName {
			failedCount := utils.ReadSeq(tmpDir, testSuite, model.FailedSequenceFilename)
			errorCount := utils.ReadSeq(tmpDir, testSuite, model.ErroredSequenceFilename)
			if failedCount > 0 && errorCount == 0 {
				suites = append(suites, testSuite)
			}
		}
	}
	// Add errors
	for _, m := range matches {
		testSuite := filepath.Base(m)
		if testSuite != model.GlobalConfigTestSuiteName {
			errorCount := utils.ReadSeq(tmpDir, testSuite, model.ErroredSequenceFilename)
			if errorCount > 0 {
				suites = append(suites, testSuite)
			}
		}
	}
	return
}

func forgeWorkDirectoryPath(token string) (tempDirPath string, err error) {
	if token == "" {
		token, err = utils.ForgeContextualToken()
	}
	if err != nil {
		return
	}
	tempDirName := fmt.Sprintf("%s-%s", tempDirPrefix, token)
	tempDirPath = filepath.Join(os.TempDir(), tempDirName)
	err = os.MkdirAll(tempDirPath, 0700)
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

func testDirectoryPath(testSuite, token string, seq int) (testDir string, err error) {
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

/*
func persistSuiteContext0(testSuite, token string, config model.Context) (err error) {
	var contextFilepath string
	contextFilepath, err = testsuiteConfigFilepath(testSuite, token)
	if err != nil {
		return
	}
	content, err := yaml.Marshal(config)
	if err != nil {
		return
	}
	logger.Debug("Persisting context", "context", content, "file", contextFilepath)
	err = os.WriteFile(contextFilepath, content, 0600)
	if err != nil {
		err = fmt.Errorf("cannot persist context: %w", err)
		return
	}
	return
}

func persistSuiteContext1(config model.Context) (err error) {
	testSuite := config.TestSuite
	token := config.Token
	var contextFilepath string
	contextFilepath, err = testsuiteConfigFilepath(testSuite, token)
	if err != nil {
		return
	}
	content, err := yaml.Marshal(config)
	if err != nil {
		return
	}
	logger.Debug("Persisting context", "context", content, "file", contextFilepath)
	err = os.WriteFile(contextFilepath, content, 0600)
	if err != nil {
		err = fmt.Errorf("cannot persist context: %w", err)
		return
	}
	return
}
*/

func clearSuiteWorkspace(token, testSuite string) (err error) {
	var workDir string
	workDir, err = testSuiteDirectoryPath(testSuite, token)
	if err != nil {
		return
	}
	logger.Debug("Clearing suite workspace", "suite", testSuite, "workDir", workDir)
	err = os.RemoveAll(workDir)
	return
}

func persistSuiteConfig(token string, cfg model.Config) (err error) {
	testSuite := cfg.TestSuite.Get()
	err = initWorkspaceIfNot(token, testSuite)
	if err != nil {
		return
	}
	//err = cfg.TestSuite.IfPresent(func(testSuite string) error {
	contextFilepath, err2 := testsuiteConfigFilepath(testSuite, token)
	if err2 != nil {
		return err2
	}
	content, err2 := yaml.Marshal(&cfg)
	if err2 != nil {
		return err2
	}
	logger.Debug("Persisting config", "suite", testSuite, "file", contextFilepath, "cfg", cfg)
	err2 = os.WriteFile(contextFilepath, content, 0600)
	if err2 != nil {
		err2 = fmt.Errorf("cannot persist context: %w", err2)
		return err2
	}
	return nil
	//})
	//return
}

func readConfig(name, token string) (config model.Config, err error) {
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

func cmdLogFiles(testSuite, token string, seq int) (stdoutFile, stderrFile, reportFile *os.File, err error) {
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
