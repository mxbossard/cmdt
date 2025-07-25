## Report v2

- Suites can now be re-reported.
- Add support to report sync & async suites simultaneously.

### Report (@report=)

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