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
	MaxTestNameLength = 70
)

const (
	MessageColor = ansi.HiPurple
	TestColor    = ansi.HiCyan
	SuccessColor = ansi.BoldGreen
	FailureColor = ansi.BoldRed
	ReportColor  = ansi.Yellow
	WarningColor = ansi.BoldHiYellow
	ErrorColor   = ansi.Red
	ResetColor   = ansi.Reset
)

type TestDisplayer interface {
	Title()
	Outcome(model.TestOutcome)
	Stdout(string)
	Stderr(string)
	Errors(...error)
	//Flush()
	Open()
	Close()
}

type basicTestDisplayer struct {
	dpl                      Displayer
	ctx                      facade.TestContext
	opened, titled, outcomed bool
	printer                  printz.Printer
	notQuietPrinter          printz.Printer
	// Buffered printer are flushed after test outcome
	bufPrinter         printz.Printer
	bufNotQuietPrinter printz.Printer
	errors             []error
	outFormatter       inout.Formatter
	errFormatter       inout.Formatter
}

func (d *basicTestDisplayer) title(ctx facade.TestContext) {
	if ctx.Config.Verbose.Get() == model.SHOW_REPORTS_ONLY {
		return
	}
	defer d.flush()
	d.titled = true

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
}

func (d *basicTestDisplayer) Title() {
	d.title(d.ctx)
}

func (d *basicTestDisplayer) Outcome(outcome model.TestOutcome) {
	if d.ctx.Config.Verbose.Get() == model.SHOW_REPORTS_ONLY {
		return
	}
	defer func() {
		d.outcomed = true
		d.titled = false
		d.flush()
	}()

	// FIXME get outcome from ctx
	cfg := d.ctx.Config
	verbose := cfg.Verbose.Get()
	testDuration := outcome.Duration

	if verbose < model.SHOW_PASSED && outcome.Outcome != model.PASSED && outcome.Outcome != model.IGNORED {
		// Print back test title not printed yet
		clone := d.ctx
		clone.Config.Verbose.Set(model.SHOW_PASSED)
		d.title(clone)
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
		d.printer.Errf(" (after %s)\n", d.ctx.Config.Timeout.Get())
	case model.ERRORED:
		d.printer.ColoredErrf(WarningColor, "ERRORED")
		d.printer.Errf(" (not executed)\n")
	case model.UNKNOWN:
		d.printer.ColoredErrf(WarningColor, "UNKNOWN")
		d.printer.Err("\n")
	case model.IGNORED:
		if verbose > model.SHOW_FAILED_OUTS {
			d.printer.ColoredErrf(WarningColor, "IGNORED")
			d.printer.Err("\n")
		}
	default:
		panic(fmt.Sprintf("unknown outcome: %s", outcome.Outcome))
	}

	if outcome.Outcome == model.ERRORED {
		d.printer.Errf("\tSupplied cmd: \t\t[%s]\n", CmdTitle(d.ctx))
	} else if verbose >= model.SHOW_FAILED_ONLY && outcome.Outcome != model.PASSED && outcome.Outcome != model.IGNORED || verbose >= model.SHOW_PASSED_OUTS {
		d.printer.Errf("\tExecuting cmd: \t\t[%s]\n", CmdTitle(d.ctx))
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

func (d basicTestDisplayer) assertionResult(result model.AssertionResult) {
	defer d.flush()
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
	if !result.Success {
		expected = strings.ReplaceAll(expected, "\n", "\\n")
		if s, ok := got.(string); ok {
			s = strings.ReplaceAll(s, "\n", "\\n")
			got = s

			stringifiedGot = ansi.TruncateMid(s, 100, "[...]")
		} else {
			stringifiedGot = fmt.Sprintf("%v", got)
			//panic(fmt.Sprintf("unable to stringify rule %s value: [%v]", assertName, got))
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

// Write on stdout
func (d basicTestDisplayer) Stdout(s string) {
	prtr := d.bufNotQuietPrinter
	if !d.titled {
		// Delay stdout flushing after test is outcomed
		defer d.flush()
		prtr = d.notQuietPrinter
	}
	if s != "" {
		prtr.Out(s)
		//d.bufPrinter.Out(s)
	}
	// if !d.opened {
	// 	d.bufNotQuietPrinter.Flush()
	// 	d.dpl.Flush()
	// }
}

// Write on stderr
func (d basicTestDisplayer) Stderr(s string) {
	prtr := d.bufNotQuietPrinter
	if !d.titled {
		// Delay stdout flushing after test is closed
		defer d.flush()
		prtr = d.notQuietPrinter
	}
	if s != "" {
		prtr.Err(s)
		//d.bufPrinter.Err(s)
	}
	// if !d.opened {
	// 	d.bufNotQuietPrinter.Flush()
	// 	d.dpl.Flush()
	// }
}

func (d *basicTestDisplayer) Errors(errors ...error) {
	defer d.flush()
	// Delay error display when test is closed

	d.errors = append(d.errors, errors...)
	// Display errors
	for _, err := range d.errors {
		if err != nil {
			d.bufNotQuietPrinter.ColoredErrf(ErrorColor, "ERROR: %s", err)
		}
	}
	// Clear errors list
	d.errors = make([]error, 0)
}

func (d basicTestDisplayer) flush() {
	if !d.titled {
		err := d.bufPrinter.Flush()
		if err != nil {
			panic(err)
		}
		err = d.bufNotQuietPrinter.Flush()
		if err != nil {
			panic(err)
		}
	}

	err := d.printer.Flush()
	if err != nil {
		panic(err)
	}
	err = d.notQuietPrinter.Flush()
	if err != nil {
		panic(err)
	}

	err = d.dpl.Flush()
	if err != nil {
		panic(err)
	}
}

func (d *basicTestDisplayer) Open() {
	if d.opened {
		panic(fmt.Sprintf("Test: [%s] already opened !", d.ctx.TestId()))
	}
	if d.outcomed {
		panic(fmt.Sprintf("Test: [%s] already outcomed !", d.ctx.TestId()))
	}
	logger.Debug("Opening test", "suite", d.ctx.Config.TestSuite, "seq", d.ctx.Seq)
	d.opened = true
}

func (d *basicTestDisplayer) Close() {
	logger.Debug("Closing test", "suite", d.ctx.Config.TestSuite, "seq", d.ctx.Seq)
	if !d.outcomed {
		logger.Debug("Closing not outcomed test.", "suite", d.ctx.Config.TestSuite, "seq", d.ctx.Seq)
		// Display a nice outcome if test not closed
		var to model.TestOutcome
		if len(d.errors) > 0 {
			// ERRORED outcome
			to = d.ctx.ErroredTestOutcome(d.errors...)
		} else {
			// UNKNOWN outcome
			to = d.ctx.UnknownTestOutcome()
		}
		d.Outcome(to)

		// display errors
		d.Errors()

		d.flush()
	}

	d.opened = false
}

func NewTestDisplayer(d Displayer, ctx facade.TestContext, printer, notQuietPrinter printz.Printer, outFormatter, errFormatter inout.Formatter) *basicTestDisplayer {
	bufPrinter := printz.Buffered(printer)
	bufNotQuietPrinter := printz.Buffered(notQuietPrinter)
	td := &basicTestDisplayer{
		dpl:                d,
		ctx:                ctx,
		printer:            printer,
		bufPrinter:         bufPrinter,
		notQuietPrinter:    notQuietPrinter,
		bufNotQuietPrinter: bufNotQuietPrinter,
		outFormatter:       outFormatter,
		errFormatter:       errFormatter,
	}

	return td
}
