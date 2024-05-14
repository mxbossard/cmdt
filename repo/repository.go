package repo

import (
	"errors"
	"fmt"
	"log"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"

	"mby.fr/cmdtest/model"
	"mby.fr/cmdtest/utils"
)

const (
	TEMP_DIR_PREFIX = "cmdtest"
)

var (
	logger                        = slog.New(slog.NewTextHandler(os.Stderr, model.DefaultLoggerOpts))
	testSuiteNameSanitizerPattern = regexp.MustCompile("[^a-zA-Z0-9]")
)

func New(token, isolation string) (repo FileRepo) {
	repo.token = token
	repo.isolation = isolation
	logger.Warn("new repo", "token", token, "isolation", isolation)
	path, err := forgeWorkDirectoryPath(token, isolation)
	if err != nil {
		log.Fatal(err)
	}

	repo.dbRepo, err = newDbRepo(path)
	if err != nil {
		log.Fatal(err)
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
