package display

import (
	"fmt"
	"io"
	"strings"
	"time"

	"mby.fr/cmdtest/facade"
	"mby.fr/cmdtest/model"
	"mby.fr/cmdtest/repo"
	"mby.fr/utils/ansi"
	"mby.fr/utils/filez"
	"mby.fr/utils/format"
	"mby.fr/utils/inout"
	"mby.fr/utils/printz"
	"mby.fr/utils/zlog"
)

var (
	logger = zlog.New()
)

type asyncDisplay struct {
	token, isolation   string
	printers           *asyncPrinters
	stdPrinter         printz.Printer
	clearAnsiFormatter inout.Formatter
	outFormatter       inout.Formatter
	errFormatter       inout.Formatter
	verbose            model.VerboseLevel
	quiet              bool
}

func (d asyncDisplay) Global(ctx facade.GlobalContext) {
	if d.quiet {
		return
	}
	if ctx.Config.Verbose.Get() >= model.SHOW_FAILED_OUTS {
		printer := d.printers.printer("", 0)
		printer.ColoredErrf(messageColor, "## New config (token: %s)\n", ctx.Token)
	}
}

func (d asyncDisplay) Suite(ctx facade.SuiteContext) {
	if d.quiet {
		return
	}
	if ctx.Config.Verbose.Get() >= model.SHOW_PASSED {
		suite := ctx.Config.TestSuite.Get()
		printer := d.printers.printer(suite, 0)
		printer.ColoredErrf(messageColor, "## Test suite [%s] (token: %s)\n", suite, ctx.Token)
	}
}

func (d asyncDisplay) TestTitle(ctx facade.TestContext) {
	if d.quiet {
		return
	}
	if ctx.Config.Verbose.Get() == model.SHOW_REPORTS_ONLY {
		return
	}

	seq := ctx.Seq

	cfg := ctx.Config
	timecode := int(time.Since(cfg.SuiteStartTime.Get()).Milliseconds())
	qualifiedName := testQualifiedName(ctx, testColor)
	qualifiedName = format.TruncateRight(qualifiedName, MaxTestNameLength)

	title := fmt.Sprintf("[%05d] Test %s #%02d... ", timecode, qualifiedName, seq)
	title = format.PadRight(title, MaxTestNameLength+23)

	printer := d.printers.printer(cfg.TestSuite.Get(), int(seq))

	if ctx.Config.Verbose.Get() > model.SHOW_FAILED_OUTS && ctx.Config.Ignore.Is(true) {
		if ctx.Config.Verbose.Get() >= model.SHOW_FAILED_OUTS {
			printer.ColoredErrf(warningColor, title)
		}
		return
	}

	if ctx.Config.Verbose.Get() >= model.SHOW_PASSED {
		printer.ColoredErrf(testColor, title)
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
}

func (d asyncDisplay) TestOutcome(ctx facade.TestContext, outcome model.TestOutcome) {
	if d.quiet {
		return
	}
	if ctx.Config.Verbose.Get() == model.SHOW_REPORTS_ONLY {
		return
	}
	// FIXME get outcome from ctx
	cfg := ctx.Config
	suite := cfg.TestSuite.Get()
	verbose := cfg.Verbose.Get()
	testDuration := outcome.Duration

	if verbose < model.SHOW_PASSED && outcome.Outcome != model.PASSED && outcome.Outcome != model.IGNORED {
		// Print back test title not printed yed
		clone := ctx
		clone.Config.Verbose.Set(model.SHOW_PASSED)
		d.TestTitle(clone)
	}

	printer := d.printers.printer(suite, int(outcome.Seq))
	defer d.printers.testEnded(suite, int(outcome.Seq))

	switch outcome.Outcome {
	case model.PASSED:
		if verbose >= model.SHOW_PASSED {
			printer.ColoredErrf(successColor, "PASSED")
			printer.Errf(" (in %s)\n", testDuration)
		}
	case model.FAILED:
		printer.ColoredErrf(failureColor, "FAILED")
		printer.Errf(" (in %s)\n", testDuration)
	case model.TIMEOUT:
		printer.ColoredErrf(failureColor, "TIMEOUT")
		printer.Errf(" (after %s)\n", ctx.Config.Timeout.Get())
	case model.ERRORED:
		printer.ColoredErrf(warningColor, "ERRORED")
		printer.Errf(" (not executed)\n")
	case model.IGNORED:
		if verbose > model.SHOW_FAILED_OUTS {
			printer.ColoredErrf(warningColor, "IGNORED")
			printer.Err("\n")
		}
	default:
	}

	if verbose >= model.SHOW_FAILED_ONLY && outcome.Outcome != model.PASSED && outcome.Outcome != model.IGNORED || verbose >= model.SHOW_PASSED_OUTS {
		printer.Errf("\tExecuting cmd: \t\t[%s]\n", cmdTitle(ctx))
	}

	if outcome.Err != nil {
		printer.ColoredErrf(model.ErrorColor, "\t%s\n", outcome.Err)
	}

	if len(outcome.AssertionResults) > 0 {
		for _, asseriontResult := range outcome.AssertionResults {
			d.assertionResult(printer, asseriontResult)
		}
	}

	if verbose >= model.SHOW_FAILED_OUTS && (len(outcome.AssertionResults) > 0 || outcome.Outcome == model.TIMEOUT || outcome.Outcome == model.ERRORED) || verbose >= model.SHOW_PASSED_OUTS {
		printer.Errf(d.outFormatter.Format(outcome.Stdout))
		printer.Errf(d.errFormatter.Format(outcome.Stderr))
		printer.Errf("\n")
	}

}

func (d asyncDisplay) TestStdout(ctx facade.TestContext, s string) {
	if s != "" {
		printer := d.printers.printer(ctx.Config.TestSuite.Get(), int(ctx.Seq))
		printer.Err(d.outFormatter.Format(s))
	}
}

func (d asyncDisplay) TestStderr(ctx facade.TestContext, s string) {
	if s != "" {
		printer := d.printers.printer(ctx.Config.TestSuite.Get(), int(ctx.Seq))
		printer.Err(d.errFormatter.Format(s))
	}
}

func (d asyncDisplay) assertionResult(printer printz.Printer, result model.AssertionResult) {
	hlClr := reportColor
	//log.Printf("failedResult: %v\n", result)
	assertPrefix := result.Rule.Prefix
	assertName := result.Rule.Name
	assertOp := result.Rule.Op
	expected := result.Rule.Expected
	got := result.Value

	if result.ErrMessage != "" {
		printer.ColoredErrf(errorColor, result.ErrMessage+"\n")
	}

	assertLabel := format.Sprintf(testColor, "%s%s", assertPrefix, assertName)

	if assertName == "success" || assertName == "fail" {
		printer.Errf("\t%sExpected%s %s\n", hlClr, resetColor, assertLabel)
		//d.Stdout(cmd.StdoutRecord())
		//d.Stderr(cmd.StderrRecord())
		/*
			if cmd.StderrRecord() != "" {
				d.printer.Errf("sdterr> %s\n", cmd.StderrRecord())
			}
		*/
		return
	} else if assertName == "cmd" {
		printer.Errf("\t%sExpected%s %s=%s to succeed\n", hlClr, resetColor, assertLabel, expected)
		return
	} else if assertName == "exists" {
		printer.Errf("\t%sExpected%s file %s=%s file to exists\n", hlClr, resetColor, assertLabel, expected)
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
			printer.Errf("\t%sExpected%s %s \n\t\t%sto be%s: \t\t[%s]\n\t\t%sbut got%s: \t[%v]\n", hlClr, resetColor, assertLabel, hlClr, resetColor, expected, hlClr, resetColor, stringifiedGot)
		} else if assertOp == ":" || assertOp == "@:" {
			printer.Errf("\t%sExpected%s %s \n\t\t%sto contains%s: \t[%s]\n\t\t%sbut got%s: \t[%v]\n", hlClr, resetColor, assertLabel, hlClr, resetColor, expected, hlClr, resetColor, stringifiedGot)
		} else if assertOp == "!:" {
			printer.Errf("\t%sExpected%s %s \n\t\t%snot to contains%s: \t[%s]\n\t\t%sbut got%s: \t[%v]\n", hlClr, resetColor, assertLabel, hlClr, resetColor, expected, hlClr, resetColor, stringifiedGot)
		} else if assertOp == "~" {
			printer.Errf("\t%sExpected%s %s \n\t\t%sto match%s: \t[%s]\n\t\t%sbut got%s: \t[%v]\n", hlClr, resetColor, assertLabel, hlClr, resetColor, expected, hlClr, resetColor, stringifiedGot)
		} else if assertOp == "!~" {
			printer.Errf("\t%sExpected%s %s \n\t\t%snot to match%s: \t[%s]\n\t\t%sbut got%s: \t[%v]\n", hlClr, resetColor, assertLabel, hlClr, resetColor, expected, hlClr, resetColor, stringifiedGot)
		}
	} else {
		printer.Errf("assertion %s%s%s failed\n", assertLabel, assertOp, expected)
	}
}

func (d asyncDisplay) reportSuite(outcome model.SuiteOutcome, padding int) {
	testCount := outcome.TestCount
	ignoredCount := outcome.IgnoredCount
	failedCount := outcome.FailedCount
	errorCount := outcome.ErroredCount
	passedCount := outcome.PassedCount
	tooMuchCount := outcome.TooMuchCount

	testSuite := outcome.TestSuite
	testSuiteLabel := format.New(testColor, testSuite)
	testSuiteLabel.LeftPad = padding

	// if ctx.Config.Verbose.Get() >= model.SHOW_PASSED {
	// 	d.printer.ColoredErrf(messageColor, "Reporting [%s] test suite (%s) ...\n", testSuite, ctx.Token)
	// }

	//printer := d.stdPrinter // Do not print async
	printer := d.printers.printer(testSuite, 99)

	ignoredMessage := ""
	if ignoredCount > 0 {
		ignoredMessage = fmt.Sprintf(" (%d ignored)", ignoredCount)
	}
	duration := outcome.Duration
	fmtDuration := NormalizeDurationInSec(duration)
	if failedCount == 0 && errorCount == 0 {
		printer.ColoredErrf(successColor, "Successfuly ran  [ %s ] test suite in %10s (%3d success)", testSuiteLabel, fmtDuration, passedCount)
		printer.ColoredErrf(warningColor, "%s", ignoredMessage)
		printer.Errf("\n")
	} else {
		printer.ColoredErrf(failureColor, "Failures running [ %s ] test suite in %10s (%3d success, %3d failures, %3d errors on %3d tests)", testSuiteLabel, fmtDuration, passedCount, failedCount, errorCount, testCount)
		printer.ColoredErrf(warningColor, "%s", ignoredMessage)
		printer.Errf("\n")
		for _, report := range outcome.FailureReports {
			report = strings.TrimSpace(report)
			if report != "" {
				//report = format.PadRight(report, 60)
				printer.ColoredErrf(reportColor, "%s\n", report)
			}
		}
	}
	if tooMuchCount > 0 {
		printer.ColoredErrf(warningColor, "Too much failures (%d tests not executed)\n", tooMuchCount)
	}
}

func (d asyncDisplay) ReportSuite(outcome model.SuiteOutcome) {
	if d.quiet {
		return
	}
	d.reportSuite(outcome, MinReportSuiteLabelPadding)
}

func (d asyncDisplay) ReportSuites(outcomes []model.SuiteOutcome) {
	if d.quiet {
		return
	}
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

func (d asyncDisplay) ReportAllFooter(globalCtx facade.GlobalContext) {
	if d.quiet {
		return
	}
	printer := d.stdPrinter // Do not print async
	globalStartTime := globalCtx.Config.GlobalStartTime.Get()
	globalDuration := model.NormalizeDurationInSec(time.Since(globalStartTime))
	printer.ColoredErrf(messageColor, "Global duration time: %s\n", globalDuration)
}

func (d asyncDisplay) TooMuchFailures(ctx facade.SuiteContext, testSuite string) {
	if d.quiet {
		return
	}
	if ctx.Config.Verbose.Get() == model.SHOW_REPORTS_ONLY {
		return
	}
	printer := d.printers.printer(testSuite, 0)
	printer.ColoredErrf(warningColor, "Too much failure for [%s] test suite. Stop testing.\n", testSuite)
}

func (d asyncDisplay) Error(err error) {

}

func (d asyncDisplay) Flush() error {
	return nil
}

func (d *asyncDisplay) Quiet(quiet bool) {
	d.quiet = quiet
}

func (d *asyncDisplay) SetVerbose(level model.VerboseLevel) {
	d.verbose = level
}

func (d *asyncDisplay) DisplayRecorded(suite string) error {
	p := logger.PerfTimer("suite", suite)
	defer p.End()

	stdoutFile, stderrFile, err := repo.DaemonSuiteReportFilepathes(suite, d.token, d.isolation)
	if err != nil {
		return err
	}

	logger.Debug("displaying recorded outs", "suite", suite,
		"outFile", stdoutFile,
		"out", func() string { s, _ := filez.ReadString(stdoutFile); return s },
		"errFile", stderrFile,
		"err", func() string { s, _ := filez.ReadString(stderrFile); return s })

	err = d.printers.flush(suite, true)
	if err != nil {
		return err
	}

	outR, errR, err := newFileReaders(stdoutFile, stderrFile)
	if err != nil {
		return err
	}

	_, err = io.Copy(d.stdPrinter.Outputs().Out(), outR)
	if err != nil {
		return err
	}
	_, err = io.Copy(d.stdPrinter.Outputs().Err(), errR)
	if err != nil {
		return err
	}
	d.stdPrinter.Outputs().Flush()
	return nil
}

func (d *asyncDisplay) DisplayAllRecorded() (err error) {
	p := logger.PerfTimer()
	defer p.End()

	/*
		err = d.DisplayRecorded("")
		if err != nil {
			return
		}
	*/
	recordedSuites := d.printers.recordedSuites()
	for _, suite := range recordedSuites {
		err = d.DisplayRecorded(suite)
		if err != nil {
			return
		}
	}
	return
}

func (d *asyncDisplay) StartDisplayRecorded(suite string) {
	// Launch suitepPrinters flush async
	go func() {
		d.DisplayRecorded(suite)
	}()
}

func (d *asyncDisplay) StartDisplayAllRecorded() {
	// Launch suitepPrinters flush async
	go func() {
		d.DisplayAllRecorded()
	}()
}

func (d *asyncDisplay) EndDisplayRecorded(suite string) {
	// Something to do ?
}

func (d *asyncDisplay) EndDisplayAllRecorded() {
	// Something to do ?
}

func NewAsync(token, isolation string) *asyncDisplay {
	p := printz.NewStandard()
	d := &asyncDisplay{
		token:              token,
		isolation:          isolation,
		stdPrinter:         p,
		clearAnsiFormatter: inout.AnsiFormatter{AnsiFormat: ansi.Reset},
		outFormatter:       inout.PrefixFormatter{Prefix: fmt.Sprintf("%sout%s>", testColor, resetColor)},
		errFormatter:       inout.PrefixFormatter{Prefix: fmt.Sprintf("%serr%s>", reportColor, resetColor)},
		verbose:            model.DefaultVerboseLevel,
		quiet:              false,
		printers:           newAsyncPrinters(token, isolation, nil, nil),
	}
	return d
}
