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

	d := NewAsync(token, isol)
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

	d := NewAsync(token, isol)
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

func displayTestTitle(t *testing.T, d Displayer, token, isol string, suite int, seq int) {
	testSuite := fmt.Sprintf("suite-%d", suite)
	ctx, err := facade.NewTestContext(token, isol, testSuite, uint16(seq), model.Config{}, uint32(42))
	require.NoError(t, err)
	ctx.CmdExec = cmdz.Cmd("true")
	outcome := model.TestOutcome{
		Outcome:  model.FAILED,
		Duration: 3 * time.Millisecond,
		TestSignature: model.TestSignature{
			TestSuite: testSuite,
			TestName:  "",
			Seq:       uint16(seq),
		},
	}
	d.OpenTest(ctx)
	d.TestTitle(ctx)
	d.TestOutcome(ctx, outcome)
}

func displayTestOut(t *testing.T, d Displayer, token, isol string, suite int, seq int) {
	ctx, err := facade.NewTestContext(token, isol, fmt.Sprintf("suite-%d", suite), uint16(seq), model.Config{}, uint32(42))
	require.NoError(t, err)
	ctx.CmdExec = cmdz.Cmd("true")
	d.OpenTest(ctx)
	d.TestStdout(ctx, fmt.Sprintf("suite-%d-%d-out\n", suite, seq))
}

func displayTestErr(t *testing.T, d Displayer, token, isol string, suite int, seq int) {
	ctx, err := facade.NewTestContext(token, isol, fmt.Sprintf("suite-%d", suite), uint16(seq), model.Config{}, uint32(42))
	require.NoError(t, err)
	ctx.CmdExec = cmdz.Cmd("true")
	d.OpenTest(ctx)
	d.TestStderr(ctx, fmt.Sprintf("suite-%d-%d-err\n", suite, seq))
}

func displayEndTest(t *testing.T, d Displayer, token, isol string, suite int, seq int) {
	ctx, err := facade.NewTestContext(token, isol, fmt.Sprintf("suite-%d", suite), uint16(seq), model.Config{}, uint32(42))
	require.NoError(t, err)
	ctx.CmdExec = cmdz.Cmd("true")
	d.OpenTest(ctx)
	d.CloseTest(ctx)
}

func globalInitPattern(token string) string {
	return fmt.Sprintf(`## New config \(token: %s\)\n`, token)
}

func suiteInitRegexp(token string, suite int) string {
	return fmt.Sprintf(`## Test suite \[suite-%d\] \(token: %s\)\n`, suite, token)
}

func testTitleRegexp(suite, seq int) string {
	return fmt.Sprintf(`\[\d+\] Test \[suite-%d\]\(on host\)>true #0%d...\s*FAILED \(in \dms\)\n\s+Executing cmd:\s+\[\w+\]\s*\n`, suite, seq)
}

func testStdoutRegexp(suite, seq int) string {
	return fmt.Sprintf(`suite-%d-%d-out\n`, suite, seq)
}

func testStderrRegexp(suite, seq int) string {
	return fmt.Sprintf(`suite-%d-%d-err\n`, suite, seq)
}

func reportSuitePattern(suite int) string {
	return fmt.Sprintf(`Successfuly ran  \[ suite-%d\s* \] test suite in    [\d.]+ s \(\s*\d+ success\)\s*\n`, suite)
}

func TestBlockTail(t *testing.T) {
	//t.Skip()
	token := "foo10"
	isol := "bar10"
	var err error

	err = repo.ClearWorkDirectory(token, isol)
	require.NoError(t, err)

	d := NewAsync(token, isol)
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
	displaySuite(d, token, isol, 1) // 100- Init suite1

	// Simulate outputs sent disordered
	displayTestTitle(t, d, token, isol, 1, 1)
	displayTestOut(t, d, token, isol, 1, 1)
	displayTestErr(t, d, token, isol, 1, 1)
	displayEndTest(t, d, token, isol, 1, 1)

	displayTestTitle(t, d, token, isol, 1, 3)
	displayTestOut(t, d, token, isol, 1, 3)
	displayTestErr(t, d, token, isol, 1, 3)
	displayEndTest(t, d, token, isol, 1, 3)

	displayTestTitle(t, d, token, isol, 1, 2)
	displayTestOut(t, d, token, isol, 1, 2)
	displayTestErr(t, d, token, isol, 1, 2)
	displayEndTest(t, d, token, isol, 1, 2)

	displayReport(d, 1)

	assert.Empty(t, outW.String())
	assert.Empty(t, errW.String())

	require.NoError(t, err)
	err = d.AsyncFlush("suite-1", 100*time.Millisecond)
	require.NoError(t, err)
	err = d.BlockTail("suite-1", 100*time.Millisecond)
	require.NoError(t, err)

	outScenarioRegexp := regexp.MustCompile("^" +
		testStdoutRegexp(1, 1) +
		testStdoutRegexp(1, 2) +
		testStdoutRegexp(1, 3) +
		"$")
	assert.Regexp(t, outScenarioRegexp, ansi.Unformat(outW.String()))
	// Expect scénario to be oredred test1, test2, test4
	errScenarioRegexp := regexp.MustCompile("^" +
		suiteInitRegexp(token, 1) +
		testTitleRegexp(1, 1) +
		testStderrRegexp(1, 1) +
		testTitleRegexp(1, 2) +
		testStderrRegexp(1, 2) +
		testTitleRegexp(1, 3) +
		testStderrRegexp(1, 3) +
		reportSuitePattern(1) +
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

	d := NewAsync(token, isol)
	d.SetVerbose(model.SHOW_ALL)

	// Replace stdPrinter std outputs by 2 string builders
	outW := &strings.Builder{}
	errW := &strings.Builder{}
	d.stdPrinter = printz.New(printz.NewOutputs(outW, errW))

	assert.Empty(t, outW.String())
	assert.Empty(t, errW.String())

	// Start 1 test in one suite
	displaySuite(d, token, isol, 1) // 100- Init suite1

	displayTestTitle(t, d, token, isol, 1, 1)
	displayTestOut(t, d, token, isol, 1, 1)
	displayTestErr(t, d, token, isol, 1, 1)
	displayEndTest(t, d, token, isol, 1, 1)

	displayReport(d, 1)

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
		suiteInitRegexp(token, 1) +
		testTitleRegexp(1, 1) +
		testStdoutRegexp(1, 1) +
		testStderrRegexp(1, 1) +
		reportSuitePattern(1) +
		"$")
	assert.Regexp(t, scenarioRegexp, ansi.Unformat(errW.String()))

	outW.Reset()
	errW.Reset()

	// Start another test in reinited suite
	displaySuite(d, token, isol, 1) // 100- Init suite1

	displayTestTitle(t, d, token, isol, 1, 1)
	displayTestOut(t, d, token, isol, 1, 1)
	//displayTestErr(t, d, token, isol, 1, 1)
	displayEndTest(t, d, token, isol, 1, 1)

	displayReport(d, 1)

	assert.Empty(t, outW.String())
	assert.Empty(t, errW.String())

	require.NoError(t, err)
	err = d.AsyncFlush("suite-1", 100*time.Millisecond)
	require.NoError(t, err)
	err = d.BlockTail("suite-1", 100*time.Millisecond)
	require.NoError(t, err)

	// Expect scénario to be test2
	scenarioRegexp = regexp.MustCompile("^" +
		suiteInitRegexp(token, 1) +
		testTitleRegexp(1, 1) +
		testStdoutRegexp(1, 1) +
		//testStderrRegexp(1, 1) +
		reportSuitePattern(1) +
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

	d := NewAsync(token, isol)
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
	displaySuite(d, token, isol, 1) // 100- Init suite1

	// Simulate outputs sent disordered
	displayTestTitle(t, d, token, isol, 1, 1)
	displayTestOut(t, d, token, isol, 1, 1)
	displayTestErr(t, d, token, isol, 1, 1)
	displayEndTest(t, d, token, isol, 1, 1)

	err = d.AsyncFlush("suite-1", 100*time.Millisecond)
	require.NoError(t, err)

	time.Sleep(10 * time.Millisecond)

	displayTestTitle(t, d, token, isol, 1, 3)
	displayTestOut(t, d, token, isol, 1, 3)
	displayTestErr(t, d, token, isol, 1, 3)
	displayEndTest(t, d, token, isol, 1, 3)

	time.Sleep(10 * time.Millisecond)

	displayTestTitle(t, d, token, isol, 1, 2)
	displayTestOut(t, d, token, isol, 1, 2)
	displayTestErr(t, d, token, isol, 1, 2)
	displayEndTest(t, d, token, isol, 1, 2)

	displayReport(d, 1)

	assert.Empty(t, outW.String())
	assert.Empty(t, errW.String())

	err = d.BlockTail("suite-1", 100*time.Millisecond)
	require.NoError(t, err)

	assert.Empty(t, ansi.Unformat(outW.String()))
	// Expect scénario to be oredred test1, test2, test3
	scenarioRegexp := regexp.MustCompile("^" +
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

	d := NewAsync(token, isol)
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
	displaySuite(d, token, isol, 1) // 100- Init suite1

	// Simulate outputs sent disordered
	displayTestTitle(t, d, token, isol, 1, 1)
	displayTestOut(t, d, token, isol, 1, 1)
	displayTestErr(t, d, token, isol, 1, 1)
	displayEndTest(t, d, token, isol, 1, 1)

	displayTestTitle(t, d, token, isol, 1, 3)
	displayTestOut(t, d, token, isol, 1, 3)
	displayTestErr(t, d, token, isol, 1, 3)
	displayEndTest(t, d, token, isol, 1, 3)

	displayTestTitle(t, d, token, isol, 1, 2)
	displayTestOut(t, d, token, isol, 1, 2)
	displayTestErr(t, d, token, isol, 1, 2)
	displayEndTest(t, d, token, isol, 1, 2)

	displayReport(d, 1)

	assert.Empty(t, outW.String())
	assert.Empty(t, errW.String())

	err = d.AsyncFlushAll(100 * time.Millisecond)
	require.NoError(t, err)
	err = d.BlockTailAll(100 * time.Millisecond)
	require.NoError(t, err)

	assert.Empty(t, ansi.Unformat(outW.String()))
	// Expect scénario to be oredred test1, test2, test3
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

	d := NewAsync(token, isol)
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
	displaySuite(d, token, isol, 1) // 100- Init suite1
	displayTestTitle(t, d, token, isol, 1, 1)
	displayTestTitle(t, d, token, isol, 1, 3)
	displayTestTitle(t, d, token, isol, 1, 2)

	assert.Empty(t, outW.String())
	assert.Empty(t, errW.String())

	err = d.AsyncFlushAll(1000 * time.Millisecond)
	require.NoError(t, err)

	// Simulate outputs sent disordered
	displayTestOut(t, d, token, isol, 1, 1)
	displayTestErr(t, d, token, isol, 1, 1)
	displayEndTest(t, d, token, isol, 1, 1)

	time.Sleep(10 * time.Millisecond)

	displayTestOut(t, d, token, isol, 1, 3)
	displayTestErr(t, d, token, isol, 1, 3)
	displayEndTest(t, d, token, isol, 1, 3)

	time.Sleep(10 * time.Millisecond)

	displayTestOut(t, d, token, isol, 1, 2)
	displayTestErr(t, d, token, isol, 1, 2)
	displayEndTest(t, d, token, isol, 1, 2)

	displayReport(d, 1)

	err = d.BlockTailAll(100 * time.Millisecond)
	require.NoError(t, err)

	assert.Empty(t, ansi.Unformat(outW.String()))
	// Expect scénario to be oredred test1, test2, test3
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

	d := NewAsync(token, isol)
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

	displaySuite(d, token, isol, 1) // 100- Init suite1
	displayTestTitle(t, d, token, isol, 1, 1)
	displayTestOut(t, d, token, isol, 1, 1)
	displayTestErr(t, d, token, isol, 1, 1)
	displayEndTest(t, d, token, isol, 1, 1)

	displayTestTitle(t, d, token, isol, 1, 2)
	displayTestOut(t, d, token, isol, 1, 2)
	displayTestErr(t, d, token, isol, 1, 2)
	displayEndTest(t, d, token, isol, 1, 2)
	displayTestTitle(t, d, token, isol, 1, 3)
	displayTestOut(t, d, token, isol, 1, 3)
	displayTestErr(t, d, token, isol, 1, 3)
	displayEndTest(t, d, token, isol, 1, 3)
	displayReport(d, 1)
	displaySuite(d, token, isol, 2) // 200- Init suite2
	displayTestTitle(t, d, token, isol, 2, 1)
	displayTestOut(t, d, token, isol, 2, 1)
	displayTestErr(t, d, token, isol, 2, 1)
	displayEndTest(t, d, token, isol, 2, 1)
	displayTestTitle(t, d, token, isol, 2, 2)
	displayTestOut(t, d, token, isol, 2, 2)
	displayTestErr(t, d, token, isol, 2, 2)
	displayEndTest(t, d, token, isol, 2, 2)
	displayReport(d, 2)

	displaySuite(d, token, isol, 3) // 300- Init suite3
	displayTestTitle(t, d, token, isol, 3, 1)
	displayTestOut(t, d, token, isol, 3, 1)
	displayTestErr(t, d, token, isol, 3, 1)
	displayEndTest(t, d, token, isol, 3, 1)
	displayTestTitle(t, d, token, isol, 3, 2)
	displayTestOut(t, d, token, isol, 3, 2)
	displayTestErr(t, d, token, isol, 3, 2)
	displayEndTest(t, d, token, isol, 3, 2)
	displayReport(d, 3)

	assert.Empty(t, outW.String())
	assert.Empty(t, errW.String())

	err = d.AsyncFlushAll(100 * time.Millisecond)
	require.NoError(t, err)
	err = d.BlockTailAll(100 * time.Millisecond)
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
	//t.Skip()
	token := "foo60"
	isol := "bar60"

	var err error
	err = repo.ClearWorkDirectory(token, isol)
	require.NoError(t, err)

	d := NewAsync(token, isol)
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

	displaySuite(d, token, isol, 1) // 100- Init suite1

	displayTestTitle(t, d, token, isol, 1, 1) // 110- Test suite1 #1
	displayTestOut(t, d, token, isol, 1, 1)   // 111- Test suite1 #1 out>
	displayTestErr(t, d, token, isol, 1, 1)   // 112- Test suite1 #1 err>
	displayEndTest(t, d, token, isol, 1, 1)

	displaySuite(d, token, isol, 2) // 200- Init suite2
	displaySuite(d, token, isol, 3) // 300- Init suite3

	displayTestTitle(t, d, token, isol, 3, 1) // 310- Test suite3 #1
	displayTestOut(t, d, token, isol, 3, 1)   // 311- Test suite3 #1 out>

	displayTestTitle(t, d, token, isol, 1, 2) // 120- Test suite1 #2
	displayTestTitle(t, d, token, isol, 2, 1) // 210- Test suite2 #1

	displayTestOut(t, d, token, isol, 1, 2) // 121- Test suite1 #2 out>
	displayTestErr(t, d, token, isol, 1, 2) // 122- Test suite1 #2 err>
	displayEndTest(t, d, token, isol, 1, 2)

	displayTestOut(t, d, token, isol, 2, 1) // 211- Test suite2 #1 out>
	displayTestErr(t, d, token, isol, 2, 1) // 212- Test suite2 #1 err>
	displayEndTest(t, d, token, isol, 2, 1)

	displayTestTitle(t, d, token, isol, 2, 2) // 220- Test suite2 #2

	displayTestErr(t, d, token, isol, 3, 1) // 312- Test suite3 #1 err>
	displayEndTest(t, d, token, isol, 3, 1)

	displayTestOut(t, d, token, isol, 2, 2) // 221- Test suite2 #2 out>
	displayTestErr(t, d, token, isol, 2, 2) // 222- Test suite2 #2 err>
	displayEndTest(t, d, token, isol, 2, 2)

	displayReport(d, 2) // 270- Report suite2

	displayTestTitle(t, d, token, isol, 1, 3) // 130- Test suite1 #3
	displayTestOut(t, d, token, isol, 1, 3)   // 131- Test suite1 #3 out>
	displayTestErr(t, d, token, isol, 1, 3)   // 132- Test suite1 #3 err>
	displayEndTest(t, d, token, isol, 1, 3)

	displayReport(d, 1) // 170- Report suite1

	displayTestTitle(t, d, token, isol, 3, 2) // 320- Test suite3 #2
	displayTestOut(t, d, token, isol, 3, 2)   // 321- Test suite3 #2 out>
	displayTestErr(t, d, token, isol, 3, 2)   // 322- Test suite3 #2 err>
	displayEndTest(t, d, token, isol, 3, 2)

	displayReport(d, 3) // 370- Report suite3

	assert.Empty(t, outW.String())
	assert.Empty(t, errW.String())

	err = d.AsyncFlushAll(100 * time.Millisecond)
	require.NoError(t, err)
	err = d.BlockTailAll(100 * time.Millisecond)
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
	//t.Skip()
	token := "foo70"
	isol := "bar70"

	var err error
	err = repo.ClearWorkDirectory(token, isol)
	require.NoError(t, err)

	d := NewAsync(token, isol)
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

	displaySuite(d, token, isol, 1) // 100- Init suite1

	displayTestTitle(t, d, token, isol, 1, 1) // 110- Test suite1 #1
	displayTestTitle(t, d, token, isol, 1, 2) // 120- Test suite1 #2

	displayTestOut(t, d, token, isol, 1, 1) // 111- Test suite1 #1 out>

	displayTestOut(t, d, token, isol, 1, 2) // 121- Test suite1 #2 out>
	displayTestErr(t, d, token, isol, 1, 2) // 122- Test suite1 #2 err>
	displayEndTest(t, d, token, isol, 1, 2)

	displayTestErr(t, d, token, isol, 1, 1) // 112- Test suite1 #1 err>
	displayEndTest(t, d, token, isol, 1, 1)

	displaySuite(d, token, isol, 2) // 200- Init suite2
	displaySuite(d, token, isol, 3) // 300- Init suite3

	displayTestTitle(t, d, token, isol, 3, 1) // 310- Test suite3 #1
	displayTestOut(t, d, token, isol, 3, 1)   // 311- Test suite3 #1 out>

	displayTestTitle(t, d, token, isol, 2, 1) // 210- Test suite2 #1
	displayTestOut(t, d, token, isol, 2, 1)   // 211- Test suite2 #1 out>

	displayTestTitle(t, d, token, isol, 2, 2) // 220- Test suite2 #2
	displayTestOut(t, d, token, isol, 2, 2)   // 221- Test suite2 #2 out>

	displayTestErr(t, d, token, isol, 2, 1) // 212- Test suite2 #1 err>
	displayEndTest(t, d, token, isol, 2, 1)

	displayTestErr(t, d, token, isol, 2, 2) // 222- Test suite2 #2 err>
	displayEndTest(t, d, token, isol, 2, 2)
	displayReport(d, 2) // 270- Report suite2

	displayTestTitle(t, d, token, isol, 1, 3) // 130- Test suite1 #3
	displayTestOut(t, d, token, isol, 1, 3)   // 131- Test suite1 #3 out>
	displayTestErr(t, d, token, isol, 1, 3)   // 132- Test suite1 #3 err>
	displayEndTest(t, d, token, isol, 1, 3)
	displayReport(d, 1) // 170- Report suite1

	displayTestTitle(t, d, token, isol, 3, 2) // 320- Test suite3 #2

	displayTestErr(t, d, token, isol, 3, 1) // 312- Test suite3 #1 err>
	displayEndTest(t, d, token, isol, 3, 1)

	displayTestOut(t, d, token, isol, 3, 2) // 321- Test suite3 #2 out>
	displayTestErr(t, d, token, isol, 3, 2) // 322- Test suite3 #2 err>
	displayEndTest(t, d, token, isol, 3, 2)

	displayReport(d, 3) // 370- Report suite3

	assert.Empty(t, outW.String())
	assert.Empty(t, errW.String())

	err = d.AsyncFlushAll(100 * time.Millisecond)
	require.NoError(t, err)
	err = d.BlockTailAll(100 * time.Millisecond)
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
