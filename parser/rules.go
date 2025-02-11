package parser

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

type Rule struct {
	name        string
	operators   []*Operator
	aliases     []string
	prefixMask  string // @: Standard behavior: rules prefixed by PREFIX ; -: allow - and -- ;
	multiValued bool
}

type Operator struct {
	op         string
	validaters []Validater
}

type RuleSet struct {
	name              string
	mutuallyExclusive bool
	rules             []*Rule
	defaults          []*Rule
	//ruleSets          []RuleSet
	//dependsOn []*RuleSet
}

type Validater func(string) bool

type Config struct {
	prefix   string
	rule     Rule
	operator string
	value    string
}

var (

	// VALIDATERS
	isUint8         = buildValidater(func(val string) bool { return true })
	isUint16        = buildValidater(func(val string) bool { return true })
	isInt32         = buildValidater(func(val string) bool { return true })
	isUint32        = buildValidater(func(val string) bool { return true })
	isString        = buildValidater(func(val string) bool { return true })
	isStringOrEmpty = buildValidater(func(val string) bool { return true })
	isBoolean       = buildValidater(func(val string) bool { return true })
	isFilepath      = buildValidater(func(val string) bool { return true })
	isSuiteName     = buildValidater(func(val string) bool { return true })
	isTestName      = buildValidater(func(val string) bool { return true })
	isDuration      = buildValidater(func(val string) bool { return true })
	isCmd           = buildValidater(func(val string) bool { return true })
	isMock          = buildValidater(func(val string) bool { return true })
	//isRuleName      = buildValidater(func(val string) bool { return true })

	// OPERATORS
	noOp = buildOp("")
	// strings
	equalOneChar      = buildOp("=", isString) // TODO: implements equalString(minLength, maxLength)
	equalString       = buildOp("=", isString)
	equalEmptyString  = buildOp("=", isStringOrEmpty)
	notEqualString    = buildOp("!=", isString)
	containsString    = buildOp(":", isString)
	notContainsString = buildOp("!:", isString)
	matchString       = buildOp("~", isString)
	notMatchString    = buildOp("!~", isString)
	// files
	equalFileContent    = buildOp("@=", isFilepath)
	containsFileContent = buildOp("@:", isFilepath)
	equalFilepath       = buildOp("=", isFilepath)
	// numbers
	equalUint8 = buildOp("=", isUint8) // TODO: implements equalInt(min, max)
	// booleans
	equalBoolean = buildOp("=", isBoolean)
	// durations
	equalDuration = buildOp("=", isDuration)
	// patterns
	equalSuiteName = buildOp("=", isSuiteName)
	equalTestName  = buildOp("=", isTestName)
	// cmd & mock
	equalCmd  = buildOp("=", isCmd)
	equalMock = buildOp("=", isMock)
	//equalRuleName       = buildOp("=", isRuleName)

	// ACTIONS
	global = buildRule("global", nil, "g")
	suite  = buildRule("suite", ops(noOp, equalSuiteName), "s", "init", "i")
	test   = buildRule("test", ops(noOp, equalTestName), "t")
	report = buildRule("report", ops(noOp, equalSuiteName), "r")
	help   = buildRule("help", ops(noOp), "h")
	usage  = buildRule("usage", nil) // ???

	// ISOLATION
	token     = buildRule("token", ops(equalString))
	isolation = buildRule("isolation", ops(equalString), "isol")

	// TOKEN_ACTIONS
	printToken  = buildRule("printToken", nil, "print_token")
	exportToken = buildRule("exportToken", nil, "export_token")

	// PARSING
	prefix = buildRule("prefix", ops(equalString))

	// VERBOSITY
	quiet   = buildRule("quiet", ops(noOp, equalBoolean), "q")
	verbose = buildMvRule("verbose", ops(noOp, equalUint8), "v")
	debug   = buildMvRule("debug", ops(noOp, equalUint8), "d", "x")

	// SUITE CONFIG
	fork          = buildRule("fork", ops(noOp, equalUint8))
	suiteTimeout  = buildRule("suiteTimeout", ops(equalDuration), "suite_timeout", "timeoutSuite", "timeout_suite")
	async         = buildRule("async", ops(noOp, equalBoolean))
	stopOnFailure = buildRule("stopOnFailure", ops(noOp, equalBoolean), "stop_on_failure")
	failuresLimit = buildRule("failuresLimit", ops(equalUint8), "failures_limit")
	beforeSuite   = buildMvRule("beforeSuite", ops(equalCmd), "before_suite")
	afterSuite    = buildMvRule("afterSuite", ops(equalCmd), "after_suite")

	// TEST CONFIG
	wait           = buildRule("wait", ops(noOp, equalBoolean))
	ignore         = buildRule("ignore", ops(noOp, equalBoolean))
	keepStdout     = buildRule("keepStdout", ops(noOp, equalBoolean), "keep_stdout", "keepOut", "keep_out")
	keepStderr     = buildRule("keepStderr", ops(noOp, equalBoolean), "keep_stderr", "keepErr", "keep_err")
	keepOutputs    = buildRule("keepOutputs", ops(noOp, equalBoolean), "keep_outputs", "keepOuts", "keep_outs")
	timeout        = buildRule("timeout", ops(equalDuration))
	runCount       = buildRule("runCount", ops(equalUint8), "run_count")
	mock           = buildRule("mock", ops(equalMock))
	before         = buildMvRule("before", ops(equalCmd))
	after          = buildMvRule("after", ops(equalCmd))
	container      = buildRule("container", nil)
	dirtyContainer = buildRule("dirtyContainer", nil, "dirty_container")

	// REPORT CONFIG
	keepReport = buildRule("keep", nil)

	// WHERE ?
	//parallel = buildRule("parallel", nil)

	// ASSERTIONS CONFIG
	success = buildRule("success", nil)
	failure = buildRule("failure", nil, "fail")
	exit    = buildRule("exit", ops(equalUint8), "rc")
	stdout  = buildMvRule("stdout", ops(equalEmptyString, equalString, notEqualString, containsString, notContainsString, matchString, notMatchString, equalFileContent, containsFileContent), "out")
	stderr  = buildMvRule("stderr", ops(equalEmptyString, equalString, notEqualString, containsString, notContainsString, matchString, notMatchString, equalFileContent, containsFileContent), "err")
	cmd     = buildMvRule("cmd", ops(equalCmd))
	exists  = buildMvRule("exists", ops(equalFilepath))
)

var ruleConfig = []*RuleSet{
	buildMERS("action", rules(global, suite, test, report, help, usage), test),
	buildRS("isolation", rules(token, isolation), nil),
	buildMERS("tokenAction", rules(printToken, exportToken), nil),
	buildRS("parsing", rules(prefix), nil),
	buildRS("verbosity", rules(quiet, verbose, debug), nil),
	buildRS("suiteConfig", rules(fork, suiteTimeout, async, stopOnFailure, failuresLimit,
		beforeSuite, afterSuite), nil),
	buildRS("testConfig", rules(wait, ignore, keepStdout, keepStderr, keepOutputs, timeout, runCount,
		mock, before, after, container, dirtyContainer), nil),
	buildRS("reportConfig", rules(keepReport), nil),
	buildMERS("outcomeAssertions", rules(success, failure, exit), success),
	buildRS("assertions", rules(stdout, stderr, cmd, exists), nil),
}

func rules(rules ...*Rule) []*Rule {
	return rules
}

func ops(ops ...*Operator) []*Operator {
	return ops
}

func buildRule(name string, ops []*Operator, aliases ...string) *Rule {
	return &Rule{
		name:        name,
		operators:   ops,
		aliases:     aliases,
		multiValued: false,
	}
}

func buildMvRule(name string, ops []*Operator, aliases ...string) *Rule {
	return &Rule{
		name:        name,
		operators:   ops,
		aliases:     aliases,
		multiValued: true,
	}
}

func buildOp(op string, validaters ...Validater) *Operator {
	return &Operator{
		op:         op,
		validaters: validaters,
	}
}

func buildValidater(f func(val string) bool) Validater {
	return f
}

func buildMERS(name string, rules []*Rule, def *Rule) *RuleSet {
	return &RuleSet{
		name:              name,
		mutuallyExclusive: true,
		rules:             rules,
		defaults:          []*Rule{def},
	}
}

func buildRS(name string, rules []*Rule, def []*Rule) *RuleSet {
	return &RuleSet{
		name:              name,
		mutuallyExclusive: false,
		rules:             rules,
		defaults:          def,
	}
}
