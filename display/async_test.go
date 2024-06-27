package display

import (
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"mby.fr/cmdtest/facade"
	"mby.fr/cmdtest/model"
	"mby.fr/utils/cmdz"
	"mby.fr/utils/printz"
	"mby.fr/utils/zlog"
)

func TestMain(m *testing.M) {
	// test context initialization here
	// zlog.ColoredConfig()
	zlog.SetLogLevelThreshold(zlog.LevelPerf)
	os.Exit(m.Run())
}

func TestSuitePrinters_suitePrint(t *testing.T) {
	outW := &strings.Builder{}
	errW := &strings.Builder{}
	sps := newSuitePrinters("suite")
	sps.outW = outW
	sps.errW = errW

	p, err := sps.suitePrinter()
	require.NoError(t, err)

	outW.Reset()
	errW.Reset()
	expectedOut := "fooOut"
	p.Out(expectedOut)
	assert.Empty(t, outW.String())
	assert.Empty(t, errW.String())

	done, err := sps.flush()
	require.NoError(t, err)
	assert.True(t, done)
	assert.Equal(t, expectedOut, outW.String())
	assert.Empty(t, errW.String())

	outW.Reset()
	errW.Reset()
	expectedOut = "barOut"
	p.Out(expectedOut)
	assert.Empty(t, outW.String())
	assert.Empty(t, errW.String())

	done, err = sps.flush()
	require.NoError(t, err)
	assert.True(t, done)
	assert.Equal(t, expectedOut, outW.String())
	assert.Empty(t, errW.String())

	outW.Reset()
	errW.Reset()
	expectedErr := "fooErr"
	p.Err(expectedErr)
	assert.Empty(t, outW.String())
	assert.Empty(t, errW.String())

	done, err = sps.flush()
	require.NoError(t, err)
	assert.True(t, done)
	assert.Empty(t, outW.String())
	assert.Equal(t, expectedErr, errW.String())

}

func TestSuitePrinters_testPrint(t *testing.T) {
	outW := &strings.Builder{}
	errW := &strings.Builder{}
	sps := newSuitePrinters("suite")
	sps.outW = outW
	sps.errW = errW

	p, err := sps.testPrinter(1)
	require.NoError(t, err)

	outW.Reset()
	errW.Reset()
	expectedOut := "fooOut"
	p.Out(expectedOut)
	assert.Empty(t, outW.String())
	assert.Empty(t, errW.String())

	done, err := sps.flush()
	require.NoError(t, err)
	assert.True(t, done)
	assert.Equal(t, expectedOut, outW.String())
	assert.Empty(t, errW.String())

	outW.Reset()
	errW.Reset()
	expectedOut = "barOut"
	p.Out(expectedOut)
	assert.Empty(t, outW.String())
	assert.Empty(t, errW.String())

	done, err = sps.flush()
	require.NoError(t, err)
	assert.True(t, done)
	assert.Equal(t, expectedOut, outW.String())
	assert.Empty(t, errW.String())

	outW.Reset()
	errW.Reset()
	expectedErr := "fooErr"
	p.Err(expectedErr)
	assert.Empty(t, outW.String())
	assert.Empty(t, errW.String())

	done, err = sps.flush()
	require.NoError(t, err)
	assert.True(t, done)
	assert.Empty(t, outW.String())
	assert.Equal(t, expectedErr, errW.String())

}

func TestAsyncPrinters_globalPrint(t *testing.T) {
	outW := &strings.Builder{}
	errW := &strings.Builder{}
	outs := printz.NewOutputs(outW, errW)
	prtr := printz.New(outs)
	aps := newAsyncPrinters(prtr)

	p := aps.printer("", 0)
	outW.Reset()
	errW.Reset()
	expectedOut := "fooOut"
	p.Out(expectedOut)
	assert.Empty(t, outW.String())
	assert.Empty(t, errW.String())

	err := aps.flush("foo", true)
	require.NoError(t, err)
	assert.Empty(t, outW.String())
	assert.Empty(t, errW.String())

	//return
	outW.Reset()
	errW.Reset()
	err = aps.flush("", true)
	require.NoError(t, err)
	assert.Equal(t, expectedOut, outW.String())
	assert.Empty(t, errW.String())
}

func TestAsyncPrinters_testPrint(t *testing.T) {
	outW := &strings.Builder{}
	errW := &strings.Builder{}
	outs := printz.NewOutputs(outW, errW)
	prtr := printz.New(outs)
	aps := newAsyncPrinters(prtr)

	suite1 := "suite1"
	suite2 := "suite2"
	p1 := aps.printer(suite1, 0)
	p2 := aps.printer(suite2, 0)

	outW.Reset()
	errW.Reset()
	expectedOut1 := suite1 + "Out"
	expectedOut2 := suite2 + "Out"
	p1.Out(expectedOut1)
	p2.Out(expectedOut2)
	assert.Empty(t, outW.String())
	assert.Empty(t, errW.String())

	err := aps.flush("", true)
	require.NoError(t, err)
	assert.Empty(t, outW.String())
	assert.Empty(t, errW.String())

	return

	err = aps.flush("", true)
	require.NoError(t, err)
	assert.Empty(t, outW.String())
	assert.Empty(t, errW.String())

	err = aps.flush(suite1, true)
	require.NoError(t, err)
	assert.Equal(t, expectedOut1, outW.String())
	assert.Empty(t, errW.String())

	outW.Reset()
	errW.Reset()
	err = aps.flush(suite1, true)
	require.NoError(t, err)
	assert.Empty(t, outW.String())
	assert.Empty(t, errW.String())

	outW.Reset()
	errW.Reset()
	err = aps.flush(suite2, true)
	require.NoError(t, err)
	assert.Equal(t, expectedOut2, outW.String())
	assert.Empty(t, errW.String())

	outW.Reset()
	errW.Reset()
	err = aps.flush(suite2, true)
	require.NoError(t, err)
	assert.Empty(t, outW.String())
	assert.Empty(t, errW.String())

	outW.Reset()
	errW.Reset()
	err = aps.flush(suite1, true)
	require.NoError(t, err)
	assert.Empty(t, outW.String())
	assert.Empty(t, errW.String())

}

func TestUsage(t *testing.T) {
	t.Skip()
	d := NewAsync("foo", "bar")
	// Replace stdPrinter std outputs by 2 string builders
	outW := &strings.Builder{}
	errW := &strings.Builder{}
	d.stdPrinter = printz.New(printz.NewOutputs(outW, errW))

	assert.Empty(t, outW.String())
	assert.Empty(t, errW.String())

	// Writing async
	outMsg := "stdout"
	errMsg := "stderr"
	d.Stdout(outMsg)
	d.Stderr(errMsg)

	assert.Empty(t, outW.String())
	assert.Empty(t, errW.String())

	err := d.DisplayAllRecorded()
	require.NoError(t, err)

	assert.Equal(t, outMsg, outW.String())
	assert.Equal(t, errMsg, errW.String())

	expectedTitle := "Test [suite]/true"
	ctx := facade.NewTestContext("token", "isol", "suite", 2, model.Config{}, 42)
	ctx.CmdExec = cmdz.Cmd("true")

	d.TestTitle(ctx)

	err = d.DisplayAllRecorded()
	require.NoError(t, err)

	assert.Equal(t, expectedTitle, errW.String())

}
