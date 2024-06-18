package display

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"mby.fr/utils/printz"
)

func TestUsage(t *testing.T) {
	out := &strings.Builder{}
	err := &strings.Builder{}
	d := NewAsync("foo", "bar")
	d.stdPrinter = printz.New(printz.NewOutputs(out, err))

	assert.Empty(t, out.String())
	assert.Empty(t, err.String())

	// Writing async
	outMsg := "stdout"
	errMsg := "stderr"
	d.Stdout(outMsg)
	d.Stderr(errMsg)

	assert.Empty(t, out.String())
	assert.Empty(t, err.String())

	d.DisplayAllRecorded()

	assert.Equal(t, outMsg, out.String())
	assert.Equal(t, errMsg, err.String())

}
