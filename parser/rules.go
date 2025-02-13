package parser

import "mby.fr/cmdtest/model"

/**
## Rules
OPEN: comment gérer ces alias: @help et -h ?
OPEN: comment gérer help qui est particulier car tout est ME en mode doc derriere help ?
OPEN: comment gérer le multi value ?

## Parser rules
- do not allow rules after @--
- consider all args not "rule parsable" as command and args to test
- while no command is parsed allow prefix to be - for 1 char aliases or -- for aliases ?
*/

type ruleMatcher interface {
	Name() string
	//Match([]string) (bool, error)
	Mutate([]string, *model.Config) error
}

type rule[T any] struct {
	name        string
	operators   []*operator[T]
	mutater     *configMutater[T]
	aliases     []string
	prefixMask  string // @: Standard behavior: rules prefixed by PREFIX ; -: allow - and -- ;
	multiValued bool
}

func (r rule[T]) Name() string {
	return r.name
}

func (r rule[T]) Mutate([]string, *model.Config) error {
	// TODO
	return nil
}

type operator[T any] struct {
	op         string
	mapper     *mapper[T]
	validaters []*validater[T]
}

type ruleSet struct {
	name              string
	mutuallyExclusive bool
	dependsOn         []ruleMatcher
	defaults          []ruleMatcher
	rules             []ruleMatcher
	//ruleSets          []RuleSet
}

type mapper[T any] func(value string) (T, error)
type validater[T any] func(value T) error
type configMutater[T any] func(cfg *model.Config, op string, value T)

type Config struct {
	prefix   string
	rule     ruleMatcher
	operator string
	value    string
}

var (

	// ACTIONS
	global = buildRule("global", ops(noOp[string]()), nil, "g")
	suite  = buildRule("suite", ops(noOp[string](), equalSuiteName),
		mutater(),
		"s", "init", "i")
	test = buildRule("test", ops(noOp[string](), equalTestName),
		mutater(),
		"t")
	report = buildRule("report", ops(noOp[string](), equalSuiteName),
		mutater(),
		"r")
	help  = buildRule("help", ops(noOp[string]()), nil, "h")
	usage = buildRule("usage", ops(noOp[string]()), nil) // ???

	// ISOLATION
	token = buildRule("token", ops(equalString),
		mutater())
	isolation = buildRule("isolation", ops(equalString),
		mutater(),
		"isol")

	// TOKEN_ACTIONS
	printToken  = buildRule("printToken", ops(noOp[string]()), nil, "print_token")
	exportToken = buildRule("exportToken", ops(noOp[string]()), nil, "export_token")

	// PARSING
	prefix = buildRule("prefix", ops(equalString),
		mutater())

	// VERBOSITY
	quiet = buildRule("quiet", ops(noOp[bool](), equalBoolean),
		mutater(),
		"q")
	verbose = buildMvRule("verbose", ops(noOp[int](), equalUint8),
		mutater(),
		"v")
	debug = buildMvRule("debug", ops(noOp[int](), equalUint8),
		mutater(),
		"d", "x")

	// SUITE CONFIG
	fork = buildRule("fork", ops(noOp[int](), equalUint8),
		mutater())
	suiteTimeout = buildRule("suiteTimeout", ops(equalDuration),
		mutater(),
		"suite_timeout", "timeoutSuite", "timeout_suite")
	async = buildRule("async", ops(noOp[bool](), equalBoolean),
		mutater())
	stopOnFailure = buildRule("stopOnFailure", ops(noOp[bool](), equalBoolean),
		mutater(),
		"stop_on_failure")
	failuresLimit = buildRule("failuresLimit", ops(equalUint8),
		mutater(),
		"failures_limit")
	beforeSuite = buildMvRule("beforeSuite", ops(equalCmd),
		mutater(),
		"before_suite")
	afterSuite = buildMvRule("afterSuite", ops(equalCmd),
		mutater(),
		"after_suite")

	// TEST CONFIG
	wait = buildRule("wait", ops(noOp[bool](),
		mutater(),
		equalBoolean))
	ignore = buildRule("ignore", ops(noOp[bool](),
		mutater(),
		equalBoolean))
	keepStdout = buildRule("keepStdout", ops(noOp[bool](), equalBoolean),
		mutater(),
		"keep_stdout", "keepOut", "keep_out")
	keepStderr = buildRule("keepStderr", ops(noOp[bool](), equalBoolean),
		mutater(),
		"keep_stderr", "keepErr", "keep_err")
	keepOutputs = buildRule("keepOutputs", ops(noOp[bool](), equalBoolean),
		mutater(),
		"keep_outputs", "keepOuts", "keep_outs")
	timeout = buildRule("timeout", ops(equalDuration),
		mutater())
	runCount = buildRule("runCount", ops(equalUint8),
		mutater(),
		"run_count")
	mock = buildRule("mock", ops(equalMock),
		mutater())
	before = buildMvRule("before", ops(equalCmd),
		mutater())
	after = buildMvRule("after", ops(equalCmd),
		mutater())
	container = buildRule("container", ops(noOp[bool]()),
		mutater())
	dirtyContainer = buildRule("dirtyContainer", ops(noOp[bool]()),
		mutater(),
		"dirty_container")

	// REPORT CONFIG
	keepReport = buildRule("keep", ops(noOp[bool]()), nil)

	// WHERE ?
	//parallel = buildRule("parallel", nil)

	// ASSERTIONS CONFIG
	success = buildRule("success", ops(noOp[bool]()),
		mutater())
	failure = buildRule("failure", ops(noOp[bool]()),
		mutater(),
		"fail")
	exit = buildRule("exit", ops(equalUint8),
		mutater(),
		"rc")
	stdout = buildMvRule("stdout", ops(equalStringOrEmpty, equalString, notEqualString, containsString,
		notContainsString, matchString, notMatchString, equalFileContent, containsFileContent),
		mutater(),
		"out")
	stderr = buildMvRule("stderr", ops(equalStringOrEmpty, equalString, notEqualString, containsString,
		notContainsString, matchString, notMatchString, equalFileContent, containsFileContent),
		mutater(),
		"err")
	cmd = buildMvRule("cmd", ops(equalCmd),
		mutater())
	exists = buildMvRule("exists", ops(equalFilepath),
		mutater())
)

var ruleTree = []*ruleSet{
	buildMERS("action", nil, rules(global, suite, test, report, help, usage), test),
	buildRS("isolation", nil, rules(token, isolation), nil),
	buildMERS("tokenAction", nil, rules(printToken, exportToken), nil),
	buildRS("parsing", nil, rules(prefix), nil),
	buildRS("verbosity", nil, rules(quiet, verbose, debug), nil),
	buildRS("suiteConfig", rules(global, suite), rules(fork, suiteTimeout, async, stopOnFailure,
		failuresLimit, beforeSuite, afterSuite), nil),
	buildRS("testConfig", rules(global, suite, test), rules(wait, ignore, keepStdout, keepStderr,
		keepOutputs, timeout, runCount, mock, before, after, container, dirtyContainer), nil),
	buildRS("reportConfig", rules(report), rules(keepReport), nil),
	buildMERS("outcomeAssertions", rules(test), rules(success, failure, exit), success),
	buildRS("stackableAssertions", rules(test), rules(stdout, stderr, cmd, exists), nil),
}

func rules(rules ...ruleMatcher) []ruleMatcher {
	return rules
}

func ops[T any](ops ...*operator[T]) []*operator[T] {
	return ops
}

func buildRule[T any](name string, ops []*operator[T], mutater *configMutater[T], aliases ...string) *rule[T] {
	return &rule[T]{
		name:        name,
		operators:   ops,
		mutater:     mutater,
		aliases:     aliases,
		multiValued: false,
	}
}

func buildMvRule[T any](name string, ops []*operator[T], mutater *configMutater[T], aliases ...string) *rule[T] {
	return &rule[T]{
		name:        name,
		operators:   ops,
		mutater:     mutater,
		aliases:     aliases,
		multiValued: true,
	}
}

func mutater[T any](f func(*model.Config, string, T)) *configMutater[T] {
	var res configMutater[T] = func(cfg *model.Config, op string, val T) {
		//TODO
	}
	return &res
}

func buildMERS(name string, dependsOn []ruleMatcher, r []ruleMatcher, def ruleMatcher) *ruleSet {
	return &ruleSet{
		name:              name,
		mutuallyExclusive: true,
		dependsOn:         dependsOn,
		defaults:          rules(def),
		rules:             r,
	}
}

func buildRS(name string, dependsOn []ruleMatcher, r []ruleMatcher, def []ruleMatcher) *ruleSet {
	return &ruleSet{
		name:              name,
		mutuallyExclusive: false,
		dependsOn:         dependsOn,
		defaults:          def,
		rules:             r,
	}
}
