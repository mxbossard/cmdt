package fork

import "cmdt/internal/model"

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

func QueueTestDef(testDef model.TestDefinition) (err error) {
	err = instance()
	if err != nil {
		return err
	}

	return
}

var sched *scheduler

func instance() (err error) {
	if sched != nil {
		return
	}

	sched = &scheduler{
		suiteQueues: make(map[string]suiteQueue),
		maxRunners:  0,
	}
	return
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

type suiteQueue struct {
	priority   int
	runnerPool chan bool
	queue      []model.TestDefinition
	done       []model.TestDefinition
}

type scheduler struct {
	suiteQueues map[string]suiteQueue
	maxRunners  int
}

func (s *scheduler) update(testDef model.TestDefinition) (err error) {
	forkCount := int(testDef.Config.ForkCount.Get())
	q, ok := s.suiteQueues[testDef.TestSuite]
	if !ok {
		q = suiteQueue{
			priority:   1,
			runnerPool: make(chan bool, forkCount),
		}
	}

	q.queue = append(q.queue, testDef)
	s.suiteQueues[testDef.TestSuite] = q
	s.maxRunners = max(s.maxRunners, forkCount)
	return
}

func (s *scheduler) run() (err error) {
	for {
		// 1- Check for runner availability

		// 2- Elect highest priority queue which can be dequed and ran
		for suite, q := range s.suiteQueues {
			_ = suite
			_ = q
		}

		// 3- Dequeue elected suite and launch test

	}
	return
}

func (s scheduler) forkTest(testDef model.TestDefinition) (err error) {
	return
}
