package display

import (
	"os"

	"mby.fr/utils/printz"
)

type FiledDisplay struct {
	*basicDisplay
}

func (d *FiledDisplay) SetOutputFiles(out, err *os.File) {
	outs := printz.NewOutputs(out, err)
	d.basicDisplay.notQuietPrinter = printz.New(outs)
}

func NewFiled() (d *FiledDisplay) {
	return &FiledDisplay{
		basicDisplay: New(),
	}
}
