package repo

import (
	"bytes"
	"encoding/gob"
	"errors"
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v2"
	"mby.fr/cmdtest/model"
	"mby.fr/utils/collections"
)

type OperationKind string

const (
	TestKind   = OperationKind("test")
	ReportKind = OperationKind("report")
)

type Operater interface {
	Kind() string
	Suite() string
	Seq() uint16
	Block() bool
	ExitCode() uint16
	SetExitCode(uint16)
	Id() uint16
	SetId(uint16)
}

type OperationBase struct {
	//Token     string
	Type      string
	TestSuite string
	Sequence  uint16
	Blocking  bool
	exit      uint16
	id        uint16
}

func (o OperationBase) Kind() string {
	return o.Type
}

func (o OperationBase) Suite() string {
	return o.TestSuite
}

func (o OperationBase) Seq() uint16 {
	return o.Sequence
}

func (o OperationBase) Block() bool {
	return o.Blocking
}

func (o OperationBase) ExitCode() uint16 {
	return o.exit
}

func (o *OperationBase) SetExitCode(code uint16) {
	o.exit = code
}

func (o OperationBase) Id() uint16 {
	return o.id
}

func (o *OperationBase) SetId(id uint16) {
	o.id = id
}

type TestOp struct {
	OperationBase //`yaml:",inline"`
	Definition    model.TestDefinition
}

func TestOperation(suite string, seq uint16, blocking bool, def model.TestDefinition) TestOp {
	return TestOp{
		OperationBase: OperationBase{
			Type:      string(TestKind),
			TestSuite: suite,
			Sequence:  seq,
			Blocking:  blocking,
		},
		Definition: def,
	}
}

type ReportOp struct {
	OperationBase
	Definition model.ReportDefinition
}

func ReportOperation(suite string, blocking bool, def model.ReportDefinition) ReportOp {
	return ReportOp{
		OperationBase: OperationBase{
			Type:      string(ReportKind),
			TestSuite: suite,
			Blocking:  blocking,
		},
		Definition: def,
	}
}

type ReportAllOp struct {
	OperationBase
	Definition model.ReportDefinition
}

func ReportAllOperation(blocking bool, def model.ReportDefinition) ReportAllOp {
	return ReportAllOp{
		OperationBase: OperationBase{
			Type:      string(ReportKind),
			TestSuite: "__global",
			Blocking:  blocking,
		},
		Definition: def,
	}
}

type serializedOp struct {
	Test      *TestOp      `yaml:",omitempty"`
	Report    *ReportOp    `yaml:",omitempty"`
	ReportAll *ReportAllOp `yaml:",omitempty"`
}

func serializeOp0(op Operater) (sop serializedOp) {
	switch o := op.(type) {
	case *TestOp:
		sop.Test = o
	case *ReportOp:
		sop.Report = o
	case *ReportAllOp:
		sop.ReportAll = o
	default:
		err := fmt.Errorf("unable to serialize operation")
		panic(err)
	}
	return
}

func serializeOp(op Operater) ([]byte, error) {
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	sop := serializeOp0(op)
	err := enc.Encode(sop)
	if err != nil {
		return nil, err
	}
	b := buf.Bytes()
	return b, nil
}

func deserializeOp0(sop serializedOp) (op Operater) {
	if sop.Test != nil {
		return sop.Test
	} else if sop.Report != nil {
		return sop.Report
	} else if sop.ReportAll != nil {
		return sop.ReportAll
	}
	err := fmt.Errorf("unable to deserialize operation")
	panic(err)
}

func deserializeOp(b []byte) (op Operater, err error) {
	buf := bytes.NewReader(b)
	dec := gob.NewDecoder(buf)
	var sop serializedOp
	err = dec.Decode(&sop)
	op = deserializeOp0(sop)
	return
}

type OperationQueue struct {
	//TestSuite  string
	Operations []serializedOp
	Blocking   *serializedOp
}

type OperationQueueRepo struct {
	backingFilepath string
	QueuedSuites    []string
	Queues          map[string]OperationQueue
	OpenedSuites    []string
	lastUpdate      time.Time
}

func (r *OperationQueueRepo) Queue(op Operater) {
	testSuite := op.Suite()
	q, ok := r.Queues[testSuite]
	if !ok {
		r.QueuedSuites = append(r.QueuedSuites, testSuite)
		q = OperationQueue{}
	}
	logger.Debug("Queue()", "testSuite", testSuite, "operation", op)

	sop := serializeOp0(op)
	q.Operations = append(q.Operations, sop)
	r.Queues[testSuite] = q
}

func (r *OperationQueueRepo) Unqueue() (ok bool, op Operater) {
	err := r.Update()
	if err != nil {
		panic(err)
	}
	if len(r.QueuedSuites) == 0 {
		return
	}
	logger.Debug("Unqueue()", "QueuedSuites", r.QueuedSuites, "OpenedSuites", r.OpenedSuites)
	var electedSuite string
	for _, suite := range r.OpenedSuites {
		// Elect first open not blocked queue
		q := r.Queues[suite]
		if q.Blocking != nil {
			// blocked queue => cannot elect it
			continue
		}
		electedSuite = r.QueuedSuites[0]
		break
	}

	if electedSuite == "" {
		// Elect first not opened queued suite
		for _, suite := range r.QueuedSuites {
			if !collections.Contains(&r.OpenedSuites, suite) {
				electedSuite = suite
				break
			}
		}
	}

	if electedSuite != "" {
		// open the queue if not done already
		if !collections.Contains(&r.OpenedSuites, electedSuite) {
			r.OpenedSuites = append(r.OpenedSuites, electedSuite)
		}
	} else {
		// no queue available to unqueue
		return
	}

	q := r.Queues[electedSuite]
	if q.Blocking != nil {
		err = fmt.Errorf("q %s was elected but is blocked", electedSuite)
		panic(err)
	}
	size := len(q.Operations)
	logger.Debug("Unqueue()", "electedSuite", electedSuite, "size", size)

	if size > 0 {
		// Unqueue operation
		ok = true
		sop := q.Operations[0]
		op = deserializeOp0(sop)
		q.Operations = q.Operations[1:]
		if op.Block() {
			q.Blocking = &sop
		}
		r.Queues[electedSuite] = q
	}

	if len(q.Operations) == 0 {
		logger.Debug("Unqueue() clearing QueuedSuites")
		// Empty queue => remove it
		if len(r.QueuedSuites) == 1 {
			r.QueuedSuites = []string{}
		} else {
			for p, s := range r.QueuedSuites {
				if s == electedSuite {
					r.QueuedSuites = append(r.QueuedSuites[:p], r.QueuedSuites[p+1:]...)
					break
				}
			}
		}
		delete(r.Queues, electedSuite)
	}

	logger.Warn("Unqueue()", "ok", ok, "ok2", op != nil, "opened", r.OpenedSuites, "electedSuite", electedSuite, "remaining", len(q.Operations), "blocked", q.Blocking != nil)

	return
}

func (r *OperationQueueRepo) Unblock(op Operater) {
	if op == nil {
		return
	}
	suite := op.Suite()
	q := r.Queues[suite]
	if q.Blocking != nil {
		blocking := deserializeOp0(*q.Blocking)
		if blocking.Kind() == op.Kind() && blocking.Seq() == op.Seq() {
			q.Blocking = nil
			logger.Warn("Unblock", "suite", suite)
		}
	}
}

func (r *OperationQueueRepo) WaitEmptyQueue(testSuite string, timeout time.Duration) {
	start := time.Now()
	for time.Since(start) < timeout {
		err := r.Update()
		if err != nil {
			panic(err)
		}
		if q, ok := r.Queues[testSuite]; ok {
			logger.Warn("WaitEmptyQueue()", "q", q)
			if len(q.Operations) == 0 {
				// Queue is empty
				return
			}
		} else {
			// Queue does not exists
			return
		}
		time.Sleep(100 * time.Millisecond)
	}
	err := errors.New("WaitEmptyQueue() timed out")
	panic(err)
}

func (r *OperationQueueRepo) WaitAllEmpty(timeout time.Duration) {
	start := time.Now()
	for time.Since(start) < timeout {
		err := r.Update()
		if err != nil {
			panic(err)
		}
		if len(r.Queues) == 0 {
			// No Queue anymore
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	err := errors.New("WaitAllEmpty() timed out")
	panic(err)
}

func (r OperationQueueRepo) Persist() (err error) {
	content, err := yaml.Marshal(r)
	if err != nil {
		return
	}
	logger.Debug("Persisting queue", "context", content, "file", r.backingFilepath)
	err = os.WriteFile(r.backingFilepath, content, 0600)
	if err != nil {
		err = fmt.Errorf("cannot persist context: %w", err)
		return
	}
	return
}

func (r *OperationQueueRepo) Update() (err error) {
	if time.Since(r.lastUpdate) < 10*time.Millisecond {
		// Update once every 10 ms
		return nil
	}

	var content []byte
	content, err = os.ReadFile(r.backingFilepath)
	if os.IsNotExist(err) {
		r.Queues = make(map[string]OperationQueue)
		err = nil
		return
	} else if err != nil {
		return
	}
	//logger.Debug("Update()", "backingFilepath", r.backingFilepath, "content", content)
	err = yaml.Unmarshal(content, r)
	r.lastUpdate = time.Now()
	return
}

func LoadOperationQueueRepo(backingFilepath string) (repo OperationQueueRepo, err error) {
	repo.backingFilepath = backingFilepath
	repo.Queues = map[string]OperationQueue{}
	err = repo.Update()
	return
}
