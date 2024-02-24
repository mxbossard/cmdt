package model

import (
	"time"

	"mby.fr/utils/utilz"
)

type Config2 struct {
	// TestSuite only

	TestSuite utilz.Optional[string] `yaml:""`
	TestName  utilz.Optional[string] `yaml:""`

	Prefix       string                        `yaml:""`
	StartTime    utilz.Optional[time.Time]     `yaml:""`
	LastTestTime utilz.Optional[time.Time]     `yaml:""`
	SuiteTimeout utilz.Optional[time.Duration] `yaml:""`
	ForkCount    utilz.Optional[int]           `yaml:""`
	BeforeSuite  utilz.Optional[[][]string]    `yaml:""`
	AfterSuite   utilz.Optional[[][]string]    `yaml:""`

	// Test or TestSuite
	PrintToken    utilz.Optional[bool]          `yaml:""`
	ExportToken   utilz.Optional[bool]          `yaml:""`
	ReportAll     utilz.Optional[bool]          `yaml:""`
	Quiet         utilz.Optional[bool]          `yaml:""`
	Ignore        utilz.Optional[bool]          `yaml:""`
	StopOnFailure utilz.Optional[bool]          `yaml:""`
	KeepStdout    utilz.Optional[bool]          `yaml:""`
	KeepStderr    utilz.Optional[bool]          `yaml:""`
	Timeout       utilz.Optional[time.Duration] `yaml:""`
	RunCount      utilz.Optional[int]           `yaml:""`
	Parallel      utilz.Optional[int]           `yaml:""`

	Mocks  utilz.Optional[[]CmdMock]  `yaml:""`
	Before utilz.Optional[[][]string] `yaml:""`
	After  utilz.Optional[[][]string] `yaml:""`

	ContainerDisabled utilz.Optional[bool]        `yaml:""`
	ContainerImage    utilz.Optional[string]      `yaml:""`
	ContainerDirties  utilz.Optional[string]      `yaml:""`
	ContainerId       utilz.Optional[string]      `yaml:""`
	ContainerScope    utilz.Optional[ConfigScope] `yaml:""`
}

func (c Config) PersistGlobal() (err error) {
	// TODO
	return
}

func (c Config) PersistSuite() (err error) {
	// TODO
	return
}

func (c Config) Merge(new Config) (err error) {
	// TODO
	return
}

func LoadGlobalConfig(token, suite string) (cfg Config) {
	return
}

func LoadSuiteConfig(token, suite string) (cfg Config) {
	return
}
