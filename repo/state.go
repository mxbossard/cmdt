package repo

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/gofrs/flock"
	"gopkg.in/yaml.v2"
	"mby.fr/utils/collections"
)

type State struct {
	OperationsDone []*TestOperation
}

type FileState struct {
	backingFilepath string
	fileLock        *flock.Flock
	state           State
}

func (s FileState) lock() (err error) {
	s.fileLock = flock.New(s.backingFilepath)
	lockCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	locked, err := s.fileLock.TryLockContext(lockCtx, time.Millisecond)
	if err != nil {
		return
	}
	if !locked {
		err = errors.New("unable to acquire FileState lock")
	}
	return
}

func (s FileState) unlock() (err error) {
	if s.fileLock != nil {
		err = s.fileLock.Unlock()
	}
	return
}

func (s FileState) persist() (err error) {
	content, err := yaml.Marshal(&s.state)
	if err != nil {
		return
	}
	logger.Warn("persisting state...", "file", s.backingFilepath, "content", content)
	err = os.WriteFile(s.backingFilepath, content, 0600)
	if err != nil {
		err = fmt.Errorf("cannot persist context: %w", err)
		return
	}
	return
}

func (s *FileState) update() (err error) {
	var content []byte
	err = s.lock()
	if err != nil {
		return
	}
	content, err = os.ReadFile(s.backingFilepath)
	if os.IsNotExist(err) {
		err = nil
		return
	} else if err != nil {
		return
	}
	logger.Warn("updating state...", "file", s.backingFilepath, "content", content)
	err = s.unlock()
	if err != nil {
		return
	}
	err = yaml.Unmarshal(content, &s.state)
	return
}
func (s FileState) ReportOperationDone(op *TestOperation) (err error) {
	err = s.lock()
	if err != nil {
		return
	}
	s.state.OperationsDone = append(s.state.OperationsDone, op)
	s.persist()
	err = s.unlock()
	if err != nil {
		return
	}
	return
}

func (s FileState) WaitOperationDone(op *TestOperation, timeout time.Duration) (err error) {
	start := time.Now()
	for time.Since(start) < timeout {
		s.update()
		if collections.Contains(&s.state.OperationsDone, op) {
			return
		}
		time.Sleep(1 * time.Millisecond)
	}
	err = errors.New("timed out")
	return
}
