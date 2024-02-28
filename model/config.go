package model

import (
	"log/slog"
	"regexp"
	"strings"
	"time"

	"mby.fr/utils/utilz"
)

const (
	DefaultVerboseLevel = BETTER_ASSERTION_REPORT
	DefaultDebugLevel   = INFO
)

var (
	DefaultLoggerOpts = &slog.HandlerOptions{
		Level: slog.LevelWarn,
	}
)

func NewGlobalDefaultConfig() Config {
	return Config{
		Prefix:            utilz.OptionalOf(DefaultRulePrefix),
		Verbose:           utilz.OptionalOf(DefaultVerboseLevel),
		Debug:             utilz.OptionalOf(DefaultDebugLevel),
		GlobalStartTime:   utilz.OptionalOf(time.Now()),
		ForkCount:         utilz.OptionalOf(1),
		Quiet:             utilz.OptionalOf(false),
		Ignore:            utilz.OptionalOf(false),
		StopOnFailure:     utilz.OptionalOf(false),
		KeepStdout:        utilz.OptionalOf(false),
		KeepStderr:        utilz.OptionalOf(false),
		Timeout:           utilz.OptionalOf(10 * time.Second),
		RunCount:          utilz.OptionalOf(1),
		Parallel:          utilz.OptionalOf(1),
		ContainerDisabled: utilz.OptionalOf(true),
		ContainerImage:    utilz.OptionalOf(DefaultContainerImage),
		ContainerDirties:  utilz.OptionalOf(DirtyBeforeSuite),
		ContainerScope:    utilz.OptionalOf(SUITE_SCOPE),
	}
}

func NewSuiteDefaultConfig() Config {
	return Config{
		SuiteStartTime: utilz.OptionalOf(time.Now()),
		SuiteTimeout:   utilz.OptionalOf(120 * time.Second),
	}
}

func NewInitlessSuiteDefaultConfig() Config {
	return Config{
		SuiteStartTime: utilz.OptionalOf(time.Now()),
		SuiteTimeout:   utilz.OptionalOf(3600 * time.Second),
	}
}

type ConfigScope int

const (
	GLOBAL_SCOPE ConfigScope = iota // How to use this ?
	SUITE_SCOPE                     // can be placed on suite init only
	TEST_SCOPE                      // can be placed on test or on suite to configure all tests
	RUN_SCOPE
)

type DirtyScope string

const (
	DirtyBeforeSuite = DirtyScope("beforeSuite")
	DirtyAfterSuite  = DirtyScope("afterSuite")
	DirtyBeforeTest  = DirtyScope("beforeTest")
	DirtyAfterTest   = DirtyScope("afterTest")
	DirtyBeforeRun   = DirtyScope("beforeRun")
	DirtyAfterRun    = DirtyScope("afterRun")
)

type VerboseLevel int

const (
	FAILED_ONLY VerboseLevel = iota
	BETTER_ASSERTION_REPORT
	SHOW_PASSED
	SHOW_PASSED_AND_OUTPUTS
)

type DebugLevel int

const (
	DEBUG DebugLevel = iota
	INFO
	WARN
	ERROR
)

type CmdMock struct {
	Op               string
	Cmd              string
	Args             []string
	StdinOp          string
	Stdin            *string
	Stdout           string
	Stderr           string
	ExitCode         int
	Delegate         bool
	OnCallCmdAndArgs []string
}

type Config struct {
	// TestSuite only
	Token     utilz.Optional[string]
	Action    utilz.Optional[Action] `yaml:""`
	TestSuite utilz.Optional[string] `yaml:""`
	TestName  utilz.Optional[string] `yaml:""`

	Prefix          utilz.Optional[string]        `yaml:""`
	CmdAndArgs      []string                      `yaml:""`
	GlobalStartTime utilz.Optional[time.Time]     `yaml:""`
	SuiteStartTime  utilz.Optional[time.Time]     `yaml:""`
	LastTestTime    utilz.Optional[time.Time]     `yaml:""`
	SuiteTimeout    utilz.Optional[time.Duration] `yaml:""`
	ForkCount       utilz.Optional[int]           `yaml:""`
	BeforeSuite     [][]string                    `yaml:""`
	AfterSuite      [][]string                    `yaml:""`

	// Test or TestSuite
	PrintToken    utilz.Optional[bool]          `yaml:""`
	ExportToken   utilz.Optional[bool]          `yaml:""`
	ReportAll     utilz.Optional[bool]          `yaml:""`
	Verbose       utilz.Optional[VerboseLevel]  `yaml:""`
	Debug         utilz.Optional[DebugLevel]    `yaml:""`
	Quiet         utilz.Optional[bool]          `yaml:""`
	Ignore        utilz.Optional[bool]          `yaml:""`
	StopOnFailure utilz.Optional[bool]          `yaml:""`
	KeepStdout    utilz.Optional[bool]          `yaml:""`
	KeepStderr    utilz.Optional[bool]          `yaml:""`
	Timeout       utilz.Optional[time.Duration] `yaml:""`
	RunCount      utilz.Optional[int]           `yaml:""`
	Parallel      utilz.Optional[int]           `yaml:""`

	Mocks  []CmdMock  `yaml:""`
	Before [][]string `yaml:""`
	After  [][]string `yaml:""`

	ContainerDisabled utilz.Optional[bool]        `yaml:""`
	ContainerImage    utilz.Optional[string]      `yaml:""`
	ContainerDirties  utilz.Optional[DirtyScope]  `yaml:""`
	ContainerId       utilz.Optional[string]      `yaml:""`
	ContainerScope    utilz.Optional[ConfigScope] `yaml:""`
}

func (c Config) IsRule(s string) bool {
	prefix := c.Prefix.Get()
	return strings.HasPrefix(s, prefix)
}

func (c Config) SplitRuleExpr(ruleExpr string) (ok bool, r Rule) {
	ok = false
	prefix := c.Prefix.Get()
	assertionRulePattern := regexp.MustCompile("^" + prefix + "([a-zA-Z]+)([=~:!]{1,2})?(.+)?$")
	submatch := assertionRulePattern.FindStringSubmatch(ruleExpr)
	if submatch != nil {
		ok = true
		r.Prefix = prefix
		r.Name = submatch[1]
		r.Op = submatch[2]
		r.Expected = submatch[3]
	}
	return
}

func (c *Config) Merge(right Config) {
	c.Action.Merge(right.Action)
	c.TestSuite.Merge(right.TestSuite)
	c.TestSuite.Merge(right.TestSuite)
	c.TestName.Merge(right.TestName)

	c.Prefix.Merge(right.Prefix)
	c.SuiteStartTime.Merge(right.SuiteStartTime)
	c.SuiteStartTime.Merge(right.SuiteStartTime)
	c.LastTestTime.Merge(right.LastTestTime)
	c.SuiteTimeout.Merge(right.SuiteTimeout)
	c.ForkCount.Merge(right.ForkCount)
	if len(right.CmdAndArgs) > 0 {
		c.CmdAndArgs = right.CmdAndArgs
	}
	if len(right.BeforeSuite) > 0 {
		c.BeforeSuite = append(c.BeforeSuite, right.BeforeSuite...)
	}
	if len(right.AfterSuite) > 0 {
		c.AfterSuite = append(c.AfterSuite, right.AfterSuite...)
	}

	c.PrintToken.Merge(right.PrintToken)
	c.ExportToken.Merge(right.ExportToken)
	c.ReportAll.Merge(right.ReportAll)
	c.Quiet.Merge(right.Quiet)
	c.Ignore.Merge(right.Ignore)
	c.StopOnFailure.Merge(right.StopOnFailure)
	c.KeepStdout.Merge(right.KeepStdout)
	c.KeepStderr.Merge(right.KeepStderr)
	c.Timeout.Merge(right.Timeout)
	c.RunCount.Merge(right.RunCount)
	c.Parallel.Merge(right.Parallel)
	c.Verbose.Merge(right.Verbose)
	c.Debug.Merge(right.Debug)

	if len(right.Mocks) > 0 {
		c.Mocks = append(c.Mocks, right.Mocks...)
	}
	if len(right.Before) > 0 {
		c.Before = append(c.Before, right.Before...)
	}
	if len(right.After) > 0 {
		c.After = append(c.After, right.After...)
	}

	c.ContainerDisabled.Merge(right.ContainerDisabled)
	c.ContainerImage.Merge(right.ContainerImage)
	c.ContainerDirties.Merge(right.ContainerDirties)
	c.ContainerId.Merge(right.ContainerId)
	c.ContainerScope.Merge(right.ContainerScope)
}
