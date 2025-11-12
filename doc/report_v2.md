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