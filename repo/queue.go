package repo

import (
	"errors"
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v2"
	"mby.fr/cmdtest/model"
	"mby.fr/utils/collections"
)

/*
type OperationKind string

const (
	TestOperation   = OperationKind("Test")
	ReportOperation = OperationKind("Report")
)

type ReportOperation struct {
	TestSuite string
	Wait      bool
}

type Operation[T TestOperation | ReportOperation] interface {
	Wait() bool
	Op() T
}
*/

type TestOperation struct {
	TestSuite string
	Def       model.TestDefinition
	Blocking  bool
	ExitCode  int
}

type OperationQueue struct {
	//TestSuite  string
	Operations []TestOperation
	Blocked    bool
}

type OperationQueueRepo struct {
	backingFilepath string
	QueuedSuites    []string
	Queues          map[string]OperationQueue
	OpenedSuites    []string
	OperationsDone  []*TestOperation
	lastUpdate      time.Time
	//LastUnqueued map[string]*TestOperation
	//BlockingOperations []*TestOperation
}

func (r *OperationQueueRepo) ReportOperationDone(op *TestOperation) (err error) {
	r.OperationsDone = append(r.OperationsDone, op)
	err = r.Persist()
	return
}

func (r *OperationQueueRepo) WaitOperationDone(op *TestOperation, timeout time.Duration) (err error) {
	start := time.Now()
	for time.Since(start) < timeout {
		if collections.Contains(&r.OperationsDone, op) {
			return
		}
		time.Sleep(1 * time.Millisecond)
	}
	err = errors.New("WaitOperationDone() timed out")
	return
}

func (r *OperationQueueRepo) Queue(op TestOperation) {
	testSuite := op.Def.TestSuite
	q, ok := r.Queues[testSuite]
	if !ok {
		r.QueuedSuites = append(r.QueuedSuites, testSuite)
		q = OperationQueue{}
	}
	logger.Debug("Queue()", "testSuite", testSuite, "operation", op)
	q.Operations = append(q.Operations, op)
	r.Queues[testSuite] = q
}

func (r *OperationQueueRepo) Unqueue() (ok bool, op *TestOperation) {
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
		q := r.Queues[suite]
		if q.Blocked {
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
		r.OpenedSuites = append(r.OpenedSuites, electedSuite)
	} else {
		// no queue available to unqueue
		return
	}

	q := r.Queues[electedSuite]
	size := len(q.Operations)
	logger.Debug("Unqueue()", "electedSuite", electedSuite, "size", size)

	if size > 0 {
		// Unqueue operation
		ok = true
		op = &q.Operations[0]
		q.Operations = q.Operations[1:]
		q.Blocked = op.Blocking
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

	return
}

func (r *OperationQueueRepo) Unblock(op *TestOperation) {
	q := r.Queues[op.TestSuite]
	q.Blocked = false
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
	err = repo.Update()
	return
}
