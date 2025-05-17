package fork

import (
	"cmdt/internal/model"
	"cmdt/internal/service"
	"fmt"
	"sync"
	"time"

	"github.com/mxbossard/utilz/collectionz"
)

const (
	workerInactivityTimeout      = 50 * time.Millisecond
	workerSleepPeriod            = 1 * time.Microsecond
	schedulerQueueElectionPeriod = 1 * time.Microsecond
	schedulerMaxWorker           = 20
	waitQueueCompleteSleepPeriod = 1 * time.Millisecond
)

/*

## Fork ideas:
- before/after and dirtiesScope cause problems to run in //
- for // running must unqueue several tests in //
  - if possible unqueue test from same suite but if waiting unqueue from another suite
- @tests could always be added in a queue, a background process (daemon) in charge of executing it
- one queue by token/repo
- group outputs by test suite : first output take output priority (like mass do)
- new test : enqueue test, wait some ms ?, check daemon is started or start it
- no test in queue : wait some secs, stop daemon
- @report should block until all tests done => @async=false by default on report
- only one dameon running at a time by token/repo
- new rules @async[=true/false] @wait[=true/false] @_daemon
- do not run daemon in container => no queue in container
- @async only needed for @test & @report
- @test @async=false should queue test and wait for all previous suite test to be finished before running it
- how to queue and unqueue a test by suite ? For // to work must unqueue in priority from suite but in some case must unqueue from another suite.
  - a queue by suite
  - unqueue in priority of already unqueued suite
  - if previously unqueued a waiting test must not unqueue from this suite until test done
  - suite choice responsibility given to repo
  - unqueue by suite
  - if suite waiting how repo know it
  - SaveTestOutcome could release wait
  - Unqueue should not block if possible but block if nothing to unqueue ???
- Daemon wait for finished test with sql repo
- Do not queue async report for simplicity sake => report always in sync
*/

func QueueTestDef(testDef model.TestDefinition, op *model.TestOp) (err error) {
	sched, err = instance()
	if err != nil {
		return err
	}

	suite := testDef.TestSuite
	forkCount := int(testDef.Config.ForkCount.Get())
	work := func() {
		exitCode := service.ProcessTestDef(testDef, true)
		op.SetExitCode(uint16(exitCode))
	}
	_, err = sched.schedule(suite, forkCount, work)
	return
}

func QueueOperation0(op *model.Operater) (err error) {
	sched, err = instance()
	if err != nil {
		return err
	}

	// TODO ?
	return
}

func WaitQueueComplete(name string) (err error) {
	sched, err = instance()
	if err != nil {
		return err
	}

	err = sched.waitQueueComplete(name)
	return
}

func WaitAllQueuesComplete() (err error) {
	sched, err = instance()
	if err != nil {
		return err
	}

	err = sched.waitAllQueuesComplete()
	return
}

func ClearQueue(name string) (err error) {
	sched, err = instance()
	if err != nil {
		return err
	}

	err = sched.clearSuite(name)
	return
}

var sched *scheduler

func instance() (*scheduler, error) {
	if sched != nil {
		return sched, nil
	}

	sched = &scheduler{
		queues:     make(map[string]*queue),
		maxWorkers: schedulerMaxWorker,
	}

	go sched.run()

	return sched, nil
}

/*

- 1 Test queue by suite
- Change suite queue priority
- 1 Scheduler to unqueue and run tests
- What to do with fork at suite level ?
  - suite A with fork = 5
  - suite B with fort = 1
  - suite C with fork = 10
  - => IDEA: scheduler could have max(5,1,10) runners

- 1- Shceduler scan all suite queue by priority
- 2- Scheduler maintain a pool for each suite queue of suite fork size
- 3- Scheduler maintain a meta pool of max fork size
*/

type work func()

type task struct {
	work work
	done chan bool
}

type queue struct {
	name        string
	priority    int
	forkCount   int
	workerCount int
	tasks       chan task
	quit        chan bool
	enable      bool
	scheduled   int
	done        int
}

type scheduler struct {
	sync.Mutex
	queues      map[string]*queue
	maxWorkers  int
	workerCount int
}

func (s *scheduler) clearSuite(suite string) (err error) {
	q, ok := s.queues[suite]
	if ok {
		// stop suite workers
		//q.quit <- true
		close(q.tasks)
		//close(q.done)
		s.Lock()
		delete(s.queues, suite)
		s.Unlock()
	}
	return
}

func (s *scheduler) schedule(queueName string, forkCount int, w work) (chan bool, error) {
	s.Lock()
	defer s.Unlock()

	q, ok := s.queues[queueName]
	if !ok {
		q = &queue{
			name:      queueName,
			priority:  1,
			forkCount: forkCount,
			//runnerPool: make(chan bool, forkCount),
			tasks: make(chan task, 128),
			//done:  make(chan model.TestDefinition, 128),
			quit:   make(chan bool),
			enable: true,
		}
	}

	if !q.enable {
		err := fmt.Errorf("cannot schedule work on disabled queue: %s", queueName)
		return nil, err
	}

	t := task{
		work: w,
		done: make(chan bool, 1),
	}
	q.scheduled++
	q.tasks <- t
	s.queues[queueName] = q
	s.maxWorkers = max(s.maxWorkers, forkCount)
	return t.done, nil
}

func (s *scheduler) waitQueueComplete(queueName string) (err error) {
	s.Lock()
	q, ok := s.queues[queueName]
	s.Unlock()
	if !ok {
		return
	}

	// Disable q => should not add more tasks
	q.enable = false

	for q.done != q.scheduled {
		time.Sleep(waitQueueCompleteSleepPeriod)
	}

	// fmt.Printf("queue: %s completed %d/%d tasks (worker count: %d)\n", queueName, q.done, q.scheduled, q.workerCount)

	return
}

func (s *scheduler) waitAllQueuesComplete() (err error) {
	s.Lock()
	queues := collectionz.Keys(s.queues)
	s.Unlock()

	for _, q := range queues {
		err = s.waitQueueComplete(q)
		if err != nil {
			return err
		}
	}

	return
}

func (s *scheduler) worker(q *queue) {
	touched := time.Now()
End:
	for {
		if time.Since(touched) > workerInactivityTimeout {
			// Quit worker after inactivity timeout
			break End
		}

		select {
		case task, ok := <-q.tasks:
			if !ok {
				// Channel was closed => terminate the worker
				break End
			}
			task.work()
			task.done <- true
			q.done++
			// FIXME: is done chan needed ?
			//q.done <- task
			touched = time.Now()

		//case <- q.quit:
		//	break End

		default:
			time.Sleep(workerSleepPeriod)
		}
	}
	q.workerCount--
	s.workerCount--
}

func (s *scheduler) run() (err error) {
	for {
		// 1- Elect highest priority queue which can be dequed with missing worker
		var electedQueue *queue
		s.Lock()
		for suite, q := range s.queues {
			_ = suite
			if len(q.tasks) > 0 && q.workerCount < q.forkCount {
				// q is electable
				if electedQueue == nil || q.priority < electedQueue.priority {
					electedQueue = q
				}
			}
		}

		if electedQueue != nil {
			// 2- Check for worker availability and launch worker if possible
			if s.workerCount < s.maxWorkers {
				// spawn a worker
				electedQueue.workerCount++
				s.workerCount++
				go s.worker(electedQueue)
				// fmt.Printf("Started a new worker for queue: %s (worker count: %d)\n", electedQueue.name, electedQueue.workerCount)
			}
		}

		s.Unlock()
		time.Sleep(schedulerQueueElectionPeriod)
	}
}

/*
- @fork=N => N workers
- @maxFork => limit global number of forks
- scheduler in charge of spawning suite q workers
- 1 task chan by suite q


*/
