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

func NewGlobalContext(token string, inputCfg model.Config) GlobalContext {
	if token == "" {
		var err error
		token, err = utils.ForgeContextualToken()
		if err != nil {
			log.Fatal(err)
		}
	}

	repo := repo.New(token)

	cfg, err := repo.LoadGlobalConfig()
	if err != nil {
		log.Fatal(err)
	}
	cfg.Merge(inputCfg)

	c := GlobalContext{
		Token:  token,
		Repo:   repo,
		Config: cfg,
	}

	//c.SetRulePrefix(cfg.Prefix.Get())
	return c
}

func NewSuiteContext(token, testSuite string, action model.Action, inputCfg model.Config) SuiteContext {
	globalCtx := NewGlobalContext(token, model.Config{})
	suiteCfg, err := globalCtx.Repo.LoadSuiteConfig(testSuite)
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
	//c.SetRulePrefix(cfg.Prefix.Get())
	return suiteCtx
}

func NewTestContext(token, testSuite string, inputCfg model.Config) TestContext {
	suiteCtx := NewSuiteContext(token, testSuite, model.TestAction, model.Config{})
	mergedCfg := suiteCtx.Config
	mergedCfg.Merge(inputCfg)
	suiteCtx.Config = mergedCfg

	cmdAndArgs := mergedCfg.CmdAndArgs
	if len(cmdAndArgs) == 0 {
		err := fmt.Errorf("no command supplied to test")
		suiteCtx.Fatal(err)
	}
	cmd := cmdz.Cmd(cmdAndArgs[0])
	if len(cmdAndArgs) > 1 {
		cmd.AddArgs(cmdAndArgs[1:]...)
	}

	if mergedCfg.Timeout.IsPresent() {
		cmd.Timeout(mergedCfg.Timeout.Get())
	}

	if !mergedCfg.TestName.IsPresent() || mergedCfg.TestName.Is("") {
		mergedCfg.TestName = utilz.OptionalOf(cmdTitle(cmd))
	}

	testCtx := TestContext{
		SuiteContext: suiteCtx,
	}
	err := testCtx.initExecuter()
	if err != nil {
		testCtx.NoErrorOrFatal(err)
	}

	//c.SetRulePrefix(cfg.Prefix.Get())
	return testCtx
}

/*
type Context struct {
	Token                string
	Action               model.Action
	rulePrefix           string         // TODEL ?
	assertionRulePattern *regexp.Regexp // TODEL

	Repo   repo.FileRepo
	Config model.Config

	TestOutcome  utilz.AnyOptional[model.TestOutcome]
	SuiteOutcome utilz.AnyOptional[model.SuiteOutcome]
}

func (c Context) MergeConfig(newCfg model.Config) {
	c.Config.Merge(newCfg)
}

func (c Context) Save() error {
	if c.Action == model.GlobalAction {
		return c.Repo.SaveGlobalConfig(c.Config)
	} else {
		return c.Repo.SaveSuiteConfig(c.Config)
	}
}

func (c Context) TestId() (id string) {
	// TODO
	log.Fatal("not implemented yet")
	return
}

func (c Context) TestQualifiedName() (name string) {
	name = fmt.Sprintf("[%s]/%s", c.Config.TestSuite.Get(), c.Config.TestName.Get())
	return
}

func (c Context) TestTitle() (title string) {
	cfg := c.Config
	timecode := int(time.Since(cfg.SuiteStartTime.Get()).Milliseconds())
	qualifiedName := c.TestQualifiedName()
	seq := c.TestCount()
	title = fmt.Sprintf("[%05d] Test: %s #%02d... ", timecode, qualifiedName, seq)
	return
}

func (c Context) RulePrefix() string {
	return c.rulePrefix
}

func (c *Context) SetRulePrefix(prefix string) {
	if prefix != "" {
		c.rulePrefix = prefix
		c.assertionRulePattern = regexp.MustCompile("^" + c.RulePrefix() + "([a-zA-Z]+)([=~:!]{1,2})?(.+)?$")
	}
}

func (c Context) IsRule(s string) bool {
	return strings.HasPrefix(s, c.RulePrefix())
}

func (c Context) SplitRuleExpr(ruleExpr string) (ok bool, r model.Rule) {
	ok = false
	submatch := c.assertionRulePattern.FindStringSubmatch(ruleExpr)
	if submatch != nil {
		ok = true
		r.Prefix = c.RulePrefix()
		r.Name = submatch[1]
		r.Op = submatch[2]
		r.Expected = submatch[3]
	}
	return
}

func (c Context) IncrementTestCount() (n int) {
	return c.Repo.IncrementSuiteSeq(c.Config.TestName.Get(), model.TestSequenceFilename)
}

func (c Context) IncrementPassedCount() (n int) {
	return c.Repo.IncrementSuiteSeq(c.Config.TestName.Get(), model.PassedSequenceFilename)
}

func (c Context) IncrementIgnoredCount() (n int) {
	return c.Repo.IncrementSuiteSeq(c.Config.TestName.Get(), model.IgnoredSequenceFilename)
}

func (c Context) IncrementFailedCount() (n int) {
	return c.Repo.IncrementSuiteSeq(c.Config.TestName.Get(), model.FailedSequenceFilename)
}

func (c Context) IncrementErroredCount() (n int) {
	return c.Repo.IncrementSuiteSeq(c.Config.TestName.Get(), model.ErroredSequenceFilename)
}

func (c Context) SuiteError(v ...any) error {
	return c.SuiteErrorf("%s", fmt.Sprint(v...))
}

func (c Context) SuiteErrorf(format string, v ...any) error {
	c.IncrementErroredCount()
	return fmt.Errorf(format, v...)
}

func (c Context) Fatal(v ...any) {
	c.IncrementErroredCount()
	log.Fatal(v...)
}

func (c Context) Fatalf(format string, v ...any) {
	c.Fatal(fmt.Sprintf(format, v...))
}

func (c Context) NoErrorOrFatal(err error) {
	if err != nil {
		c.Config.TestSuite.IfPresent(func(testSuite string) error {
			c.Repo.UpdateLastTestTime(testSuite)
			c.Fatal(err)
			return nil
		})
		log.Fatal(err)
	}
}
*/

type GlobalContext struct {
	Token string

	Repo   repo.FileRepo
	Config model.Config
}

func (c GlobalContext) MergeConfig(newCfg model.Config) {
	c.Config.Merge(newCfg)
}

func (c GlobalContext) Save() error {
	return c.Repo.SaveGlobalConfig(c.Config)
}

/*
func (c GlobalContext) RulePrefix() string {
	return c.rulePrefix
}

func (c *GlobalContext) SetRulePrefix(prefix string) {
	if prefix != "" {
		c.rulePrefix = prefix
		c.assertionRulePattern = regexp.MustCompile("^" + c.RulePrefix() + "([a-zA-Z]+)([=~:!]{1,2})?(.+)?$")
	}
}

func (c GlobalContext) IsRule(s string) bool {
	return strings.HasPrefix(s, c.RulePrefix())
}

func (c GlobalContext) SplitRuleExpr(ruleExpr string) (ok bool, r model.Rule) {
	ok = false
	submatch := c.assertionRulePattern.FindStringSubmatch(ruleExpr)
	if submatch != nil {
		ok = true
		r.Prefix = c.RulePrefix()
		r.Name = submatch[1]
		r.Op = submatch[2]
		r.Expected = submatch[3]
	}
	return
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

func (c SuiteContext) IncrementTestCount() (n int) {
	return c.Repo.IncrementSuiteSeq(c.Config.TestSuite.Get(), model.TestSequenceFilename)
}

func (c SuiteContext) IncrementPassedCount() (n int) {
	return c.Repo.IncrementSuiteSeq(c.Config.TestSuite.Get(), model.PassedSequenceFilename)
}

func (c SuiteContext) IncrementIgnoredCount() (n int) {
	return c.Repo.IncrementSuiteSeq(c.Config.TestSuite.Get(), model.IgnoredSequenceFilename)
}

func (c SuiteContext) IncrementFailedCount() (n int) {
	return c.Repo.IncrementSuiteSeq(c.Config.TestSuite.Get(), model.FailedSequenceFilename)
}

func (c SuiteContext) IncrementErroredCount() (n int) {
	return c.Repo.IncrementSuiteSeq(c.Config.TestSuite.Get(), model.ErroredSequenceFilename)
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
		log.Fatal(err)
	}
}

type TestContext struct {
	SuiteContext

	CmdExec     cmdz.Executer
	TestOutcome utilz.AnyOptional[model.TestOutcome]
}

func (c SuiteContext) TestId() (id string) {
	// TODO
	log.Fatal("not implemented yet")
	return
}

func (c TestContext) TestQualifiedName() (name string) {
	var testName string
	if c.Config.TestName.IsPresent() && !c.Config.TestName.Is("") {
		testName = c.Config.TestName.Get()
	} else {
		testName = cmdTitle(c.CmdExec)
	}
	name = fmt.Sprintf("[%s]/%s", c.Config.TestSuite.Get(), testName)
	return
}

func (c TestContext) initTestOutcome(seq int) (outcome model.TestOutcome) {
	testSuite := c.Config.TestSuite.Get()
	outcome.TestSuite = testSuite
	outcome.Seq = seq
	outcome.TestQualifiedName = c.TestQualifiedName()
	outcome.ExitCode = -1
	outcome.CmdTitle = cmdTitle(c.CmdExec)
	outcome.Duration = c.CmdExec.Duration()
	outcome.Stdout = c.CmdExec.StdoutRecord()
	outcome.Stderr = c.CmdExec.StderrRecord()
	return
}

func (c TestContext) IgnoredTestOutcome(seq int) (outcome model.TestOutcome) {
	outcome = c.initTestOutcome(seq)
	outcome.Outcome = model.IGNORED
	return
}

func (c TestContext) AssertCmdExecBlocking(seq int, assertions []model.Assertion) (outcome model.TestOutcome) {
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
		outcome.ExitCode = exitCode

		var failedResults []model.AssertionResult
		for _, assertion := range assertions {
			var result model.AssertionResult
			result, err = assertion.Asserter(c.CmdExec)
			result.Assertion = assertion
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

	// TODO: Record outcome
	err = c.Repo.SaveTestOutcome(outcome)
	c.NoErrorOrFatal(err)

	return
}

func (c *TestContext) initExecuter() (err error) {
	cfg := c.Config
	cmdAndArgs := cfg.CmdAndArgs
	if len(cmdAndArgs) == 0 {
		err := fmt.Errorf("no command supplied to test")
		c.Fatal(err)
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

	// Mocking
	currentPath := os.Getenv("PATH")
	if len(cfg.Mocks) > 0 {
		// Put mockDir in PATH
		testWorkDir := c.Repo.BackingFilepath()
		var mockDir string
		mockDir, err = mock.ProcessMocking(testWorkDir, cfg.Mocks)
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

	c.CmdExec = cmd
	return
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
