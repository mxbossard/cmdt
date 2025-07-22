package daemon

import (
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"cmdt/internal/repo"

	"github.com/gofrs/flock"
	"github.com/mxbossard/utilz/filez"
	"github.com/mxbossard/utilz/utilz"
)

var stopChan = make(chan bool)
var watcher *activityWatcher

func heartBeat(rep repo.Repo) {
	fmt.Printf("\n/!\\ HEARTBEAT STARTED /!\\\n")
	successives := 0
Loop:
	for {
		select {
		case <-stopChan:
			// Stop heartBeat
			break Loop
		default:
			// Continue looping
		}
		// Report daemon activity in DB
		err := rep.ReportActivity()
		if err != nil {
			successives++
			logger.Errorf("HEARTBEAT ERROR: %s", err)
			fmt.Printf("\n/!\\ HEARTBEAT ERROR: /!\\\n%v\n", err)
		} else {
			successives = 0
		}
		if successives > maxWatcherSuccessiveErrors {
			logger.Errorf("Stop daemon after %d heartbeat errors.", successives)
			fmt.Printf("\n/!\\ Stop daemon after %d heartbeat errors./!\\\n", successives)
			err = rep.ClearDaemonPid(os.Getpid())
			if err != nil {
				logger.Error("HEARTBEAT ERROR", "error", err)
			}
			os.Exit(1)
		}
		time.Sleep(daemonWatcherPeriod)
	}
	fmt.Printf("\n/!\\ HEARTBEAT STOPPED /!\\\n")
}

func startHeartBeat(rep repo.Repo) {
	watcher = NewActivityWatcher(filepath.Join(rep.BackingFilepath(), "activity.ping"))
	watcher.StartDaemon(daemonWatcherPeriod)
	// go heartBeat(rep)
}

func stopHeartBeat() {
	watcher.StopDaemon()
	// stopChan <- true
}

func WatchDaemonActivity0(daemonToken, daemonIsol string) {
	rep := repo.New(daemonToken, daemonIsol)
	daemonRestart := 0

	for {
		inactivityPeriod, err := rep.InactivityDuration()
		if err != nil {
			panic(err)
		}

		if inactivityPeriod > (maxWatcherSuccessiveErrors+2)*daemonWatcherPeriod {
			// What to do ?

			// Restart Daemon ?
			if daemonRestart >= maxDaemonRestart {
				logger.Errorf("Stop restarting daemon after %d restart", daemonRestart)
				fmt.Printf("\n/!\\ Stop restarting daemon after %d restart /!\\\n", daemonRestart)
				os.Exit(1)
				return
			}
			fmt.Printf("\n/!\\ Restarting daemon token: %s ... /!\\\n", daemonToken)
			pid, err := rep.GetDaemonPid()
			if err != nil {
				logger.Error("WATCHER ERROR", "error", err)
			}

			if pid > 0 {
				// TODO force kill daemon ?

				// Need to clean Daemon
				err = rep.ClearDaemonPid(pid)
				if err != nil {
					logger.Error("WATCHER ERROR", "error", err)
				}
			}

			err = LanchProcessIfNeeded(daemonToken, daemonIsol)
			if err != nil {
				logger.Error("WATCHER ERROR", "error", err)
				panic(err)
			}
			daemonRestart++
			time.Sleep(time.Second)
		}
		time.Sleep(daemonWatcherPeriod)
	}
}

func WatchDaemonActivity(daemonToken, daemonIsol string) {
	rep := repo.New(daemonToken, daemonIsol)
	daemonRestart := 0

	watcher := NewActivityWatcher(filepath.Join(rep.BackingFilepath(), "activity.ping"))
	for {
		ok, err := watcher.IsInactiveFor(maxWatcherInactivityPeriod)
		if err != nil {
			logger.Error("WATCHER ERROR", "error", err)
		}
		if ok {
			if daemonRestart >= maxDaemonRestart {
				logger.Errorf("Stop restarting daemon after %d restart", daemonRestart)
				fmt.Printf("\n/!\\ Stop restarting daemon after %d restart /!\\\n", daemonRestart)
				os.Exit(1)
				return
			}

			fmt.Printf("\n/!\\ Restarting daemon token: %s ... /!\\\n", daemonToken)
			pid, err := rep.GetDaemonPid()
			if err != nil {
				logger.Error("WATCHER ERROR", "error", err)
			}

			if pid > 0 {
				// TODO force kill daemon ?

				// Need to clean Daemon
				err = rep.ClearDaemonPid(pid)
				if err != nil {
					logger.Error("WATCHER ERROR", "error", err)
				}
			}

			err = LanchProcessIfNeeded(daemonToken, daemonIsol)
			if err != nil {
				logger.Error("WATCHER ERROR", "error", err)
				panic(err)
			}
			daemonRestart++
			time.Sleep(time.Second)
		}
		time.Sleep(daemonWatcherPeriod)
	}
}

type activityWatcher struct {
	lock          *flock.Flock
	daemonRunning bool
	stopChan      chan bool
}

func (w *activityWatcher) Ping() error {
	err := utilz.FileLock(w.lock, time.Second)
	if err != nil {
		return err
	}
	defer utilz.FileUnlock(w.lock)

	now := time.Now().UnixNano()
	b := make([]byte, 8)
	binary.LittleEndian.PutUint64(b, uint64(now))
	err = os.WriteFile(w.lock.Path(), b, filez.DefaultFilePerms)
	if err != nil {
		return err
	}

	return nil
}

func (w *activityWatcher) InactiveFor() (time.Duration, error) {
	err := utilz.FileLock(w.lock, time.Second)
	if err != nil {
		return time.Duration(0), err
	}
	defer utilz.FileUnlock(w.lock)

	b, err := os.ReadFile(w.lock.Path())
	if err != nil {
		return time.Duration(0), err
	}
	if b == nil {
		return time.Duration(0), nil
	}
	lastPingTimeNs := binary.LittleEndian.Uint64(b)
	lastPingTime := time.Unix(0, int64(lastPingTimeNs))

	return time.Since(lastPingTime), nil
}

func (w *activityWatcher) IsInactiveFor(d time.Duration) (bool, error) {
	InactivityDuration, err := w.InactiveFor()
	if err != nil {
		return false, err
	}
	return InactivityDuration > d, nil
}

func (w *activityWatcher) startDaemon(period time.Duration) {
	if w.daemonRunning {
		return
	}
	w.daemonRunning = true
	successivErrors := 0
	logger.Error("WATCHER Start", "period", period)
	fmt.Printf("\n/!\\ WATCHER STARTED [%d](%s) /!\\\n", os.Getpid(), w.lock.Path())
Loop:
	for {
		select {
		case <-w.stopChan:
			// Stop heartBeat
			break Loop
		default:
			// Continue looping
		}

		err := w.Ping()
		if err != nil {
			successivErrors++
			logger.Error("WATCHER ERROR", "err", err)
		}

		if successivErrors > maxWatcherSuccessiveErrors {
			logger.Errorf("Stoped watcher daemon after %d heartbeat errors.", successivErrors)
			break Loop
		}

		time.Sleep(period)
	}
	logger.Error("WATCHER Stopped")
	//fmt.Printf("\n/!\\ WATCHER STOPPED [%d](%s) /!\\\n", os.Getpid(), w.lock.Path())
	w.daemonRunning = false
	w.stopChan <- false
}

func (w *activityWatcher) StartDaemon(period time.Duration) {
	go w.startDaemon(period)
}

func (w *activityWatcher) StopDaemon() {
	//fmt.Printf("\n/!\\ WATCHER STOPPING ... (%s) /!\\\n", w.lock.Path())
	w.stopChan <- true
	// Wait for daemon stopped ?
	<-w.stopChan
}

func NewActivityWatcher(filepath string) *activityWatcher {
	return &activityWatcher{
		lock:     flock.New(filepath),
		stopChan: make(chan bool),
	}
}
