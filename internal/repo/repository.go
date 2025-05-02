package repo

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"time"

	"cmdt/internal/model"
	"cmdt/internal/utils"

	"github.com/mxbossard/utilz/errorz"
	"github.com/mxbossard/utilz/zlog"
)

const (
	TEMP_DIR_PREFIX = "cmdtest"
)

var (
	logger                        = zlog.New() //slog.New(slog.NewTextHandler(os.Stderr, model.DefaultLoggerOpts))
	testSuiteNameSanitizerPattern = regexp.MustCompile("[^a-zA-Z0-9]")
)

type Repo interface {
	Init() error

	Close() error

	BackingFilepath() string

	MockDirectoryPath(testSuite string, testId uint16) (mockDir string, err error)

	SaveGlobalConfig(cfg model.Config) (err error)

	GetGlobalConfig() (cfg model.Config, err error)

	InitSuite(cfg model.Config) (err error)

	SaveSuiteConfig(cfg model.Config) (err error)

	GetSuiteConfig(testSuite string, initless bool) (cfg model.Config, err error)

	ClearSuite(testSuite string) (err error)

	ListReportableSuites() (suites []string, err error)
	ListReportableSuitesByMode(asyncMode, all bool) (suites []string, err error)
	ListAllSuites() (suites []string, err error)
	ListSyncSuites() (suites []string, err error)
	ListAsyncSuites() (suites []string, err error)
	ListReportedAsyncSuites() (suites []string, err error)

	SaveTestOutcome(outcome model.TestOutcome) (err error)

	SaveSuiteOutcome(outcome model.SuiteOutcome) (err error)

	UpdateLastTestTime(testSuite string)

	LoadSuiteOutcome(testSuite string) (outcome model.SuiteOutcome, err error)

	MarkSuiteReported(suite string) (err error)
	MarkSuitesReported() (err error)

	SuiteStatus(suite string) (exists, reported, kept bool, err error)

	IncrementSuiteSeq(testSuite, name string) (n uint16)

	NotReportedTestCount() (n uint16)
	TestCount(testSuite string) (n uint16)
	ToReportTestCountByMode(asyncMode, all bool) (n uint16)
	ToReportTestCountBySuiteAndMode(testSuite string, asyncMode, all bool) (n uint16)

	PassedCount(testSuite string) (n uint16)

	IgnoredCount(testSuite string) (n uint16)

	FailedCount(testSuite string) (n uint16)

	ErroredCount(testSuite string) (n uint16)

	TooMuchCount(testSuite string) (n uint16)

	IgnoredSuiteCount(reportAll bool) (n uint16)

	QueueOperation(op model.Operater) (err error)

	UnqueueOperation() (op model.Operater, err error)

	Done(op model.Operater) (err error)

	WaitOperationDone(op model.Operater, timeout time.Duration) (exitCode int16, opErr, err error)

	WaitEmptyQueue(testSuite string, timeout time.Duration) (err error)

	WaitAllEmpty(timeout time.Duration) (err error)
}

func New(token, isolation string) (repo dbRepo) {
	p := logger.PerfTimer("token", token, "isolation", isolation)
	defer p.End()

	path, err := forgeWorkDirectoryPath(token, isolation)
	if err != nil {
		errorz.Fatal(err)
	}

	repo, err = newDbRepo(path, isolation, token)
	if err != nil {
		errorz.Fatal(err)
	}

	//fmt.Printf("\n<<>> Built repo ; token: %s ; isol: %s\n", token, isolation)
	return
}

func NewFile(token, isolation string) (repo FileRepo) {
	p := logger.PerfTimer("token", token, "isolation", isolation)
	defer p.End()

	repo.token = token
	repo.isolation = isolation
	path, err := forgeWorkDirectoryPath(token, isolation)
	if err != nil {
		errorz.Fatal(err)
	}

	repo.dbRepo, err = newDbRepo(path, isolation, token)
	if err != nil {
		errorz.Fatal(err)
	}

	return
}

func forgeWorkDirectoryPath(token, isol string) (tempDirPath string, err error) {
	token, err = utils.ForgeContextualToken(token)
	if err != nil {
		return
	}
	isolatedToken := utils.IsolatedToken(token, isol)
	tempDirName := fmt.Sprintf("%s-%s", TEMP_DIR_PREFIX, isolatedToken)
	tempDirPath = filepath.Join(os.TempDir(), tempDirName)
	err = os.MkdirAll(tempDirPath, 0700)
	//logger.Warn("forgeWorkDirectoryPath", "workDir", tempDirPath)
	return
}

func ClearWorkDirectory(token, isol string) (err error) {
	dir, err := forgeWorkDirectoryPath(token, isol)
	logger.Debug("ClearWorkDirectory()", "dir", dir, "token", token, "isolation", isol)
	if err != nil {
		return
	}
	if _, err := os.Stat(dir); err == nil {
		err = os.RemoveAll(dir)
		if err != nil {
			return err
		}
	}
	return
}

func initWorkspaceIfNot(testSuite, token, isol string) (err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("cannot init workspace: %w", err)
		}
	}()

	// init the tmp directory
	var tmpDir string
	tmpDir, err = testSuiteDirectoryPath(testSuite, token, isol)
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

func sanitizeTestSuiteName(s string) string {
	return testSuiteNameSanitizerPattern.ReplaceAllString(s, "_")
}
