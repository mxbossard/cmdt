## Report v2

- Suites can now be re-reported.
- Add support to report sync & async suites simultaneously.

### Suite Report (@report=)

### Global Report (@report)
- Report not already reported sync suites first from cli side.
- Ask daemon to report not already reported async suites.
- Start tailing async display.
- Wait for async global report op to be done.



### Global Report all (@report @all)
- Report all sync suites (reported and not reported) first from cli side.
- Report already reported async suites from cli side.
- Ask daemon to report not already reported async suites.
- Start tailing async display.
- Wait for async global report op to be done.


## NEW: Report v3
All report done only from cli side.
Must still async display tests outputs.

### Suite Report (@report=)
- Wait for all suite tests to be performed and report the suite.
- Reporting 0 test should fail. RC=1

### Global Report (@report)
- Wait for all tests in not reported suites to be performed.
- When a suite is completed report it.
- Do report ignored suites even if already reported (to keep warning the user).
- Reporting 0 test should fail. RC=1

### Global Report all (@report @all)
- Wait for all tests to be performed.
- When a suite is completed report it.
- Reporting 0 test should not fail because it's a more informative command. RC=0

### Empty suite report
An empty suite should not pollute output :
- global report should report it first time. RC=0
- global report @all should always report it. RC=0
- suite report must raise an error because it's empty. RC=1

### Ignored suite report
An ignored suite should always be reported to warn that it is ignored.
If only ignored suite are reported should fail, because no test was executed. RC=1

### Re-Reporting
It's possible to re-report.
- Re-reporting one suite : display the suite outcome
- Re-reporting global : display nothing because no more suites not reported. 
- Re-reporting global @all : display all suites outcomes.
When re-reporting re display suite outcomes with a low verbosity.
Re-reporting with higher verbosity SHOULD re-display tests display.

### Re-Opening suite
After a suite is reported it can be reopen to add more test in it. 
This suite should not be cleaned unless explicitly asked with a @init=suite.
Adding more test to the suite allow to report all the suite including previous tests already reported.

## TODO
- fix daemon : how to update daemon openedSuites field ? Is it necessary ?
  - need to ensure daemon is in sync
  - FOR NOW:
    - init a suite is not an operation queued
    - first test of a suite dequeued init a suite on daemon side
    - report dequeued close a suite on daemon side
  - NEEDS:
    - need to open a suite on daemon side to print on async screen ? => NO could write on async screen on client side
    - need to close a suite on daemon side ? => NO
- end global report
- do the same for suite report
