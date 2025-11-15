## Displayed infos
### Verbosity
In order from less verbose to most verbose
- SHOW_REPORTS_ONLY [@verbose=0]    : Display only reports infos
- SHOW_FAILED_ONLY  [@verbose=1]    : Display failed tests with outcomes.
- SHOW_FAILED_OUTS  [@verbose=2] (Default for inited suite)     : Display failed tests outputs
- SHOW_PASSED       [@verbose=3] (Default for not inited suite) : Display passed tests with outcomes
- SHOW_PASSED_OUTS  [@verbose=4]    : Display passed tests outputs
- SHOW_ALL          [@verbose=5]    : Display all datas

### What to display ?
- Global config
- Suite config ?
- Suite init
- Suite opening ?
- Test
- Test outcome
- Suite report 
- Global report
- Global report @all


Split "Suite init" action into "Suite opening" & "Suite config".
- @init=suite initialize a suite with a config : "Suite config".
- The start of the first test of the suite perform the "Suite opeoning".

  ACTION
- Global config             Following suite config/init will inherit config. 
- Suite config ?            Following sync test will inherit config.  Need a blocking operation to queue suite config.
- Suite opening ?           Sync suite: done on cli side        Async suite: done on cli side with async display
- Test                      Sync suite: done on cli side        Async suite: done on daemon side with async display
- Test outcome              Sync suite: done on cli side        Async suite: done on daemon side with async display
- Suite report              Always done cli side
- Global report             Always done cli side
- Global report @all        Always done cli side
