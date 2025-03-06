package parser

import (
	"time"

	"mby.fr/cmdtest/model"
)

var (

	// ACTIONS
	global = buildRule("global", ops(noOp[string]()), nil, "g")
	suite  = buildRule("suite", ops(noOp[string](), equalSuiteName),
		mutater(func(cfg *model.Config, op string, val string) { cfg.TestSuite.Set(val) }),
		"s", "init", "i")
	test = buildRule("test", ops(noOp[string](), equalTestName),
		mutater(func(cfg *model.Config, op string, val string) {
			// FIXME: config suite and test
			cfg.TestName.Set(val)
		}),
		"t")
	report = buildRule("report", ops(noOp[string](), equalSuiteName),
		mutater(func(cfg *model.Config, op string, val string) {
			if op == "=" {
				cfg.ReportAll.Set(false)
				cfg.TestSuite.Set(val)
			} else {
				cfg.ReportAll.Set(true)
			}
		}),
		"r")
	help  = buildRule("help", ops(noOp[string]()), nil, "h")
	usage = buildRule("usage", ops(noOp[string]()), nil) // ???

	// ISOLATION
	token = buildRule("token", ops(equalString),
		mutater(func(cfg *model.Config, op string, val string) { cfg.Token.Set(val) }))
	isolation = buildRule("isolation", ops(equalString),
		mutater(func(cfg *model.Config, op string, val string) { cfg.Isol.Set(val) }),
		"isol")

	// TOKEN_ACTIONS
	printToken  = buildRule("printToken", ops(noOp[string]()), nil, "print_token")
	exportToken = buildRule("exportToken", ops(noOp[string]()), nil, "export_token")

	// PARSING
	prefix = buildRule("prefix", ops(equalString),
		mutater(func(cfg *model.Config, op string, val string) { cfg.Prefix.Set(val) }))

	// VERBOSITY
	quiet = buildRule("quiet", ops(noOp[bool](), equalBoolean),
		mutater(func(cfg *model.Config, op string, val bool) { cfg.Quiet.Set(val) }),
		"q")
	verbose = buildMvRule("verbose", ops(noOp[uint16](), equalUint16),
		mutater(func(cfg *model.Config, op string, val uint16) {
			// FIXME: implements noOp behavior
			cfg.Verbose.Set(model.VerboseLevel(val))
		}),
		"v")
	debug = buildMvRule("debug", ops(noOp[int16](), equalInt16),
		mutater(func(cfg *model.Config, op string, val int16) {
			// FIXME: implements noOp behavior
			cfg.Debug.Set(model.DebugLevel(val))
		}),
		"d", "x")

	// SUITE CONFIG
	fork = buildRule("fork", ops(noOp[uint16](), equalUint16),
		mutater(func(cfg *model.Config, op string, val uint16) { cfg.ForkCount.Set(val) }))
	suiteTimeout = buildRule("suiteTimeout", ops(equalDuration),
		mutater(func(cfg *model.Config, op string, val time.Duration) { cfg.SuiteTimeout.Set(val) }),
		"suite_timeout", "timeoutSuite", "timeout_suite")
	async = buildRule("async", ops(noOp[bool](), equalBoolean),
		mutater(func(cfg *model.Config, op string, val bool) { cfg.Async.Set(val) }))
	stopOnFailure = buildRule("stopOnFailure", ops(noOp[bool](), equalBoolean),
		mutater(func(cfg *model.Config, op string, val bool) { cfg.StopOnFailure.Set(val) }),
		"stop_on_failure")
	failuresLimit = buildRule("failuresLimit", ops(equalInt32),
		mutater(func(cfg *model.Config, op string, val int32) { cfg.TooMuchFailures.Set(val) }),
		"failures_limit")
	beforeSuite = buildMvRule("beforeSuite", ops(equalCmd),
		mutater(func(cfg *model.Config, op string, val []string) { cfg.BeforeSuite = append(cfg.BeforeSuite, val) }),
		"before_suite")
	afterSuite = buildMvRule("afterSuite", ops(equalCmd),
		mutater(func(cfg *model.Config, op string, val []string) { cfg.AfterSuite = append(cfg.BeforeSuite, val) }),
		"after_suite")

	// TEST CONFIG
	wait = buildRule("wait", ops(noOp[bool](), equalBoolean),
		mutater(func(cfg *model.Config, op string, val bool) { cfg.Wait.Set(val) }))
	ignore = buildRule("ignore", ops(noOp[bool](), equalBoolean),
		mutater(func(cfg *model.Config, op string, val bool) { cfg.Ignore.Set(val) }))
	// FIXME: if keepOutputs is used, should not have keepStdout nor keepStderr
	keepStdout = buildRule("keepStdout", ops(noOp[bool](), equalBoolean),
		mutater(func(cfg *model.Config, op string, val bool) { cfg.KeepStdout.Set(val) }),
		"keep_stdout", "keepOut", "keep_out")
	keepStderr = buildRule("keepStderr", ops(noOp[bool](), equalBoolean),
		mutater(func(cfg *model.Config, op string, val bool) { cfg.KeepStderr.Set(val) }),
		"keep_stderr", "keepErr", "keep_err")
	keepOutputs = buildRule("keepOutputs", ops(noOp[bool](), equalBoolean),
		mutater(func(cfg *model.Config, op string, val bool) {
			cfg.KeepStdout.Set(val)
			cfg.KeepStderr.Set(val)
		}),
		"keep_outputs", "keepOuts", "keep_outs")
	timeout = buildRule("timeout", ops(equalDuration),
		mutater(func(cfg *model.Config, op string, val time.Duration) { cfg.Timeout.Set(val) }))
	runCount = buildRule("runCount", ops(equalUint16),
		mutater(func(cfg *model.Config, op string, val uint16) { cfg.RunCount.Set(val) }),
		"run_count")
	mock = buildRule("mock", ops(equalMock),
		mutater(func(cfg *model.Config, op string, val model.CmdMock) { cfg.Mocks = append(cfg.Mocks, val) }))
	before = buildMvRule("before", ops(equalCmd),
		mutater(func(cfg *model.Config, op string, val []string) { cfg.Before = append(cfg.Before, val) }))
	after = buildMvRule("after", ops(equalCmd),
		mutater(func(cfg *model.Config, op string, val []string) { cfg.After = append(cfg.After, val) }))
	container = buildRule("container", ops(noOp[string](), equalString),
		mutater(func(cfg *model.Config, op string, image string) {
			cfg.ContainerDisabled.Set(false)
			// Erase ContainerId because we will want a new Container
			cfg.ContainerId.Set("")
			if image == "true" {
				cfg.ContainerImage.Set(model.DefaultContainerImage)
			} else if image == "false" {
				cfg.ContainerImage.Clear()
				cfg.ContainerDisabled.Set(true)
			} else if image != "" {
				cfg.ContainerImage.Set(image)
			} else {
				cfg.ContainerImage.Set(model.DefaultContainerImage)
			}
		}))
	dirtyContainer = buildRule("dirtyContainer", ops(equalDirtyScope),
		mutater(func(cfg *model.Config, op string, val model.DirtyScope) { cfg.ContainerDirties.Set(val) }),
		"dirty_container")

	// REPORT CONFIG
	keepReport = buildRule("keep", ops(noOp[bool]()), nil)

	// WHERE ?
	//parallel = buildRule("parallel", nil)

)

var (
	// ASSERTIONS CONFIG
	success = buildAssertRule("success", ops(noOp[bool]()), nil)
	failure = buildAssertRule("failure", ops(noOp[bool]()), nil, "fail")
	exit    = buildAssertRule("exit", ops(equalUint8), nil, "rc")
	// Multi valued
	cmd    = buildMvAssertRule("cmd", ops(equalCmd), nil)
	exists = buildMvAssertRule("exists", ops(equalFilepath), nil)
	// Multi valued or exclusive by Ops
	stdout = buildMvExclOpsAssertRule("stdout",
		ops(equalString, equalStringOrEmpty, notEqualString, containsString, notContainsString, matchString, notMatchString, equalFileContent, containsFileContent),
		ops(notEqualString, containsString, notContainsString, matchString, notMatchString, containsFileContent),
		ops(equalStringOrEmpty, equalString, equalFileContent),
		nil, "out")
	stderr = buildMvExclOpsAssertRule("stderr",
		ops(equalString, equalStringOrEmpty, notEqualString, containsString, notContainsString, matchString, notMatchString, equalFileContent, containsFileContent),
		ops(notEqualString, containsString, notContainsString, matchString, notMatchString, containsFileContent),
		ops(equalStringOrEmpty, equalString, equalFileContent),
		nil, "err")
)

var (
	// RULES SETS
	rsActions      = buildMERS("action", nil, rules(global, suite, test, report, help, usage), test)
	rsIsolation    = buildRS("isolation", nil, rules(token, isolation), nil)
	rsTokenActions = buildMERS("tokenAction", nil, rules(printToken, exportToken), nil)
	rsParsing      = buildRS("parsing", nil, rules(prefix), nil)
	rsVerbosity    = buildRS("verbosity", rules(global, suite, test, report), rules(quiet, verbose, debug), nil)
	rsSuiteConfig  = buildRS("suiteConfig", rules(global, suite), rules(fork, suiteTimeout, async, stopOnFailure,
		failuresLimit, beforeSuite, afterSuite), nil)
	rsTestConfig = buildRS("testConfig", rules(global, suite, test), rules(wait, ignore, keepStdout, keepStderr,
		keepOutputs, timeout, runCount, mock, before, after, container, dirtyContainer), nil)
	rsReportConfig        = buildRS("reportConfig", rules(report), rules(keepReport), nil)
	rsOutcomeAssertions   = buildMERS("outcomeAssertions", rules(test), rules(success, failure, exit), success)
	rsStackableAssertions = buildRS("stackableAssertions", rules(test), rules(stdout, stderr, cmd, exists), nil)
)
