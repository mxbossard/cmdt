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
	err = errors.New("timed out")
	return
}

func (r *OperationQueueRepo) Queue(op TestOperation) {
	testSuite := op.Def.TestSuite
	q, ok := r.Queues[testSuite]
	if !ok {
		r.QueuedSuites = append(r.QueuedSuites, testSuite)
		q = OperationQueue{}
		r.Queues[testSuite] = q
	}

	q.Operations = append(q.Operations, op)
}

func (r *OperationQueueRepo) Unqueue() (ok bool, op *TestOperation) {
	if len(r.QueuedSuites) == 0 {
		return
	}

	var electedSuite string
	for _, suite := range r.OpenedSuites {
		q := r.Queues[suite]
		if q.Blocked {
			// blocked queue => cannot elect it
			continue
		}
		if len(q.Operations) == 0 {
			// Empty queue => remove it
			for p, s := range r.QueuedSuites {
				if s == suite {
					r.QueuedSuites = append(r.QueuedSuites[:p], r.QueuedSuites[p+1:]...)
					break
				}
			}
			delete(r.Queues, suite)
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
				r.OpenedSuites = append(r.OpenedSuites, suite)
				break
			}
		}
	}

	if electedSuite == "" {
		// not queue available to unqueue
		return
	}

	ok = true
	q := r.Queues[electedSuite]
	// Unqueue operation
	op = &q.Operations[0]
	q.Operations = q.Operations[1:]
	q.Blocked = op.Blocking

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

func LoadOperationQueueRepo(backingFilepath string) (repo OperationQueueRepo, err error) {
	var content []byte
	content, err = os.ReadFile(backingFilepath)
	if os.IsNotExist(err) {
		repo.backingFilepath = backingFilepath
		repo.Queues = make(map[string]OperationQueue)
		err = nil
		return
	} else if err != nil {
		return
	}
	err = yaml.Unmarshal(content, &repo)
	return
}
