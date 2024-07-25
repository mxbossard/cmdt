package display

import (
	"fmt"
	"io"
	"os"
	"sync"
	"time"

	"mby.fr/cmdtest/repo"
	"mby.fr/utils/printz"
)

const (
	FlushPeriod       = 1 * time.Millisecond
	NoActivityTimeout = 30 * time.Second
)

/**
## Purpose
- everithing wich can be displayed async should be done here
- all tests can be run in // but should be displayed in order
- in async mode display only on @report request

## Ideas
- display vs recorder ? => display reports, record everything else
- how to display if // report concurrent display request ? => ok: display record only once
- how to link suite deletion with record ? => ok: no need
- display recorded event after async call of StartDisplayRecorded()
- do I need to call a StopDisplayRecord() ?
- how to decide when a suite is done recording ? => ok: on StartDisplayRecorded() we can wait for all started test for a complete display (count TestTitle & TestOutcome by suite)

## Implem
- a global buffered printer
- for each suite a buffered printer
- for each test a buffered printer
- a scheduler wich flush buffers in order : global, suite1, test1, suite1, test2, suite1, ..., testN, suite1, global, suite2, ...
- StartDisplayRecorded() start the scheduler
- StopDisplayRecord() stop the scheduler
- can start flushing global instantly
- can start flushing suite after @init or first @test
- can start flushing test after init and after @test
- every async operations to display are buffered and flush in order into files.
- StartDisplayRecorded() will tail in order those files and print them into stdout & stderr alternatively.

## Open questions
- @report actions is never async ? => always print to stdout/stderr ?
- other actions should be writen in two separete files (stdout/stderr) ?
-

May need to split actions in half : init by cmdt ; processing by daemon => cmdt async return must guarantee action is inited and will be processed by daemon.
Or simpler may display testTitle on queueing the test ?
*/

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

func closeSuite(token, isol, suite string) error {
	_, _, doneFile, _, err := repo.DaemonSuiteReportFilepathes(suite, token, isol)
	if err != nil {
		return err
	}
	f, err := os.Create(doneFile)
	if err != nil {
		return err
	}
	err = f.Close()
	return err
}

type keepClosedFileWriter struct {
	filepath string
}

func (w keepClosedFileWriter) Write(b []byte) (int, error) {
	f, err := os.OpenFile(w.filepath, os.O_WRONLY+os.O_APPEND+os.O_CREATE, 0644)
	if err != nil {
		return 0, err
	}
	defer f.Close()
	return f.Write(b)
}

func newFileWriters(outFile, errFile string) (io.Writer, io.Writer, error) {
	//return keepClosedFileWriter{outFile}, keepClosedFileWriter{errFile}, nil
	outW, err := os.OpenFile(outFile, os.O_WRONLY+os.O_APPEND+os.O_CREATE, 0644)
	if err != nil {
		return nil, nil, err
	}
	errW, err := os.OpenFile(errFile, os.O_WRONLY+os.O_APPEND+os.O_CREATE, 0644)
	if err != nil {
		return nil, nil, err
	}
	return outW, errW, nil
}

func newFileReaders(outFile, errFile string) (*os.File, *os.File, error) {
	outW, err := os.OpenFile(outFile, os.O_RDONLY+os.O_CREATE, 0644)
	if err != nil {
		return nil, nil, err
	}
	errW, err := os.OpenFile(errFile, os.O_RDONLY+os.O_CREATE, 0644)
	if err != nil {
		return nil, nil, err
	}
	return outW, errW, nil
}

func newSuitePrinters(token, isol, suite string) *suitePrinters {
	err := clearFileWriters(token, isol, suite)
	if err != nil {
		panic(err)
	}

	return &suitePrinters{
		token:     token,
		isol:      isol,
		suite:     suite,
		tests:     make(map[int]printz.Printer),
		startTime: time.Now(),
		cursor:    1,
	}
}

type suitePrinters struct {
	inited             bool
	suite, token, isol string
	outW, errW         io.Writer
	//main               printz.Printer
	tests              map[int]printz.Printer
	cursor, ended, max int
	startTime          time.Time
}

/*
func (p *suitePrinters) suitePrinter() (printz.Printer, error) {
	if p.main == nil {
		if p.outW == nil {
			stdout, stderr, err := repo.DaemonSuiteReportFilepathes(p.suite, p.token, p.isol)
			if err != nil {
				return nil, err
			}
			p.outW, p.errW, err = newFileWriters(stdout, stderr)
			if err != nil {
				return nil, err
			}
			logger.Debug("initialized new suite file recorder", "suite", p.suite, "stdout", stdout, "stderr", stderr)
		}
		bufferedOuts := printz.NewBufferedOutputs(printz.NewOutputs(p.outW, p.errW))
		prtr := printz.New(bufferedOuts)
		p.main = prtr
		//p.mainBuffer = bufferedOuts
		logger.Debug("created new suite printer", "suite", p.suite)
	}
	p.startTime = time.Now()
	return p.main, nil
}
*/

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
		bufferedOuts := printz.NewBufferedOutputs(printz.NewOutputs(p.outW, p.errW))
		printer = printz.New(bufferedOuts)
		p.tests[seq] = printer
		// p.testsBuffer[seq] = bufferedOuts
		logger.Debug("created new test printer", "suite", p.suite, "seq", seq)
	}
	return printer, nil
}

func (p *suitePrinters) testEnded(seq int) {
	p.ended = seq
}

func (p *suitePrinters) flush() (done bool, err error) {
	// Current implem need all test printers to be registered before starting to flush.

	// 1- flush suite until first test printer is open
	// 2- flush cursor test printer if available until ended
	// 3- increment cursor
	// 4- flush suite

	if time.Since(p.startTime) > NoActivityTimeout {
		err = fmt.Errorf("timeout flushing async display after %s", NoActivityTimeout)
	}

	/*
		if !p.inited {
			// flush suite printer on init
			logger.Debug("flushing suite printer", "suite", p.suite)
			if p.main != nil {
				//p.main.Err("flush>0")
				p.main.Flush()
			}
			p.inited = true
		}
	*/

	//prtr := p.tests[p.cursor]
	for i := 0; i <= p.max; i++ {
		if prtr, ok := p.tests[i]; ok && prtr != nil {
			// flush cursor test printer
			logger.Debug("flushing test printer", "suite", p.suite, "seq", p.cursor)
			//prtr.Err("flush>1")
			prtr.Flush()
		}
	}

	if p.cursor <= p.ended {
		// current printer is done
		p.cursor++
		p.startTime = time.Now()
	}

	/*
		if p.main != nil {
			// flush suite printer
			logger.Debug("flushing suite printer", "suite", p.suite)
			//p.main.Err("flush>2")
			p.main.Flush()
		}
	*/

	if p.cursor >= len(p.tests) {
		// All printers are done
		done = true
	}
	return
}

func newAsyncPrinters(token, isol string, outW, errW io.Writer) *asyncPrinters {
	return &asyncPrinters{
		Mutex: &sync.Mutex{},

		token: token,
		isol:  isol,
		//globalPrinter:  global,
		suitesPrinters: make(map[string]*suitePrinters),
		outW:           outW,
		errW:           errW,
	}
}

type asyncPrinters struct {
	*sync.Mutex

	token, isol string
	//globalPrinter  printz.Printer
	suites         []string
	suitesPrinters map[string]*suitePrinters
	//currentSuite   string
	outW, errW io.Writer
}

func (p *asyncPrinters) printer(suite string, seq int) printz.Printer {
	/*
		if suite == "" {
			if p.globalPrinter == nil {
				stdOuts := printz.NewStandardOutputs()
				bufferedOuts := printz.NewBufferedOutputs(stdOuts)
				prtr := printz.New(bufferedOuts)
				p.globalPrinter = prtr
			}
			return p.globalPrinter
		}
	*/

	// Select the printer by suite
	var sprtr *suitePrinters
	var ok bool
	if sprtr, ok = p.suitesPrinters[suite]; !ok {
		sprtr = newSuitePrinters(p.token, p.isol, suite)
		p.suites = append(p.suites, suite)
		p.suitesPrinters[suite] = sprtr
		sprtr.outW = p.outW
		sprtr.errW = p.errW
		logger.Debug("created new suitePrinters", "suite", suite, "stored", p.suitesPrinters)
	}

	var prtr printz.Printer
	var err error
	if seq == 0 {
		//prtr, err = sprtr.suitePrinter()
		prtr, err = sprtr.testPrinter(seq)
	} else {
		prtr, err = sprtr.testPrinter(seq)
	}
	if err != nil {
		panic(err)
	}
	return prtr
}

func (p *asyncPrinters) flush(suite string, once bool) (err error) {
	p.Lock()
	defer p.Unlock()

	// logger.Debug("flushing global printer")
	// p.printer("", 0).Flush()

	// Must flush a suite only once
	var suitePrinters *suitePrinters
	var ok bool
	if suitePrinters, ok = p.suitesPrinters[suite]; ok {
		delete(p.suitesPrinters, suite)
		logger.Debug("removed suitePrinters", "suite", suite)
	} else {
		// If suitePrinters not in map, nothing to flush
		logger.Trace("nothing to flush", "suite", suite)
		//err = fmt.Errorf("no: [%s] suite to flush", suite)
		return
	}

	for done, err := suitePrinters.flush(); !once && !done; {
		if err != nil {
			return err
		}
		time.Sleep(FlushPeriod)
	}
	return
}

func (p *asyncPrinters) testEnded(suite string, seq int) {
	if sp, ok := p.suitesPrinters[suite]; ok {
		sp.testEnded(seq)
	}
}

func (p *asyncPrinters) clear(suite string) {
	delete(p.suitesPrinters, suite)
}

func (p *asyncPrinters) recordedSuites() (suites []string) {
	for _, suite := range p.suites {
		suites = append(suites, suite)
	}
	logger.Debug("listed recorded suites", "suites", suites)
	return
}

func (p *asyncPrinters) flushAll(once bool) (err error) {
	// Current implem need all suites printers to be registered before starting to flush.
	p.printer("", 0).Flush()

	for suite, _ := range p.suitesPrinters {
		if suite != "" {
			err = p.flush(suite, once)
			if err != nil {
				return
			}
		}
	}

	return
}
