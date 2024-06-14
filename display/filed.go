package display

import (
	"os"

	"mby.fr/utils/printz"
)

type FiledDisplay struct {
	*BasicDisplay
}

func (d *FiledDisplay) SetOutputFiles(out, err *os.File) {
	outs := printz.NewOutputs(out, err)
	d.BasicDisplay.notQuietPrinter = printz.New(outs)
}

func NewFiled() (d *FiledDisplay) {
	return &FiledDisplay{
		BasicDisplay: New(),
	}
}
