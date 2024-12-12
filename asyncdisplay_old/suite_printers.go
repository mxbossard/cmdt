package asyncdisplay_old

import (
	"fmt"
	"io"
	"os"
	"time"

	"mby.fr/cmdtest/repo"
	"mby.fr/utils/inout"
	"mby.fr/utils/printz"
)

func clearFileWriters(token, isol, suite string) error {
	outFile, errFile, doneFile, flushedFile, err := repo.DaemonSuiteReportFilepathes(suite, token, isol)
	if err != nil {
		panic(err)
	}
	logger.Info("removing recorder files", "token", token, "isol", isol, "suite", suite, "outFile", outFile, "errFile", errFile, "doneFile", doneFile, "flushedFile", flushedFile)

	if _, err := os.Stat(outFile); err == nil {
		err = os.Remove(outFile)
		if err != nil {
			return err
		}
	}
	if _, err := os.Stat(errFile); err == nil {
		err = os.Remove(errFile)
		if err != nil {
			return err
		}
	}
	if _, err := os.Stat(doneFile); err == nil {
		err = os.Remove(doneFile)
		if err != nil {
			return err
		}
	}
	if _, err := os.Stat(flushedFile); err == nil {
		err = os.Remove(flushedFile)
		if err != nil {
			return err
		}
	}
	return nil
}

func newSuitePrinters(token, isol, suite string) *suitePrinters { //, outW, errW io.Writer
	err := clearFileWriters(token, isol, suite)
	if err != nil {
		panic(err)
	}

	return &suitePrinters{
		token: token,
		isol:  isol,
		suite: suite,
		// outW:      outW,
		// errW:      errW,
		tests:     make(map[int]printz.Printer),
		closed:    make(map[int]bool),
		startTime: time.Now(),
		cursor:    0,
	}
}

type suitePrinters struct {
	suite, token, isol string
	outW, errW         io.Writer
	tests              map[int]printz.Printer
	closed             map[int]bool
	cursor, ended, max int
	startTime          time.Time
}

func (p *suitePrinters) testPrinter(seq int) (printz.Printer, error) {
	p.max = max(p.max, seq)
	printer, ok := p.tests[seq]
	if !ok {
		if p.outW == nil {
			stdout, stderr, _, _, err := repo.DaemonSuiteReportFilepathes(p.suite, p.token, p.isol)
			if err != nil {
				return nil, err
			}
			p.outW, p.errW, err = newFileWriters(stdout, stderr)
			if err != nil {
				return nil, err
			}
			logger.Debug("initialized new test file recorder", "suite", p.suite, "stdout", stdout, "stderr", stderr)
		}
		debOut := inout.CallbackWriter{
			Nested: p.outW,
			Callback: func(b []byte) {
				logger.Debug("test printer writing on out", "suite", p.suite, "seq", seq, "printed", string(b))
			},
		}
		outs := printz.NewOutputs(&debOut, p.errW)
		//outs := printz.NewOutputs(p.outW, p.errW)
		//bufferedOuts := printz.NewBufferedOutputs(outs)
		printer = printz.New(outs)
		p.tests[seq] = printer
		// p.testsBuffer[seq] = bufferedOuts
		logger.Debug("created new test printer", "suite", p.suite, "seq", seq)
	}
	return printer, nil
}

func (p *suitePrinters) testEnded(seq int) {
	p.ended = max(p.ended, seq)
	p.closed[seq] = true
}

func (p *suitePrinters) flush0() (done bool, err error) {
	logger.Debug("flushing suite printers", "suite", p.suite, "max", p.max, "cursor", p.cursor, "end", p.ended)

	// FIXME: Current implem need all test printers to be registered before starting to flush.

	// 1- flush suite until first test printer is open
	// 2- flush cursor test printer if available until ended
	// 3- increment cursor
	// 4- flush suite

	if time.Since(p.startTime) > NoActivityTimeout {
		err = fmt.Errorf("timeout flushing async display after %s", NoActivityTimeout)
	}

	//prtr := p.tests[p.cursor]
	for i := 0; i <= p.max; i++ {
		if prtr, ok := p.tests[i]; ok && prtr != nil {
			// flush cursor test printer
			logger.Debug("flushing test printer", "suite", p.suite, "seq", i)
			//prtr.Err("flush>1")
			prtr.Flush()
		} else if i > 0 {
			// Next printer not available yet
			logger.Debug("test printer not available yet", "suite", p.suite, "seq", i)
			break
		}
	}

	if p.cursor <= p.ended {
		// current printer is done
		p.cursor++
		p.startTime = time.Now()
	}

	if p.cursor >= len(p.tests) {
		// All printers are done
		done = true
	}
	return
}

func (p *suitePrinters) flush() (done bool, err error) {
	logger.Debug("flushing suite printers", "suite", p.suite, "max", p.max, "cursor", p.cursor, "end", p.ended)

	// Flush is done when all registered test printer are ended and where flushed
	// Flush all printers in order, stop when a printer is missing.
	// Printer 0 is the suite printer

	// cursor: current not ended printer seq to flush
	// max: max printer seq registered
	// ended: highest printer seq ended

	if time.Since(p.startTime) > NoActivityTimeout {
		err = fmt.Errorf("timeout flushing async display after %s", NoActivityTimeout)
	}

	lastFlushed := 0
	for i := p.cursor; i <= p.max; i++ {
		if prtr, ok := p.tests[i]; ok && prtr != nil {
			// flush cursor test printer
			logger.Debug("flushing test printer", "suite", p.suite, "seq", i)
			prtr.Flush()
			lastFlushed = i
		} else if i > 0 {
			// Next printer not available yet
			logger.Debug("test printer not available yet", "suite", p.suite, "seq", i)
			if prtr, ok := p.tests[0]; ok && prtr != nil {
				// Flush printer 0
				logger.Debug("flushing suite printer", "suite", p.suite)
				prtr.Flush()
			}
			break
		}

		if _, ok := p.closed[i]; i > 0 && !ok {
			// if printer not closed stop flushing
			logger.Debug("printer not closed", "suite", p.suite, "i", i)
			break
		} else {
			if prtr, ok := p.tests[0]; ok && prtr != nil {
				// Flush printer 0
				logger.Debug("flushing suite printer", "suite", p.suite)
				prtr.Flush()
			}
		}
	}

	p.cursor = lastFlushed
	if _, ok := p.closed[lastFlushed]; lastFlushed != 0 && ok {
		// current printer is done
		p.cursor = lastFlushed + 1
		p.startTime = time.Now()
	}

	logger.Debug("flushed suite printers", "suite", p.suite, "max", p.max, "cursor", p.cursor, "end", p.ended, "lastFlushed", lastFlushed)

	if p.max == lastFlushed && p.max == p.ended {
		// All printers are done
		done = true
	}
	return
}
