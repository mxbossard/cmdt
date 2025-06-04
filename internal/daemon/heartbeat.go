package daemon

import (
	"fmt"
	"os"
	"time"

	"cmdt/internal/repo"
)

var stopChan = make(chan bool)

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
		if successives > maxHeartBeatSuccessiveErrors {
			logger.Errorf("Stop daemon after %d heartbeat errors.", successives)
			fmt.Printf("\n/!\\ Stop daemon after %d heartbeat errors./!\\\n", successives)
			err = rep.ClearDaemonPid(os.Getpid())
			if err != nil {
				logger.Error("HEARTBEAT ERROR", "error", err)
			}
			os.Exit(1)
		}
		time.Sleep(daemonHeartBeatPeriod)
	}
	fmt.Printf("\n/!\\ HEARTBEAT STOPPED /!\\\n")
}

func startHeartBeat(rep repo.Repo) {
	go heartBeat(rep)
}

func stopHeartBeat() {
	stopChan <- true
}

func WatchDaemonActivity(daemonToken, daemonIsol string) {
	rep := repo.New(daemonToken, daemonIsol)
	daemonRestart := 0
	for {
		inactivityPeriod, err := rep.InactivityDuration()
		if err != nil {
			panic(err)
		}

		if inactivityPeriod > (maxHeartBeatSuccessiveErrors+2)*daemonHeartBeatPeriod {
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
		time.Sleep(daemonHeartBeatPeriod)
	}
}
