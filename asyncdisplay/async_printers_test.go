package asyncdisplay

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"mby.fr/cmdtest/repo"
	"mby.fr/utils/filez"
)

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

func TestAsyncPrinters_testFlushOrder(t *testing.T) {
	outW := &strings.Builder{}
	errW := &strings.Builder{}
	aps := newAsyncPrinters("token", "isol", outW, errW)

	suite1 := "suite1"
	suite2 := "suite2"

	p1 := aps.printer(suite1, 1)
	p3 := aps.printer(suite1, 3)
	p4 := aps.printer(suite1, 4)
	p1.Out("test1\n")
	aps.testEnded(suite1, 1)
	time.Sleep(100 * time.Millisecond)

	p3.Out("test3\n")
	p4.Out("test4\n")
	assert.Empty(t, outW.String())
	assert.Empty(t, errW.String())
	aps.testEnded(suite1, 4)
	aps.testEnded(suite1, 3)
	time.Sleep(100 * time.Millisecond)

	p2 := aps.printer(suite1, 2)
	p2.Out("test2\n")
	assert.Empty(t, outW.String())
	assert.Empty(t, errW.String())
	aps.testEnded(suite1, 2)
	time.Sleep(100 * time.Millisecond)

	aps.flush(suite2, false)
	assert.Empty(t, outW.String())
	assert.Empty(t, errW.String())

	p5 := aps.printer(suite2, 1)
	p5.Out("test5\n")

	aps.flush(suite2, true)
	assert.Equal(t, "test5\n", outW.String())
	assert.Empty(t, errW.String())

	aps.flush(suite1, true)
	aps.flush(suite1, true)
	aps.flush(suite1, true)
	aps.flush(suite1, true)
	assert.NotEmpty(t, outW.String())
	assert.Equal(t, "test5\ntest1\ntest2\ntest3\ntest4\n", outW.String())
	assert.Empty(t, errW.String())
}

func TestAsyncPrinters_testFlushOrder_nilWriters(t *testing.T) {
	aps := newAsyncPrinters("token", "isol", nil, nil)

	suite1 := "suite1"
	suite2 := "suite2"

	stdoutFile, stderrFile, _, _, err := repo.DaemonSuiteReportFilepathes(suite1, "token", "isol")
	require.NoError(t, err)

	p1 := aps.printer(suite1, 1)
	p3 := aps.printer(suite1, 3)
	p4 := aps.printer(suite1, 4)
	p1.Out("test1\n")
	aps.testEnded(suite1, 1)
	time.Sleep(100 * time.Millisecond)

	p3.Out("test3\n")
	p4.Out("test4\n")
	assert.Empty(t, func() string { s, _ := filez.ReadString(stdoutFile); return s }())
	assert.Empty(t, func() string { s, _ := filez.ReadString(stderrFile); return s }())
	aps.testEnded(suite1, 4)
	aps.testEnded(suite1, 3)
	time.Sleep(100 * time.Millisecond)

	p2 := aps.printer(suite1, 2)
	p2.Out("test2\n")
	assert.Empty(t, func() string { s, _ := filez.ReadString(stdoutFile); return s }())
	assert.Empty(t, func() string { s, _ := filez.ReadString(stderrFile); return s }())
	aps.testEnded(suite1, 2)
	time.Sleep(100 * time.Millisecond)

	aps.flush(suite2, false)
	assert.Empty(t, func() string { s, _ := filez.ReadString(stdoutFile); return s }())
	assert.Empty(t, func() string { s, _ := filez.ReadString(stderrFile); return s }())

	p5 := aps.printer(suite2, 1)
	p5.Out("test5\n")

	aps.flush(suite2, true)

	aps.flush(suite1, true)
	aps.flush(suite1, true)
	aps.flush(suite1, true)
	aps.flush(suite1, true)
	assert.NotEmpty(t, func() string { s, _ := filez.ReadString(stdoutFile); return s }())
	assert.Equal(t, "test1\ntest2\ntest3\ntest4\n", func() string { s, _ := filez.ReadString(stdoutFile); return s }())
	assert.Empty(t, func() string { s, _ := filez.ReadString(stderrFile); return s }())
}

func TestAsyncPrinters_testFlushOrder_nilWriters_2(t *testing.T) {
	aps := newAsyncPrinters("token", "isol", nil, nil)

	suite1 := "suite1"

	stdoutFile, stderrFile, _, _, err := repo.DaemonSuiteReportFilepathes(suite1, "token", "isol")
	require.NoError(t, err)

	p1 := aps.printer(suite1, 1)
	p3 := aps.printer(suite1, 3)
	p4 := aps.printer(suite1, 4)
	p1.Out("test1\n")
	aps.testEnded(suite1, 1)
	time.Sleep(100 * time.Millisecond)

	p3.Out("test3\n")
	p4.Out("test4\n")
	assert.Empty(t, func() string { s, _ := filez.ReadString(stdoutFile); return s }())
	assert.Empty(t, func() string { s, _ := filez.ReadString(stderrFile); return s }())
	aps.testEnded(suite1, 4)
	aps.testEnded(suite1, 3)
	time.Sleep(100 * time.Millisecond)

	aps.flush(suite1, true)
	aps.flush(suite1, true)
	aps.flush(suite1, true)

	assert.Equal(t, "test1\n", func() string { s, _ := filez.ReadString(stdoutFile); return s }())
	assert.Empty(t, func() string { s, _ := filez.ReadString(stderrFile); return s }())

	p2 := aps.printer(suite1, 2)
	p2.Out("test2\n")
	assert.Equal(t, "test1\n", func() string { s, _ := filez.ReadString(stdoutFile); return s }())
	assert.Empty(t, func() string { s, _ := filez.ReadString(stderrFile); return s }())
	aps.testEnded(suite1, 2)
	time.Sleep(100 * time.Millisecond)

	aps.flush(suite1, false)
	assert.Equal(t, "test1\ntest2\ntest3\ntest4\n", func() string { s, _ := filez.ReadString(stdoutFile); return s }())
	assert.Empty(t, func() string { s, _ := filez.ReadString(stderrFile); return s }())
}
