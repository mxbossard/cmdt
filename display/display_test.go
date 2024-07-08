package display

import (
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
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
	ctx := facade.NewTestContext("token", "isol", "suite", 12, model.Config{}, uint32(42))
	ctx.CmdExec = cmdz.Cmd("true")
	d.TestStdout(ctx, outMsg)
	d.TestStderr(ctx, errMsg)

	assert.Empty(t, outW.String())
	assert.Empty(t, errW.String())

	d.Flush()

	// assert.Equal(t, d.outFormatter.Format(outMsg), outW.String())
	// assert.Equal(t, d.errFormatter.Format(errMsg), errW.String())

	assert.Empty(t, outW.String())
	assert.Equal(t, d.outFormatter.Format(outMsg)+d.errFormatter.Format(errMsg), errW.String())
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

	ctx := facade.NewTestContext("token", "isol", "suite", 2, model.Config{}, 42)
	ctx.CmdExec = cmdz.Cmd("true")

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
	displayTestTitle(d, token, isol, 1, 1)
	displayTestOut(d, token, isol, 1, 1)
	displayTestErr(d, token, isol, 1, 1)

	displayTestTitle(d, token, isol, 1, 2)
	displayTestOut(d, token, isol, 1, 2)
	displayTestErr(d, token, isol, 1, 2)
	displayTestTitle(d, token, isol, 1, 3)
	displayTestOut(d, token, isol, 1, 3)
	displayTestErr(d, token, isol, 1, 3)
	displayReport(d, 1)
	displaySuite(d, token, isol, 2) // 200- Init suite2
	displayTestTitle(d, token, isol, 2, 1)
	displayTestOut(d, token, isol, 2, 1)
	displayTestErr(d, token, isol, 2, 1)
	displayTestTitle(d, token, isol, 2, 2)
	displayTestOut(d, token, isol, 2, 2)
	displayTestErr(d, token, isol, 2, 2)
	displayReport(d, 2)

	displaySuite(d, token, isol, 3) // 300- Init suite3
	displayTestTitle(d, token, isol, 3, 1)
	displayTestOut(d, token, isol, 3, 1)
	displayTestErr(d, token, isol, 3, 1)
	displayTestTitle(d, token, isol, 3, 2)
	displayTestOut(d, token, isol, 3, 2)
	displayTestErr(d, token, isol, 3, 2)
	displayReport(d, 3)

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
