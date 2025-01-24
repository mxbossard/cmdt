package display

import (
	"fmt"
	"os"
	"strings"
	"time"

	"mby.fr/cmdtest/facade"
	"mby.fr/cmdtest/model"
	"mby.fr/utils/ansi"
	"mby.fr/utils/format"
	"mby.fr/utils/inout"
	"mby.fr/utils/printz"
)

const (
	MinReportSuiteLabelPadding = 20
)

type Displayer interface {
	Global(facade.GlobalContext)

	OpenSuite(facade.SuiteContext)
	CloseSuite(facade.SuiteContext)

	SuiteTitle(facade.SuiteContext)

	OpenTest(facade.TestContext) TestDisplayer
	CloseTest(facade.TestContext)

	// FIXME: remove following which are implemented in TestDisplayer
	//TestTitle(facade.TestContext)
	//TestOutcome(facade.TestContext, model.TestOutcome)
	//TestStdout(facade.TestContext, string)
	//TestStderr(facade.TestContext, string)

	ReportSuite(model.SuiteOutcome)
	ReportSuites([]model.SuiteOutcome)
	ReportAllFooter(facade.GlobalContext)
	TooMuchFailures(facade.SuiteContext, string)

	Errors(...error)
	GlobalErrors(facade.GlobalContext, ...error)
	SuiteErrors(facade.SuiteContext, ...error)
	TestErrors(facade.TestContext, ...error)

	Flush() error
	Quiet(bool)
	SetVerbose(model.VerboseLevel)
}

type basicDisplay struct {
	printer            printz.Printer
	notQuietPrinter    printz.Printer
	clearAnsiFormatter inout.Formatter
	outFormatter       inout.Formatter
	errFormatter       inout.Formatter
	verbose            model.VerboseLevel
	openedTest         *basicTestDisplayer
}

func (d basicDisplay) Global(ctx facade.GlobalContext) {
	defer d.Flush()
	// Do nothing ?
	if ctx.Config.Verbose.Get() >= model.SHOW_FAILED_OUTS {
		d.printer.ColoredErrf(MessageColor, "## New config (token: %s)\n", ctx.Token)
	}
}

func (d basicDisplay) SuiteTitle(ctx facade.SuiteContext) {
	defer d.Flush()
	if ctx.Config.Verbose.Get() >= model.SHOW_PASSED {
		d.printer.ColoredErrf(MessageColor, "## Test suite [%s] (token: %s)\n", ctx.Config.TestSuite.Get(), ctx.Token)
	}
}

func (d basicDisplay) OpenSuite(ctx facade.SuiteContext) {
	// Nothing to do
}

func (d basicDisplay) CloseSuite(ctx facade.SuiteContext) {
	// Nothing to do
}

func (d *basicDisplay) OpenTest(ctx facade.TestContext) TestDisplayer {
	if d.openedTest != nil {
		// Close the current open test
		d.CloseTest(d.openedTest.ctx)
	}

	// bufPrinter := printz.New(d.printer.Outputs())
	// bufNotQuietPrinter := printz.New(d.notQuietPrinter.Outputs())
	//bufPrinter := printz.Buffered(d.printer)
	//bufNotQuietPrinter := printz.Buffered(d.notQuietPrinter)
	td := NewTestDisplayer(d, ctx, d.printer, d.notQuietPrinter, d.outFormatter, d.errFormatter)
	d.openedTest = td
	return td
}

func (d basicDisplay) TestTitle(ctx facade.TestContext) {
	d.openedTest.Title()
}

/*
func (d basicDisplay) TestTitle0(ctx facade.TestContext) {
	if ctx.Config.Verbose.Get() == model.SHOW_REPORTS_ONLY {
		return
	}
	defer d.Flush()

	cfg := ctx.Config
	timecode := int(time.Since(cfg.SuiteStartTime.Get()).Milliseconds())
	qualifiedName := TestQualifiedName(ctx, TestColor)
	qualifiedName = format.TruncateRight(qualifiedName, MaxTestNameLength)

	seq := ctx.Seq
	title := fmt.Sprintf("[%05d] Test %s #%02d... ", timecode, qualifiedName, seq)
	title = format.PadRight(title, MaxTestNameLength+23)

	if ctx.Config.Verbose.Get() > model.SHOW_FAILED_OUTS && ctx.Config.Ignore.Is(true) {
		if ctx.Config.Verbose.Get() >= model.SHOW_FAILED_OUTS {
			d.printer.ColoredErrf(WarningColor, title)
		}
		return
	}

	if ctx.Config.Verbose.Get() >= model.SHOW_PASSED {
		d.printer.ColoredErrf(TestColor, title)
	}

	/*
		if ctx.Config.Verbose.Get() <= model.SHOW_PASSED {
			d.printer.ColoredErrf(testColor, title)
			if ctx.Config.KeepStdout.Is(true) || ctx.Config.KeepStderr.Is(true) {
				// NewLine because we expect cmd outputs
				//d.printer.Errf("\n")
			}
		}
*/
// }

func (d basicDisplay) TestOutcome(ctx facade.TestContext, outcome model.TestOutcome) {
	d.openedTest.Outcome(outcome)
}

/*
	func (d basicDisplay) TestOutcome0(ctx facade.TestContext, outcome model.TestOutcome) {
		if ctx.Config.Verbose.Get() == model.SHOW_REPORTS_ONLY {
			return
		}
		// FIXME get outcome from ctx
		cfg := ctx.Config
		verbose := cfg.Verbose.Get()
		testDuration := outcome.Duration
		defer d.Flush()

		if verbose < model.SHOW_PASSED && outcome.Outcome != model.PASSED && outcome.Outcome != model.IGNORED {
			// Print back test title not printed yed
			clone := ctx
			clone.Config.Verbose.Set(model.SHOW_PASSED)
			d.TestTitle(clone)
		}

		switch outcome.Outcome {
		case model.PASSED:
			if verbose >= model.SHOW_PASSED {
				d.printer.ColoredErrf(SuccessColor, "PASSED")
				d.printer.Errf(" (in %s)\n", testDuration)
			}
		case model.FAILED:
			d.printer.ColoredErrf(FailureColor, "FAILED")
			d.printer.Errf(" (in %s)\n", testDuration)
		case model.TIMEOUT:
			d.printer.ColoredErrf(FailureColor, "TIMEOUT")
			d.printer.Errf(" (after %s)\n", ctx.Config.Timeout.Get())
		case model.ERRORED:
			d.printer.ColoredErrf(WarningColor, "ERRORED")
			d.printer.Errf(" (not executed)\n")
		case model.IGNORED:
			if verbose > model.SHOW_FAILED_OUTS {
				d.printer.ColoredErrf(WarningColor, "IGNORED")
				d.printer.Err("\n")
			}
		default:
		}

		if verbose >= model.SHOW_FAILED_ONLY && outcome.Outcome != model.PASSED && outcome.Outcome != model.IGNORED || verbose >= model.SHOW_PASSED_OUTS {
			d.printer.Errf("\tExecuting cmd: \t\t[%s]\n", CmdTitle(ctx))
		}

		if outcome.Err != nil {
			d.printer.ColoredErrf(model.ErrorColor, "\t%s\n", outcome.Err)
		}

		if len(outcome.AssertionResults) > 0 {
			for _, asseriontResult := range outcome.AssertionResults {
				d.assertionResult(asseriontResult)
			}
		}

		if verbose >= model.SHOW_FAILED_OUTS && (len(outcome.AssertionResults) > 0 || outcome.Outcome == model.TIMEOUT || outcome.Outcome == model.ERRORED) || verbose >= model.SHOW_PASSED_OUTS {
			d.printer.Errf(d.outFormatter.Format(outcome.Stdout))
			d.printer.Errf(d.errFormatter.Format(outcome.Stderr))
			d.printer.Errf("\n")
		}

}
*/
func (d basicDisplay) TestStdout(ctx facade.TestContext, s string) {
	d.openedTest.Stdout(s)
}

/*
	func (d basicDisplay) TestStdout0(ctx facade.TestContext, s string) {
		if s != "" {
			d.notQuietPrinter.Out(d.outFormatter.Format(s))
		}
	}
*/
func (d basicDisplay) TestStderr(ctx facade.TestContext, s string) {
	d.openedTest.Stderr(s)
}

/*
	func (d basicDisplay) TestStderr0(ctx facade.TestContext, s string) {
		if s != "" {
			d.notQuietPrinter.Err(d.errFormatter.Format(s))
		}
	}
*/
func (d *basicDisplay) CloseTest(ctx facade.TestContext) {
	if d.openedTest == nil {
		panic("no test currently open")
	}

	d.openedTest.Close()
	d.openedTest = nil
}

func (d basicDisplay) assertionResult(result model.AssertionResult) {
	defer d.Flush()
	hlClr := ReportColor
	//log.Printf("failedResult: %v\n", result)
	assertPrefix := result.Rule.Prefix
	assertName := result.Rule.Name
	assertOp := result.Rule.Op
	expected := result.Rule.Expected
	got := result.Value

	if result.ErrMessage != "" {
		d.printer.ColoredErrf(ErrorColor, result.ErrMessage+"\n")
	}

	assertLabel := format.Sprintf(TestColor, "%s%s", assertPrefix, assertName)

	if assertName == "success" || assertName == "fail" {
		d.printer.Errf("\t%sExpected%s %s\n", hlClr, ResetColor, assertLabel)
		//d.Stdout(cmd.StdoutRecord())
		//d.Stderr(cmd.StderrRecord())
		/*
			if cmd.StderrRecord() != "" {
				d.printer.Errf("sdterr> %s\n", cmd.StderrRecord())
			}
		*/
		return
	} else if assertName == "cmd" {
		d.printer.Errf("\t%sExpected%s %s=%s to succeed\n", hlClr, ResetColor, assertLabel, expected)
		return
	} else if assertName == "exists" {
		d.printer.Errf("\t%sExpected%s file %s=%s file to exists\n", hlClr, ResetColor, assertLabel, expected)
		return
	}

	var stringifiedGot string
	if expected != got {
		expected = strings.ReplaceAll(expected, "\n", "\\n")
		if s, ok := got.(string); ok {
			s = strings.ReplaceAll(s, "\n", "\\n")
			got = s

			stringifiedGot = ansi.TruncateMid(s, 100, "[...]")
		}

		if assertOp == "=" || assertOp == "@=" {
			d.printer.Errf("\t%sExpected%s %s \n\t\t%sto be%s: \t\t[%s]\n\t\t%sbut got%s: \t[%v]\n", hlClr, ResetColor, assertLabel, hlClr, ResetColor, expected, hlClr, ResetColor, stringifiedGot)
		} else if assertOp == ":" || assertOp == "@:" {
			d.printer.Errf("\t%sExpected%s %s \n\t\t%sto contains%s: \t[%s]\n\t\t%sbut got%s: \t[%v]\n", hlClr, ResetColor, assertLabel, hlClr, ResetColor, expected, hlClr, ResetColor, stringifiedGot)
		} else if assertOp == "!:" {
			d.printer.Errf("\t%sExpected%s %s \n\t\t%snot to contains%s: \t[%s]\n\t\t%sbut got%s: \t[%v]\n", hlClr, ResetColor, assertLabel, hlClr, ResetColor, expected, hlClr, ResetColor, stringifiedGot)
		} else if assertOp == "~" {
			d.printer.Errf("\t%sExpected%s %s \n\t\t%sto match%s: \t[%s]\n\t\t%sbut got%s: \t[%v]\n", hlClr, ResetColor, assertLabel, hlClr, ResetColor, expected, hlClr, ResetColor, stringifiedGot)
		} else if assertOp == "!~" {
			d.printer.Errf("\t%sExpected%s %s \n\t\t%snot to match%s: \t[%s]\n\t\t%sbut got%s: \t[%v]\n", hlClr, ResetColor, assertLabel, hlClr, ResetColor, expected, hlClr, ResetColor, stringifiedGot)
		}
	} else {
		d.printer.Errf("assertion %s%s%s failed\n", assertLabel, assertOp, expected)
	}
}

func (d basicDisplay) reportSuite(outcome model.SuiteOutcome, padding int) {
	defer d.Flush()
	testCount := outcome.TestCount
	ignoredCount := outcome.IgnoredCount
	failedCount := outcome.FailedCount
	errorCount := outcome.ErroredCount
	passedCount := outcome.PassedCount
	tooMuchCount := outcome.TooMuchCount

	testSuite := outcome.TestSuite
	testSuiteLabel := format.New(TestColor, testSuite)
	testSuiteLabel.LeftPad = padding

	// if ctx.Config.Verbose.Get() >= model.SHOW_PASSED {
	// 	d.printer.ColoredErrf(messageColor, "Reporting [%s] test suite (%s) ...\n", testSuite, ctx.Token)
	// }

	ignoredMessage := ""
	if ignoredCount > 0 {
		ignoredMessage = fmt.Sprintf(" (%d ignored)", ignoredCount)
	}
	duration := outcome.Duration
	fmtDuration := NormalizeDurationInSec(duration)
	if failedCount == 0 && errorCount == 0 {
		d.printer.ColoredErrf(SuccessColor, "Successfuly ran  [ %s ] test suite in %10s (%3d success)", testSuiteLabel, fmtDuration, passedCount)
		d.printer.ColoredErrf(WarningColor, "%s", ignoredMessage)
		d.printer.Errf("\n")
	} else {
		d.printer.ColoredErrf(FailureColor, "Failures running [ %s ] test suite in %10s (%3d success, %3d failures, %3d errors on %3d tests)", testSuiteLabel, fmtDuration, passedCount, failedCount, errorCount, testCount)
		d.printer.ColoredErrf(WarningColor, "%s", ignoredMessage)
		d.printer.Errf("\n")
		for _, report := range outcome.FailureReports {
			report = strings.TrimSpace(report)
			if report != "" {
				//report = format.PadRight(report, 60)
				d.printer.ColoredErrf(ReportColor, "%s\n", report)
			}
		}
	}
	if tooMuchCount > 0 {
		d.printer.ColoredErrf(WarningColor, "Too much failures (%d tests not executed)\n", tooMuchCount)
	}
}

func (d basicDisplay) ReportSuite(outcome model.SuiteOutcome) {
	d.reportSuite(outcome, MinReportSuiteLabelPadding)
}

func (d basicDisplay) ReportSuites(outcomes []model.SuiteOutcome) {
	maxSuiteNameSize := 0
	for _, outcome := range outcomes {
		if len(outcome.TestSuite) > maxSuiteNameSize {
			maxSuiteNameSize = len(outcome.TestSuite)
		}
	}
	for _, outcome := range outcomes {
		d.reportSuite(outcome, max(MinReportSuiteLabelPadding, maxSuiteNameSize))
	}
}

func (d basicDisplay) ReportAllFooter(globalCtx facade.GlobalContext) {
	defer d.Flush()

	globalStartTime := globalCtx.Config.GlobalStartTime.Get()
	globalDuration := model.NormalizeDurationInSec(time.Since(globalStartTime))
	d.printer.ColoredErrf(MessageColor, "Global duration time: %s\n", globalDuration)
}

func (d basicDisplay) TooMuchFailures(ctx facade.SuiteContext, testSuite string) {
	if ctx.Config.Verbose.Get() == model.SHOW_REPORTS_ONLY {
		return
	}
	defer d.Flush()
	d.printer.ColoredErrf(WarningColor, "Too much failure for [%s] test suite. Stop testing.\n", testSuite)
}

func (d basicDisplay) Errors(errors ...error) {
	//  An Error is Fatal
	for _, err := range errors {
		fmt.Fprintln(os.Stderr, err)
	}
	os.Exit(1)
}

func (d basicDisplay) GlobalErrors(ctx facade.GlobalContext, errors ...error) {
	if d.openedTest != nil {
		// Delegate error display to test display
		d.openedTest.Errors(errors...)
	} else {
		d.Errors(errors...)
	}
}

func (d basicDisplay) SuiteErrors(ctx facade.SuiteContext, errors ...error) {
	if d.openedTest != nil {
		// Delegate error display to test display
		d.openedTest.Errors(errors...)
	} else {
		d.Errors(errors...)
	}
}

func (d basicDisplay) TestErrors(ctx facade.TestContext, errors ...error) {
	if d.openedTest != nil {
		d.openedTest.Errors(errors...)
	} else {
		d.SuiteErrors(ctx.SuiteContext, errors...)
	}
}

func (d basicDisplay) Flush() error {
	return d.printer.Flush()
}

func (d *basicDisplay) Quiet(quiet bool) {
	if quiet {
		d.printer = printz.NewDiscarding()
	} else {
		d.printer = d.notQuietPrinter
	}
}

func (d *basicDisplay) SetVerbose(level model.VerboseLevel) {
	d.verbose = level
}

func New() *basicDisplay {
	d := &basicDisplay{
		notQuietPrinter:    printz.NewStandard(),
		clearAnsiFormatter: inout.AnsiFormatter{AnsiFormat: ansi.Reset},
		outFormatter:       inout.PrefixFormatter{Prefix: fmt.Sprintf("%sout%s>", TestColor, ResetColor)},
		errFormatter:       inout.PrefixFormatter{Prefix: fmt.Sprintf("%serr%s>", ReportColor, ResetColor)},
		verbose:            model.DefaultVerboseLevel,
	}
	d.printer = d.notQuietPrinter
	return d
}
