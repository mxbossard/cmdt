package repo

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/gofrs/flock"
	"gopkg.in/yaml.v2"
)

type State struct {
	Done []Operater
}

type FileState struct {
	backingFilepath string
	fileLock        *flock.Flock
	state           State
	lastUpdate      time.Time
}

func (s *FileState) lock() (err error) {
	if s.fileLock == nil {
		s.fileLock = flock.New(s.backingFilepath)
	}
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
	//logger.Debug("persisting state...", "file", s.backingFilepath, "content", content)
	err = os.WriteFile(s.backingFilepath, content, 0600)
	if err != nil {
		err = fmt.Errorf("cannot persist context: %w", err)
		return
	}
	return
}

func (s *FileState) update() (err error) {
	if time.Since(s.lastUpdate) < 10*time.Millisecond {
		// Update once every 10 ms
		return nil
	}
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
	//logger.Debug("updating state...", "file", s.backingFilepath, "content", content)
	err = s.unlock()
	if err != nil {
		return
	}
	err = yaml.Unmarshal(content, &s.state)
	//logger.Debug("updated state", "state", s.state)
	s.lastUpdate = time.Now()
	return
}
func (s *FileState) ReportOperationDone(op Operater) (err error) {
	logger.Debug("ReportOperationDone()", "operation", op)
	err = s.lock()
	if err != nil {
		return
	}
	s.state.Done = append(s.state.Done, op)
	err = s.persist()
	if err != nil {
		return
	}
	err = s.unlock()
	if err != nil {
		return
	}
	return
}

func (s *FileState) WaitOperationDone(op Operater, timeout time.Duration) (exitCode int, err error) {
	logger.Debug("waiting operation done...", "operation", op, "timeout", timeout)
	start := time.Now()
	for time.Since(start) < timeout {
		s.update()
		for _, done := range s.state.Done {
			if done.Suite() == op.Suite() && done.Seq() == op.Seq() {
				logger.Debug("operation finished.", "operation", op)
				return
			}
		}
		/*
			if collections.ContainsAny(&s.state.OperationsDone, op) {
				return
			}
		*/
		time.Sleep(1 * time.Millisecond)
	}
	err = errors.New("timed out")
	return
}
