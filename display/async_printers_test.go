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
	zlog.ColoredConfig()
	zlog.SetLogLevelThreshold(zlog.LevelPerf)
	os.Exit(m.Run())
}

/*
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
*/

func TestSuitePrinters_testPrint(t *testing.T) {
	outW := &strings.Builder{}
	errW := &strings.Builder{}
	sps := newSuitePrinters("token", "isol", "suiteA")
	sps.outW = outW
	sps.errW = errW

	p, err := sps.testPrinter(0)
	require.NoError(t, err)
	p.Out("zero")
	done, err := sps.flush()
	require.NoError(t, err)
	assert.True(t, done)
	assert.Equal(t, "zero", outW.String())
	assert.Empty(t, errW.String())

	p, err = sps.testPrinter(1)
	require.NoError(t, err)

	outW.Reset()
	errW.Reset()
	expectedOut := "fooOut"
	p.Out(expectedOut)
	assert.Empty(t, outW.String())
	assert.Empty(t, errW.String())

	done, err = sps.flush()
	require.NoError(t, err)
	assert.False(t, done)
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
	assert.False(t, done)
	assert.Equal(t, expectedOut, outW.String())
	assert.Empty(t, errW.String())

	outW.Reset()
	errW.Reset()
	expectedErr := "fooErr"
	p.Err(expectedErr)
	sps.testEnded(1)
	assert.Empty(t, outW.String())
	assert.Empty(t, errW.String())

	done, err = sps.flush()
	require.NoError(t, err)
	assert.True(t, done)
	assert.Empty(t, outW.String())
	assert.Equal(t, expectedErr, errW.String())
}

func TestSuitePrinters_testFlushOrder(t *testing.T) {
	outW := &strings.Builder{}
	errW := &strings.Builder{}
	sps := newSuitePrinters("token", "isol", "suiteB")
	sps.outW = outW
	sps.errW = errW

	// Test flush before printer creation
	done, err := sps.flush()
	require.NoError(t, err)
	assert.True(t, done)

	// Expect nothing flushed
	assert.Empty(t, outW.String())
	assert.Empty(t, errW.String())

	p, err := sps.testPrinter(1)
	require.NoError(t, err)

	// Test flush before print
	done, err = sps.flush()
	require.NoError(t, err)
	assert.False(t, done)

	// Expect nothing flushed
	assert.Empty(t, outW.String())
	assert.Empty(t, errW.String())

	// First print
	expectedOut1 := "foo1"
	p.Out(expectedOut1)

	// First flush not ended
	assert.Empty(t, outW.String())
	assert.Empty(t, errW.String())
	done, err = sps.flush()
	require.NoError(t, err)
	assert.False(t, done)

	// Expect expectedOut1 flushed
	assert.Equal(t, expectedOut1, outW.String())
	assert.Empty(t, errW.String())

	// End first test
	sps.testEnded(1)
	done, err = sps.flush()
	require.NoError(t, err)
	assert.True(t, done)

	// Expect expectedOut1 flushed
	assert.Equal(t, expectedOut1, outW.String())
	assert.Empty(t, errW.String())

	// Print on third printer (bad order)
	p, err = sps.testPrinter(3)
	require.NoError(t, err)
	expectedOut3 := "bar3"
	p.Out(expectedOut3)
	sps.testEnded(3)

	// Second flush
	done, err = sps.flush()
	require.NoError(t, err)
	assert.False(t, done)

	// Expect nothing flushed (printer 2 does not exists yet)
	assert.Equal(t, expectedOut1, outW.String())
	assert.Empty(t, errW.String())

	// // Print on second printer (bad order)
	p, err = sps.testPrinter(2)
	require.NoError(t, err)
	expectedOut2 := "baz2"
	p.Out(expectedOut2)

	// Third flush
	done, err = sps.flush()
	require.NoError(t, err)
	assert.False(t, done)

	assert.Equal(t, expectedOut1+expectedOut2, outW.String())
	assert.Empty(t, errW.String())

	sps.testEnded(2)

	done, err = sps.flush()
	require.NoError(t, err)
	assert.True(t, done)

	assert.Equal(t, expectedOut1+expectedOut2+expectedOut3, outW.String())
	assert.Empty(t, errW.String())
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
	assert.Len(t, aps.recordedSuites(), 1)
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
