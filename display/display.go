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

var (
	messageColor = ansi.HiPurple
	testColor    = ansi.HiCyan
	successColor = ansi.BoldGreen
	failureColor = ansi.BoldRed
	reportColor  = ansi.Yellow
	warningColor = ansi.BoldHiYellow
	errorColor   = ansi.Red
	resetColor   = ansi.Reset
)

/*
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
*/
type BasicDisplay struct {
	printer            printz.Printer
	clearAnsiFormatter inout.Formatter
	outFormatter       inout.Formatter
	errFormatter       inout.Formatter
}

func (d BasicDisplay) Global(ctx facade.GlobalContext) {
	defer d.Flush()
	// Do nothing ?
	if ctx.Config.Verbose.Get() <= model.BETTER_ASSERTION_REPORT {
		d.printer.ColoredErrf(messageColor, "## New config (token: %s)\n", ctx.Token)
	}
}

func (d BasicDisplay) Suite(ctx facade.SuiteContext) {
	defer d.Flush()
	if ctx.Config.Verbose.Get() <= model.BETTER_ASSERTION_REPORT {
		d.printer.ColoredErrf(messageColor, "## New test suite: [%s] (token: %s)\n", ctx.Config.TestSuite.Get(), ctx.Token)
	}
}

func (d BasicDisplay) TestTitle(ctx facade.TestContext, seq int) {
	defer d.Flush()
	maxTestNameLength := 50

	cfg := ctx.Config
	timecode := int(time.Since(cfg.SuiteStartTime.Get()).Milliseconds())
	qualifiedName := ctx.TestQualifiedName()
	//seq := c.Repo.TestCount(c.Config.TestSuite.Get())
	qualifiedName = format.TruncateRight(qualifiedName, maxTestNameLength)

	title := fmt.Sprintf("[%05d] Test: %s #%02d... ", timecode, qualifiedName, seq)
	title = format.PadRight(title, maxTestNameLength+23)

	//title := ctx.TestTitle()

	if ctx.Config.Ignore.Is(true) {
		if ctx.Config.Verbose.Get() <= model.BETTER_ASSERTION_REPORT {
			d.printer.ColoredErrf(warningColor, title)
		}
		return
	}

	if ctx.Config.Verbose.Get() <= model.SHOW_PASSED {
		d.printer.ColoredErrf(testColor, title)
		if ctx.Config.KeepStdout.Is(true) || ctx.Config.KeepStderr.Is(true) {
			// NewLine because we expect cmd outputs
			//d.printer.Errf("\n")
		}
	}
}

func (d BasicDisplay) TestOutcome(ctx facade.TestContext, outcome model.TestOutcome) {
	// FIXME get outcome from ctx
	testDuration := outcome.Duration
	defer d.Flush()
	switch outcome.Outcome {
	case model.PASSED:
		if ctx.Config.Verbose.Get() <= model.SHOW_PASSED {
			d.printer.ColoredErrf(successColor, "PASSED")
			d.printer.Errf(" (in %s)\n", testDuration)
		}
	case model.FAILED:
		if ctx.Config.Verbose.Get() > model.SHOW_PASSED {
			ctx.Config.Verbose.Set(model.SHOW_PASSED)
			d.TestTitle(ctx, outcome.Seq)
		}
		d.printer.ColoredErrf(failureColor, "FAILED")
		d.printer.Errf(" (in %s)\n", testDuration)
		d.printer.Errf("\tFailure calling cmd: \t[%s]\n", outcome.CmdTitle)
	case model.TIMEOUT:
		d.printer.ColoredErrf(failureColor, "FAILED")
		d.printer.Errf(" (timed out after %s)\n", ctx.Config.Timeout.Get())
	case model.ERRORED:
		d.printer.ColoredErrf(warningColor, "ERRORED")
		d.printer.Errf(" (not executed)\n")
		d.printer.Errf("\tFailure calling cmd: \t[%s]\n", outcome.CmdTitle)
	case model.IGNORED:
		if ctx.Config.Verbose.Get() >= model.BETTER_ASSERTION_REPORT {
			d.printer.ColoredErrf(warningColor, "IGNORED")
			d.printer.Err("\n")
		}
	default:
	}
	if outcome.Err != nil {
		d.printer.ColoredErrf(model.ErrorColor, "\t%s\n", outcome.Err)
	}

	if ctx.Config.Verbose.Get() <= model.SHOW_PASSED &&
		(ctx.Config.KeepStdout.Is(true) || ctx.Config.KeepStderr.Is(true)) {
		// NewLine in printer to print test result in a new line
		//d.printer.Errf("        ")
		d.printer.Flush()
	}
}

func (d BasicDisplay) AssertionResult(result model.AssertionResult) {
	defer d.Flush()
	hlClr := messageColor
	//log.Printf("failedResult: %v\n", result)
	assertPrefix := result.Assertion.Prefix
	assertName := result.Assertion.Name
	assertOp := result.Assertion.Op
	expected := result.Assertion.Expected
	got := result.Value

	if result.ErrMessage != "" {
		d.printer.ColoredErrf(errorColor, result.ErrMessage+"\n")
	}

	assertLabel := ansi.Sprintf(testColor, "%s%s", assertPrefix, assertName)

	if assertName == "success" || assertName == "fail" {
		d.printer.Errf("\t%sExpected%s %s\n", hlClr, resetColor, assertLabel)
		//d.Stdout(cmd.StdoutRecord())
		//d.Stderr(cmd.StderrRecord())
		/*
			if cmd.StderrRecord() != "" {
				d.printer.Errf("sdterr> %s\n", cmd.StderrRecord())
			}
		*/
		return
	} else if assertName == "cmd" {
		d.printer.Errf("\t%sExpected%s %s=%s to succeed\n", hlClr, resetColor, assertLabel, expected)
		return
	} else if assertName == "exists" {
		d.printer.Errf("\t%sExpected%s file %s=%s file to exists\n", hlClr, resetColor, assertLabel, expected)
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
			d.printer.Errf("\t%sExpected%s %s \n\t\t%sto be%s: \t\t[%s]\n\t\t%sbut got%s: \t[%v]\n", hlClr, resetColor, assertLabel, hlClr, resetColor, expected, hlClr, resetColor, got)
		} else if assertOp == ":" {
			d.printer.Errf("\t%sExpected%s %s \n\t\t%sto contains%s: \t[%s]\n\t\t%sbut got%s: \t[%v]\n", hlClr, resetColor, assertLabel, hlClr, resetColor, expected, hlClr, resetColor, got)
		} else if assertOp == "!:" {
			d.printer.Errf("\t%sExpected%s %s \n\t\t%snot to contains%s: \t[%s]\n\t\t%sbut got%s: \t[%v]\n", hlClr, resetColor, assertLabel, hlClr, resetColor, expected, hlClr, resetColor, got)
		} else if assertOp == "~" {
			d.printer.Errf("\t%sExpected%s %s \n\t\t%sto match%s: \t[%s]\n\t\t%sbut got%s: \t[%v]\n", hlClr, resetColor, assertLabel, hlClr, resetColor, expected, hlClr, resetColor, got)
		} else if assertOp == "!~" {
			d.printer.Errf("\t%sExpected%s %s \n\t\t%snot to match%s: \t[%s]\n\t\t%sbut got%s: \t[%v]\n", hlClr, resetColor, assertLabel, hlClr, resetColor, expected, hlClr, resetColor, got)
		}
	} else {
		d.printer.Errf("assertion %s%s%s failed\n", assertLabel, assertOp, expected)
	}
}

func (d BasicDisplay) ReportSuite(ctx facade.SuiteContext, outcome model.SuiteOutcome) {
	defer d.Flush()
	testCount := outcome.TestCount
	ignoredCount := outcome.IgnoredCount
	failedCount := outcome.FailedCount
	errorCount := outcome.ErroredCount
	//passedCount := testCount - failedCount - ignoredCount
	passedCount := outcome.PassedCount

	testSuite := ctx.Config.TestSuite.Get()
	if ctx.Config.Verbose.Get() <= model.SHOW_PASSED {
		d.printer.ColoredErrf(messageColor, "Reporting [%s] test suite (%s) ...\n", testSuite, ctx.Token)
	}

	ignoredMessage := ""
	if ignoredCount > 0 {
		ignoredMessage = fmt.Sprintf(" (%d ignored)", ignoredCount)
	}
	startTime := ctx.Config.SuiteStartTime.Get()
	endTime := ctx.Config.LastTestTime.GetOr(time.Now())
	duration := endTime.Sub(startTime)
	fmtDuration := NormalizeDurationInSec(duration)
	if failedCount == 0 && errorCount == 0 {
		d.printer.ColoredErrf(successColor, "Successfuly ran [%s] test suite (%d success in %s)", testSuite, passedCount, fmtDuration)
		d.printer.ColoredErrf(warningColor, "%s", ignoredMessage)
		d.printer.Errf("\n")
	} else {
		d.printer.ColoredErrf(failureColor, "Failures in [%s] test suite (%d success, %d failures, %d errors on %d tests in %s)", testSuite, passedCount, failedCount, errorCount, testCount, fmtDuration)
		d.printer.ColoredErrf(warningColor, "%s", ignoredMessage)
		d.printer.Errf("\n")
		for _, report := range outcome.FailureReports {
			report = strings.TrimSpace(report)
			if report != "" {
				//report = format.PadRight(report, 60)
				d.printer.ColoredErrf(reportColor, "%s\n", report)
			}
		}
	}
}

func (d BasicDisplay) ReportAllFooter(globalCtx facade.GlobalContext) {
	defer d.Flush()

	globalStartTime := globalCtx.Config.GlobalStartTime.Get()
	globalDuration := model.NormalizeDurationInSec(time.Since(globalStartTime))
	d.printer.ColoredErrf(messageColor, "Global duration time: %s\n", globalDuration)
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
