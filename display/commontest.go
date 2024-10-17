package display

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"mby.fr/cmdtest/facade"
	"mby.fr/cmdtest/model"
	"mby.fr/utils/cmdz"
)

func DisplaySuite(d Displayer, token, isol string, suite int) {
	ctx := facade.NewSuiteContext(token, isol, fmt.Sprintf("suite-%d", suite), true, model.InitAction, model.Config{})
	d.Suite(ctx)
}

func DisplayReport(d Displayer, suite int) {
	outcome := model.SuiteOutcome{
		TestSuite:   fmt.Sprintf("suite-%d", suite),
		Duration:    3 * time.Millisecond,
		TestCount:   4,
		PassedCount: 4,
		Outcome:     model.PASSED,
	}
	d.ReportSuite(outcome)
}

func DisplayTestTitle(t *testing.T, d Displayer, token, isol string, suite int, seq int) {
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

func DisplayTestOut(t *testing.T, d Displayer, token, isol string, suite int, seq int) {
	ctx, err := facade.NewTestContext(token, isol, fmt.Sprintf("suite-%d", suite), uint16(seq), model.Config{}, uint32(42))
	require.NoError(t, err)
	ctx.CmdExec = cmdz.Cmd("true")
	d.OpenTest(ctx)
	d.TestStdout(ctx, fmt.Sprintf("suite-%d-%d-out\n", suite, seq))
}

func DisplayTestErr(t *testing.T, d Displayer, token, isol string, suite int, seq int) {
	ctx, err := facade.NewTestContext(token, isol, fmt.Sprintf("suite-%d", suite), uint16(seq), model.Config{}, uint32(42))
	require.NoError(t, err)
	ctx.CmdExec = cmdz.Cmd("true")
	d.OpenTest(ctx)
	d.TestStderr(ctx, fmt.Sprintf("suite-%d-%d-err\n", suite, seq))
}

func DisplayEndTest(t *testing.T, d Displayer, token, isol string, suite int, seq int) {
	ctx, err := facade.NewTestContext(token, isol, fmt.Sprintf("suite-%d", suite), uint16(seq), model.Config{}, uint32(42))
	require.NoError(t, err)
	ctx.CmdExec = cmdz.Cmd("true")
	d.OpenTest(ctx)
	d.CloseTest(ctx)
}

func GlobalInitPattern(token string) string {
	return fmt.Sprintf(`## New config \(token: %s\)\n`, token)
}

func SuiteInitRegexp(token string, suite int) string {
	return fmt.Sprintf(`## Test suite \[suite-%d\] \(token: %s\)\n`, suite, token)
}

func TestTitleRegexp(suite, seq int) string {
	return fmt.Sprintf(`\[\d+\] Test \[suite-%d\]\(on host\)>true #0%d...\s*FAILED \(in \dms\)\n\s+Executing cmd:\s+\[\w+\]\s*\n`, suite, seq)
}

func TestStdoutRegexp(suite, seq int) string {
	return fmt.Sprintf(`suite-%d-%d-out\n`, suite, seq)
}

func TestStderrRegexp(suite, seq int) string {
	return fmt.Sprintf(`suite-%d-%d-err\n`, suite, seq)
}

func ReportSuitePattern(suite int) string {
	return fmt.Sprintf(`Successfuly ran  \[ suite-%d\s* \] test suite in    [\d.]+ s \(\s*\d+ success\)\s*\n`, suite)
}
