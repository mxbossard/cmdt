package display

import (
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
	sps := newSuitePrinters("token", "isol", "suite")
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
	sps := newSuitePrinters("token", "isol", "suite")
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
	aps := newAsyncPrinters("token", "isol", outW, errW)

	p := aps.printer("", 0)
	outW.Reset()
	errW.Reset()
	expectedOut := "fooOut"
	p.Out(expectedOut)
	assert.Empty(t, outW.String())
	assert.Empty(t, errW.String())
	assert.Len(t, aps.recordedSuites(), 1)
	assert.Contains(t, aps.recordedSuites(), "")

	err := aps.flush("foo", true)
	assert.NoError(t, err)
	assert.Empty(t, outW.String())
	assert.Empty(t, errW.String())
	assert.Len(t, aps.recordedSuites(), 1)
	assert.Contains(t, aps.recordedSuites(), "")

	outW.Reset()
	errW.Reset()
	err = aps.flush("", true)
	require.NoError(t, err)
	assert.Equal(t, expectedOut, outW.String())
	assert.Empty(t, errW.String())
	assert.Len(t, aps.recordedSuites(), 0)
}

func TestAsyncPrinters_testPrint(t *testing.T) {
	outW := &strings.Builder{}
	errW := &strings.Builder{}
	aps := newAsyncPrinters("token", "isol", outW, errW)

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
	assert.NoError(t, err)
	assert.Empty(t, outW.String())
	assert.Empty(t, errW.String())

	err = aps.flush("", true)
	assert.NoError(t, err)
	assert.Empty(t, outW.String())
	assert.Empty(t, errW.String())

	err = aps.flush(suite1, true)
	require.NoError(t, err)
	assert.Equal(t, expectedOut1, outW.String())
	assert.Empty(t, errW.String())

	outW.Reset()
	errW.Reset()
	err = aps.flush(suite1, true)
	assert.NoError(t, err)
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
	assert.NoError(t, err)
	assert.Empty(t, outW.String())
	assert.Empty(t, errW.String())

	outW.Reset()
	errW.Reset()
	err = aps.flush(suite1, true)
	assert.NoError(t, err)
	assert.Empty(t, outW.String())
	assert.Empty(t, errW.String())

}
