package facade

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"cmdt/internal/mock"
	"cmdt/internal/model"
	"cmdt/internal/repo"
	"cmdt/internal/utils"

	"github.com/mxbossard/utilz/cmdz"
	"github.com/mxbossard/utilz/errorz"
	"github.com/mxbossard/utilz/poolz"
	"github.com/mxbossard/utilz/utilz"
	"github.com/mxbossard/utilz/zlog"
)

var logger = zlog.New() //slog.New(slog.NewTextHandler(os.Stderr, model.DefaultLoggerOpts))

var _repo = make(map[string]repo.Repo)

var _repoPool poolz.Pool[*repo.DbRepo]
var _repoPoolMap = make(map[string]poolz.Pool[*repo.DbRepo])

func Repo(token, isolation string) repo.Repo {
	return NewRepo(token, isolation)
}

func CachedRepo(token, isolation string) repo.Repo {
	key := fmt.Sprintf("%s;%s", token, isolation)
	if ok := _repo[key]; ok == nil {
		r := repo.New(token, isolation)
		_repo[key] = &r
	}
	return _repo[key]
}

func NewRepo(token, isolation string) repo.Repo {
	r := repo.New(token, isolation)
	return &r
}

var n = 0

func Repo1(token, isolation string) repo.Repo {
	if _repoPool == nil {
		_repoPool = poolz.New(50, func() (*repo.DbRepo, error) {
			r := repo.New(token, isolation)
			n++
			if n > 3 {
				available, inUse := _repoPool.Count()
				fmt.Printf("\n<<>> built single pool DbRepo #%d (avail: %d ; inUse: %d)\n", n, available, inUse)
			}
			return &r, nil
		})
	}
	r, err := _repoPool.Open()
	if err != nil {
		panic(err)
	}

	return r
}

func PooledRepo(token, isolation string) repo.Repo {
	key := fmt.Sprintf("%s;%s", token, isolation)
	if ok := _repoPoolMap[key]; ok == nil {
		r := poolz.New(50, func() (*repo.DbRepo, error) {
			repo := repo.New(token, isolation)
			// n++
			// if n > 3 {
			// 	available, inUse := _repoPoolMap[key].Count()
			// 	fmt.Printf("\n<<>> built single pool DbRepo #%d (avail: %d ; inUse: %d)\n", n, available, inUse)
			// }
			return &repo, nil
		})
		r.SetMaxLifeTime(1000 * time.Millisecond)
		_repoPoolMap[key] = r
	}

	r, err := _repoPoolMap[key].Open()
	if err != nil {
		panic(err)
	}

	return r
}

func NewGlobalContext(token, isolation string, inputCfg model.Config, pooledRepo bool) GlobalContext {
	logger.Debug("Building Global context", "token", token, "isolation", isolation)
	var err error
	token, err = utils.ForgeContextualToken(token)
	if err != nil {
		errorz.Fatal(err)
	}

	var repo repo.Repo
	if pooledRepo {
		repo = PooledRepo(token, isolation)
		//repo = NewRepo(token, isolation)
	} else {
		repo = CachedRepo(token, isolation)
		//repo = NewRepo(token, isolation)
	}

	cfg, err := repo.GetGlobalConfig()
	if err != nil {
		errorz.Fatal(err)
	}
	cfg.Merge(inputCfg)

	c := GlobalContext{
		Token:     token,
		Isolation: isolation,
		Repo:      repo,
		Config:    cfg,
	}
	logger.Debug("Builded Global context", "token", token, "isolation", isolation)
	return c
}

func NewSuiteContext(token, isolation, testSuite string, initless bool, action model.Action, inputCfg model.Config, pooledRepo bool) SuiteContext {
	logger.Debug("Building Suite context", "suite", testSuite, "token", token, "isolation", isolation)
	globalCtx := NewGlobalContext(token, isolation, model.Config{}, pooledRepo)
	suiteCfg, err := globalCtx.Repo.GetSuiteConfig(testSuite, initless)
	if err != nil {
		errorz.Fatal(err)
	}

	mergedCfg := globalCtx.Config
	mergedCfg.Merge(suiteCfg)
	mergedCfg.Merge(inputCfg)
	globalCtx.Config = mergedCfg

	suiteCtx := SuiteContext{
		GlobalContext: globalCtx,
		Action:        action,
	}
	logger.Debug("Builded Suite context", "suite", testSuite, "token", token, "isolation", isolation, "async", suiteCtx.Config.Async.Get(), "suiteCfgAsync", suiteCfg.Async.Get())
	return suiteCtx
}

func NewTestContext(token, isolation, testSuite string, seq uint16, inputCfg model.Config, ppid uint32, pooledRepo bool) (testCtx TestContext, err error) {
	logger.Debug("Building Test context", "suite", testSuite, "seq", seq)
	suiteCtx := NewSuiteContext(token, isolation, testSuite, true, model.TestAction, model.Config{}, pooledRepo)
	mergedCfg := suiteCtx.Config
	mergedCfg.Merge(inputCfg)

	testCtx = TestContext{
		SuiteContext: suiteCtx,
	}
	testCtx.Config = mergedCfg
	testCtx.Suite = suiteCtx
	testCtx.Seq = seq
	err = testCtx.initExecuter(ppid)
	if err != nil {
		return
	}

	if ok, ctId := utils.ReadEnvValue(model.EnvContainerIdKey); ok {
		testCtx.ContainerId = ctId
		_, testCtx.ContainerScope = utils.ReadEnvValue(model.EnvContainerScopeKey)
		_, testCtx.ContainerImage = utils.ReadEnvValue(model.EnvContainerImageKey)
	}
	logger.Debug("Builded Test context", "suite", testSuite, "seq", seq)
	return
}

func NewTestContext2(testDef model.TestDefinition, pooledRepo bool) (TestContext, error) {
	return NewTestContext(testDef.Token, testDef.Isolation, testDef.TestSuite, testDef.Seq, testDef.Config, testDef.Ppid, pooledRepo)
}

type GlobalContext struct {
	Token     string
	Isolation string

	Repo   repo.Repo
	Config model.Config
}

func (c GlobalContext) Close() error {
	return c.Repo.PoolClose()
}

func (c GlobalContext) MergeConfig(newCfg model.Config) {
	c.Config.Merge(newCfg)
}

func (c GlobalContext) Save() error {
	return c.Repo.SaveGlobalConfig(c.Config)
}

/*
	func (c GlobalContext) Fatal(v ...any) {
		fmt.Fprintln(os.Stderr, v...)
		os.Exit(1)
	}
*/

/*
func (c GlobalContext) NoErrorOrFatal(err error) {
	if err != nil {
		c.Config.TestSuite.IfPresent(func(testSuite string) error {
			c.Repo.UpdateLastTestTime(testSuite)
			c.Fatal(err)
			return nil
		})
		c.Fatal(err)
	}
}
*/

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

/*
func (c SuiteContext) Fatal(v ...any) {
	c.IncrementErroredCount()
	c.GlobalContext.Fatal(v...)
}
*/

/*
func (c SuiteContext) Fatalf(format string, v ...any) {
	c.Fatal(fmt.Sprintf(format, v...))
}
*/

/*
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
*/

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
	//errorz.Fatal("not implemented yet")
	return fmt.Sprintf("%s__%d", c.Suite.Config.TestSuite.Get(), c.Seq)
}

func (c *TestContext) IncrementTestCount() (n uint16) {
	logger.Debug("Incrementing Test count")
	if utils.IsWithinContainer() {
		// Do not increment seq
		n = utils.ReadEnvTestSeq()
	} else {
		n = c.SuiteContext.IncrementTestCount()
	}
	c.Seq = n
	logger.Debug("Incremented Test count", "n", n)
	return n
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
	if c.CmdExec != nil {
		outcome.Duration = c.CmdExec.Duration()
		outcome.Stdout = c.CmdExec.StdoutRecord()
		outcome.Stderr = c.CmdExec.StderrRecord()
	}

	if c.Config.TestName.IsPresent() && !c.Config.TestName.Is("") {
		outcome.TestName = c.Config.TestName.Get()
	} else if c.CmdExec != nil {
		outcome.TestName = cmdTitle(c.CmdExec)
	} else {
		outcome.TestName = "UNKNONW"
	}

	return
}

func (c TestContext) IgnoredTestOutcome() (outcome model.TestOutcome) {
	outcome = c.initTestOutcome(c.Seq)
	outcome.Outcome = model.IGNORED
	return
}

func (c TestContext) ErroredTestOutcome(errors ...error) (outcome model.TestOutcome) {
	outcome = c.initTestOutcome(c.Seq)
	outcome.Outcome = model.ERRORED
	if len(errors) > 0 {
		outcome.Err = errors[0]
	}
	return
}

func (c TestContext) UnknownTestOutcome() (outcome model.TestOutcome) {
	outcome = c.initTestOutcome(c.Seq)
	outcome.Outcome = model.UNKNOWN
	return
}

func (c TestContext) AssertCmdExecBlocking(seq uint16, assertions []model.Assertion) (outcome model.TestOutcome, err error) {
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
		logger.Debug("Asserting command execution ...", "seq", seq, "assertions", assertions)
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
		logger.Info("Asserted command execution", "seq", seq, "outcome", outcome.Outcome, "failedResults", failedResults)
	}

	err = c.Repo.SaveTestOutcome(outcome)

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
