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
	"mby.fr/utils/trust"
	"mby.fr/utils/utilz"
)

const (
	tempDirPrefix = "cmdtest"
)

var (
	logger                        = slog.New(slog.NewTextHandler(os.Stderr, nil))
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

func (r FileRepo) SaveGlobalConfig(cfg model.Config) (err error) {
	cfg.TestSuite = utilz.OptionnalOf(model.GlobalConfigTestSuiteName)
	return persistSuiteConfig(r.token, cfg)
}

func (r FileRepo) LoadGlobalConfig(testSuite string) (model.Config, error) {
	return loadGlobalConfig(r.token)
}

func (r FileRepo) SaveSuiteConfig(cfg model.Config) (err error) {
	return persistSuiteConfig(r.token, cfg)
}

func (r FileRepo) LoadSuiteConfig(testSuite string) (model.Config, error) {
	return loadSuiteConfig(r.token, testSuite)
}

func (r FileRepo) SaveTestOutcome(outcome model.TestOutcome) (err error) {
	initWorkspaceIfNot(r.token, outcome.Context.Config.TestName.Get())
	// TODO
	return
}

func (r FileRepo) LoadSuiteOutcome(testSuite string) (outcome model.SuiteOutcome, err error) {
	initWorkspaceIfNot(r.token, testSuite)
	// TODO
	return
}

func (r FileRepo) ListTestSuites(token string) (suites []string, err error) {
	var tmpDir string
	tmpDir, err = forgeWorkDirectoryPath(token)
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

func (r FileRepo) TestCount(ctx model.Context) (n int) {
	return readSeq(ctx, model.TestSequenceFilename)
}

func (r FileRepo) PassedCount(ctx model.Context) (n int) {
	return readSeq(ctx, model.PassedSequenceFilename)
}

func (r FileRepo) IgnoredCount(ctx model.Context) (n int) {
	return readSeq(ctx, model.IgnoredSequenceFilename)
}

func (r FileRepo) FailedCount(ctx model.Context) (n int) {
	return readSeq(ctx, model.FailedSequenceFilename)
}

func (r FileRepo) ErroredCount(ctx model.Context) (n int) {
	return readSeq(ctx, model.ErroredSequenceFilename)
}

func (r FileRepo) IncrementTestCount(ctx model.Context) (n int) {
	return incrementSeq(ctx, model.TestSequenceFilename)
}

func (r FileRepo) IncrementPassedCount(ctx model.Context) (n int) {
	return incrementSeq(ctx, model.PassedSequenceFilename)
}

func (r FileRepo) IncrementIgnoredCount(ctx model.Context) (n int) {
	return incrementSeq(ctx, model.IgnoredSequenceFilename)
}

func (r FileRepo) IncrementFailedCount(ctx model.Context) (n int) {
	return incrementSeq(ctx, model.FailedSequenceFilename)
}

func (r FileRepo) IncrementErroredCount(ctx model.Context) (n int) {
	return incrementSeq(ctx, model.ErroredSequenceFilename)
}

func (r FileRepo) SuiteError(ctx model.Context, v ...any) error {
	return r.SuiteErrorf(ctx, "%s", fmt.Sprint(v...))
}

func (r FileRepo) SuiteErrorf(ctx model.Context, format string, v ...any) error {
	r.IncrementErroredCount(ctx)
	return fmt.Errorf(format, v...)
}

func (r FileRepo) Fatal(ctx model.Context, v ...any) {
	r.IncrementErroredCount(ctx)
	log.Fatal(v...)
}

func (r FileRepo) Fatalf(ctx model.Context, format string, v ...any) {
	r.Fatal(ctx, fmt.Sprintf(format, v...))
}

func (r FileRepo) NoErrorOrFatal(ctx model.Context, err error) {
	if err != nil {
		ctx.Config.TestSuite.IfPresent(func(testSuite string) error {
			updateLastTestTime(testSuite, r.token)
			r.Fatal(ctx, err)
			return nil
		})
		log.Fatal(err)
	}
}

func forgeContextualToken() (string, error) {
	// If no token supplied use Workspace dir + ppid to forge tmp directory path
	workDirPath, err := os.Getwd()
	if err != nil {
		//log.Fatalf("cannot find workspace dir: %s", err)
		return "", fmt.Errorf("cannot find workspace dir: %w", err)
	}
	ppid := os.Getppid()
	ppidStr := fmt.Sprintf("%d", ppid)
	ppidStartTime, err := utils.GetProcessStartTime(ppid)
	if err != nil {
		//log.Fatalf("cannot find parent process start time: %s", err)
		return "", fmt.Errorf("cannot find parent process start time: %w", err)
	}
	ppidStartTimeStr := fmt.Sprintf("%d", ppidStartTime)
	token, err := trust.SignStrings(workDirPath, "--", ppidStr, "--", ppidStartTimeStr)
	if err != nil {
		err = fmt.Errorf("cannot hash workspace dir: %w", err)
	}
	//log.Printf("contextual token: %s base on workDirPath: %s and ppid: %s\n", token, workDirPath, ppid)
	return token, err
}

func forgeWorkDirectoryPath(token string) (tempDirPath string, err error) {
	if token == "" {
		token, err = forgeContextualToken()
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
	content, err2 := yaml.Marshal(cfg)
	if err2 != nil {
		return err2
	}
	logger.Debug("Persisting context", "context", content, "file", contextFilepath)
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
	err = config.Merge(suiteCfg)
	if err != nil {
		return
	}
	//SetRulePrefix(config.Prefix)
	logger.Debug("Merges context", "merged", config)
	return
}

func loadGlobalConfig(token string) (config model.Config, err error) {
	config, err = readConfig(model.GlobalConfigTestSuiteName, token)
	config.TestSuite = utilz.OptionnalOf("")
	return
}

func updateLastTestTime(testSuite, token string) {
	cfg, err := loadSuiteConfig(testSuite, token)
	if err != nil {
		log.Fatal(err)
	}
	cfg.LastTestTime = utilz.OptionnalOf(time.Now())
	err = persistSuiteConfig(token, cfg)
	if err != nil {
		log.Fatal(err)
	}
}

func readSeq(ctx model.Context, name string) (n int) {
	suiteDir, err := testSuiteDirectoryPath(ctx.Config.TestSuite.Get(), ctx.Token)
	if err != nil {
		log.Fatal(err)
	}
	n = utils.ReadSeq(suiteDir, name)
	return
}

func incrementSeq(ctx model.Context, name string) (n int) {
	suiteDir, err := testSuiteDirectoryPath(ctx.Config.TestSuite.Get(), ctx.Token)
	if err != nil {
		log.Fatal(err)
	}
	n = utils.IncrementSeq(suiteDir, name)
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
