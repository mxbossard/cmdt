## Client / Daemon
- For sync tests, all operations are performed by the client.
- For async tests, tests operations are queued in the DB and the cli return RC=0 immediatly.
- The daemon dequeue test operations from DB and perform tests in //.
- The daemon is the same client binary but launched with dedicated arguments.

## Daemon process
- When needed the client start the daemon in another process.
- The pid of the daemon is stored in the DB.
- If a pid is already in the DB, the client do not start another daemon.


## Heartbeat
- A heartbeat is integrated in the daemon which allow the
 client to check if the daemon is alive.
- The client can start a new daemon if the daemon is not alive.

## Reports

### Report suite Operation
- For sync suites, a report is performed instantly
- For async suites, the client wait for all operations queued to be performed by the daemon before performing a report of the suite.

### Report @all Operation
- Report all sync suites
- Report all already reported async suites
- Then wait for all operations to be performed by the daemon
- Report all remaining suites


## Software Architecture

### Idea
- main
  - service (TO split into cli + a common pkg)
    - common TODO
    - cli TODO
      - model
      - background
      - common ?
      - repo
      - display
      - help
      - parser
      - utils
    - daemon
      - model
      - background
      - common ?
      - repo
      - asyncdisplay
      - fork
      - utils


### Facade / Context
Bad design : to rewrite

### Service
A lot of logic to split.
Keep internal logic not attached to daemon or cli ?

#### Inventory
- GlobalConfig(facade.GlobalContext)
- globalReport(facade.GlobalContext, async)
- ProcessGlobalReportDef(model.ReportDefinition, async)
- reportTestSuite(facade.SuiteContext, global, all)
- ProcessReportDef(model.ReportDefinition)
- performTest(odel.TestDefinition, facade.TestContext)
- ProcessTestDef(model.TestDefinition, pooledRepo)
- ProcessArgs(allArgs)
- ProcessGlobalError(facade.GlobalContext, err)
- ProcessSuiteError(facade.SuiteContext, err)
- ProcessTestError(facade.TestContext, err)

#### Split
- Error management
- Operation processing

### CliFacade
Facade dedicated to CLI operations
Features:
- Global Init Config
- Suite Init config
- Test
- Report
- Help
- Launch Daemon ?

### DaemonFacade
Facade dedicated do Daemon operations
- Dequeue operation ?
- HeartBeat ?
- 

### Repo

### DAO

#### GlobalDao
Keep global config.
global table not trivial fields :
- global.daemonPid:     PID of daemon. NULL if daemon never started. 0 if Daemon PID cleared.
- global.activity:      EPOCH timestamp of last reported activity.

#### SuiteDao
Keep suite config.
suite table not trivial fields :
- suite.name:           suite name. /!\ FIXME: "" suite name is a special suite to store global config /!\.
- suite.confg:          Serialized model.Config attached to suite.
- suite.tooMuch:        TODO
- suite.outcome:        model.Outcome str representation. 'Z' if not outcomed.
- suite.outcomeOrder:   sorting field to display outcomes in order.
- suite.reportedCount:  number of already reported tests in suite.
- suite.async:          1 if suite is configured async.
- suite.ignored:        1 if suite is configured ignored.

#### QueueDao
Store operations for daemon to perform asynchronously.
suite_queue table not trivial fields :
- suite_queue.open:     1 if opened, daemon is dequeueing the suite.
- suite_queue.blocking: 1 if blocking, the daemon should not dequeue another suite while this one is blocking.

operation_queue table not trivial fields :
- operation_queue.unqueued: 0 when queued ; -1 when done ; daemon PID while in process
- operation_queue.exitCode: NULL when queued ; >=0 when done (Result Code)
- operation_queue.error:    NULL when queued ; "" if no error or "ERROR_MSG" if error encountered while processing operation.
- operation_queue.block:    1 if blocking. IS IT USED ?

#### TestDao
Store performed test with assertion results.

tested table not trivial fields:
- tested.seq:           Unique id of a test, incremented seqentialy.
- tested.name:          Test name
- tested.outcome:       model.Outcome str representation.
- tested.errorMsg:      "" if no error, or "ERROR_MSG" if error encountered performing test.
- tested.duration:      Test durations in µs.
- tested.passed:        1 if test PASSED
- tested.failed:        1 if test FAILED
- tested.errored:       1 if test ERRORED
- tested.ignored:       1 if test IGNORED
- tested.timeouted:     1 if test TIMEOUTED
- tested.exitCode:      exit code of executed command. -1 if no exit code supplied
- tested.stdout:        stdout of executed command. "" if nothing
- tested.stderr:        stderr of executed command. "" if nothing
- tested.report:        IS IT USED ?


assertion_result table not trivial fields:
- assertion_result.prefix:      Prefix used declaring the assertion (@ for example)
- assertion_result.name:        Assertion name (stderr for example)
- assertion_result.op:          Assertion Operator (= for example)
- assertion_result.expected:    Assertion result expected (foo for example)
- assertion_result.value:       Value get after performing the test
- assertion_result.errorMsg:    "" if no error, or "ERROR_MSG" if error encountered performing the assertion
- assertion_result.success:     1 if assertion succeed.
