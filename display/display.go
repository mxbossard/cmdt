package display

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"mby.fr/cmdtest/facade"
	"mby.fr/cmdtest/model"
	"mby.fr/utils/ansi"
	"mby.fr/utils/cmdz"
	"mby.fr/utils/inout"
	"mby.fr/utils/printz"
	"mby.fr/utils/utilz"
)

var (
	messageColor = ansi.HiPurple
	testColor    = ansi.HiCyan
	successColor = ansi.BoldGreen
	failureColor = ansi.BoldRed
	reportColor  = ansi.Yellow
	warningColor = ansi.BoldHiYellow
	errorColor   = ansi.Red
)

type Displayer interface {
	Global(facade.Context)
	Suite(facade.Context)
	TestTitle(facade.Context)
	TestOutcome(ctx facade.Context)
	AssertionResult(model.AssertionResult)
	ReportSuite(facade.Context)
	ReportAll(facade.Context)
	Stdout(string)
	Stderr(string)
	Error(error)
	Flush() error
}

type BasicDisplay struct {
	printer            printz.Printer
	clearAnsiFormatter inout.Formatter
	outFormatter       inout.Formatter
	errFormatter       inout.Formatter
}

func (d BasicDisplay) Global(ctx facade.GlobalContext) {
	defer d.Flush()
	// Do nothing ?
	if ctx.Config.Verbose.Get() >= model.BETTER_ASSERTION_REPORT {
		d.printer.ColoredErrf(messageColor, "## New config (token: %s)\n", ctx.Token)
	}
}

func (d BasicDisplay) Suite(ctx facade.SuiteContext) {
	defer d.Flush()
	if ctx.Config.Verbose.Get() >= model.BETTER_ASSERTION_REPORT {
		d.printer.ColoredErrf(messageColor, "## New test suite: [%s] (token: %s)\n", ctx.TestQualifiedName(), ctx.Token)
	}
}

func (d BasicDisplay) TestTitle(ctx facade.TestContext) {
	// FIXME: get seq from ctx
	defer d.Flush()
	title := ctx.TestTitle()

	if ctx.Config.Ignore.Is(true) {
		if ctx.Config.Verbose.Get() >= model.BETTER_ASSERTION_REPORT {
			d.printer.ColoredErrf(warningColor, title)
		}
		return
	}

	if ctx.Config.Verbose.Get() >= model.SHOW_PASSED {
		d.printer.ColoredErrf(testColor, title)
		if ctx.Config.KeepStdout.Is(true) || ctx.Config.KeepStderr.Is(true) {
			// NewLine because we expect cmd outputs
			d.printer.Errf("\n")
		}
	}
}

func (d BasicDisplay) TestOutcome(ctx facade.TestContext, seq int, outcome model.Outcome, cmd cmdz.Executer, testDuration time.Duration, err error) {
	// FIXME get outcome from ctx
	// FIXME get cmd, duration and error from outcome
	defer d.Flush()
	switch outcome {
	case model.PASSED:
		if ctx.Config.Verbose.Get() >= model.SHOW_PASSED {
			d.printer.ColoredErrf(successColor, "PASSED")
			d.printer.Errf(" (in %s)\n", testDuration)
		}
	case model.FAILED:
		if ctx.Config.Verbose.Get() < model.SHOW_PASSED {
			ctx.Config.Verbose = utilz.OptionalOf(model.SHOW_PASSED)
			d.TestTitle(ctx)
		}
		if err == nil {
			//IncrementSeq(tmpDir, FailureSequenceFilename)
			d.printer.ColoredErrf(failureColor, "FAILED")
			d.printer.Errf(" (in %s)\n", testDuration)
		} else {
			if errors.Is(err, context.DeadlineExceeded) {
				// Swallow error
				err = nil
				//IncrementSeq(tmpDir, FailureSequenceFilename)
				d.printer.ColoredErrf(failureColor, "FAILED")
				d.printer.Errf(" (timed out after %s)\n", ctx.Config.Timeout.Get())
			}
		}
		d.printer.Errf("Failure calling cmd: <|%s|>\n", cmd)
	case model.ERRORED:
		d.printer.ColoredErrf(warningColor, "ERROR")
		d.printer.Errf(" (not executed)\n")
		d.printer.Errf("Failure calling cmd: <|%s|>\n", cmd)
	case model.IGNORED:
		if ctx.Config.Verbose.Get() >= model.BETTER_ASSERTION_REPORT {
			d.printer.ColoredErrf(warningColor, "IGNORED")
			d.printer.Err("\n")
		}
	default:
	}
	if err != nil {
		d.printer.ColoredErrf(model.ErrorColor, "%s\n", err)
	}

	if ctx.Config.Verbose.Get() >= model.SHOW_PASSED &&
		(ctx.Config.KeepStdout.Is(true) || ctx.Config.KeepStderr.Is(true)) {
		// NewLine in printer to print test result in a new line
		d.printer.Errf("        ")
		d.printer.Flush()
	}
}

func (d BasicDisplay) AssertionResult(cmd cmdz.Executer, result model.AssertionResult) {
	defer d.Flush()
	//log.Printf("failedResult: %v\n", result)
	assertPrefix := result.Assertion.Prefix
	assertName := result.Assertion.Name
	assertOp := result.Assertion.Op
	expected := result.Assertion.Expected
	got := result.Value

	if result.Message != "" {
		d.printer.ColoredErrf(errorColor, result.Message+"\n")
	}

	if assertName == "success" || assertName == "fail" {
		d.printer.Errf("Expected %s%s\n", assertPrefix, assertName)
		d.Stdout(cmd.StdoutRecord())
		d.Stderr(cmd.StderrRecord())
		/*
			if cmd.StderrRecord() != "" {
				d.printer.Errf("sdterr> %s\n", cmd.StderrRecord())
			}
		*/
		return
	} else if assertName == "cmd" {
		d.printer.Errf("Expected %s%s=%s to succeed\n", assertPrefix, assertName, expected)
		return
	} else if assertName == "exists" {
		d.printer.Errf("Expected file %s%s=%s file to exists\n", assertPrefix, assertName, expected)
		return
	}

	if expected != got {

		if s, ok := got.(string); ok {
			got = strings.ReplaceAll(s, "\n", "\\n")
			/*
				const sliceSize = 16
				minStrLen := min(len(s), len(expected))
				for k := range minStrLen / sliceSize {
					left := expected[k*sliceSize : min(len(expected), (k+1)*sliceSize-1)]
					right := s[k*sliceSize : min(len(s), (k+1)*sliceSize-1)]
					if left == right {
						continue
					} else {
						shortenExpected := ""
						if k > 0 {
							shortenExpected += "[...]"
						}
						shortenExpected += left
						if k*minStrLen < len(expected) {
							shortenExpected += "[...]"
						}
						shortenGot := ""
						if k > 0 {
							shortenGot += "[...]"
						}
						shortenGot += right
						if k*minStrLen < len(s) {
							shortenGot += "[...]"
						}
						expected = shortenExpected
						s = shortenGot
					}
				}
					expected = d.clearAnsiFormatter.Format(expected)
					got = d.clearAnsiFormatter.Format(s)
			*/
		}
		if assertOp == "=" {
			d.printer.Errf("Expected %s%s to be: [%s] but got: [%v]\n", assertPrefix, assertName, expected, got)
		} else if assertOp == ":" {
			d.printer.Errf("Expected %s%s to contains: [%s] but got: [%v]\n", assertPrefix, assertName, expected, got)
		} else if assertOp == "!:" {
			d.printer.Errf("Expected %s%s not to contains: [%s] but got: [%v]\n", assertPrefix, assertName, expected, got)
		} else if assertOp == "~" {
			d.printer.Errf("Expected %s%s to match: [%s] but got: [%v]\n", assertPrefix, assertName, expected, got)
		} else if assertOp == "!~" {
			d.printer.Errf("Expected %s%s not to match: [%s] but got: [%v]\n", assertPrefix, assertName, expected, got)
		}
	} else {
		d.printer.Errf("assertion %s%s%s%s failed\n", assertPrefix, assertName, assertOp, expected)
	}
}

func (d BasicDisplay) ReportSuite(ctx facade.SuiteContext, failedReports []string) {
	defer d.Flush()
	testCount := ctx.TestCount()
	ignoredCount := ctx.IgnoredCount()
	failedCount := ctx.FailedCount()
	errorCount := ctx.ErroredCount()
	passedCount := testCount - failedCount - ignoredCount

	testSuite := ctx.Config.TestSuite.Get()
	if ctx.Config.Verbose.Get() >= model.SHOW_PASSED {
		d.printer.ColoredErrf(messageColor, "Reporting [%s] test suite (%s) ...\n", testSuite, ctx.Token)
	}

	ignoredMessage := ""
	if ignoredCount > 0 {
		ignoredMessage = fmt.Sprintf(" (%d ignored)", ignoredCount)
	}
	duration := ctx.Config.LastTestTime.Get().Sub(ctx.Config.SuiteStartTime.Get())
	fmtDuration := NormalizeDurationInSec(duration)
	if failedCount == 0 && errorCount == 0 {
		d.printer.ColoredErrf(successColor, "Successfuly ran [%s] test suite (%d tests in %s)", testSuite, passedCount, fmtDuration)
		d.printer.ColoredErrf(warningColor, "%s", ignoredMessage)
		d.printer.Errf("\n")
	} else {
		d.printer.ColoredErrf(failureColor, "Failures in [%s] test suite (%d success, %d failures, %d errors on %d tests in %s)", testSuite, passedCount, failedCount, errorCount, testCount, fmtDuration)
		d.printer.ColoredErrf(warningColor, "%s", ignoredMessage)
		d.printer.Errf("\n")
		for _, report := range failedReports {
			d.printer.ColoredErrf(reportColor, "%s\n", report)
		}
	}
}

func (d BasicDisplay) ReportAllFooter(globalCtx facade.GlobalContext) {
	defer d.Flush()

	globalStartTime := globalCtx.Config.GlobalStartTime.Get()
	globalDuration := model.NormalizeDurationInSec(time.Since(globalStartTime))
	d.printer.ColoredErrf(reportColor, "Global duration time: %s\n", globalDuration)
}

func (d BasicDisplay) Stdout(s string) {
	if s != "" {
		d.printer.Err(d.outFormatter.Format(s))
	}
}

func (d BasicDisplay) Stderr(s string) {
	if s != "" {
		d.printer.Err(d.errFormatter.Format(s))
	}
}

func (d BasicDisplay) Error(err error) {

}

func (d BasicDisplay) Flush() error {
	return d.printer.Flush()
}

func New() BasicDisplay {
	return BasicDisplay{
		printer:            printz.NewStandard(),
		clearAnsiFormatter: inout.AnsiFormatter{AnsiFormat: ansi.Reset},
		outFormatter:       inout.PrefixFormatter{Prefix: "out> "},
		errFormatter:       inout.PrefixFormatter{Prefix: "err> "},
	}
}

func NormalizeDurationInSec(d time.Duration) (duration string) {
	duration = fmt.Sprintf("%.3f s", float32(d.Milliseconds())/1000)
	return
}
