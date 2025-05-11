package display

import (
	"fmt"
	"os"
	"strings"
	"time"

	"cmdt/internal/facade"
	"cmdt/internal/model"

	"github.com/mxbossard/utilz/anzi"
	"github.com/mxbossard/utilz/formatz"
	"github.com/mxbossard/utilz/inoutz"
	"github.com/mxbossard/utilz/printz"
	"github.com/mxbossard/utilz/zlog"
)

const (
	MinReportSuiteLabelPadding = 20
)

var (
	logger = zlog.New() //slog.New(slog.NewTextHandler(os.Stderr, model.DefaultLoggerOpts))
)

type Displayer interface {
	Global(facade.GlobalContext)

	OpenSuite(facade.SuiteContext)
	CloseSuite(facade.SuiteContext)
	ClearSuite(string)

	SuiteTitle(facade.SuiteContext)

	OpenTest(facade.TestContext) TestDisplayer
	CloseTest(facade.TestContext)

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
	clearAnsiFormatter inoutz.Formatter
	outFormatter       inoutz.Formatter
	errFormatter       inoutz.Formatter
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

func (d basicDisplay) ClearSuite(name string) {
	// Nothing to do
}

func (d *basicDisplay) OpenTest(ctx facade.TestContext) TestDisplayer {
	if d.openedTest != nil {
		if ctx.Seq == d.openedTest.ctx.Seq {
			// Return the currently already opened test
			return d.openedTest
		} else {
			// Close the current open test
			d.CloseTest(d.openedTest.ctx)
		}
	}

	flusher := func() error {
		return d.Flush()
	}
	td := NewTestDisplayer(flusher, ctx, d.printer, d.notQuietPrinter, d.outFormatter, d.errFormatter)
	d.openedTest = td
	return td
}

func (d basicDisplay) TestTitle(ctx facade.TestContext) {
	d.openedTest.Title()
}

func (d basicDisplay) TestOutcome(ctx facade.TestContext, outcome model.TestOutcome) {
	d.openedTest.Outcome(outcome)
}

func (d basicDisplay) TestStdout(ctx facade.TestContext, s string) {
	d.openedTest.Stdout(s)
}

func (d basicDisplay) TestStderr(ctx facade.TestContext, s string) {
	d.openedTest.Stderr(s)
}

func (d *basicDisplay) CloseTest(ctx facade.TestContext) {
	if d.openedTest == nil {
		panic("no test currently open")
	}

	d.openedTest.Close()
	d.openedTest = nil
}

func (d basicDisplay) reportSuite(outcome model.SuiteOutcome, padding int) {
	defer d.Flush()
	testCount := outcome.TestCount
	ignoredCount := outcome.IgnoredCount
	failedCount := outcome.FailedCount
	errorCount := outcome.ErroredCount
	timeoutCount := outcome.TimeoutedCount
	passedCount := outcome.PassedCount
	tooMuchCount := outcome.TooMuchCount

	testSuite := outcome.TestSuite
	testSuiteLabel := formatz.New(TestColor, testSuite)
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
	if outcome.Outcome == model.EMPTY {
		d.printer.ColoredErrf(WarningColor, "Empty not ran    [ %s ] test suite !", testSuiteLabel)
		d.printer.Errf("\n")
	} else if outcome.Outcome == model.IGNORED {
		d.printer.ColoredErrf(WarningColor, "Ignored not ran  [ %s ] test suite !", testSuiteLabel)
		if ignoredCount > 0 {
			d.printer.ColoredErrf(WarningColor, "%s", ignoredMessage)
		}
		d.printer.Errf("\n")
	} else if failedCount == 0 && errorCount == 0 && timeoutCount == 0 {
		d.printer.ColoredErrf(SuccessColor, "Successfully ran [ %s ] test suite in %10s (%3d success)", testSuiteLabel, fmtDuration, passedCount)
		d.printer.ColoredErrf(WarningColor, "%s", ignoredMessage)
		d.printer.Errf("\n")
	} else {
		d.printer.ColoredErrf(FailureColor, "Failures running [ %s ] test suite in %10s (%3d success", testSuiteLabel, fmtDuration, passedCount)
		if failedCount > 0 {
			d.printer.ColoredErrf(FailureColor, ", %3d failures", failedCount)
		}
		if errorCount > 0 {
			d.printer.ColoredErrf(FailureColor, ", %3d errors", errorCount)
		}
		if timeoutCount > 0 {
			d.printer.ColoredErrf(FailureColor, "%3d timeouts", timeoutCount)
		}
		d.printer.ColoredErrf(FailureColor, " on %3d tests)", testCount)
		if ignoredMessage != "" {
			d.printer.ColoredErrf(WarningColor, "%s", ignoredMessage)
		}
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
	lastReportTime := globalCtx.Config.LastReportTime.GetOr(globalStartTime)
	if globalStartTime.After(lastReportTime) {
		lastReportTime = globalStartTime
	}

	sessionDuration := model.NormalizeDurationInSec(time.Since(lastReportTime))
	globalDuration := model.NormalizeDurationInSec(time.Since(globalStartTime))
	if sessionDuration == globalDuration {
		d.printer.ColoredErrf(MessageColor, "Session duration: %s\n", sessionDuration)
	} else {
		d.printer.ColoredErrf(MessageColor, "Session duration: %s (global duration: %s)\n", sessionDuration, globalDuration)
	}
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
	var errored bool
	for _, err := range errors {
		if err != nil {
			errored = true
			fmt.Fprintln(os.Stderr, err)
		}
	}
	if errored {
		os.Exit(1)
	}
}

func (d basicDisplay) GlobalErrors(ctx facade.GlobalContext, errors ...error) {
	if len(errors) == 0 {
		return
	}
	d.Errors(errors...)
}

func (d basicDisplay) SuiteErrors(ctx facade.SuiteContext, errors ...error) {
	if len(errors) == 0 {
		return
	}
	d.Errors(errors...)
}

func (d basicDisplay) TestErrors(ctx facade.TestContext, errors ...error) {
	if len(errors) == 0 {
		return
	}
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
		clearAnsiFormatter: inoutz.AnsiFormatter{AnsiFormat: anzi.Reset},
		outFormatter:       inoutz.PrefixFormatter{Prefix: fmt.Sprintf("%sout%s>", TestColor, ResetColor)},
		errFormatter:       inoutz.PrefixFormatter{Prefix: fmt.Sprintf("%serr%s>", ReportColor, ResetColor)},
		verbose:            model.DefaultVerboseLevel,
	}
	d.printer = d.notQuietPrinter
	return d
}
