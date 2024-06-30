package display

import (
	"fmt"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"mby.fr/cmdtest/facade"
	"mby.fr/cmdtest/model"
	"mby.fr/utils/ansi"
	"mby.fr/utils/cmdz"
	"mby.fr/utils/printz"
)

func TestAsyncDisplay_Stdout(t *testing.T) {
	//t.Skip()
	d := NewAsync("foo", "bar1")
	// Replace stdPrinter std outputs by 2 string builders
	outW := &strings.Builder{}
	errW := &strings.Builder{}
	d.stdPrinter = printz.New(printz.NewOutputs(outW, errW))

	assert.Empty(t, outW.String())
	assert.Empty(t, errW.String())

	// Writing async
	outMsg := "stdout\n"
	errMsg := "stderr\n"
	d.Stdout(outMsg)
	d.Stderr(errMsg)

	assert.Empty(t, outW.String())
	assert.Empty(t, errW.String())

	err := d.DisplayAllRecorded()
	require.NoError(t, err)

	assert.Equal(t, d.outFormatter.Format(outMsg), outW.String())
	assert.Equal(t, d.errFormatter.Format(errMsg), errW.String())
}

func TestAsyncDisplay_TestTitle(t *testing.T) {
	//t.Skip()
	d := NewAsync("foo", "bar2")
	// Replace stdPrinter std outputs by 2 string builders
	outW := &strings.Builder{}
	errW := &strings.Builder{}
	d.stdPrinter = printz.New(printz.NewOutputs(outW, errW))

	err := d.DisplayAllRecorded()
	require.NoError(t, err)

	assert.Empty(t, outW.String())
	assert.Empty(t, errW.String())

	ctx := facade.NewTestContext("token", "isol", "suite", 2, model.Config{}, 42)
	ctx.CmdExec = cmdz.Cmd("true")

	d.TestTitle(ctx)

	err = d.DisplayAllRecorded()
	require.NoError(t, err)

	assert.Empty(t, outW.String())
	expectedTitlePattern := `\[\d+\] Test \[suite\]\(on host\)>true #02...\s*`
	assert.Regexp(t, regexp.MustCompile(expectedTitlePattern), ansi.Unformat(errW.String()))

}

func displaySuite(d Displayer, token, isol string, suite int) {
	ctx := facade.NewSuiteContext(token, isol, fmt.Sprintf("suite-%d", suite), true, model.InitAction, model.Config{})
	d.Suite(ctx)
}

func displayReport(d Displayer, suite int) {
	outcome := model.SuiteOutcome{
		TestSuite:   fmt.Sprintf("suite-%d", suite),
		Duration:    3 * time.Millisecond,
		TestCount:   4,
		PassedCount: 4,
		Outcome:     model.PASSED,
	}
	d.ReportSuite(outcome)
}

func displayTestTitle(d Displayer, token, isol string, suite int, seq int) {
	ctx := facade.NewTestContext(token, isol, fmt.Sprintf("suite-%d", suite), uint16(seq), model.Config{}, uint32(42))
	ctx.CmdExec = cmdz.Cmd("true")
	outcome := model.TestOutcome{
		Outcome:  model.PASSED,
		Duration: 3 * time.Millisecond,
	}
	d.TestTitle(ctx)
	d.TestOutcome(ctx, outcome)
}

func displayTestTestOut(d Displayer, token, isol string, suite int, seq int) {
	ctx := facade.NewTestContext(token, isol, fmt.Sprintf("suite-%d", suite), uint16(seq), model.Config{}, uint32(42))
	ctx.CmdExec = cmdz.Cmd("true")
	d.Stdout(fmt.Sprintf("suite-%d-%d-out\n", suite, seq))
}

func displayTestTestErr(d Displayer, token, isol string, suite int, seq int) {
	ctx := facade.NewTestContext(token, isol, fmt.Sprintf("suite-%d", suite), uint16(seq), model.Config{}, uint32(42))
	ctx.CmdExec = cmdz.Cmd("true")
	d.Stderr(fmt.Sprintf("suite-%d-%d-err\n", suite, seq))
}

func globalInitPattern(token string) string {
	return fmt.Sprintf(`## New config \(token: %s\)\n`, token)
}

func suiteInitRegexp(token string, suite int) string {
	return fmt.Sprintf(`## Test suite \[suite-%d\] \(token: %s\)\n`, suite, token)
}

func testTitleRegexp(suite, seq int) string {
	return fmt.Sprintf(`\[\d+\] Test \[suite-%d\]\(on host\)>true #0%d...\s*PASSED \(in \dms\)\n`, suite, seq)
}

func testStdoutRegexp(suite, seq int) string {
	return fmt.Sprintf(`out>suite-%d-%d-out\n`, suite, seq)
}

func testStderrRegexp(suite, seq int) string {
	return fmt.Sprintf(`err>suite-%d-%d-err\n`, suite, seq)
}

func reportSuitePattern(suite int) string {
	return fmt.Sprintf(`Successfuly ran  \[ suite-%d\s* \] test suite in    [\d.]+ s \(\s*\d+ success\)\s*\n`, suite)
}

func TestAsyncDisplayUsage_SerialSuitesSerialTests(t *testing.T) {
	//t.Skip()
	token := "foo"
	isol := "bar3"
	d := NewAsync(token, isol)
	// Replace stdPrinter std outputs by 2 string builders
	outW := &strings.Builder{}
	errW := &strings.Builder{}
	d.stdPrinter = printz.New(printz.NewOutputs(outW, errW))

	assert.Empty(t, outW.String())
	assert.Empty(t, errW.String())

	// Scénario: Writing async on 3 suites with test ran serial
	// 000- Start with Global
	// 100- Init suite1
	// 110- Test suite1 #1
	// 111- Test suite1 #1 out>
	// 112- Test suite1 #1 err>
	// 120- Test suite1 #2
	// 121- Test suite1 #2 out>
	// 122- Test suite1 #2 err>
	// 130- Test suite1 #3
	// 131- Test suite1 #3 out>
	// 132- Test suite1 #3 err>
	// 170- Report suite1
	// 200- Init suite2
	// 210- Test suite2 #1
	// 211- Test suite2 #1 out>
	// 212- Test suite2 #1 err>
	// 220- Test suite2 #2
	// 221- Test suite2 #2 out>
	// 222- Test suite2 #2 err>
	// 270- Report suite2
	// 300- Init suite3
	// 310- Test suite3 #1
	// 311- Test suite3 #1 out>
	// 312- Test suite3 #1 err>
	// 320- Test suite3 #2
	// 321- Test suite3 #2 out>
	// 322- Test suite3 #2 err>
	// 370- Report suite3

	d.SetVerbose(model.SHOW_ALL)

	gctx := facade.NewGlobalContext(token, isol, model.Config{})
	d.Global(gctx)

	displaySuite(d, token, isol, 1) // 100- Init suite1
	displayTestTitle(d, token, isol, 1, 1)
	displayTestTestOut(d, token, isol, 1, 1)
	displayTestTestErr(d, token, isol, 1, 1)

	displayTestTitle(d, token, isol, 1, 2)
	displayTestTestOut(d, token, isol, 1, 2)
	displayTestTestErr(d, token, isol, 1, 2)
	displayTestTitle(d, token, isol, 1, 3)
	displayTestTestOut(d, token, isol, 1, 3)
	displayTestTestErr(d, token, isol, 1, 3)
	displayReport(d, 1)
	displaySuite(d, token, isol, 2) // 200- Init suite2
	displayTestTitle(d, token, isol, 2, 1)
	displayTestTestOut(d, token, isol, 2, 1)
	displayTestTestErr(d, token, isol, 2, 1)
	displayTestTitle(d, token, isol, 2, 2)
	displayTestTestOut(d, token, isol, 2, 2)
	displayTestTestErr(d, token, isol, 2, 2)
	displayReport(d, 2)

	displaySuite(d, token, isol, 3) // 300- Init suite3
	displayTestTitle(d, token, isol, 3, 1)
	displayTestTestOut(d, token, isol, 3, 1)
	displayTestTestErr(d, token, isol, 3, 1)
	displayTestTitle(d, token, isol, 3, 2)
	displayTestTestOut(d, token, isol, 3, 2)
	displayTestTestErr(d, token, isol, 3, 2)
	displayReport(d, 3)

	assert.Empty(t, outW.String())
	assert.Empty(t, errW.String())

	err := d.DisplayAllRecorded()
	require.NoError(t, err)

	assert.Empty(t, ansi.Unformat(outW.String()))

	scenarioRegexp := regexp.MustCompile("^" +
		globalInitPattern(token) +
		suiteInitRegexp(token, 1) +
		testTitleRegexp(1, 1) +
		testStdoutRegexp(1, 1) +
		testStderrRegexp(1, 1) +
		testTitleRegexp(1, 2) +
		testStdoutRegexp(1, 2) +
		testStderrRegexp(1, 2) +
		testTitleRegexp(1, 3) +
		testStdoutRegexp(1, 3) +
		testStderrRegexp(1, 3) +
		reportSuitePattern(1) +

		suiteInitRegexp(token, 2) +
		testTitleRegexp(2, 1) +
		testStdoutRegexp(2, 1) +
		testStderrRegexp(2, 1) +
		testTitleRegexp(2, 2) +
		testStdoutRegexp(2, 2) +
		testStderrRegexp(2, 2) +
		reportSuitePattern(2) +

		suiteInitRegexp(token, 3) +
		testTitleRegexp(3, 1) +
		testStdoutRegexp(3, 1) +
		testStderrRegexp(3, 1) +
		testTitleRegexp(3, 2) +
		testStdoutRegexp(3, 2) +
		testStderrRegexp(3, 2) +
		reportSuitePattern(3) +
		"$")
	assert.Regexp(t, scenarioRegexp, ansi.Unformat(errW.String()))

}

func TestAsyncDisplayUsage_AsyncSuitesSerialTests(t *testing.T) {
	t.Skip()
	token := "foo"
	isol := "bar4"
	d := NewAsync(token, isol)
	// Replace stdPrinter std outputs by 2 string builders
	outW := &strings.Builder{}
	errW := &strings.Builder{}
	d.stdPrinter = printz.New(printz.NewOutputs(outW, errW))

	assert.Empty(t, outW.String())
	assert.Empty(t, errW.String())

	// Scénario: Writing async on 3 suites with test ran serial
	// 000- Start with Global
	// 100- Init suite1
	// 110- Test suite1 #1
	// 111- Test suite1 #1 out>
	// 112- Test suite1 #1 err>
	// 200- Init suite2
	// 300- Init suite3
	// 310- Test suite3 #1
	// 311- Test suite3 #1 out>
	// 120- Test suite1 #2
	// 210- Test suite2 #1
	// 121- Test suite1 #2 out>
	// 122- Test suite1 #2 err>
	// 211- Test suite2 #1 out>
	// 212- Test suite2 #1 err>
	// 220- Test suite2 #2
	// 312- Test suite3 #1 err>
	// 221- Test suite2 #2 out>
	// 222- Test suite2 #2 err>
	// 270- Report suite2
	// 130- Test suite1 #3
	// 131- Test suite1 #3 out>
	// 132- Test suite1 #3 err>
	// 170- Report suite1
	// 320- Test suite3 #2
	// 321- Test suite3 #2 out>
	// 322- Test suite3 #2 err>
	// 370- Report suite3

	gctx := facade.NewGlobalContext(token, isol, model.Config{})
	d.Global(gctx)

	displaySuite(d, token, isol, 1) // 100- Init suite1

	displayTestTitle(d, token, isol, 1, 1)   // 110- Test suite1 #1
	displayTestTestOut(d, token, isol, 1, 1) // 111- Test suite1 #1 out>
	displayTestTestErr(d, token, isol, 1, 1) // 112- Test suite1 #1 err>

	displaySuite(d, token, isol, 2) // 200- Init suite2
	displaySuite(d, token, isol, 3) // 300- Init suite3

	displayTestTitle(d, token, isol, 3, 1)   // 310- Test suite3 #1
	displayTestTestOut(d, token, isol, 3, 1) // 311- Test suite3 #1 out>

	displayTestTitle(d, token, isol, 1, 2) // 120- Test suite1 #2
	displayTestTitle(d, token, isol, 2, 1) // 210- Test suite2 #1

	displayTestTestOut(d, token, isol, 1, 2) // 121- Test suite1 #2 out>
	displayTestTestErr(d, token, isol, 1, 2) // 122- Test suite1 #2 err>

	displayTestTestOut(d, token, isol, 2, 1) // 211- Test suite2 #1 out>
	displayTestTestErr(d, token, isol, 2, 1) // 212- Test suite2 #1 err>

	displayTestTitle(d, token, isol, 2, 2) // 220- Test suite2 #2

	displayTestTestErr(d, token, isol, 3, 1) // 312- Test suite3 #1 err>

	displayTestTestOut(d, token, isol, 2, 2) // 221- Test suite2 #2 out>
	displayTestTestErr(d, token, isol, 2, 2) // 222- Test suite2 #2 err>

	displayReport(d, 2) // 270- Report suite2

	displayTestTitle(d, token, isol, 1, 3)   // 130- Test suite1 #3
	displayTestTestOut(d, token, isol, 1, 3) // 131- Test suite1 #3 out>
	displayTestTestErr(d, token, isol, 1, 3) // 132- Test suite1 #3 err>

	displayReport(d, 1) // 170- Report suite1

	displayTestTitle(d, token, isol, 3, 2)   // 320- Test suite3 #2
	displayTestTestOut(d, token, isol, 3, 2) // 321- Test suite3 #2 out>
	displayTestTestErr(d, token, isol, 3, 2) // 322- Test suite3 #2 err>

	displayReport(d, 3) // 370- Report suite3

	assert.Empty(t, outW.String())
	assert.Empty(t, errW.String())

	err := d.DisplayAllRecorded()
	require.NoError(t, err)

	assert.Empty(t, ansi.Unformat(outW.String()))

	scenarioRegexp := regexp.MustCompile("^" +
		globalInitPattern(token) +
		suiteInitRegexp(token, 1) +
		testTitleRegexp(1, 1) +
		testStdoutRegexp(1, 1) +
		testStderrRegexp(1, 1) +
		testTitleRegexp(1, 2) +
		testStdoutRegexp(1, 2) +
		testStderrRegexp(1, 2) +
		testTitleRegexp(1, 3) +
		testStdoutRegexp(1, 3) +
		testStderrRegexp(1, 3) +
		reportSuitePattern(1) +

		suiteInitRegexp(token, 2) +
		testTitleRegexp(2, 1) +
		testStdoutRegexp(2, 1) +
		testStderrRegexp(2, 1) +
		testTitleRegexp(2, 2) +
		testStdoutRegexp(2, 2) +
		testStderrRegexp(2, 2) +
		reportSuitePattern(2) +

		suiteInitRegexp(token, 3) +
		testTitleRegexp(3, 1) +
		testStdoutRegexp(3, 1) +
		testStderrRegexp(3, 1) +
		testTitleRegexp(3, 2) +
		testStdoutRegexp(3, 2) +
		testStderrRegexp(3, 2) +
		reportSuitePattern(3) +
		"$")
	assert.Regexp(t, scenarioRegexp, ansi.Unformat(errW.String()))
}

func TestAsyncDisplayUsage_AsyncSuitesAsyncTests(t *testing.T) {
	t.Skip()
	token := "foo"
	isol := "bar5"
	d := NewAsync(token, isol)
	// Replace stdPrinter std outputs by 2 string builders
	outW := &strings.Builder{}
	errW := &strings.Builder{}
	d.stdPrinter = printz.New(printz.NewOutputs(outW, errW))

	assert.Empty(t, outW.String())
	assert.Empty(t, errW.String())

	// Scénario: Writing async on 3 suites with tests ran async too
	// 000- Start with Global
	// 100- Init suite1
	// 110- Test suite1 #1
	// 120- Test suite1 #2
	// 111- Test suite1 #1 out>
	// 121- Test suite1 #2 out>
	// 122- Test suite1 #2 err>
	// 112- Test suite1 #1 err>
	// 200- Init suite2
	// 300- Init suite3
	// 310- Test suite3 #1
	// 311- Test suite3 #1 out>
	// 210- Test suite2 #1
	// 211- Test suite2 #1 out>
	// 220- Test suite2 #2
	// 221- Test suite2 #2 out>
	// 212- Test suite2 #1 err>
	// 222- Test suite2 #2 err>
	// 270- Report suite2
	// 130- Test suite1 #3
	// 131- Test suite1 #3 out>
	// 132- Test suite1 #3 err>
	// 170- Report suite1
	// 320- Test suite3 #2
	// 312- Test suite3 #1 err>
	// 321- Test suite3 #2 out>
	// 322- Test suite3 #2 err>
	// 370- Report suite3

	gctx := facade.NewGlobalContext(token, isol, model.Config{})
	d.Global(gctx)

	displaySuite(d, token, isol, 1) // 100- Init suite1

	displayTestTitle(d, token, isol, 1, 1) // 110- Test suite1 #1
	displayTestTitle(d, token, isol, 1, 2) // 120- Test suite1 #2

	displayTestTestOut(d, token, isol, 1, 1) // 111- Test suite1 #1 out>

	displayTestTestOut(d, token, isol, 1, 2) // 121- Test suite1 #2 out>
	displayTestTestErr(d, token, isol, 1, 2) // 122- Test suite1 #2 err>

	displayTestTestErr(d, token, isol, 1, 1) // 112- Test suite1 #1 err>

	displaySuite(d, token, isol, 2) // 200- Init suite2
	displaySuite(d, token, isol, 3) // 300- Init suite3

	displayTestTitle(d, token, isol, 3, 1)   // 310- Test suite3 #1
	displayTestTestOut(d, token, isol, 3, 1) // 311- Test suite3 #1 out>

	displayTestTitle(d, token, isol, 2, 1)   // 210- Test suite2 #1
	displayTestTestOut(d, token, isol, 2, 1) // 211- Test suite2 #1 out>

	displayTestTitle(d, token, isol, 2, 2)   // 220- Test suite2 #2
	displayTestTestOut(d, token, isol, 2, 2) // 221- Test suite2 #2 out>

	displayTestTestErr(d, token, isol, 2, 1) // 212- Test suite2 #1 err>

	displayTestTestErr(d, token, isol, 2, 2) // 222- Test suite2 #2 err>
	displayReport(d, 2)                      // 270- Report suite2

	displayTestTitle(d, token, isol, 1, 3)   // 130- Test suite1 #3
	displayTestTestOut(d, token, isol, 1, 3) // 131- Test suite1 #3 out>
	displayTestTestErr(d, token, isol, 1, 3) // 132- Test suite1 #3 err>
	displayReport(d, 1)                      // 170- Report suite1

	displayTestTitle(d, token, isol, 3, 2) // 320- Test suite3 #2

	displayTestTestErr(d, token, isol, 3, 1) // 312- Test suite3 #1 err>

	displayTestTestOut(d, token, isol, 3, 2) // 321- Test suite3 #2 out>
	displayTestTestErr(d, token, isol, 3, 2) // 322- Test suite3 #2 err>

	displayReport(d, 3) // 370- Report suite3

	assert.Empty(t, outW.String())
	assert.Empty(t, errW.String())

	err := d.DisplayAllRecorded()
	require.NoError(t, err)

	assert.Empty(t, ansi.Unformat(outW.String()))

	scenarioRegexp := regexp.MustCompile("^" +
		globalInitPattern(token) +
		suiteInitRegexp(token, 1) +
		testTitleRegexp(1, 1) +
		testStdoutRegexp(1, 1) +
		testStderrRegexp(1, 1) +
		testTitleRegexp(1, 2) +
		testStdoutRegexp(1, 2) +
		testStderrRegexp(1, 2) +
		testTitleRegexp(1, 3) +
		testStdoutRegexp(1, 3) +
		testStderrRegexp(1, 3) +
		reportSuitePattern(1) +

		suiteInitRegexp(token, 2) +
		testTitleRegexp(2, 1) +
		testStdoutRegexp(2, 1) +
		testStderrRegexp(2, 1) +
		testTitleRegexp(2, 2) +
		testStdoutRegexp(2, 2) +
		testStderrRegexp(2, 2) +
		reportSuitePattern(2) +

		suiteInitRegexp(token, 3) +
		testTitleRegexp(3, 1) +
		testStdoutRegexp(3, 1) +
		testStderrRegexp(3, 1) +
		testTitleRegexp(3, 2) +
		testStdoutRegexp(3, 2) +
		testStderrRegexp(3, 2) +
		reportSuitePattern(3) +
		"$")
	assert.Regexp(t, scenarioRegexp, ansi.Unformat(errW.String()))
}
