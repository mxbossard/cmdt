package display

import (
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

func TestDisplay_Stdout(t *testing.T) {
	//t.Skip()
	d := New()
	d.SetVerbose(model.SHOW_ALL)

	// Replace notQuietPrinter std outputs by 2 string builders
	outW := &strings.Builder{}
	errW := &strings.Builder{}
	d.printer = printz.New(printz.NewOutputs(outW, errW))
	d.notQuietPrinter = d.printer

	assert.Empty(t, outW.String())
	assert.Empty(t, errW.String())

	// Writing async
	outMsg := "stdout\n"
	errMsg := "stderr\n"
	ctx, err := facade.NewTestContext("token", "isol", "suite", 12, model.Config{}, uint32(42))
	require.NoError(t, err)
	ctx.CmdExec = cmdz.Cmd("true")
	d.OpenTest(ctx)
	d.TestStdout(ctx, "beforeOut\n")
	d.TestStderr(ctx, "beforeErr\n")
	d.TestTitle(ctx)
	d.TestStdout(ctx, outMsg)
	d.TestStderr(ctx, errMsg)
	to := model.TestOutcome{
		TestSignature: model.TestSignature{TestSuite: "suite", Seq: 12},
		Duration:      3 * time.Millisecond,
		Outcome:       model.FAILED,
	}
	d.TestOutcome(ctx, to)

	// assert.Empty(t, outW.String())
	// assert.Empty(t, errW.String())
	assert.Equal(t, "beforeOut\n", ansi.Unformat(outW.String()))
	assert.Equal(t, "beforeErr\n", ansi.Unformat(errW.String()))

	d.CloseTest(ctx)

	// assert.Empty(t, outW.String())
	// assert.Empty(t, errW.String())

	// d.Flush()

	// assert.Equal(t, d.outFormatter.Format(outMsg), outW.String())
	// assert.Equal(t, d.errFormatter.Format(errMsg), errW.String())

	assert.Equal(t, "beforeOut\n"+outMsg, ansi.Unformat(outW.String()))
	assert.Regexp(t, `beforeErr\n\[\d+\] Test \[suite\]\(on host\)>true #12\s+FAILED (in 3 ms)\n`+
		`\s+Executing cmd:\s+ \[true\]\s*\n`+errMsg, ansi.Unformat(errW.String()))
}

func TestDisplay_Errors(t *testing.T) {
}

func TestDisplay_TestTitle(t *testing.T) {
	//t.Skip()
	d := New()
	d.SetVerbose(model.SHOW_ALL)

	// Replace notQuietPrinter std outputs by 2 string builders
	outW := &strings.Builder{}
	errW := &strings.Builder{}
	d.printer = printz.New(printz.NewOutputs(outW, errW))
	d.notQuietPrinter = d.printer

	assert.Empty(t, outW.String())
	assert.Empty(t, errW.String())

	ctx, err := facade.NewTestContext("token", "isol", "suite", 2, model.Config{}, 42)
	require.NoError(t, err)
	ctx.CmdExec = cmdz.Cmd("true")

	d.OpenTest(ctx)
	d.TestTitle(ctx)

	assert.Empty(t, outW.String())
	expectedTitlePattern := `\[\d+\] Test \[suite\]\(on host\)>true #02...\s*`
	assert.Regexp(t, regexp.MustCompile(expectedTitlePattern), ansi.Unformat(errW.String()))

}

func TestDisplayUsage_SerialSuitesSerialTests(t *testing.T) {
	//t.Skip()
	token := "foo"
	isol := "bar3"
	d := New()
	d.SetVerbose(model.SHOW_ALL)

	// Replace notQuietPrinter std outputs by 2 string builders
	outW := &strings.Builder{}
	errW := &strings.Builder{}
	d.printer = printz.New(printz.NewOutputs(outW, errW))
	d.notQuietPrinter = d.printer

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
	displayTestTitle(t, d, token, isol, 1, 1)
	displayTestOut(t, d, token, isol, 1, 1)
	displayTestErr(t, d, token, isol, 1, 1)

	displayTestTitle(t, d, token, isol, 1, 2)
	displayTestOut(t, d, token, isol, 1, 2)
	displayTestErr(t, d, token, isol, 1, 2)
	displayTestTitle(t, d, token, isol, 1, 3)
	displayTestOut(t, d, token, isol, 1, 3)
	displayTestErr(t, d, token, isol, 1, 3)
	displayReport(d, 1)
	displaySuite(d, token, isol, 2) // 200- Init suite2
	displayTestTitle(t, d, token, isol, 2, 1)
	displayTestOut(t, d, token, isol, 2, 1)
	displayTestErr(t, d, token, isol, 2, 1)
	displayTestTitle(t, d, token, isol, 2, 2)
	displayTestOut(t, d, token, isol, 2, 2)
	displayTestErr(t, d, token, isol, 2, 2)
	displayReport(d, 2)

	displaySuite(d, token, isol, 3) // 300- Init suite3
	displayTestTitle(t, d, token, isol, 3, 1)
	displayTestOut(t, d, token, isol, 3, 1)
	displayTestErr(t, d, token, isol, 3, 1)
	displayTestTitle(t, d, token, isol, 3, 2)
	displayTestOut(t, d, token, isol, 3, 2)
	displayTestErr(t, d, token, isol, 3, 2)
	displayReport(d, 3)

	outScenarioRegexp := regexp.MustCompile("^" +
		testStdoutRegexp(1, 1) +
		testStdoutRegexp(1, 2) +
		testStdoutRegexp(1, 3) +

		testStdoutRegexp(2, 1) +
		testStdoutRegexp(2, 2) +

		testStdoutRegexp(3, 1) +
		testStdoutRegexp(3, 2) +
		"$")
	// assert.Empty(t, ansi.Unformat(outW.String()))
	assert.Regexp(t, outScenarioRegexp, ansi.Unformat(outW.String()))

	errScenarioRegexp := regexp.MustCompile("^" +
		globalInitPattern(token) +
		suiteInitRegexp(token, 1) +
		testTitleRegexp(1, 1) +
		testStderrRegexp(1, 1) +
		testTitleRegexp(1, 2) +
		testStderrRegexp(1, 2) +
		testTitleRegexp(1, 3) +
		testStderrRegexp(1, 3) +
		reportSuitePattern(1) +

		suiteInitRegexp(token, 2) +
		testTitleRegexp(2, 1) +
		testStderrRegexp(2, 1) +
		testTitleRegexp(2, 2) +
		testStderrRegexp(2, 2) +
		reportSuitePattern(2) +

		suiteInitRegexp(token, 3) +
		testTitleRegexp(3, 1) +
		testStderrRegexp(3, 1) +
		testTitleRegexp(3, 2) +
		testStderrRegexp(3, 2) +
		reportSuitePattern(3) +
		"$")
	assert.Regexp(t, errScenarioRegexp, ansi.Unformat(errW.String()))

}
