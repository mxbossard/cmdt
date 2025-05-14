package asyncdisplay

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"cmdt/internal/display"
	"cmdt/internal/facade"
	"cmdt/internal/model"

	"github.com/mxbossard/utilz/errorz"
	"github.com/mxbossard/utilz/formatz"
	"github.com/mxbossard/utilz/inoutz"
	"github.com/mxbossard/utilz/printz"
	"github.com/mxbossard/utilz/zcreen"
	"github.com/mxbossard/utilz/zlog"
)

const (
	SuiteBeginPrinterName   = "__sessionBEGIN__"
	SuiteEndPrinterName     = "__sessionEND__"
	RecordedFileFlushPeriod = 10 * time.Millisecond
	RecordedFileTailPeriod  = 10 * time.Millisecond
)

var (
	logger = zlog.New()
)

func logErrors(errors ...error) {
	for _, err := range errors {
		logger.Errorf("ERROR: %s", err)
		fmt.Printf("ERROR: %s\n", err)
	}
}
func testDisplayerKey(ctx facade.TestContext) string {
	return fmt.Sprintf("%s//%d", ctx.Config.TestSuite.Get(), ctx.Seq)
}

type AsyncDisplay struct {
	*sync.Mutex

	verbose model.VerboseLevel
	quiet   bool
	tmpDir  string

	screen zcreen.Sink
	tailer zcreen.Tailer

	outFormatter   inoutz.Formatter
	errFormatter   inoutz.Formatter
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

	err := d.screen.Resync()
	if err != nil {
		panic(err)
	}

	suite := ctx.Config.TestSuite.Get()
	logger.Info("Opening suite", "suite", suite)
	session := d.screen.Session(suite, 0)
	//fmt.Printf("\n<<>> opened session: %s\n", session)
	err = session.Start(ctx.Config.SuiteTimeout.Get())
	if err != nil {
		panic(err)
	}
	err = session.Flush()
	if err != nil {
		panic(err)
	}
}

func (d AsyncDisplay) CloseSuite(ctx facade.SuiteContext) {
	suite := ctx.Config.TestSuite.Get()
	session := d.screen.Session(suite, 0)
	err := session.End(fmt.Sprintf("closing suite: %s", suite))
	if err != nil {
		panic(err)
	}
	err = session.Flush()
	if err != nil {
		panic(err)
	}
}

func (d AsyncDisplay) ClearSuite(name string) {
	if d.tailer != nil {
		err := d.tailer.ClearSession(name)
		if err != nil {
			panic(err)
		}
	}
	zcreen.ClearSession(d.tmpDir, name)
	logger.Info("cleared async suite", "suite", name)
}

func (d AsyncDisplay) SuiteTitle(ctx facade.SuiteContext) {
	suite := ctx.Config.TestSuite.Get()
	session := d.screen.Session(suite, 0)
	if ctx.Config.Verbose.Get() >= model.SHOW_PASSED {
		printer := session.Printer(SuiteBeginPrinterName, 0)
		printer.ColoredErrf(display.MessageColor, "## Test suite [%s] (token: %s)\n", suite, ctx.Token)
		printer.Flush()
		session.ClosePrinter(SuiteBeginPrinterName, "low verbosity")
	}
	err := session.Flush()
	if err != nil {
		panic(err)
	}
}

func (d AsyncDisplay) OpenTest(ctx facade.TestContext) display.TestDisplayer {
	d.Lock()
	defer d.Unlock()

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
	d.Lock()
	defer d.Unlock()

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
	session.ClosePrinter(ctx.TestId(), "closing test")
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
	testSuiteLabel := formatz.New(display.TestColor, testSuite)
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
		//fmt.Printf("flushed end suite printer & session\n")
		// err = session.ClosePrinter(SuiteEndPrinterName)
		// if err != nil {
		// 	panic(err)
		// }

		// err = session.End()
		// if err != nil {
		// 	panic(err)
		// }
	}()

	//fmt.Printf("reporting suite ...\n")

	ignoredMessage := ""
	if ignoredCount > 0 {
		ignoredMessage = fmt.Sprintf(" (%d ignored)", ignoredCount)
	}
	duration := outcome.Duration
	fmtDuration := display.NormalizeDurationInSec(duration)
	if failedCount == 0 && errorCount == 0 && timeoutCount == 0 {
		printer.ColoredErrf(display.SuccessColor, "Successfully ran [ %s ] test suite in %10s (%3d success)", testSuiteLabel, fmtDuration, passedCount)
		printer.ColoredErrf(display.WarningColor, "%s", ignoredMessage)
		printer.Errf("\n")
	} else {
		//printer.ColoredErrf(display.FailureColor, "Failures running [ %s ] test suite in %10s (%3d success, %3d failures, %3d errors, %3d timeouts on %3d tests)", testSuiteLabel, fmtDuration, passedCount, failedCount, errorCount, timeoutCount, testCount)
		//printer.ColoredErrf(display.WarningColor, "%s", ignoredMessage)
		printer.ColoredErrf(display.FailureColor, "Failures running [ %s ] test suite in %10s (%3d success", testSuiteLabel, fmtDuration, passedCount)
		if failedCount > 0 {
			printer.ColoredErrf(display.FailureColor, ", %3d failures", failedCount)
		}
		if errorCount > 0 {
			printer.ColoredErrf(display.FailureColor, ", %3d errors", errorCount)
		}
		if timeoutCount > 0 {
			printer.ColoredErrf(display.FailureColor, "%3d timeouts", timeoutCount)
		}
		printer.ColoredErrf(display.FailureColor, " on %3d tests)", testCount)
		if ignoredMessage != "" {
			printer.ColoredErrf(display.WarningColor, "%s", ignoredMessage)
		}
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
	defer printer.Flush()
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
	defer printer.Flush()
	printer.ColoredErrf(display.WarningColor, "Too much failure for [%s] test suite. Stop testing.\n", testSuite)
}

func (d AsyncDisplay) Errors(errors ...error) {
	logErrors(errors...)
	//  An Error cannot be Fatal
	printer := d.screen.NotifyPrinter()
	defer printer.Flush()
	for _, err := range errors {
		printer.ColoredErrf(display.ErrorColor, "ERROR: %s\n", err)
	}
}

func (d AsyncDisplay) GlobalErrors(ctx facade.GlobalContext, errors ...error) {
	logErrors(errors...)
	printer := d.screen.NotifyPrinter()
	defer printer.Flush()
	for _, err := range errors {
		printer.ColoredErrf(display.ErrorColor, "ERROR: %s\n", err)
	}
}

func (d AsyncDisplay) SuiteErrors(ctx facade.SuiteContext, errors ...error) {
	logErrors(errors...)
	testSuite := ctx.Config.TestSuite.Get()
	session := d.screen.Session(testSuite, 0)
	printer := session.NotifyPrinter()
	defer printer.Flush()
	for _, err := range errors {
		printer.ColoredErrf(display.ErrorColor, "ERROR: %s\n", err)
	}
}

func (d AsyncDisplay) TestErrors(ctx facade.TestContext, errors ...error) {
	d.Lock()
	defer d.Unlock()

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
	err := d.WaitForTailerInit(timeout)
	if err != nil {
		return err
	}
	updatedTimeout := timeout - time.Since(startTime)
	logger.Debug("TailBlocking ...", "suite", suite)
	// FIXME: use TailOnlyBlocking because TailBlocking keep re-tailing all previously tailed sessions.
	return d.tailer.TailOnlyBlocking(suite, updatedTimeout)
}

func (d *AsyncDisplay) TailAllBlocking(timeout time.Duration) error {
	// Wait for tailer to be started
	startTime := time.Now()
	err := d.WaitForTailerInit(timeout)
	if err != nil {
		return err
	}
	updatedTimeout := timeout - time.Since(startTime)
	logger.Debug("TailAllBlocking ...")
	return d.tailer.TailAllBlocking(updatedTimeout)
}

func (d *AsyncDisplay) TailSuppliedBlocking(suites []string, timeout time.Duration) error {
	// Wait for tailer to be started
	startTime := time.Now()
	err := d.WaitForTailerInit(timeout)
	if err != nil {
		return err
	}
	updatedTimeout := timeout - time.Since(startTime)
	logger.Debug("TailSuppliedBlocking ...")
	return d.tailer.TailSuppliedBlocking(suites, updatedTimeout)
}

func (d *AsyncDisplay) WaitForTailerInit(timeout time.Duration) error {
	startTime := time.Now()
	for d.tailer == nil {
		if time.Since(startTime) > timeout {
			return fmt.Errorf("timeout reached waiting for screen tailer: [%s]", timeout)
		}
		time.Sleep(1 * time.Millisecond)
	}
	return nil
}

func zcreenTmpDir(tmpDir string) string {
	return filepath.Join(tmpDir, "zcreen")
}

func New(tmpDir string, init bool, outs printz.Outputs) *AsyncDisplay {
	openedTests := make(map[string]display.TestDisplayer, 0)
	zcreenTmpDir := zcreenTmpDir(tmpDir)
	logger.Info("Building new async display", "zcreenTmpDir", zcreenTmpDir)

	d := &AsyncDisplay{
		Mutex:          &sync.Mutex{},
		tmpDir:         zcreenTmpDir,
		outFormatter:   inoutz.PrefixFormatter{Prefix: fmt.Sprintf("%sout%s>", display.TestColor, display.ResetColor)},
		errFormatter:   inoutz.PrefixFormatter{Prefix: fmt.Sprintf("%serr%s>", display.ReportColor, display.ResetColor)},
		verbose:        model.DefaultVerboseLevel,
		quiet:          false,
		testDisplayers: openedTests,
	}

	if init {
		d.screen = zcreen.NewAsyncScreen(zcreenTmpDir)
	}
	go func() {
		// Tailer should be build later after daemon initialized the screen
		d.tailer = zcreen.NewAsyncScreenTailerWaiting(outs, zcreenTmpDir, 2*time.Second)
	}()
	return d
}

func NewWaitingTailer(tmpDir string, init bool, outs printz.Outputs) *AsyncDisplay {
	openedTests := make(map[string]display.TestDisplayer, 0)
	zcreenTmpDir := zcreenTmpDir(tmpDir)
	logger.Info("Building new async display", "zcreenTmpDir", zcreenTmpDir)

	d := &AsyncDisplay{
		Mutex:          &sync.Mutex{},
		tmpDir:         zcreenTmpDir,
		outFormatter:   inoutz.PrefixFormatter{Prefix: fmt.Sprintf("%sout%s>", display.TestColor, display.ResetColor)},
		errFormatter:   inoutz.PrefixFormatter{Prefix: fmt.Sprintf("%serr%s>", display.ReportColor, display.ResetColor)},
		verbose:        model.DefaultVerboseLevel,
		quiet:          false,
		testDisplayers: openedTests,
	}

	if init {
		d.screen = zcreen.NewAsyncScreen(zcreenTmpDir)
	}
	// Tailer should be build later after daemon initialized the screen
	d.tailer = zcreen.NewAsyncScreenTailerWaiting(outs, zcreenTmpDir, 2*time.Second)
	return d
}

func ClearSuite(tmpDir, name string) error {
	zcreenTmpDir := zcreenTmpDir(tmpDir)
	return zcreen.ClearSession(zcreenTmpDir, name)
}
