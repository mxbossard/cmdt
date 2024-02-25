package model

import (
	"regexp"
	"strings"
	"time"

	"mby.fr/utils/utilz"
)

func NewGlobalDefaultConfig() Config {
	return Config{
		Prefix:            utilz.OptionnalOf(DefaultRulePrefix),
		GlobalStartTime:   utilz.OptionnalOf(time.Now()),
		ForkCount:         utilz.OptionnalOf(1),
		Quiet:             utilz.OptionnalOf(false),
		Ignore:            utilz.OptionnalOf(false),
		StopOnFailure:     utilz.OptionnalOf(false),
		KeepStdout:        utilz.OptionnalOf(false),
		KeepStderr:        utilz.OptionnalOf(false),
		Timeout:           utilz.OptionnalOf(10 * time.Second),
		RunCount:          utilz.OptionnalOf(1),
		Parallel:          utilz.OptionnalOf(1),
		ContainerDisabled: utilz.OptionnalOf(true),
		ContainerImage:    utilz.OptionnalOf(DefaultContainerImage),
		ContainerDirties:  utilz.OptionnalOf(DirtyBeforeSuite),
		ContainerScope:    utilz.OptionnalOf(SUITE_SCOPE),
	}
}

func NewSuiteDefaultConfig() Config {
	return Config{
		StartTime:    utilz.OptionnalOf(time.Now()),
		SuiteTimeout: utilz.OptionnalOf(120 * time.Second),
	}
}

func NewInitlessSuiteDefaultConfig() Config {
	return Config{
		StartTime:    utilz.OptionnalOf(time.Now()),
		SuiteTimeout: utilz.OptionnalOf(3600 * time.Second),
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
	SHOW_PASSED_OUTPUTS
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
	Token     utilz.Optional[string] `yaml:""`
	Action    utilz.Optional[Action] `yaml:""`
	TestSuite utilz.Optional[string] `yaml:""`
	TestName  utilz.Optional[string] `yaml:""`

	Prefix          utilz.Optional[string]        `yaml:""`
	GlobalStartTime utilz.Optional[time.Time]     `yaml:""`
	StartTime       utilz.Optional[time.Time]     `yaml:""`
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

func (c Config) Merge(new Config) (err error) {
	if new.TestSuite.IsPresent() {
		c.TestSuite = new.TestSuite
	}
	if new.TestName.IsPresent() {
		c.TestName = new.TestName
	}

	if new.Prefix.IsPresent() {
		c.Prefix = new.Prefix
	}
	if new.StartTime.IsPresent() {
		c.StartTime = new.StartTime
	}
	if new.LastTestTime.IsPresent() {
		c.LastTestTime = new.LastTestTime
	}
	if new.SuiteTimeout.IsPresent() {
		c.SuiteTimeout = new.SuiteTimeout
	}
	if new.ForkCount.IsPresent() {
		c.ForkCount = new.ForkCount
	}
	if len(new.BeforeSuite) > 0 {
		c.BeforeSuite = append(c.BeforeSuite, new.BeforeSuite...)
	}
	if len(new.AfterSuite) > 0 {
		c.AfterSuite = append(c.AfterSuite, new.AfterSuite...)
	}

	if new.PrintToken.IsPresent() {
		c.PrintToken = new.PrintToken
	}
	if new.ExportToken.IsPresent() {
		c.ExportToken = new.ExportToken
	}
	if new.ReportAll.IsPresent() {
		c.ReportAll = new.ReportAll
	}
	if new.Quiet.IsPresent() {
		c.Quiet = new.Quiet
	}
	if new.Ignore.IsPresent() {
		c.Ignore = new.Ignore
	}
	if new.StopOnFailure.IsPresent() {
		c.StopOnFailure = new.StopOnFailure
	}
	if new.KeepStdout.IsPresent() {
		c.KeepStdout = new.KeepStdout
	}
	if new.KeepStderr.IsPresent() {
		c.KeepStderr = new.KeepStderr
	}
	if new.Timeout.IsPresent() {
		c.Timeout = new.Timeout
	}
	if new.RunCount.IsPresent() {
		c.RunCount = new.RunCount
	}
	if new.Parallel.IsPresent() {
		c.Parallel = new.Parallel
	}

	if len(new.Mocks) > 0 {
		c.Mocks = append(c.Mocks, new.Mocks...)
	}
	if len(new.Before) > 0 {
		c.Before = append(c.Before, new.Before...)
	}
	if len(new.After) > 0 {
		c.After = append(c.After, new.After...)
	}

	if new.ContainerDisabled.IsPresent() {
		c.ContainerDisabled = new.ContainerDisabled
	}
	if new.ContainerImage.IsPresent() {
		c.ContainerImage = new.ContainerImage
	}
	if new.ContainerDirties.IsPresent() {
		c.ContainerDirties = new.ContainerDirties
	}
	if new.ContainerId.IsPresent() {
		c.ContainerId = new.ContainerId
	}
	if new.ContainerScope.IsPresent() {
		c.ContainerScope = new.ContainerScope
	}

	return
}
