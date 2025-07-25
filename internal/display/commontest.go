package display

import (
	"fmt"
	"testing"
	"time"

	"cmdt/internal/facade"
	"cmdt/internal/model"

	"github.com/mxbossard/utilz/cmdz"
	"github.com/stretchr/testify/require"
)

func DisplaySuite(d Displayer, token, isol string, suite int) {
	ctx := facade.NewSuiteContext(token, isol, fmt.Sprintf("suite-%d", suite), true, model.InitAction, model.Config{}, false)
	d.OpenSuite(ctx)
	d.SuiteTitle(ctx)
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

func CloseSuite(d Displayer, suite int, token, isol string) {
	ctx := facade.NewSuiteContext(token, isol, fmt.Sprintf("suite-%d", suite), true, model.InitAction, model.Config{}, false)
	d.CloseSuite(ctx, "test")
}

func DisplayOpenTitleOutcomeTest(t *testing.T, d Displayer, token, isol string, suite int, seq int) TestDisplayer {
	testSuite := fmt.Sprintf("suite-%d", suite)
	ctx, err := facade.NewTestContext(token, isol, testSuite, uint(seq), model.Config{}, uint32(42), false)
	require.NoError(t, err)
	ctx.CmdExec = cmdz.Cmd("true")
	outcome := model.TestOutcome{
		Outcome:  model.FAILED,
		Duration: 3 * time.Millisecond,
		TestSignature: model.TestSignature{
			TestSuite: testSuite,
			TestName:  "",
			Seq:       uint(seq),
		},
	}
	td := d.OpenTest(ctx)
	td.Title()
	td.Outcome(outcome)
	return td
}

func DisplayTestOut(t *testing.T, td TestDisplayer, suite int, seq int) {
	td.Stdout(fmt.Sprintf("suite-%d-%d-out\n", suite, seq))
}

func DisplayTestErr(t *testing.T, td TestDisplayer, suite int, seq int) {
	td.Stderr(fmt.Sprintf("suite-%d-%d-err\n", suite, seq))
}

func DisplayEndTest(t *testing.T, td TestDisplayer, suite int, seq int) {
	td.Close()
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
	return fmt.Sprintf(`Successfully ran \[ suite-%d\s* \] test suite in    [\d.]+ s \(\s*\d+ success\)\s*\n`, suite)
}
