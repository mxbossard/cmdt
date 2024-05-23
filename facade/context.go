package facade

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"mby.fr/cmdtest/mock"
	"mby.fr/cmdtest/model"
	"mby.fr/cmdtest/repo"
	"mby.fr/cmdtest/utils"
	"mby.fr/utils/cmdz"
	"mby.fr/utils/utilz"
)

var logger = slog.New(slog.NewTextHandler(os.Stderr, model.DefaultLoggerOpts))

var (
// g *GlobalContext
// s *SuiteContext
// t *TestContext
)

func NewGlobalContext(token, isolation string, inputCfg model.Config) GlobalContext {
	/*
		if g != nil {
			return *g
		}
	*/
	var err error
	token, err = utils.ForgeContextualToken(token)
	if err != nil {
		log.Fatal(err)
	}

	repo := repo.New(token, isolation)

	cfg, err := repo.GetGlobalConfig()
	if err != nil {
		log.Fatal(err)
	}
	cfg.Merge(inputCfg)

	c := GlobalContext{
		Token:     token,
		Isolation: isolation,
		Repo:      repo,
		Config:    cfg,
	}
	//g = &c
	return c
}

func NewSuiteContext(token, isolation, testSuite string, initless bool, action model.Action, inputCfg model.Config) SuiteContext {
	/*
		if s != nil {
			return *s
		}
	*/
	globalCtx := NewGlobalContext(token, isolation, model.Config{})
	suiteCfg, err := globalCtx.Repo.GetSuiteConfig(testSuite, initless)
	if err != nil {
		log.Fatal(err)
	}

	mergedCfg := globalCtx.Config
	logger.Debug("before suite merge", "cfg", mergedCfg)
	mergedCfg.Merge(suiteCfg)
	logger.Debug("after suite merge", "suiteCfg", suiteCfg, "mergedCfg", mergedCfg)
	mergedCfg.Merge(inputCfg)
	logger.Debug("merged suite context", "testSuite", testSuite, "inputCfg", inputCfg, "mergedCfg", mergedCfg)
	globalCtx.Config = mergedCfg

	suiteCtx := SuiteContext{
		GlobalContext: globalCtx,
		Action:        action,
	}
	//s = &suiteCtx
	return suiteCtx
}

func NewTestContext(token, isolation, testSuite string, seq uint16, inputCfg model.Config, ppid uint32) TestContext {
	/*
		if t != nil {
			return *t
		}
	*/
	suiteCtx := NewSuiteContext(token, isolation, testSuite, true, model.TestAction, model.Config{})
	mergedCfg := suiteCtx.Config
	mergedCfg.Merge(inputCfg)

	testCtx := TestContext{
		SuiteContext: suiteCtx,
	}
	testCtx.Config = mergedCfg
	testCtx.Suite = suiteCtx
	testCtx.Seq = seq
	//logger.Warn("NewTestContext", "testCtx", testCtx)
	err := testCtx.initExecuter(ppid)
	if err != nil {
		testCtx.NoErrorOrFatal(err)
	}

	if ok, ctId := utils.ReadEnvValue(model.EnvContainerIdKey); ok {
		testCtx.ContainerId = ctId
		_, testCtx.ContainerScope = utils.ReadEnvValue(model.EnvContainerScopeKey)
		_, testCtx.ContainerImage = utils.ReadEnvValue(model.EnvContainerImageKey)
	}

	//t = &testCtx
	return testCtx
}

func NewTestContext2(testDef model.TestDefinition) (ctx TestContext) {
	return NewTestContext(testDef.Token, testDef.Isolation, testDef.TestSuite, testDef.Seq, testDef.Config, testDef.Ppid)
}

type GlobalContext struct {
	Token     string
	Isolation string

	Repo   repo.Repo
	Config model.Config
}

func (c GlobalContext) MergeConfig(newCfg model.Config) {
	c.Config.Merge(newCfg)
}

func (c GlobalContext) Save() error {
	return c.Repo.SaveGlobalConfig(c.Config)
}

type SuiteContext struct {
	GlobalContext

	Action model.Action

	SuiteOutcome utilz.AnyOptional[model.SuiteOutcome]
}

func (c SuiteContext) Save() error {
	return c.Repo.SaveSuiteConfig(c.Config)
}

func (c SuiteContext) InitSuite() error {
	return c.Repo.InitSuite(c.Config)
}

func (c SuiteContext) IncrementTestCount() (n uint16) {
	s := c.Repo.IncrementSuiteSeq(c.Config.TestSuite.Get(), model.TestSequenceFilename)
	return uint16(s)
}

func (c SuiteContext) IncrementPassedCount() (n uint16) {
	return c.Repo.IncrementSuiteSeq(c.Config.TestSuite.Get(), model.PassedSequenceFilename)
}

func (c SuiteContext) IncrementIgnoredCount() (n uint16) {
	return c.Repo.IncrementSuiteSeq(c.Config.TestSuite.Get(), model.IgnoredSequenceFilename)
}

func (c SuiteContext) IncrementFailedCount() (n uint16) {
	return c.Repo.IncrementSuiteSeq(c.Config.TestSuite.Get(), model.FailedSequenceFilename)
}

func (c SuiteContext) IncrementErroredCount() (n uint16) {
	return c.Repo.IncrementSuiteSeq(c.Config.TestSuite.Get(), model.ErroredSequenceFilename)
}

func (c SuiteContext) IncrementTooMuchCount() (n uint16) {
	return c.Repo.IncrementSuiteSeq(c.Config.TestSuite.Get(), model.TooMuchSequenceFilename)
}

func (c SuiteContext) SuiteError(v ...any) error {
	return c.SuiteErrorf("%s", fmt.Sprint(v...))
}

func (c SuiteContext) SuiteErrorf(format string, v ...any) error {
	c.IncrementErroredCount()
	return fmt.Errorf(format, v...)
}

func (c SuiteContext) Fatal(v ...any) {
	c.IncrementErroredCount()
	log.Fatal(v...)
}

func (c SuiteContext) Fatalf(format string, v ...any) {
	c.Fatal(fmt.Sprintf(format, v...))
}

func (c SuiteContext) NoErrorOrFatal(err error) {
	if err != nil {
		c.Config.TestSuite.IfPresent(func(testSuite string) error {
			c.Repo.UpdateLastTestTime(testSuite)
			c.Fatal(err)
			return nil
		})
		c.Fatal(err)
	}
}

type TestContext struct {
	SuiteContext
	Suite SuiteContext

	Seq uint16

	//MockDir        string
	ContainerId    string
	ContainerScope string
	ContainerImage string
	CmdExec        cmdz.Executer
	//TestOutcome    utilz.AnyOptional[model.TestOutcome]
}

func (c TestContext) TestId() (id string) {
	// TODO
	log.Fatal("not implemented yet")
	return
}

func (c *TestContext) IncrementTestCount() (n uint16) {
	if utils.IsWithinContainer() {
		// Do not increment seq
		n = utils.ReadEnvTestSeq()
	} else {
		n = c.SuiteContext.IncrementTestCount()
	}
	c.Seq = n
	return n
}

func (c TestContext) NoErrorOrFatal(err error) {
	if err != nil {
		outcome := model.NewTestOutcome2(c.Config, c.Seq)
		outcome.Outcome = model.ERRORED
		outcome.Err = err
		err2 := c.Repo.SaveTestOutcome(outcome)
		if err2 != nil {
			logger.Error("unable to save errored test outcome", "error", err2)
		}
	}
	c.SuiteContext.NoErrorOrFatal(err)
}

func (c TestContext) ProcessTooMuchFailures() (n uint16) {
	cfg := c.Config
	testSuite := cfg.TestSuite.Get()
	failures := c.Repo.ErroredCount(testSuite) + c.Repo.FailedCount(testSuite)
	if !c.Config.TooMuchFailures.Is(model.TooMuchFailuresNoLimit) && int32(failures) >= c.Config.TooMuchFailures.Get() {
		// Too much failures do not execute more tests
		n = c.IncrementTooMuchCount()
	}
	return
}

func (c TestContext) TestQualifiedName0() (name string) {
	cfg := c.Config
	var testName string
	if cfg.TestName.IsPresent() && !cfg.TestName.Is("") {
		testName = cfg.TestName.Get()
	} else {
		testName = cmdTitle(c.CmdExec)
	}

	containerPart := ""
	if c.ContainerImage != "" {
		containerPart = fmt.Sprintf("(%s)", c.ContainerImage)
	}

	name = fmt.Sprintf("[%s]%s/%s", cfg.TestSuite.Get(), containerPart, testName)
	return
}

func (c TestContext) initTestOutcome(seq uint16) (outcome model.TestOutcome) {
	testSuite := c.Config.TestSuite.Get()
	outcome.TestSuite = testSuite
	outcome.Seq = seq
	outcome.ExitCode = -1
	outcome.Duration = c.CmdExec.Duration()
	outcome.Stdout = c.CmdExec.StdoutRecord()
	outcome.Stderr = c.CmdExec.StderrRecord()

	if c.Config.TestName.IsPresent() && !c.Config.TestName.Is("") {
		outcome.TestName = c.Config.TestName.Get()
	} else {
		outcome.TestName = cmdTitle(c.CmdExec)
	}

	return
}

func (c TestContext) IgnoredTestOutcome(seq uint16) (outcome model.TestOutcome) {
	outcome = c.initTestOutcome(seq)
	outcome.Outcome = model.IGNORED
	return
}

func (c TestContext) AssertCmdExecBlocking(seq uint16, assertions []model.Assertion) (outcome model.TestOutcome) {
	testSuite := c.Config.TestSuite.Get()
	exitCode, err := c.CmdExec.BlockRun()

	c.Repo.UpdateLastTestTime(testSuite)
	outcome = c.initTestOutcome(seq)

	if err != nil {
		// Timeout error is managed
		if errors.Is(err, context.DeadlineExceeded) {
			// Swallow error
			err = nil
			outcome.Outcome = model.TIMEOUT
		} else {
			outcome.Err = err
			outcome.Outcome = model.ERRORED
		}
		c.IncrementErroredCount()
	} else {
		outcome.ExitCode = int16(exitCode)

		var failedResults []model.AssertionResult
		for _, assertion := range assertions {
			var result model.AssertionResult
			result, err = assertion.Asserter(c.CmdExec)
			result.Rule = assertion.Rule
			if err != nil {
				// FIXME: aggregate errors
				result.ErrMessage += fmt.Sprintf("%s ", err)
				result.Success = false
			}
			if !result.Success {
				failedResults = append(failedResults, result)
			}
		}
		outcome.AssertionResults = failedResults

		if len(failedResults) == 0 {
			outcome.Outcome = model.PASSED
			c.IncrementPassedCount()
		} else {
			outcome.Outcome = model.FAILED
			c.IncrementFailedCount()
		}
	}

	err = c.Repo.SaveTestOutcome(outcome)
	c.NoErrorOrFatal(err)

	return
}

func (c *TestContext) initExecuter(ppid uint32) (err error) {
	cfg := c.Config
	cmdAndArgs := cfg.CmdAndArgs
	if len(cmdAndArgs) == 0 {
		//err := fmt.Errorf("no command supplied to test")
		//c.Fatal(err)
		return nil
	}
	cmd := cmdz.Cmd(cmdAndArgs[0])
	if len(cmdAndArgs) > 1 {
		cmd.AddArgs(cmdAndArgs[1:]...)
	}

	// Timeout
	if cfg.Timeout.IsPresent() {
		cmd.Timeout(cfg.Timeout.Get())
	}

	// Input / Outputs
	var stdout, stderr io.Writer
	if cfg.KeepStdout.Is(true) {
		stdout = os.Stdout
	}
	if cfg.KeepStderr.Is(true) {
		stderr = os.Stderr
	}
	cmd.SetOutputs(stdout, stderr)

	stat, _ := os.Stdin.Stat()
	if (stat.Mode() & os.ModeCharDevice) == 0 {
		cmd.SetInput(os.Stdin)
	}

	for _, environ := range os.Environ() {
		if !strings.HasPrefix(environ, "PATH=") {
			cmd.AddEnviron(environ)
		}
	}

	ppidStr := fmt.Sprintf("%d", ppid)
	cmd.AddEnv(model.ContextPpidEnvVarName, ppidStr)
	c.CmdExec = cmd

	return
}

func (c TestContext) ConfigMocking() (err error) {
	cfg := c.Config
	cmd := c.CmdExec
	// Mocking config
	currentPath := os.Getenv("PATH")
	var mockDir string
	mockDir, err = c.MockDirectoryPath(c.Seq)
	if err != nil {
		return
	}
	//logger.Warn("configuring mocking", "dir", mockDir, "count", len(cfg.Mocks)+len(cfg.RootMocks))

	if len(cfg.Mocks)+len(cfg.RootMocks) > 0 {
		// Put mockDir in PATH
		err = mock.ProcessMocking(mockDir, cfg.RootMocks, cfg.Mocks)
		if err != nil {
			return
		}
		cmd.AddEnv("ORIGINAL_PATH", currentPath)
		newPath := fmt.Sprintf("%s:%s", mockDir, currentPath)
		cmd.AddEnv("PATH", newPath)
		err = os.Setenv("PATH", newPath)
		if err != nil {
			return
		}
	} else {
		cmd.AddEnv("PATH", currentPath)
	}
	return
}

func (c TestContext) MockDirectoryPath(testId uint16) (mockDir string, err error) {
	return c.Repo.MockDirectoryPath(c.Config.TestSuite.Get(), testId)
}

func cmdTitle(cmd cmdz.Executer) string {
	cmdNameParts := strings.Split(cmd.String(), " ")
	shortenedCmd := filepath.Base(cmdNameParts[0])
	shortenCmdNameParts := cmdNameParts
	shortenCmdNameParts[0] = shortenedCmd
	cmdName := strings.Join(shortenCmdNameParts, " ")
	//testName = fmt.Sprintf("cmd: <|%s|>", cmdName)
	//testName := fmt.Sprintf("[%s]", cmdName)
	testName := cmdName
	return testName
}
