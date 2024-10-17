package asyncdisplay

import (
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"mby.fr/cmdtest/display"
	"mby.fr/cmdtest/facade"
	"mby.fr/cmdtest/model"
	"mby.fr/cmdtest/repo"
	"mby.fr/utils/ansi"
	"mby.fr/utils/cmdz"
	"mby.fr/utils/printz"
)

func TestAsyncDisplay_TestStdout(t *testing.T) {
	//t.Skip()
	token := "foo1"
	isol := "bar1"
	suite := "suite"
	var err error

	err = repo.ClearWorkDirectory(token, isol)
	require.NoError(t, err)

	d := New(token, isol)
	// Replace stdPrinter std outputs by 2 string builders
	outW := &strings.Builder{}
	errW := &strings.Builder{}
	d.stdPrinter = printz.New(printz.NewOutputs(outW, errW))

	assert.Empty(t, outW.String())
	assert.Empty(t, errW.String())

	// Writing async
	outMsg := "stdout\n"
	errMsg := "stderr\n"
	ctx, err := facade.NewTestContext(token, isol, suite, 1, model.Config{}, uint32(42))
	require.NoError(t, err)
	ctx.CmdExec = cmdz.Cmd("true")
	d.OpenTest(ctx)
	d.TestStdout(ctx, outMsg)
	d.TestStderr(ctx, errMsg)
	d.CloseTest(ctx)

	assert.Empty(t, outW.String())
	assert.Empty(t, errW.String())

	err = d.AsyncFlush(suite, 100*time.Millisecond)
	require.NoError(t, err)
	err = d.BlockTail(suite, 100*time.Millisecond)
	require.NoError(t, err)

	assert.Equal(t, outMsg, outW.String())
	assert.Equal(t, errMsg, errW.String())

	outW.Reset()
	errW.Reset()

	// FIXME: why AsyncFlushAll should re flush already flushed suite ?
	err = d.AsyncFlushAll(100 * time.Millisecond)
	require.NoError(t, err)
	err = d.BlockTailAll(100 * time.Millisecond)
	require.NoError(t, err)

	assert.Equal(t, outMsg, outW.String())
	assert.Equal(t, errMsg, errW.String())
}

func TestAsyncDisplay_TestTitle(t *testing.T) {
	//t.Skip()
	token := "foo2"
	isol := "bar2"
	var err error

	err = repo.ClearWorkDirectory(token, isol)
	require.NoError(t, err)

	d := New(token, isol)
	// Replace stdPrinter std outputs by 2 string builders
	outW := &strings.Builder{}
	errW := &strings.Builder{}
	d.stdPrinter = printz.New(printz.NewOutputs(outW, errW))

	err = d.AsyncFlushAll(100 * time.Millisecond)
	require.NoError(t, err)
	err = d.BlockTailAll(100 * time.Millisecond)
	require.NoError(t, err)

	assert.Empty(t, outW.String())
	assert.Empty(t, errW.String())

	ctx, err := facade.NewTestContext("token", "isol", "suite", 1, model.Config{}, 42)
	require.NoError(t, err)
	ctx.CmdExec = cmdz.Cmd("true")

	d.OpenTest(ctx)
	d.TestTitle(ctx)
	d.CloseTest(ctx)

	err = d.AsyncFlushAll(100 * time.Millisecond)
	require.NoError(t, err)
	err = d.BlockTailAll(100 * time.Millisecond)
	require.NoError(t, err)

	assert.Empty(t, outW.String())
	expectedTitlePattern := `\[\d+\] Test \[suite\]\(on host\)>true #01...\s*`
	assert.Regexp(t, regexp.MustCompile(expectedTitlePattern), ansi.Unformat(errW.String()))

}

func TestBlockTail(t *testing.T) {
	//t.Skip()
	token := "foo10"
	isol := "bar10"
	var err error

	err = repo.ClearWorkDirectory(token, isol)
	require.NoError(t, err)

	d := New(token, isol)
	d.SetVerbose(model.SHOW_ALL)

	// Replace stdPrinter std outputs by 2 string builders
	outW := &strings.Builder{}
	errW := &strings.Builder{}
	d.stdPrinter = printz.New(printz.NewOutputs(outW, errW))

	assert.Empty(t, outW.String())
	assert.Empty(t, errW.String())

	// Scénario: Writing on 3 suites in sync with test ran serial
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

	// Start 3 tests async/unordered
	display.DisplaySuite(d, token, isol, 1) // 100- Init suite1

	// Simulate outputs sent disordered
	display.DisplayTestTitle(t, d, token, isol, 1, 1)
	display.DisplayTestOut(t, d, token, isol, 1, 1)
	display.DisplayTestErr(t, d, token, isol, 1, 1)
	display.DisplayEndTest(t, d, token, isol, 1, 1)

	display.DisplayTestTitle(t, d, token, isol, 1, 3)
	display.DisplayTestOut(t, d, token, isol, 1, 3)
	display.DisplayTestErr(t, d, token, isol, 1, 3)
	display.DisplayEndTest(t, d, token, isol, 1, 3)

	display.DisplayTestTitle(t, d, token, isol, 1, 2)
	display.DisplayTestOut(t, d, token, isol, 1, 2)
	display.DisplayTestErr(t, d, token, isol, 1, 2)
	display.DisplayEndTest(t, d, token, isol, 1, 2)

	display.DisplayReport(d, 1)

	assert.Empty(t, outW.String())
	assert.Empty(t, errW.String())

	require.NoError(t, err)
	err = d.AsyncFlush("suite-1", 100*time.Millisecond)
	require.NoError(t, err)
	err = d.BlockTail("suite-1", 100*time.Millisecond)
	require.NoError(t, err)

	outScenarioRegexp := regexp.MustCompile("^" +
		display.TestStdoutRegexp(1, 1) +
		display.TestStdoutRegexp(1, 2) +
		display.TestStdoutRegexp(1, 3) +
		"$")
	assert.Regexp(t, outScenarioRegexp, ansi.Unformat(outW.String()))
	// Expect scénario to be oredred test1, test2, test4
	errScenarioRegexp := regexp.MustCompile("^" +
		display.SuiteInitRegexp(token, 1) +
		display.TestTitleRegexp(1, 1) +
		display.TestStderrRegexp(1, 1) +
		display.TestTitleRegexp(1, 2) +
		display.TestStderrRegexp(1, 2) +
		display.TestTitleRegexp(1, 3) +
		display.TestStderrRegexp(1, 3) +
		display.ReportSuitePattern(1) +
		"$")
	assert.Regexp(t, errScenarioRegexp, ansi.Unformat(errW.String()))

}

func TestBlockTail_Twice(t *testing.T) {
	//t.Skip()
	token := "foo12"
	isol := "bar12"
	var err error

	err = repo.ClearWorkDirectory(token, isol)
	require.NoError(t, err)

	d := New(token, isol)
	d.SetVerbose(model.SHOW_ALL)

	// Replace stdPrinter std outputs by 2 string builders
	outW := &strings.Builder{}
	errW := &strings.Builder{}
	d.stdPrinter = printz.New(printz.NewOutputs(outW, errW))

	assert.Empty(t, outW.String())
	assert.Empty(t, errW.String())

	// Start 1 test in one suite
	display.DisplaySuite(d, token, isol, 1) // 100- Init suite1

	display.DisplayTestTitle(t, d, token, isol, 1, 1)
	display.DisplayTestOut(t, d, token, isol, 1, 1)
	display.DisplayTestErr(t, d, token, isol, 1, 1)
	display.DisplayEndTest(t, d, token, isol, 1, 1)

	display.DisplayReport(d, 1)

	assert.Empty(t, outW.String())
	assert.Empty(t, errW.String())

	require.NoError(t, err)
	err = d.AsyncFlush("suite-1", 100*time.Millisecond)
	require.NoError(t, err)
	err = d.BlockTail("suite-1", 100*time.Millisecond)
	require.NoError(t, err)

	assert.Empty(t, ansi.Unformat(outW.String()))
	// Expect scénario to be test1
	scenarioRegexp := regexp.MustCompile("^" +
		display.SuiteInitRegexp(token, 1) +
		display.TestTitleRegexp(1, 1) +
		display.TestStdoutRegexp(1, 1) +
		display.TestStderrRegexp(1, 1) +
		display.ReportSuitePattern(1) +
		"$")
	assert.Regexp(t, scenarioRegexp, ansi.Unformat(errW.String()))

	outW.Reset()
	errW.Reset()

	// Start another test in reinited suite
	display.DisplaySuite(d, token, isol, 1) // 100- Init suite1

	display.DisplayTestTitle(t, d, token, isol, 1, 1)
	display.DisplayTestOut(t, d, token, isol, 1, 1)
	//displayTestErr(t, d, token, isol, 1, 1)
	display.DisplayEndTest(t, d, token, isol, 1, 1)

	display.DisplayReport(d, 1)

	assert.Empty(t, outW.String())
	assert.Empty(t, errW.String())

	require.NoError(t, err)
	err = d.AsyncFlush("suite-1", 100*time.Millisecond)
	require.NoError(t, err)
	err = d.BlockTail("suite-1", 100*time.Millisecond)
	require.NoError(t, err)

	// Expect scénario to be test2
	scenarioRegexp = regexp.MustCompile("^" +
		display.SuiteInitRegexp(token, 1) +
		display.TestTitleRegexp(1, 1) +
		display.TestStdoutRegexp(1, 1) +
		//testStderrRegexp(1, 1) +
		display.ReportSuitePattern(1) +
		"$")
	assert.Regexp(t, scenarioRegexp, ansi.Unformat(errW.String()))

}

func TestAsyncFlushThenDisplayThenBlockTail(t *testing.T) {
	//t.Skip()
	token := "foo20"
	isol := "bar20"
	var err error

	err = repo.ClearWorkDirectory(token, isol)
	require.NoError(t, err)

	d := New(token, isol)
	d.SetVerbose(model.SHOW_ALL)

	// Replace stdPrinter std outputs by 2 string builders
	outW := &strings.Builder{}
	errW := &strings.Builder{}
	d.stdPrinter = printz.New(printz.NewOutputs(outW, errW))

	assert.Empty(t, outW.String())
	assert.Empty(t, errW.String())

	// Scénario: Writing on 3 suites in sync with test ran serial
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

	//d.Clear("suite-1")

	// Start 3 tests async/unordered
	display.DisplaySuite(d, token, isol, 1) // 100- Init suite1

	// Simulate outputs sent disordered
	display.DisplayTestTitle(t, d, token, isol, 1, 1)
	display.DisplayTestOut(t, d, token, isol, 1, 1)
	display.DisplayTestErr(t, d, token, isol, 1, 1)
	display.DisplayEndTest(t, d, token, isol, 1, 1)

	err = d.AsyncFlush("suite-1", 100*time.Millisecond)
	require.NoError(t, err)

	time.Sleep(10 * time.Millisecond)

	display.DisplayTestTitle(t, d, token, isol, 1, 3)
	display.DisplayTestOut(t, d, token, isol, 1, 3)
	display.DisplayTestErr(t, d, token, isol, 1, 3)
	display.DisplayEndTest(t, d, token, isol, 1, 3)

	time.Sleep(10 * time.Millisecond)

	display.DisplayTestTitle(t, d, token, isol, 1, 2)
	display.DisplayTestOut(t, d, token, isol, 1, 2)
	display.DisplayTestErr(t, d, token, isol, 1, 2)
	display.DisplayEndTest(t, d, token, isol, 1, 2)

	display.DisplayReport(d, 1)

	assert.Empty(t, outW.String())
	assert.Empty(t, errW.String())

	err = d.BlockTail("suite-1", 100*time.Millisecond)
	require.NoError(t, err)

	assert.Empty(t, ansi.Unformat(outW.String()))
	// Expect scénario to be oredred test1, test2, test3
	scenarioRegexp := regexp.MustCompile("^" +
		display.SuiteInitRegexp(token, 1) +
		display.TestTitleRegexp(1, 1) +
		display.TestStdoutRegexp(1, 1) +
		display.TestStderrRegexp(1, 1) +
		display.TestTitleRegexp(1, 2) +
		display.TestStdoutRegexp(1, 2) +
		display.TestStderrRegexp(1, 2) +
		display.TestTitleRegexp(1, 3) +
		display.TestStdoutRegexp(1, 3) +
		display.TestStderrRegexp(1, 3) +
		display.ReportSuitePattern(1) +
		"$")
	assert.Regexp(t, scenarioRegexp, ansi.Unformat(errW.String()))

}

func TestBlockTailAll(t *testing.T) {
	//t.Skip()
	token := "foo30"
	isol := "bar30"
	var err error

	err = repo.ClearWorkDirectory(token, isol)
	require.NoError(t, err)

	d := New(token, isol)
	d.SetVerbose(model.SHOW_ALL)

	// Replace stdPrinter std outputs by 2 string builders
	outW := &strings.Builder{}
	errW := &strings.Builder{}
	d.stdPrinter = printz.New(printz.NewOutputs(outW, errW))

	assert.Empty(t, outW.String())
	assert.Empty(t, errW.String())

	// Scénario: Writing on 3 suites in sync with test ran serial
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

	gctx := facade.NewGlobalContext(token, isol, model.Config{})
	d.Global(gctx)

	// Start 3 tests async/unordered
	display.DisplaySuite(d, token, isol, 1) // 100- Init suite1

	// Simulate outputs sent disordered
	display.DisplayTestTitle(t, d, token, isol, 1, 1)
	display.DisplayTestOut(t, d, token, isol, 1, 1)
	display.DisplayTestErr(t, d, token, isol, 1, 1)
	display.DisplayEndTest(t, d, token, isol, 1, 1)

	display.DisplayTestTitle(t, d, token, isol, 1, 3)
	display.DisplayTestOut(t, d, token, isol, 1, 3)
	display.DisplayTestErr(t, d, token, isol, 1, 3)
	display.DisplayEndTest(t, d, token, isol, 1, 3)

	display.DisplayTestTitle(t, d, token, isol, 1, 2)
	display.DisplayTestOut(t, d, token, isol, 1, 2)
	display.DisplayTestErr(t, d, token, isol, 1, 2)
	display.DisplayEndTest(t, d, token, isol, 1, 2)

	display.DisplayReport(d, 1)

	assert.Empty(t, outW.String())
	assert.Empty(t, errW.String())

	err = d.AsyncFlushAll(100 * time.Millisecond)
	require.NoError(t, err)
	err = d.BlockTailAll(100 * time.Millisecond)
	require.NoError(t, err)

	assert.Empty(t, ansi.Unformat(outW.String()))
	// Expect scénario to be oredred test1, test2, test3
	scenarioRegexp := regexp.MustCompile("^" +
		display.GlobalInitPattern(token) +
		display.SuiteInitRegexp(token, 1) +
		display.TestTitleRegexp(1, 1) +
		display.TestStdoutRegexp(1, 1) +
		display.TestStderrRegexp(1, 1) +
		display.TestTitleRegexp(1, 2) +
		display.TestStdoutRegexp(1, 2) +
		display.TestStderrRegexp(1, 2) +
		display.TestTitleRegexp(1, 3) +
		display.TestStdoutRegexp(1, 3) +
		display.TestStderrRegexp(1, 3) +
		display.ReportSuitePattern(1) +
		"$")
	assert.Regexp(t, scenarioRegexp, ansi.Unformat(errW.String()))

}

func TestAsyncFlushAllThenDisplayThenBlockTailAll(t *testing.T) {
	//t.Skip()
	token := "foo40"
	isol := "bar40"
	var err error

	err = repo.ClearWorkDirectory(token, isol)
	require.NoError(t, err)

	d := New(token, isol)
	d.SetVerbose(model.SHOW_ALL)

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

	// Clear files on suite init
	// err = clearFileWriters(token, isol, "")
	// require.NoError(t, err)

	gctx := facade.NewGlobalContext(token, isol, model.Config{})
	d.Global(gctx)

	// Start 3 tests async/unordered
	display.DisplaySuite(d, token, isol, 1) // 100- Init suite1
	display.DisplayTestTitle(t, d, token, isol, 1, 1)
	display.DisplayTestTitle(t, d, token, isol, 1, 3)
	display.DisplayTestTitle(t, d, token, isol, 1, 2)

	assert.Empty(t, outW.String())
	assert.Empty(t, errW.String())

	err = d.AsyncFlushAll(1000 * time.Millisecond)
	require.NoError(t, err)

	// Simulate outputs sent disordered
	display.DisplayTestOut(t, d, token, isol, 1, 1)
	display.DisplayTestErr(t, d, token, isol, 1, 1)
	display.DisplayEndTest(t, d, token, isol, 1, 1)

	time.Sleep(10 * time.Millisecond)

	display.DisplayTestOut(t, d, token, isol, 1, 3)
	display.DisplayTestErr(t, d, token, isol, 1, 3)
	display.DisplayEndTest(t, d, token, isol, 1, 3)

	time.Sleep(10 * time.Millisecond)

	display.DisplayTestOut(t, d, token, isol, 1, 2)
	display.DisplayTestErr(t, d, token, isol, 1, 2)
	display.DisplayEndTest(t, d, token, isol, 1, 2)

	display.DisplayReport(d, 1)

	err = d.BlockTailAll(100 * time.Millisecond)
	require.NoError(t, err)

	assert.Empty(t, ansi.Unformat(outW.String()))
	// Expect scénario to be oredred test1, test2, test3
	scenarioRegexp := regexp.MustCompile("^" +
		display.GlobalInitPattern(token) +
		display.SuiteInitRegexp(token, 1) +
		display.TestTitleRegexp(1, 1) +
		display.TestStdoutRegexp(1, 1) +
		display.TestStderrRegexp(1, 1) +
		display.TestTitleRegexp(1, 2) +
		display.TestStdoutRegexp(1, 2) +
		display.TestStderrRegexp(1, 2) +
		display.TestTitleRegexp(1, 3) +
		display.TestStdoutRegexp(1, 3) +
		display.TestStderrRegexp(1, 3) +
		display.ReportSuitePattern(1) +
		"$")
	assert.Regexp(t, scenarioRegexp, ansi.Unformat(errW.String()))

}

func TestAsyncDisplayUsage_SerialSuitesSerialTests(t *testing.T) {
	//t.Skip()
	token := "foo50"
	isol := "bar50"
	var err error

	err = repo.ClearWorkDirectory(token, isol)
	require.NoError(t, err)

	d := New(token, isol)
	d.SetVerbose(model.SHOW_ALL)

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

	gctx := facade.NewGlobalContext(token, isol, model.Config{})
	d.Global(gctx)

	display.DisplaySuite(d, token, isol, 1) // 100- Init suite1
	display.DisplayTestTitle(t, d, token, isol, 1, 1)
	display.DisplayTestOut(t, d, token, isol, 1, 1)
	display.DisplayTestErr(t, d, token, isol, 1, 1)
	display.DisplayEndTest(t, d, token, isol, 1, 1)

	display.DisplayTestTitle(t, d, token, isol, 1, 2)
	display.DisplayTestOut(t, d, token, isol, 1, 2)
	display.DisplayTestErr(t, d, token, isol, 1, 2)
	display.DisplayEndTest(t, d, token, isol, 1, 2)
	display.DisplayTestTitle(t, d, token, isol, 1, 3)
	display.DisplayTestOut(t, d, token, isol, 1, 3)
	display.DisplayTestErr(t, d, token, isol, 1, 3)
	display.DisplayEndTest(t, d, token, isol, 1, 3)
	display.DisplayReport(d, 1)
	display.DisplaySuite(d, token, isol, 2) // 200- Init suite2
	display.DisplayTestTitle(t, d, token, isol, 2, 1)
	display.DisplayTestOut(t, d, token, isol, 2, 1)
	display.DisplayTestErr(t, d, token, isol, 2, 1)
	display.DisplayEndTest(t, d, token, isol, 2, 1)
	display.DisplayTestTitle(t, d, token, isol, 2, 2)
	display.DisplayTestOut(t, d, token, isol, 2, 2)
	display.DisplayTestErr(t, d, token, isol, 2, 2)
	display.DisplayEndTest(t, d, token, isol, 2, 2)
	display.DisplayReport(d, 2)

	display.DisplaySuite(d, token, isol, 3) // 300- Init suite3
	display.DisplayTestTitle(t, d, token, isol, 3, 1)
	display.DisplayTestOut(t, d, token, isol, 3, 1)
	display.DisplayTestErr(t, d, token, isol, 3, 1)
	display.DisplayEndTest(t, d, token, isol, 3, 1)
	display.DisplayTestTitle(t, d, token, isol, 3, 2)
	display.DisplayTestOut(t, d, token, isol, 3, 2)
	display.DisplayTestErr(t, d, token, isol, 3, 2)
	display.DisplayEndTest(t, d, token, isol, 3, 2)
	display.DisplayReport(d, 3)

	assert.Empty(t, outW.String())
	assert.Empty(t, errW.String())

	err = d.AsyncFlushAll(100 * time.Millisecond)
	require.NoError(t, err)
	err = d.BlockTailAll(100 * time.Millisecond)
	require.NoError(t, err)

	assert.Empty(t, ansi.Unformat(outW.String()))

	scenarioRegexp := regexp.MustCompile("^" +
		display.GlobalInitPattern(token) +
		display.SuiteInitRegexp(token, 1) +
		display.TestTitleRegexp(1, 1) +
		display.TestStdoutRegexp(1, 1) +
		display.TestStderrRegexp(1, 1) +
		display.TestTitleRegexp(1, 2) +
		display.TestStdoutRegexp(1, 2) +
		display.TestStderrRegexp(1, 2) +
		display.TestTitleRegexp(1, 3) +
		display.TestStdoutRegexp(1, 3) +
		display.TestStderrRegexp(1, 3) +
		display.ReportSuitePattern(1) +

		display.SuiteInitRegexp(token, 2) +
		display.TestTitleRegexp(2, 1) +
		display.TestStdoutRegexp(2, 1) +
		display.TestStderrRegexp(2, 1) +
		display.TestTitleRegexp(2, 2) +
		display.TestStdoutRegexp(2, 2) +
		display.TestStderrRegexp(2, 2) +
		display.ReportSuitePattern(2) +

		display.SuiteInitRegexp(token, 3) +
		display.TestTitleRegexp(3, 1) +
		display.TestStdoutRegexp(3, 1) +
		display.TestStderrRegexp(3, 1) +
		display.TestTitleRegexp(3, 2) +
		display.TestStdoutRegexp(3, 2) +
		display.TestStderrRegexp(3, 2) +
		display.ReportSuitePattern(3) +
		"$")
	assert.Regexp(t, scenarioRegexp, ansi.Unformat(errW.String()))

}

func TestAsyncDisplayUsage_AsyncSuitesSerialTests(t *testing.T) {
	//t.Skip()
	token := "foo60"
	isol := "bar60"

	var err error
	err = repo.ClearWorkDirectory(token, isol)
	require.NoError(t, err)

	d := New(token, isol)
	d.SetVerbose(model.SHOW_ALL)

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

	display.DisplaySuite(d, token, isol, 1) // 100- Init suite1

	display.DisplayTestTitle(t, d, token, isol, 1, 1) // 110- Test suite1 #1
	display.DisplayTestOut(t, d, token, isol, 1, 1)   // 111- Test suite1 #1 out>
	display.DisplayTestErr(t, d, token, isol, 1, 1)   // 112- Test suite1 #1 err>
	display.DisplayEndTest(t, d, token, isol, 1, 1)

	display.DisplaySuite(d, token, isol, 2) // 200- Init suite2
	display.DisplaySuite(d, token, isol, 3) // 300- Init suite3

	display.DisplayTestTitle(t, d, token, isol, 3, 1) // 310- Test suite3 #1
	display.DisplayTestOut(t, d, token, isol, 3, 1)   // 311- Test suite3 #1 out>

	display.DisplayTestTitle(t, d, token, isol, 1, 2) // 120- Test suite1 #2
	display.DisplayTestTitle(t, d, token, isol, 2, 1) // 210- Test suite2 #1

	display.DisplayTestOut(t, d, token, isol, 1, 2) // 121- Test suite1 #2 out>
	display.DisplayTestErr(t, d, token, isol, 1, 2) // 122- Test suite1 #2 err>
	display.DisplayEndTest(t, d, token, isol, 1, 2)

	display.DisplayTestOut(t, d, token, isol, 2, 1) // 211- Test suite2 #1 out>
	display.DisplayTestErr(t, d, token, isol, 2, 1) // 212- Test suite2 #1 err>
	display.DisplayEndTest(t, d, token, isol, 2, 1)

	display.DisplayTestTitle(t, d, token, isol, 2, 2) // 220- Test suite2 #2

	display.DisplayTestErr(t, d, token, isol, 3, 1) // 312- Test suite3 #1 err>
	display.DisplayEndTest(t, d, token, isol, 3, 1)

	display.DisplayTestOut(t, d, token, isol, 2, 2) // 221- Test suite2 #2 out>
	display.DisplayTestErr(t, d, token, isol, 2, 2) // 222- Test suite2 #2 err>
	display.DisplayEndTest(t, d, token, isol, 2, 2)

	display.DisplayReport(d, 2) // 270- Report suite2

	display.DisplayTestTitle(t, d, token, isol, 1, 3) // 130- Test suite1 #3
	display.DisplayTestOut(t, d, token, isol, 1, 3)   // 131- Test suite1 #3 out>
	display.DisplayTestErr(t, d, token, isol, 1, 3)   // 132- Test suite1 #3 err>
	display.DisplayEndTest(t, d, token, isol, 1, 3)

	display.DisplayReport(d, 1) // 170- Report suite1

	display.DisplayTestTitle(t, d, token, isol, 3, 2) // 320- Test suite3 #2
	display.DisplayTestOut(t, d, token, isol, 3, 2)   // 321- Test suite3 #2 out>
	display.DisplayTestErr(t, d, token, isol, 3, 2)   // 322- Test suite3 #2 err>
	display.DisplayEndTest(t, d, token, isol, 3, 2)

	display.DisplayReport(d, 3) // 370- Report suite3

	assert.Empty(t, outW.String())
	assert.Empty(t, errW.String())

	err = d.AsyncFlushAll(100 * time.Millisecond)
	require.NoError(t, err)
	err = d.BlockTailAll(100 * time.Millisecond)
	require.NoError(t, err)

	assert.Empty(t, ansi.Unformat(outW.String()))

	scenarioRegexp := regexp.MustCompile("^" +
		display.GlobalInitPattern(token) +
		display.SuiteInitRegexp(token, 1) +
		display.TestTitleRegexp(1, 1) +
		display.TestStdoutRegexp(1, 1) +
		display.TestStderrRegexp(1, 1) +
		display.TestTitleRegexp(1, 2) +
		display.TestStdoutRegexp(1, 2) +
		display.TestStderrRegexp(1, 2) +
		display.TestTitleRegexp(1, 3) +
		display.TestStdoutRegexp(1, 3) +
		display.TestStderrRegexp(1, 3) +
		display.ReportSuitePattern(1) +

		display.SuiteInitRegexp(token, 2) +
		display.TestTitleRegexp(2, 1) +
		display.TestStdoutRegexp(2, 1) +
		display.TestStderrRegexp(2, 1) +
		display.TestTitleRegexp(2, 2) +
		display.TestStdoutRegexp(2, 2) +
		display.TestStderrRegexp(2, 2) +
		display.ReportSuitePattern(2) +

		display.SuiteInitRegexp(token, 3) +
		display.TestTitleRegexp(3, 1) +
		display.TestStdoutRegexp(3, 1) +
		display.TestStderrRegexp(3, 1) +
		display.TestTitleRegexp(3, 2) +
		display.TestStdoutRegexp(3, 2) +
		display.TestStderrRegexp(3, 2) +
		display.ReportSuitePattern(3) +
		"$")
	assert.Regexp(t, scenarioRegexp, ansi.Unformat(errW.String()))
}

func TestAsyncDisplayUsage_AsyncSuitesAsyncTests(t *testing.T) {
	//t.Skip()
	token := "foo70"
	isol := "bar70"

	var err error
	err = repo.ClearWorkDirectory(token, isol)
	require.NoError(t, err)

	d := New(token, isol)
	d.SetVerbose(model.SHOW_ALL)

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

	display.DisplaySuite(d, token, isol, 1) // 100- Init suite1

	display.DisplayTestTitle(t, d, token, isol, 1, 1) // 110- Test suite1 #1
	display.DisplayTestTitle(t, d, token, isol, 1, 2) // 120- Test suite1 #2

	display.DisplayTestOut(t, d, token, isol, 1, 1) // 111- Test suite1 #1 out>

	display.DisplayTestOut(t, d, token, isol, 1, 2) // 121- Test suite1 #2 out>
	display.DisplayTestErr(t, d, token, isol, 1, 2) // 122- Test suite1 #2 err>
	display.DisplayEndTest(t, d, token, isol, 1, 2)

	display.DisplayTestErr(t, d, token, isol, 1, 1) // 112- Test suite1 #1 err>
	display.DisplayEndTest(t, d, token, isol, 1, 1)

	display.DisplaySuite(d, token, isol, 2) // 200- Init suite2
	display.DisplaySuite(d, token, isol, 3) // 300- Init suite3

	display.DisplayTestTitle(t, d, token, isol, 3, 1) // 310- Test suite3 #1
	display.DisplayTestOut(t, d, token, isol, 3, 1)   // 311- Test suite3 #1 out>

	display.DisplayTestTitle(t, d, token, isol, 2, 1) // 210- Test suite2 #1
	display.DisplayTestOut(t, d, token, isol, 2, 1)   // 211- Test suite2 #1 out>

	display.DisplayTestTitle(t, d, token, isol, 2, 2) // 220- Test suite2 #2
	display.DisplayTestOut(t, d, token, isol, 2, 2)   // 221- Test suite2 #2 out>

	display.DisplayTestErr(t, d, token, isol, 2, 1) // 212- Test suite2 #1 err>
	display.DisplayEndTest(t, d, token, isol, 2, 1)

	display.DisplayTestErr(t, d, token, isol, 2, 2) // 222- Test suite2 #2 err>
	display.DisplayEndTest(t, d, token, isol, 2, 2)
	display.DisplayReport(d, 2) // 270- Report suite2

	display.DisplayTestTitle(t, d, token, isol, 1, 3) // 130- Test suite1 #3
	display.DisplayTestOut(t, d, token, isol, 1, 3)   // 131- Test suite1 #3 out>
	display.DisplayTestErr(t, d, token, isol, 1, 3)   // 132- Test suite1 #3 err>
	display.DisplayEndTest(t, d, token, isol, 1, 3)
	display.DisplayReport(d, 1) // 170- Report suite1

	display.DisplayTestTitle(t, d, token, isol, 3, 2) // 320- Test suite3 #2

	display.DisplayTestErr(t, d, token, isol, 3, 1) // 312- Test suite3 #1 err>
	display.DisplayEndTest(t, d, token, isol, 3, 1)

	display.DisplayTestOut(t, d, token, isol, 3, 2) // 321- Test suite3 #2 out>
	display.DisplayTestErr(t, d, token, isol, 3, 2) // 322- Test suite3 #2 err>
	display.DisplayEndTest(t, d, token, isol, 3, 2)

	display.DisplayReport(d, 3) // 370- Report suite3

	assert.Empty(t, outW.String())
	assert.Empty(t, errW.String())

	err = d.AsyncFlushAll(100 * time.Millisecond)
	require.NoError(t, err)
	err = d.BlockTailAll(100 * time.Millisecond)
	require.NoError(t, err)

	assert.Empty(t, ansi.Unformat(outW.String()))

	scenarioRegexp := regexp.MustCompile("^" +
		display.GlobalInitPattern(token) +
		display.SuiteInitRegexp(token, 1) +
		display.TestTitleRegexp(1, 1) +
		display.TestStdoutRegexp(1, 1) +
		display.TestStderrRegexp(1, 1) +
		display.TestTitleRegexp(1, 2) +
		display.TestStdoutRegexp(1, 2) +
		display.TestStderrRegexp(1, 2) +
		display.TestTitleRegexp(1, 3) +
		display.TestStdoutRegexp(1, 3) +
		display.TestStderrRegexp(1, 3) +
		display.ReportSuitePattern(1) +

		display.SuiteInitRegexp(token, 2) +
		display.TestTitleRegexp(2, 1) +
		display.TestStdoutRegexp(2, 1) +
		display.TestStderrRegexp(2, 1) +
		display.TestTitleRegexp(2, 2) +
		display.TestStdoutRegexp(2, 2) +
		display.TestStderrRegexp(2, 2) +
		display.ReportSuitePattern(2) +

		display.SuiteInitRegexp(token, 3) +
		display.TestTitleRegexp(3, 1) +
		display.TestStdoutRegexp(3, 1) +
		display.TestStderrRegexp(3, 1) +
		display.TestTitleRegexp(3, 2) +
		display.TestStdoutRegexp(3, 2) +
		display.TestStderrRegexp(3, 2) +
		display.ReportSuitePattern(3) +
		"$")
	assert.Regexp(t, scenarioRegexp, ansi.Unformat(errW.String()))
}
