package asyncdisplay

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"mby.fr/cmdtest/display"
	"mby.fr/cmdtest/facade"
	"mby.fr/cmdtest/model"
	"mby.fr/utils/errorz"
	"mby.fr/utils/format"
	"mby.fr/utils/inout"
	"mby.fr/utils/printz"
	"mby.fr/utils/screen"
	"mby.fr/utils/zlog"
)

const (
	SuiteBeginPrinterName   = "__sessionBEGIN__"
	SuiteEndPrinterName     = "__sessionEND__"
	RecordedFileFlushPeriod = 20 * time.Millisecond
	RecordedFileTailPeriod  = 20 * time.Millisecond
)

var (
	logger = zlog.New()
)

func testDisplayerKey(ctx facade.TestContext) string {
	return fmt.Sprintf("%s//%d", ctx.Config.TestSuite.Get(), ctx.Seq)
}

type AsyncDisplay struct {
	verbose model.VerboseLevel
	quiet   bool

	screen screen.Sink
	tailer screen.Tailer

	outFormatter   inout.Formatter
	errFormatter   inout.Formatter
	testDisplayers map[string]display.TestDisplayer
}

func (d AsyncDisplay) Global(ctx facade.GlobalContext) {
	if d.quiet {
		return
	}

	if ctx.Config.Verbose.Get() >= model.SHOW_FAILED_OUTS {
		printer := d.screen.NotifyPrinter()
		printer.ColoredErrf(display.MessageColor, "## New config (token: %s)\n", ctx.Token)
		printer.Flush()
	}
}

func (d AsyncDisplay) Fatal(v ...any) {
	fmt.Fprintln(os.Stderr, v...)
	os.Exit(1)
}

func (d AsyncDisplay) OpenSuite(ctx facade.SuiteContext) {
	if d.quiet {
		return
	}
	suite := ctx.Config.TestSuite.Get()
	logger.Info("Opening suite", "suite", suite)
	session := d.screen.Session(suite, 0)
	err := session.Start(ctx.Config.SuiteTimeout.Get())
	if err != nil {
		panic(err)
	}
	// fmt.Printf("Opened zcreen session: %s\n", suite)
	err = session.Flush()
	if err != nil {
		panic(err)
	}
}

func (d AsyncDisplay) CloseSuite(ctx facade.SuiteContext) {
	suite := ctx.Config.TestSuite.Get()
	session := d.screen.Session(suite, 0)
	err := session.End()
	if err != nil {
		panic(err)
	}
	// Must clear session from tailer
	// err = d.screen.ClearSession(suite)
	// if err != nil {
	// 	panic(err)
	// }
	// fmt.Printf("Cleared zcreen session: %s\n", suite)
	err = session.Flush()
	if err != nil {
		panic(err)
	}
}

func (d AsyncDisplay) SuiteTitle(ctx facade.SuiteContext) {
	suite := ctx.Config.TestSuite.Get()
	session := d.screen.Session(suite, 0)
	if ctx.Config.Verbose.Get() >= model.SHOW_PASSED {
		printer := session.Printer(SuiteBeginPrinterName, 0)
		printer.ColoredErrf(display.MessageColor, "## Test suite [%s] (token: %s)\n", suite, ctx.Token)
		printer.Flush()
		session.ClosePrinter(SuiteBeginPrinterName)
	}
	err := session.Flush()
	if err != nil {
		panic(err)
	}
}

func (d AsyncDisplay) OpenTest(ctx facade.TestContext) display.TestDisplayer {
	cfg := ctx.Config

	key := testDisplayerKey(ctx)
	if td, ok := d.testDisplayers[key]; ok {
		return td
	} else {
		id := ctx.TestId()
		seq := ctx.Seq
		session := d.screen.Session(cfg.TestSuite.Get(), 0)
		printer := session.Printer(id, int(seq))
		flusher := func() error {
			return session.Flush()
		}
		td := display.NewTestDisplayer(flusher, ctx, printer, printer, d.outFormatter, d.errFormatter)
		d.testDisplayers[key] = td
		logger.Debug("new test displayer", "openedTests", d.testDisplayers)
		td.Open()
		return td
	}
}

func (d AsyncDisplay) CloseTest(ctx facade.TestContext) {
	// report end of test to suite printer
	key := testDisplayerKey(ctx)
	td, ok := d.testDisplayers[key]
	if !ok {
		//panic(fmt.Sprintf("Test: [%s] is not opened !", key))
		return
	}
	defer delete(d.testDisplayers, key)

	cfg := ctx.Config
	suite := cfg.TestSuite.Get()
	session := d.screen.Session(suite, 0)

	// Close properly test display.
	td.Close()
	session.ClosePrinter(ctx.TestId())
}

func (d AsyncDisplay) reportSuite(outcome model.SuiteOutcome, padding int) {
	testCount := outcome.TestCount
	ignoredCount := outcome.IgnoredCount
	failedCount := outcome.FailedCount
	errorCount := outcome.ErroredCount
	timeoutCount := outcome.TimeoutedCount
	passedCount := outcome.PassedCount
	tooMuchCount := outcome.TooMuchCount

	testSuite := outcome.TestSuite
	testSuiteLabel := format.New(display.TestColor, testSuite)
	testSuiteLabel.LeftPad = padding

	// if ctx.Config.Verbose.Get() >= model.SHOW_PASSED {
	// 	d.printer.ColoredErrf(messageColor, "Reporting [%s] test suite (%s) ...\n", testSuite, ctx.Token)
	// }

	suite := outcome.TestSuite
	session := d.screen.Session(suite, 0)

	printer := session.Printer(SuiteEndPrinterName, 9999)

	defer func() {
		err := printer.Flush()
		if err != nil {
			panic(err)
		}
		err = session.Flush()
		if err != nil {
			panic(err)
		}
		err = session.ClosePrinter(SuiteEndPrinterName)
		if err != nil {
			panic(err)
		}
		fmt.Printf("flushed end suite printer & session\n")
		// err = session.End()
		// if err != nil {
		// 	panic(err)
		// }
	}()

	fmt.Printf("reporting suite ...\n")

	ignoredMessage := ""
	if ignoredCount > 0 {
		ignoredMessage = fmt.Sprintf(" (%d ignored)", ignoredCount)
	}
	duration := outcome.Duration
	fmtDuration := display.NormalizeDurationInSec(duration)
	if failedCount == 0 && errorCount == 0 && timeoutCount == 0 {
		printer.ColoredErrf(display.SuccessColor, "Successfuly ran  [ %s ] test suite in %10s (%3d success)", testSuiteLabel, fmtDuration, passedCount)
		printer.ColoredErrf(display.WarningColor, "%s", ignoredMessage)
		printer.Errf("\n")
	} else {
		printer.ColoredErrf(display.FailureColor, "Failures running [ %s ] test suite in %10s (%3d success, %3d failures, %3d errors, %3d timeouts on %3d tests)", testSuiteLabel, fmtDuration, passedCount, failedCount, errorCount, timeoutCount, testCount)
		printer.ColoredErrf(display.WarningColor, "%s", ignoredMessage)
		printer.Errf("\n")
		for _, report := range outcome.FailureReports {
			report = strings.TrimSpace(report)
			if report != "" {
				//report = format.PadRight(report, 60)
				printer.ColoredErrf(display.ReportColor, "%s\n", report)
			}
		}
	}
	if tooMuchCount > 0 {
		printer.ColoredErrf(display.WarningColor, "Too much failures (%d tests not executed)\n", tooMuchCount)
	}

	/*
		err := closeSuite(d.token, d.isolation, testSuite)
		if err != nil {
			panic(err)
		}
	*/
}

func (d AsyncDisplay) ReportSuite(outcome model.SuiteOutcome) {
	if d.quiet {
		return
	}

	d.reportSuite(outcome, display.MinReportSuiteLabelPadding)
}

func (d AsyncDisplay) ReportSuites(outcomes []model.SuiteOutcome) {
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
		d.reportSuite(outcome, max(display.MinReportSuiteLabelPadding, maxSuiteNameSize))
	}
}

func (d AsyncDisplay) ReportAllFooter(globalCtx facade.GlobalContext) {
	if d.quiet {
		return
	}
	printer := d.screen.NotifyPrinter()
	globalStartTime := globalCtx.Config.GlobalStartTime.Get()
	globalDuration := model.NormalizeDurationInSec(time.Since(globalStartTime))
	printer.ColoredErrf(display.MessageColor, "Global duration time: %s\n", globalDuration)
}

func (d AsyncDisplay) TooMuchFailures(ctx facade.SuiteContext, testSuite string) {
	if d.quiet {
		return
	}
	if ctx.Config.Verbose.Get() == model.SHOW_REPORTS_ONLY {
		return
	}
	session := d.screen.Session(testSuite, 0)
	printer := session.Printer(SuiteEndPrinterName, 9999)
	printer.ColoredErrf(display.WarningColor, "Too much failure for [%s] test suite. Stop testing.\n", testSuite)
}

func (d AsyncDisplay) Errors(errors ...error) {
	//  An Error cannot be Fatal
	printer := d.screen.NotifyPrinter()
	for _, err := range errors {
		printer.ColoredErrf(display.ErrorColor, "ERROR: %s\n", err)
	}
}

func (d AsyncDisplay) GlobalErrors(ctx facade.GlobalContext, errors ...error) {
	printer := d.screen.NotifyPrinter()
	for _, err := range errors {
		printer.ColoredErrf(display.ErrorColor, "ERROR: %s\n", err)
	}
}

func (d AsyncDisplay) SuiteErrors(ctx facade.SuiteContext, errors ...error) {
	testSuite := ctx.Config.TestSuite.Get()
	session := d.screen.Session(testSuite, 0)
	printer := session.Printer(SuiteEndPrinterName, 9999)
	for _, err := range errors {
		printer.ColoredErrf(display.ErrorColor, "ERROR: %s\n", err)
	}
}

func (d AsyncDisplay) TestErrors(ctx facade.TestContext, errors ...error) {
	key := testDisplayerKey(ctx)
	if td, ok := d.testDisplayers[key]; ok {
		td.Errors(errors...)
	} else {
		d.SuiteErrors(ctx.SuiteContext, errors...)
	}
}

func (d AsyncDisplay) Flush() error {
	// FIXME: how and what to flush ?
	return nil
}

func (d *AsyncDisplay) Quiet(quiet bool) {
	d.quiet = quiet
}

func (d *AsyncDisplay) SetVerbose(level model.VerboseLevel) {
	d.verbose = level
}

/** TODO: doc */
func (d *AsyncDisplay) ClearSession(suite string) error {
	err := d.tailer.ClearSession(suite)
	fmt.Printf("async screen cleared session: [%s]\n", suite)
	return err
}

/** TODO: doc */
func (d *AsyncDisplay) Clear() error {
	err := d.tailer.Clear()
	fmt.Printf("async screen cleared\n")
	return err
}

/** Launch a goroutine to flush the display. */
func (d *AsyncDisplay) AsyncFlush(suite string, timeout time.Duration) {
	go func() {
		err := d.screen.FlushBlocking(suite, timeout)
		if err != nil && !errorz.IsTimeout(err) {
			panic(err)
		}
	}()
}

func (d *AsyncDisplay) AsyncFlushAll(timeout time.Duration) {
	go func() {
		err := d.screen.FlushAllBlocking(timeout)
		if err != nil && !errorz.IsTimeout(err) {
			panic(err)
		}
	}()
}

func (d *AsyncDisplay) TailBlocking(suite string, timeout time.Duration) error {
	// Wait for tailer to be started
	startTime := time.Now()
	for d.tailer == nil {
		if time.Since(startTime) > timeout {
			panic(fmt.Sprintf("timeout reached waiting for screen tailer: [%s]", timeout))
		}
		time.Sleep(1 * time.Millisecond)
	}
	updatedTimeout := timeout - time.Since(startTime)
	logger.Debug("TailBlocking ...", "suite", suite)
	return d.tailer.TailOnlyBlocking(suite, updatedTimeout)
}

func (d *AsyncDisplay) TailAllBlocking(timeout time.Duration) error {
	// Wait for tailer to be started
	startTime := time.Now()
	for d.tailer == nil {
		if time.Since(startTime) > timeout {
			panic(fmt.Sprintf("timeout reached waiting for screen tailer: [%s]", timeout))
		}
		time.Sleep(1 * time.Millisecond)
	}
	updatedTimeout := timeout - time.Since(startTime)
	logger.Debug("TailAllBlocking ...")
	return d.tailer.TailAllBlocking(updatedTimeout)
}

func New(tmpDir string, init bool, outs printz.Outputs) *AsyncDisplay {
	openedTests := make(map[string]display.TestDisplayer, 0)
	zcreenTmpDir := filepath.Join(tmpDir, "zcreen")
	logger.Info("Building new async display", "zcreenTmpDir", zcreenTmpDir)

	d := &AsyncDisplay{
		outFormatter:   inout.PrefixFormatter{Prefix: fmt.Sprintf("%sout%s>", display.TestColor, display.ResetColor)},
		errFormatter:   inout.PrefixFormatter{Prefix: fmt.Sprintf("%serr%s>", display.ReportColor, display.ResetColor)},
		verbose:        model.DefaultVerboseLevel,
		quiet:          false,
		testDisplayers: openedTests,
	}

	if init {
		d.screen = screen.NewAsyncScreen(zcreenTmpDir)
	}
	go func() {
		// Tailer should be build later after daemon initialized the screen
		d.tailer = screen.NewAsyncScreenTailerWaiting(outs, zcreenTmpDir, 2*time.Second)
	}()
	return d
}
