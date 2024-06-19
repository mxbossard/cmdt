package display

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"mby.fr/utils/printz"
)

func TestUsage(t *testing.T) {
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

}
