package display

import (
	"fmt"
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
	FlushFrequencyInMs     = 1
	NoActivityTimeoutInSec = 30
)

/**
## Purpose
- everithing wich can be displayed async should be done here
- all tests can be run in // but should be displayed in order
- in async mode display only on @report request

## Ideas
- display vs recorder ? => display reports, record everything else
- how to display if // report concurrent display request ? => ok: display record only once
- how to link suite deletion with record ? => ok: no need
- display recorded event after async call of StartDisplayRecorded()
- do I need to call a StopDisplayRecord() ?
- how to decide when a suite is done recording ? => ok: on StartDisplayRecorded() we can wait for all started test for a complete display (count TestTitle & TestOutcome by suite)

## Implem
- a global buffered printer
- for each suite a buffered printer
- for each test a buffered printer
- a scheduler wich flush buffers in order : global, suite1, test1, suite1, test2, suite1, ..., testN, suite1, global, suite2, ...
- StartDisplayRecorded() start the scheduler
- StopDisplayRecord() stop the scheduler
- can start flushing global instantly
- can start flushing suite after @init or first @test
- can start flushing test after init and after @test

May need to split actions in half : init by cmdt ; processing by daemon => cmdt async return must guarantee action is inited and will be processed by daemon.
Or simpler may display testTitle on queueing the test ?
*/

type suitePrinters struct {
	inited        bool
	suite         string
	main          printz.Printer
	mainBuffer    printz.Flusher
	tests         []printz.Printer
	testsBuffer   []printz.Flusher
	cursor, ended int
	startTime     time.Time
}

func (p *suitePrinters) suitePrinter() printz.Printer {
	if p.main == nil {
		stdOuts := printz.NewStandardOutputs()
		bufferedOuts := printz.NewBufferedOutputs(stdOuts)
		prtr := printz.New(bufferedOuts)
		p.main = prtr
		p.mainBuffer = bufferedOuts
	}
	p.startTime = time.Now()
	return p.main
}

func (p *suitePrinters) testPrinter(seq int) printz.Printer {
	printer := p.tests[seq]
	if printer == nil {
		stdOuts := printz.NewStandardOutputs()
		bufferedOuts := printz.NewBufferedOutputs(stdOuts)
		prtr := printz.New(bufferedOuts)
		p.tests[seq] = prtr
		p.testsBuffer[seq] = bufferedOuts
	}
	return printer
}

func (p *suitePrinters) testEnded(seq int) {
	p.ended = seq
}

func (p *suitePrinters) flush() (done bool, err error) {
	// Current implem need all test printers to be registered before starting to flush.

	// 1- flush suite until first test printer is open
	// 2- flush cursor test printer if available until ended
	// 3- increment cursor
	// 4- flush suite

	if time.Since(p.startTime) > NoActivityTimeoutInSec {
		err = fmt.Errorf("timeout flushing async display after %s", NoActivityTimeoutInSec)
	}

	prtr := p.testsBuffer[p.cursor]
	if !p.inited {
		// flush suite printer on init
		p.mainBuffer.Flush()
		p.inited = true
	}

	if prtr != nil {
		// flush cursor test printer
		prtr.Flush()
	}

	if p.cursor <= p.ended {
		// current printer is done
		p.cursor++
		// flush suite printer
		p.mainBuffer.Flush()
		p.startTime = time.Now()
	}

	if p.cursor >= len(p.testsBuffer) {
		// All printers are done
		done = true
	}
	return
}

type asyncPrinters struct {
	globalPrinter  printz.Printer
	globalBuffer   printz.Flusher
	suitesPrinters map[string]suitePrinters
	currentSuite   string
}

func (p *asyncPrinters) printer(suite string, seq int) printz.Printer {
	if suite == "" {
		if p.globalPrinter == nil {
			stdOuts := printz.NewStandardOutputs()
			bufferedOuts := printz.NewBufferedOutputs(stdOuts)
			prtr := printz.New(bufferedOuts)
			p.globalPrinter = prtr
			p.globalBuffer = bufferedOuts
		}
		return p.globalPrinter
	}

	// Select the printer by suite
	var sprtr suitePrinters
	var ok bool
	if sprtr, ok = p.suitesPrinters[suite]; !ok {
		sprtr = suitePrinters{
			suite: suite,
		}
	}

	if seq == 0 {
		return sprtr.suitePrinter()
	} else {
		return sprtr.testPrinter(seq)
	}
}

func (p *asyncPrinters) flushAll() (done bool, err error) {
	// Current implem need all suites printers to be registered before starting to flush.
	p.globalBuffer.Flush()

	for _, suitePrinters := range p.suitesPrinters {
		for done, err = suitePrinters.flush(); !done; {
			if err != nil {
				return
			}
			time.Sleep(FlushFrequencyInMs * time.Millisecond)
		}
		p.globalBuffer.Flush()
	}

	return
}

type asyncDisplay struct {
	token, isolation   string
	printers           asyncPrinters
	notQuietPrinter    printz.Printer
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
		printer := d.printers.printer(ctx.Config.TestSuite.Get(), 0)
		printer.ColoredErrf(messageColor, "## Test suite [%s] (token: %s)\n", ctx.Config.TestSuite.Get(), ctx.Token)
	}
}

func (d asyncDisplay) TestTitle(ctx facade.TestContext, seq uint16) {
	if d.quiet {
		return
	}
	if ctx.Config.Verbose.Get() == model.SHOW_REPORTS_ONLY {
		return
	}
	maxTestNameLength := 70

	cfg := ctx.Config
	timecode := int(time.Since(cfg.SuiteStartTime.Get()).Milliseconds())
	qualifiedName := testQualifiedName(ctx, testColor)
	qualifiedName = format.TruncateRight(qualifiedName, maxTestNameLength)

	title := fmt.Sprintf("[%05d] Test %s #%02d... ", timecode, qualifiedName, seq)
	title = format.PadRight(title, maxTestNameLength+23)

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
	verbose := cfg.Verbose.Get()
	testDuration := outcome.Duration
	defer d.Flush()

	if verbose < model.SHOW_PASSED && outcome.Outcome != model.PASSED && outcome.Outcome != model.IGNORED {
		// Print back test title not printed yed
		clone := ctx
		clone.Config.Verbose.Set(model.SHOW_PASSED)
		d.TestTitle(clone, outcome.Seq)
	}

	printer := d.printers.printer(cfg.TestSuite.Get(), int(outcome.Seq))

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

	printer := d.printers.printer("", 0)

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
	printer := d.printers.printer("", 0)
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

func (d asyncDisplay) Stdout(s string) {
	if s != "" {
		d.notQuietPrinter.Out(d.outFormatter.Format(s))
	}
}

func (d asyncDisplay) Stderr(s string) {
	if s != "" {
		d.notQuietPrinter.Err(d.errFormatter.Format(s))
	}
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

func newAsyncPrinter() *asyncDisplay {
	d := &asyncDisplay{
		notQuietPrinter:    printz.NewStandard(),
		clearAnsiFormatter: inout.AnsiFormatter{AnsiFormat: ansi.Reset},
		outFormatter:       inout.PrefixFormatter{Prefix: fmt.Sprintf("%sout%s>", testColor, resetColor)},
		errFormatter:       inout.PrefixFormatter{Prefix: fmt.Sprintf("%serr%s>", reportColor, resetColor)},
		verbose:            model.DefaultVerboseLevel,
	}
	d.printers = asyncPrinters{}
	return d
}
