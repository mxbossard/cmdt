#! /bin/bash
set -e -o pipefail
scriptDir=$( dirname $( readlink -f $0 ) )

. $scriptDir/buildCmdt.sh
newCmdt="$BUILT_CMDT_BIN"
ls -lh "$newCmdt"

# Trusted cmdt to works
cmdt="cmdt"
#cmdt="$newCmdt"

# Cmdt used to test
#cmdtIn="cmdt"
cmdtIn="$cmdt $@"

# Tested cmdt
params0=""
params1="@verbose @failuresLimit=-1" # Default verbose show passed test + perform all test beyond failures limit

newCmdt0="$newCmdt @isol=tested $params0"
newCmdt1="$newCmdt @isol=tested $params0 $params1"

die() {
	>&2 echo "$1"
	exit 1
}

#$cmdt @global @silent

rm -rf -- /tmp/cmdt* /tmp/cmdt.log /tmp/daemon.log 2> /dev/null || true

# Mandatory assertions
"$scriptDir/assertCmdt.sh" "$cmdt"

#"$scriptDir/assertCmdt.sh" "$newCmdt"


# Clear context
export -n __CMDT_TOKEN
#$cmdt @init=main

cannotReinitMsg="cannot erase test suite"
nothingToReportExpectedStderrMsg="you must perform some test prior to report"

>&2 echo "## Test @report without test"
$cmdtIn @init=meta0 #@verbose=4
$cmdtIn @test=meta0/ @fail @stderr:"$nothingToReportExpectedStderrMsg" @-- $newCmdt0 @report=foo #@debug=4
$cmdtIn @test=meta0/ @stderr= @-- $newCmdt0 @init=foo #@debug=4
$cmdtIn @test=meta0/ @fail @stderr:"$nothingToReportExpectedStderrMsg" @-- $newCmdt0 @report=foo #@debug=4
$cmdtIn @test=meta0/ @fail @stderr:"$nothingToReportExpectedStderrMsg" @-- $newCmdt0 @report #@debug=4

>&2 echo "## Meta1 test context not shared without token"
$cmdtIn @init=meta1 #@verbose=4
$cmdtIn @test=meta1/"without token one" @stderr:"PASSED" @stderr:"#01" @-- $newCmdt1 true #@debug
$cmdtIn @test=meta1/"without token two" @stderr:"PASSED" @stderr:"#02" @-- $newCmdt1 true #@debug
$cmdtIn @test=meta1/"command before rule stop" @fail @stderr:"before rule parsing stopper" @-- $newCmdt1 true @-- @success
$cmdtIn @test=meta1/"rule on 2 args" @stderr:"PASSED" @-- $newCmdt1 @stdout:foo bar @-- echo foo bar
$cmdtIn @test=meta1/ @exit=1 @stderr:"3 success" @stderr!:"failure" @stderr:"1 error" @-- $newCmdt0 @report=main

>&2 echo "## Test printed token"
#tk0=$( $cmdt @init @printToken 2> /dev/null )
tk0=$( $newCmdt0 @init @printToken )
>&2 echo "token: $tk0"
$cmdtIn @init=meta2 #@verbose=4
$cmdtIn @test=meta2/ @stderr:"PASSED" @stderr:"#01" @-- $newCmdt1 true @token=$tk0
$cmdtIn @test=meta2/ @stderr:"PASSED" @stderr:"#02" @-- $newCmdt1 true @token=$tk0
$cmdtIn @test=meta2/ @fail @stderr:"$nothingToReportExpectedStderrMsg" @-- $newCmdt1 @report=main
$cmdtIn @test=meta2/ @stderr:"2 success" @stderr!:"failure" @stderr!:"error" @-- $newCmdt1 @report @token=$tk0
$cmdt @report

>&2 echo "## Test exported token"
# FIXME: test seems bad for now !
eval $( $newCmdt0 @init @exportToken )
>&2 echo "token: $__CMDT_TOKEN"

tk1=$( $newCmdt0 @init @printToken )
newCmdt1_tk1="$newCmdt1 @token=$tk1"

#tk=$( $cmdt0 @init @printToken )
$cmdtIn @init=meta3 #@ignore
$cmdtIn @test=meta3/ @stderr:"PASSED" @stderr:"#01" @-- $newCmdt1_tk1 true
$cmdtIn @test=meta3/ @stderr:"PASSED" @stderr:"#02" @-- $newCmdt1_tk1 true
$cmdtIn @test=meta3/ @stderr:"Successfully ran" @-- $newCmdt1_tk1 @report=main
$cmdtIn @test=meta3/ @fail @stderr:"$nothingToReportExpectedStderrMsg" @-- $newCmdt1 @report=main @token=$tk0

$cmdtIn @init=meta4 #@ignore
$cmdtIn @test=meta4/ @stderr:"PASSED" @stderr:"#01" @-- $newCmdt1_tk1 @test=sub4/ true
$cmdtIn @test=meta4/ @stderr:"PASSED" @stderr:"#02" @-- $newCmdt1_tk1 @test=sub4/ true
$cmdtIn @test=meta4/ @stderr:"Successfully ran" @-- $newCmdt1_tk1 @report=sub4
$cmdtIn @test=meta4/ @fail @stderr:"$nothingToReportExpectedStderrMsg" @-- $newCmdt1 @report=sub4 @token=$tk0
$cmdt @report

export -n __CMDT_TOKEN


#eval $( $cmdt @init @exportToken 2> /dev/null )

>&2 echo "## Rules parsing stopper @--"
$cmdtIn @init=parsing_stopper
$cmdtIn @test=parsing_stopper/ @stdout="foo @success @fail\n" @-- echo foo @success @fail
$cmdtIn @report=parsing_stopper

# Isolate tester & tested cmdt with tokens
testerTk=$( $cmdt @init @printToken )
cmdt="$cmdt @token=$testerTk"
cmdtIn="$cmdtIn @token=$testerTk"
newTk=$( $newCmdt @isol=tested @init @printToken @debug )
newCmdt0="$newCmdt0 @token=$newTk"
newCmdt1="$newCmdt1 @token=$newTk"

>&2 echo "## Test Suite re-init"
$cmdtIn @init=reinit #@verbose=5
$cmdtIn @test=reinit/ @-- $newCmdt1 @test=sub1/ true
$cmdtIn @test=reinit/ @fail @stderr:"$cannotReinitMsg" @-- $newCmdt1 @init=sub1
$cmdtIn @test=reinit/ @stderr:"1 success" @-- $newCmdt1 @report=sub1

$cmdtIn @test=reinit/ @-- $newCmdt1 @keepOutputs @test=sub2/ true
$cmdtIn @test=reinit/ @fail @stderr:"$cannotReinitMsg" @-- $newCmdt1 @keepOutputs @init=sub2
$cmdtIn @test=reinit/ @-- $newCmdt1 @keepOutputs @test=sub2/ true
$cmdtIn @test=reinit/ @-- $newCmdt1 @keepOutputs @test=sub2/ true
$cmdtIn @test=reinit/ @stderr:"3 success" @-- $newCmdt1 @report=sub2

$cmdtIn @test=reinit/ @fail @stderr:"$nothingToReportExpectedStderrMsg" @-- $newCmdt1 @report=sub3
$cmdtIn @test=reinit/ @-- $newCmdt1 @test=sub3/ true
$cmdtIn @test=reinit/ @fail @stderr:"$cannotReinitMsg" @-- $newCmdt1 @init=sub3
$cmdtIn @test=reinit/ @-- $newCmdt1 @test=sub3/ true
$cmdtIn @test=reinit/ @stderr:"2 success" @-- $newCmdt1 @report=sub3
$cmdtIn @test=reinit/ @-- $newCmdt1 @init=sub3
$cmdtIn @test=reinit/ @-- $newCmdt1 @test=sub3/ true
$cmdtIn @test=reinit/ @stderr:"1 success" @-- $newCmdt1 @report=sub3

>&2 echo "## Test Suite re-report and @keep"
$cmdtIn @init=rereport_and_keep #@verbose=5
$cmdtIn @test=rereport_and_keep/init1 @-- $newCmdt1 @init=rereport_sub1
$cmdtIn @test=rereport_and_keep/test1 @-- $newCmdt1 @test=rereport_sub1/test true
$cmdtIn @test=rereport_and_keep/report1 @stderr:"rereport_sub1" @-- $newCmdt1 @report=rereport_sub1
$cmdtIn @test=rereport_and_keep/rereport1 @fail @stderr:"$nothingToReportExpectedStderrMsg" @-- $newCmdt1 @report=rereport_sub1

$cmdtIn @test=rereport_and_keep/init2 @-- $newCmdt1 @init=rereport_sub2
$cmdtIn @test=rereport_and_keep/test2 @-- $newCmdt1 @test=rereport_sub2/test true
$cmdtIn @test=rereport_and_keep/reportall2 @stderr:"rereport_sub2" @-- $newCmdt1 @report
$cmdtIn @test=rereport_and_keep/rereportall2 @fail @stderr:"$nothingToReportExpectedStderrMsg" @-- $newCmdt1 @report

$cmdtIn @test=rereport_and_keep/init3 @-- $newCmdt1 @init=rereport_sub3
$cmdtIn @test=rereport_and_keep/test3 @-- $newCmdt1 @test=rereport_sub3/test true
$cmdtIn @test=rereport_and_keep/report3_keeping @stderr:"rereport_sub3" @-- $newCmdt1 @report=rereport_sub3 @keep
$cmdtIn @test=rereport_and_keep/rereport3 @stderr:"rereport_sub3" @-- $newCmdt1 @report=rereport_sub3
$cmdtIn @test=rereport_and_keep/rerereport3 @fail @stderr:"$nothingToReportExpectedStderrMsg" @-- $newCmdt1 @report=rereport_sub3

$cmdtIn @test=rereport_and_keep/init4 @-- $newCmdt1 @init=rereport_sub4
$cmdtIn @test=rereport_and_keep/test4 @-- $newCmdt1 @test=rereport_sub4/test true
$cmdtIn @test=rereport_and_keep/reportall4_keeping @stderr:"rereport_sub4" @-- $newCmdt1 @report @keep
$cmdtIn @test=rereport_and_keep/rereportall4 @stderr:"rereport_sub4" @-- $newCmdt1 @report
$cmdtIn @test=rereport_and_keep/rerereportall4 @fail @stderr:"$nothingToReportExpectedStderrMsg" @-- $newCmdt1 @report

>&2 echo "## Test usage"
$cmdtIn @init=meta
$cmdtIn @test=meta/ @fail @stderr:"usage:" @-- $newCmdt0

>&2 echo "## Test cmdt basic assertions"
$cmdtIn @test=meta/ @stderr:"PASSED" @-- $newCmdt0 true
$cmdtIn @test=meta/ @stderr:"FAILED" @-- $newCmdt0 false
$cmdtIn @test=meta/ @stderr:"PASSED" @-- $newCmdt0 true @success
$cmdtIn @test=meta/ @stderr:"FAILED" @-- $newCmdt0 false @success
$cmdtIn @test=meta/ @stderr:"FAILED" @-- $newCmdt0 true @fail
$cmdtIn @test=meta/ @stderr:"PASSED" @-- $newCmdt0 false @fail
$cmdtIn @test=meta/ @stderr:"PASSED" @-- $newCmdt0 true @exit=0
$cmdtIn @test=meta/ @stderr:"FAILED" @-- $newCmdt0 false @exit=0
$cmdtIn @test=meta/ @stderr:"FAILED" @-- $newCmdt0 true @exit=1
$cmdtIn @test=meta/ @stderr:"PASSED" @-- $newCmdt0 false @exit=1

$cmdtIn @test=meta/ @stderr:"PASSED" @-- $newCmdt0 echo foo bar @stdout:foo @stderr=
$cmdtIn @test=meta/ @stderr:"FAILED" @-- $newCmdt0 echo foo bar @stdout:baz @stderr=
$cmdtIn @test=meta/ @stderr:"PASSED" @-- $newCmdt0 echo foo bar @stdout:foo @stdout:bar @stderr=
$cmdtIn @test=meta/ @stderr:"FAILED" @-- $newCmdt0 echo foo bar @stdout:baz @stdout:bar @stderr=
$cmdtIn @test=meta/ @stderr:"FAILED" @-- $newCmdt0 echo foo bar @stdout:foo @stdout:baz @stderr=
$cmdtIn @test=meta/ @stderr:"FAILED" @-- $newCmdt0 echo foo bar @stdout!:foo
$cmdtIn @test=meta/ @stderr:"PASSED" @-- $newCmdt0 echo foo bar @stdout!=foo @stdout:bar @stdout!:baz

$cmdtIn @test=meta/ @stderr:"PASSED" @-- $newCmdt0 sh -c ">&2 echo foo bar" @stderr:foo @stdout=
$cmdtIn @test=meta/ @stderr:"FAILED" @-- $newCmdt0 sh -c ">&2 echo foo bar" @stderr:baz @stdout=
$cmdtIn @test=meta/ @stderr:"PASSED" @-- $newCmdt0 sh -c ">&2 echo foo bar" @stderr:foo @stderr:bar @stdout=
$cmdtIn @test=meta/ @stderr:"FAILED" @-- $newCmdt0 sh -c ">&2 echo foo bar" @stderr:baz @stderr:bar @stdout=
$cmdtIn @test=meta/ @stderr:"FAILED" @-- $newCmdt0 sh -c ">&2 echo foo bar" @stderr:foo @stderr:baz @stdout=
$cmdtIn @test=meta/ @stderr:"FAILED" @-- $newCmdt0 sh -c ">&2 echo foo bar" @stderr!:foo
$cmdtIn @test=meta/ @stderr:"PASSED" @-- $newCmdt0 sh -c ">&2 echo foo bar" @stderr!=foo @stderr:bar @stderr!:baz
$cmdtIn @test=meta/ @fail @stderr:"11 success" @stderr:"13 failures" @-- $newCmdt1 @report=main


## Forge a token for remaining tests
#eval $( $cmdt @test=meta/ @keepStdout @-- $cmdt @init=t1 @exportToken)

>&2 echo "## Test assertions outputs"
# Init the context used in t1 test suite
$cmdtIn @init=outputs_assertions
$cmdtIn @test=outputs_assertions/ @stderr:"#01..." @stderr:"PASSED" @-- $newCmdt0 true @test=t1/
$cmdtIn @test=outputs_assertions/ @stderr:"#02..." @stderr:"PASSED" @-- $newCmdt0 true @test=t1/
$cmdtIn @test=outputs_assertions/ @stderr:"#03..." @stderr:"FAILED" @-- $newCmdt0 false @test=t1/
$cmdtIn @test=outputs_assertions/ @stderr:"#04..." @stderr:"PASSED" @-- $newCmdt0 false @fail @test=t1/
$cmdtIn @test=outputs_assertions/ @fail "@stderr~/Failures running \[.*t1.*\] test suite .*\(.*3 success.*, .*1 failures.* .*on \s*4 tests.*\).*/" @-- $newCmdt0 @report=t1


>&2 echo "## Test namings"
$cmdt @init=main 2> /dev/null
$cmdtIn @init=naming
$cmdtIn @test=naming/ @stderr~"/Test \[main\].*name1 #01.../" @stderr:"PASSED" @-- $newCmdt1 true @test=name1
$cmdtIn @test=naming/ @stderr~"/Test \[main\].*name2 #02.../" @stderr:"PASSED" @-- $newCmdt1 true @test=name2
$cmdtIn @test=naming/ @stderr~"/Test \[main\].*/" @stderr:"true" @stderr:"#03..." @stderr:"PASSED" @-- $newCmdt1 true
$cmdtIn @test=naming/ @stderr~"/Test \[main\].*/" @stderr:"true" @stderr:"#04..." @stderr:"PASSED" @-- $newCmdt1 true
$cmdtIn @test=naming/ @stderr~"/Test \[suite1\].*name1 #01.../" @stderr:"PASSED" @-- $newCmdt1 true @test=suite1/name1
$cmdtIn @test=naming/ @stderr~"/Test \[suite1\].*name2 #02.../" @stderr:"PASSED" @-- $newCmdt1 true @test=suite1/name2
$cmdtIn @test=naming/ @stderr~"/Test \[suite2\].*/" @stderr:"#01..." @stderr:"PASSED" @-- $newCmdt1 true @test=suite2/
$cmdtIn @test=naming/ @stderr~"/Test \[suite2\].*/" @stderr:"#02..." @stderr:"PASSED" @-- $newCmdt1 true @test=suite2/
$cmdtIn @test=naming/ @stderr~"/Successfully ran \[.*suite1.*\] test suite/" @-- $newCmdt1  @report=suite1
$cmdtIn @test=naming/ @stderr~"/Successfully ran \[.*suite2.*\] test suite/" @-- $newCmdt1 @report=suite2
$cmdtIn @test=naming/ @stderr~"/Successfully ran \[.*main.*\] test suite/" @-- $newCmdt1 @report=main


>&2 echo "## Test display verbosity"
$cmdtIn @init=display_verbosity
# verbose=SHOW_REPORTS_ONLY
$cmdtIn @test=display_verbosity/ @stdout= @stderr=                                                   @-- $newCmdt0 @verbose=0 echo foo 
$cmdtIn @test=display_verbosity/ @stdout= @stderr=                                                   @-- $newCmdt0 @verbose=0 echo foo @fail
$cmdtIn @test=display_verbosity/ @stdout= @stderr=                                                   @-- $newCmdt0 @verbose=0 foo
# verbose=SHOW_FAILED_ONLY
$cmdtIn @test=display_verbosity/ @stdout= @stderr=                                                   @-- $newCmdt0 @verbose=1 echo foo 
$cmdtIn @test=display_verbosity/ @stdout= @stderr:FAILED @stderr:"Executing cmd" @stderr!:">foo"     @-- $newCmdt0 @verbose=1 echo foo @fail
$cmdtIn @test=display_verbosity/ @stdout= @stderr:ERRORED @stderr:"Supplied cmd"                     @-- $newCmdt0 @verbose=1 foo
# verbose=SHOW_FAILED_OUTS
$cmdtIn @test=display_verbosity/ @stdout= @stderr=                                                   @-- $newCmdt0 @verbose=2 echo foo
$cmdtIn @test=display_verbosity/ @stdout= @stderr:FAILED @stderr:"Executing cmd" @stderr:">foo"      @-- $newCmdt0 @verbose=2 echo foo @fail
$cmdtIn @test=display_verbosity/ @stdout= @stderr:ERRORED @stderr:"Supplied cmd"                     @-- $newCmdt0 @verbose=2 foo
# verbose=SHOW_PASSED
$cmdtIn @test=display_verbosity/ @stdout= @stderr:PASSED @stderr!:"Executing cmd" @stderr!:">foo"    @-- $newCmdt0 @verbose=3 echo foo
$cmdtIn @test=display_verbosity/ @stdout= @stderr:FAILED @stderr:"Executing cmd" @stderr:">foo"      @-- $newCmdt0 @verbose=3 echo foo @fail
$cmdtIn @test=display_verbosity/ @stdout= @stderr:ERRORED @stderr:"Supplied cmd"                     @-- $newCmdt0 @verbose=3 foo
# verbose=SHOW_PASSED_OUTS
$cmdtIn @test=display_verbosity/ @stdout= @stderr:PASSED @stderr:"Executing cmd" @stderr:">foo"      @-- $newCmdt0 @verbose=4 echo foo
$cmdtIn @test=display_verbosity/ @stdout= @stderr:FAILED @stderr:"Executing cmd" @stderr:">foo"      @-- $newCmdt0 @verbose=4 echo foo @fail
$cmdtIn @test=display_verbosity/ @stdout= @stderr:ERRORED @stderr:"Supplied cmd"                     @-- $newCmdt0 @verbose=4 foo


>&2 echo "## Test rules missusage"
$cmdtIn @init=failing_rule_missusage
$cmdtIn @test=failing_rule_missusage/ @fail @-- $newCmdt1 @init true
$cmdtIn @test=failing_rule_missusage/ @fail @-- $newCmdt1 @init @test
$cmdtIn @test=failing_rule_missusage/ @fail @-- $newCmdt1 @init @report
$cmdtIn @test=failing_rule_missusage/ @fail @-- $newCmdt1 @test @report true
$cmdtIn @test=failing_rule_missusage/ @fail @-- $newCmdt1 @init @test true
$cmdtIn @test=failing_rule_missusage/ @fail @-- $newCmdt1 @init @test true
$cmdtIn @test=failing_rule_missusage/ @fail @-- $newCmdt1 @init @fail
$cmdtIn @test=failing_rule_missusage/ @fail @-- $newCmdt1 @init @success
$cmdtIn @test=failing_rule_missusage/ @fail @-- $newCmdt1 @init @exit=0
$cmdtIn @test=failing_rule_missusage/ @fail @-- $newCmdt1 true @fail @success
$cmdtIn @test=failing_rule_missusage/ @fail @-- $newCmdt1 true @fail @exit=0
$cmdtIn @test=failing_rule_missusage/ @fail @-- $newCmdt1 true @success @exit=0
$cmdtIn @test=failing_rule_missusage/ @fail @-- $newCmdt1 @report @fail
$cmdtIn @test=failing_rule_missusage/ @fail @-- $newCmdt1 @report @success
$cmdtIn @test=failing_rule_missusage/ @fail @-- $newCmdt1 @report @exit=0
$cmdtIn @test=failing_rule_missusage/ @fail @stderr:donotexist @-- $newCmdt1 true @donotexist
$cmdtIn @test=failing_rule_missusage/ @fail @stderr:donotexist @-- $newCmdt1 true @donotexist
$cmdtIn @test=failing_rule_missusage/ @fail @stderr:"can't use rule: [@async]" @-- $newCmdt1 @async=false true
$cmdtIn @test=failing_rule_missusage/ @fail @stderr:"can't use rule: [@ignore]" @-- $newCmdt1 @report=foo @ignore


>&2 echo "## Test config"
$cmdtIn @init=test_config

$cmdtIn @test=test_config/ @stderr:IGNORED @stderr!:FAILED @stderr!:PASSED @-- $newCmdt1 true @ignore
$cmdtIn @test=test_config/ @stderr!:IGNORED @stderr:PASSED @-- $newCmdt1 true
$cmdtIn @test=test_config/ @stderr:IGNORED @stderr!:FAILED @stderr!:PASSED @-- $newCmdt1 true @ignore
$cmdtIn @test=test_config/ @stderr:IGNORED @stderr!:FAILED @stderr!:PASSED @-- $newCmdt1 false @ignore
$cmdtIn @test=test_config/ @stderr!:IGNORED @stderr:PASSED @-- $newCmdt1 true

$cmdtIn @test=test_config/ @stdout!:foo @stderr!:bar @stderr:PASSED @-- $newCmdt1 @test=hide_name sh -c "echo foo; >&2 echo bar"
$cmdtIn @test=test_config/ @stdout~/^foo$/m @stderr!:bar @stderr:PASSED @-- $newCmdt1 @test=hide_name sh -c "echo foo; >&2 echo bar" @keepStdout
$cmdtIn @test=test_config/ @stdout!:foo @stderr:bar @stderr:PASSED @-- $newCmdt1 @test=hide_name sh -c "echo foo; >&2 echo bar" @keepStderr
$cmdtIn @test=test_config/ @stdout~/^foo$/m @stderr:bar @stderr:PASSED @-- $newCmdt1 @test=hide_name sh -c "echo foo; >&2 echo bar" @keepOutputs

$cmdtIn @test=test_config/ @stderr:TIMEOUT @-- $newCmdt1 sleep 0.01 @timeout=5ms
$cmdtIn @test=test_config/ @stderr:PASSED @-- $newCmdt1 sleep 0.01 @timeout=30ms

$cmdtIn @test=test_config/ @stderr:FAILED @success @-- $newCmdt1 false
$cmdtIn @test=test_config/ @stderr:FAILED @fail @-- $newCmdt1 false @stopOnFailure
$cmdtIn @test=test_config/ @stderr:FAILED @success @-- $newCmdt1 false

$cmdtIn @test=test_config/ @stderr:FAILED @-- $newCmdt1 @prefix=, ,fail true
$cmdtIn @test=test_config/ @stderr:PASSED @-- $newCmdt1 @prefix=% %fail false
$cmdtIn @test=test_config/ @stderr:PASSED @-- $newCmdt1 @prefix=% %success true

$cmdtIn @test=test_config/ @stderr= @-- $newCmdt1 @quiet true
$cmdtIn @test=test_config/ @stderr= @-- $newCmdt1 @quiet false
$cmdtIn @test=test_config/ @stdout="foo\n" @stderr= @-- $newCmdt1 @quiet @keepOutputs echo foo
$cmdtIn @test=test_config/ @stdout= @stderr="foo\n" @-- $newCmdt1 @quiet @keepOutputs sh -c ">&2 echo foo"

$cmdtIn @test=test_config/ @fail @-- $newCmdt1 @report=main


>&2 echo "## Test suite config"
$newCmdt1 @init=suite_config_quiet @quiet
$cmdtIn @init=suite_config
$cmdtIn @test=suite_config/ @stdout= @stderr= @-- $newCmdt1 @test=suite_config_quiet/ @verbose echo foo
$cmdtIn @test=suite_config/ @stderr:PASSED @-- $newCmdt1 @test=suite_config_quiet/ @verbose echo foo @quiet=false
$cmdtIn @test=suite_config/ @stdout="foo\n" @stderr= @-- $newCmdt1 @test=suite_config_quiet/ echo foo @keepOutputs
$cmdtIn @test=suite_config/ @stdout= @stderr="bar\n" @-- $newCmdt1 @test=suite_config_quiet/ sh -c ">&2 echo bar" @keepOutputs
$cmdtIn @test=suite_config/ @-- $newCmdt1 @report=suite_config_quiet

>&2 echo "## Test global config"


>&2 echo "## Test assertions"
$cmdtIn @init=assertion
$cmdtIn @test=assertion/ @stderr:PASSED @-- $newCmdt1 true
$cmdtIn @test=assertion/ @stderr:PASSED @-- $newCmdt1 true @success
$cmdtIn @test=assertion/ @stderr:FAILED @-- $newCmdt1 true @fail

$cmdtIn @test=assertion/ @stderr:FAILED @-- $newCmdt1 false
$cmdtIn @test=assertion/ @stderr:FAILED @-- $newCmdt1 false @success
$cmdtIn @test=assertion/ @stderr:PASSED @-- $newCmdt1 false @fail

$cmdtIn @test=assertion/ @stderr:PASSED @-- $newCmdt1 true @exit=0
$cmdtIn @test=assertion/ @stderr:FAILED @-- $newCmdt1 true @exit=1

$cmdtIn @test=assertion/ @stderr:FAILED @-- $newCmdt1 false @exit=0
$cmdtIn @test=assertion/ @stderr:PASSED @-- $newCmdt1 false @exit=1

$cmdtIn @test=assertion/ @stderr:PASSED @-- $newCmdt1 echo foo bar @stdout="foo bar\n"
$cmdtIn @test=assertion/ @stderr:PASSED @-- $newCmdt1 echo foo bar @stdout:foo
$cmdtIn @test=assertion/ @stderr:PASSED @-- $newCmdt1 echo foo bar @stdout:bar
$cmdtIn @test=assertion/ @stderr:PASSED @-- $newCmdt1 echo foo bar @stderr=
$cmdtIn @test=assertion/ @stderr:FAILED @-- $newCmdt1 echo foo bar @stdout="foo bar"
$cmdtIn @test=assertion/ @stderr:FAILED @-- $newCmdt1 echo foo bar @stdout=
$cmdtIn @test=assertion/ @fail @-- $newCmdt1 echo foo bar @stdout:
$cmdtIn @test=assertion/ @stderr:FAILED @-- $newCmdt1 echo foo bar @stdout:baz
$cmdtIn @test=assertion/ @fail @-- $newCmdt1 echo foo bar @stderr:

$cmdtIn @test=assertion/ @stderr:PASSED @-- $newCmdt1 sh -c ">&2 echo foo bar" @stderr="foo bar\n"
$cmdtIn @test=assertion/ @stderr:PASSED @-- $newCmdt1 sh -c ">&2 echo foo bar" @stderr:foo
$cmdtIn @test=assertion/ @stderr:PASSED @-- $newCmdt1 sh -c ">&2 echo foo bar" @stderr:bar
$cmdtIn @test=assertion/ @stderr:PASSED @-- $newCmdt1 sh -c ">&2 echo foo bar" @stdout=
$cmdtIn @test=assertion/ @stderr:FAILED @-- $newCmdt1 sh -c ">&2 echo foo bar" @stderr="foo bar"
$cmdtIn @test=assertion/ @stderr:FAILED @-- $newCmdt1 sh -c ">&2 echo foo bar" @stderr=
$cmdtIn @test=assertion/ @fail @-- $newCmdt1 sh -c ">&2 echo foo bar" @stderr:
$cmdtIn @test=assertion/ @stderr:FAILED @-- $newCmdt1 sh -c ">&2 echo foo bar" @stderr:baz
$cmdtIn @test=assertion/ @fail @-- $newCmdt1 sh -c ">&2 echo foo bar" @stdout:

$cmdtIn @test=assertion/ @stderr:FAILED @-- $newCmdt1 sh -c "rm /tmp/donotexists || true" @exists=/tmp/donotexists
$cmdtIn @test=assertion/ @stderr:PASSED @-- $newCmdt1 sh -c "touch /tmp/doexists" @exists=/tmp/doexists
$cmdtIn @test=assertion/ @stderr:PASSED @-- $newCmdt1 sh -c "chmod 640 /tmp/doexists" @exists=/tmp/doexists,-rw-r-----
$cmdtIn @test=assertion/ @stderr:FAILED @-- $newCmdt1 sh -c "chmod 640 /tmp/doexists" @exists=/tmp/doexists,-rwxr-----

$cmdtIn @test=assertion/ @stderr:PASSED @-- $newCmdt1 false @fail @cmd=true
$cmdtIn @test=assertion/ @stderr:FAILED @-- $newCmdt1 true @cmd=false
touch /tmp/doexists
$cmdtIn @test=assertion/ @stderr:PASSED @-- $newCmdt1 false @fail @cmd="ls /tmp/doexists"

$cmdtIn @test=assertion/ @fail @-- $newCmdt1 @report=main


>&2 echo "## Test stdin"
$cmdtIn @init=stdin
echo foo | $cmdt @test=stdin/ @stdout="foo\n" cat
# TODO test error raised if stdin used with @async=true


>&2 echo "## Test file ref content in assertions"
$cmdtIn @init=file_ref_content
echo foo > /tmp/fileRefContent
$cmdtIn @test=file_ref_content/ @stderr:PASSED @-- $newCmdt1 @stdout@=/tmp/fileRefContent echo foo
$cmdtIn @test=file_ref_content/ @stderr:FAILED @-- $newCmdt1 @stdout@=/tmp/fileRefContent echo -n foo
echo -n foo > /tmp/fileRefContent
$cmdtIn @test=file_ref_content/ @stderr:PASSED @-- $newCmdt1 @stdout@=/tmp/fileRefContent echo -n foo
$cmdtIn @test=file_ref_content/ @stderr:FAILED @-- $newCmdt1 @stdout@=/tmp/fileRefContent echo foo
$cmdtIn @test=file_ref_content/ @stderr:PASSED @-- $newCmdt1 @stdout@:/tmp/fileRefContent echo foo
$cmdtIn @test=file_ref_content/ @stderr:FAILED @-- $newCmdt1 @stdout@:/tmp/fileRefContent echo bar

$cmdtIn @test=file_ref_content/ @fail @-- $newCmdt1 @report=main

>&2 echo "## Test export"
export foo=bar
$cmdtIn @init=export
$cmdtIn @test=export/ @stdout~"/foo='bar'/m" sh -c "export"


>&2 echo "## Interlaced tests"
$cmdtIn @init=interlaced
$cmdtIn @test=interlaced/ @-- $newCmdt1 @init="testA" @verbose=0
$cmdtIn @test=interlaced/ @-- $newCmdt1 @init="testB" @verbose=0

$cmdtIn @test=interlaced/ @stderr:IGNORED @-- $newCmdt1 echo ignored1 @ignore @test="testA/"
$cmdtIn @test=interlaced/ @stderr:IGNORED @-- $newCmdt1 echo ignored2 @ignore @test="testA/"
$cmdtIn @test=interlaced/ @stderr:FAILED @-- $newCmdt1 false @test="testA/"

$cmdtIn @test=interlaced/ @stderr:PASSED @-- $newCmdt1 echo another test @test="testB/"
$cmdtIn @test=interlaced/ @stderr:IGNORED @-- $newCmdt1 echo ignored3 @ignore @test="testA/"
$cmdtIn @test=interlaced/ @stderr:PASSED @-- $newCmdt1 echo interlaced test @test="testA/"
$cmdtIn @test=interlaced/ @stderr:FAILED @-- $newCmdt1 false @test="testB/"
$cmdtIn @test=interlaced/ @stderr:PASSED @-- $newCmdt1 true @test="testB/"
$cmdtIn @test=interlaced/ @stderr:FAILED @-- $newCmdt1 false @test="testA/"

# should have 1 success 2 failures and 3 ignored
$cmdtIn @test="interlaced/" @fail @stderr:"1 success" @stderr:"2 failure" @stderr:"3 ignored" @-- $newCmdt1 @report="testA"
# should have 2 success 1 failure
$cmdtIn @test="interlaced/" @fail @stderr:"2 success" @stderr:"1 failure" @stderr!:"ignored" @-- $newCmdt1 @report="testB"


>&2 echo "## Mutually exclusive rules"
merExpectedMsgRule="@stderr~/mutually exclusives/"
actionExpectedMsgRule="@stderr:you can't use rule: "
# Actions are mutually exclusives
#eval $( $cmdtIn @init=mutually_exclusive_rules @exportToken )
$cmdtIn @init=mutually_exclusive_rules
$cmdtIn @test=mutually_exclusive_rules/ @fail "$merExpectedMsgRule" @-- $newCmdt1 @global @init
$cmdtIn @test=mutually_exclusive_rules/ @fail "$merExpectedMsgRule" @-- $newCmdt1 @init @global
$cmdtIn @test=mutually_exclusive_rules/ @fail "$merExpectedMsgRule" @-- $newCmdt1 @init @test
$cmdtIn @test=mutually_exclusive_rules/ @fail "$merExpectedMsgRule" @-- $newCmdt1 @init @report
$cmdtIn @test=mutually_exclusive_rules/ @fail "$merExpectedMsgRule" @-- $newCmdt1 @global @test
$cmdtIn @test=mutually_exclusive_rules/ @fail "$merExpectedMsgRule" @-- $newCmdt1 @global @report
$cmdtIn @test=mutually_exclusive_rules/ @fail "$merExpectedMsgRule" @-- $newCmdt1 @test @report

# Assertions are only accepted on test actions
for action in @global @init=baz @report; do
	for assertion in @success @fail @exit=0 @cmd=true @stdout= @stderr= @exists=foo; do
		$cmdtIn @test=mutually_exclusive_rules/ @fail "$actionExpectedMsgRule" @-- $newCmdt1 "$action" "$assertion"
	done
done

# Suite Config are only accepted on global and init actions
for action in @test @report; do
	cmd=""
	if [ "$action" = "@test" ]; then
		cmd="true"
	fi
	for assertion in @fork=5; do
		$cmdtIn @test=mutually_exclusive_rules/ @fail "$actionExpectedMsgRule" @-- $newCmdt1 "$action" "$assertion" $cmd
	done

done

# Test config are only accepted on global init and test actions
for action in @report; do
	for assertion in @quiet @keepStdout @keepStderr @keepOutputs @stopOnFailure @ignore @timeout=1s @runCount=2 @parallel=3; do
		$cmdtIn @test=mutually_exclusive_rules/ @fail "$actionExpectedMsgRule" @-- $newCmdt1 "$action" "$assertion"
	done
done


>&2 echo "## Test flow"
$cmdt @init=main 2> /dev/null # clear main test suite
$cmdtIn @init=test_flow
$cmdtIn @test=test_flow/ @-- $newCmdt1 @init=flow
$cmdtIn @test=test_flow/ @stderr:"#01" @stderr:PASSED @-- $newCmdt1 @test=flow/ @fail false
$cmdtIn @test=test_flow/ @stderr:"#02" @stderr:PASSED @-- $newCmdt1 @test=flow/ true
$cmdtIn @test=test_flow/ @fail @stderr:"you can't use rule:" @-- $newCmdt1 @test=flow/ "@test" "@fork=5" # Should error because of bad param
$cmdtIn @test=test_flow/ @stderr:"#04" @stderr:ERROR @-- $newCmdt1 @test=flow/ doNotExists @stderr:"not executed" # Should error because of not executable
$cmdtIn @test=test_flow/ @stderr:"#05" @stderr:PASSED @-- $newCmdt1 @test=flow/ true
$cmdtIn @test=test_flow/ @fail @stderr:"3 success" @stderr:"2 error" @-- $newCmdt1 @report=flow
$cmdtIn @test=test_flow/ @stderr:"#01" @stderr:PASSED @-- $newCmdt1 @test=flow/ true
$cmdtIn @test=test_flow/ @stderr:"1 success" @-- $newCmdt1 @report=flow


>&2 echo "## Test ignore"
$cmdtIn @init=test_ignore
$cmdtIn @test=test_ignore/init_suite @-- $newCmdt1 @init=test_ignore1_sub
# following test should fail, but are ignored
$cmdtIn @test=test_ignore/ @stderr:PASSED @-- $newCmdt1 @test=test_ignore1_sub/ true
$cmdtIn @test=test_ignore/ @stderr:IGNORED @-- $newCmdt1 @test=test_ignore1_sub/ false @ignore
$cmdtIn @test=test_ignore/ @stderr:IGNORED @-- $newCmdt1 @test=test_ignore1_sub/ false @ignore
$cmdtIn @test=test_ignore/report_suite @stderr:"Successfully ran" @stderr:"2 ignored" @stderr:"1 success" @stderr!:"failure" @-- $newCmdt1 @report=test_ignore1_sub

$cmdtIn @test=test_ignore/init_suite @-- $newCmdt1 @init=test_ignore2_sub
# following test should fail, but are ignored
$cmdtIn @test=test_ignore/ @stderr:IGNORED @-- $newCmdt1 @test=test_ignore2_sub/ false @ignore
$cmdtIn @test=test_ignore/ @stderr:PASSED @-- $newCmdt1 @test=test_ignore2_sub/ true
$cmdtIn @test=test_ignore/ @stderr:FAILED @-- $newCmdt1 @test=test_ignore2_sub/ false
$cmdtIn @test=test_ignore/ @stderr:IGNORED @-- $newCmdt1 @test=test_ignore2_sub/ false @ignore
$cmdtIn @test=test_ignore/report_suite @fail @stderr:"Failures running" @stderr:"2 ignored" @stderr:"1 success" @stderr:"1 failure" @-- $newCmdt1 @report=test_ignore2_sub

# Auto inited suite should not be ignored
$cmdtIn @test=test_ignore/ @stderr:IGNORED @-- $newCmdt1 @test=test_ignore3_sub/ false @ignore
$cmdtIn @test=test_ignore/ @stderr:PASSED @-- $newCmdt1 @test=test_ignore3_sub/ true
$cmdtIn @test=test_ignore/ @stderr:IGNORED @-- $newCmdt1 @test=test_ignore3_sub/ false @ignore
$cmdtIn @test=test_ignore/report_suite @stderr:"Successfully ran" @stderr:"2 ignored" @stderr:"1 success" @stderr!:"failure" @-- $newCmdt1 @report=test_ignore3_sub

$cmdtIn @test=test_ignore/ @stderr:IGNORED @-- $newCmdt1 @test=test_ignore4_sub/ false @ignore
$cmdtIn @test=test_ignore/report_suite @stderr:"Ignored not ran" @stderr:"1 ignored" @stderr!:"success" @stderr!:"failure" @-- $newCmdt1 @report=test_ignore4_sub

$cmdtIn @test=test_ignore/ @stderr:PASSED @-- $newCmdt1 @test=test_ignore5_sub/ true
$cmdtIn @test=test_ignore/report_suite @stderr:"Successfully ran" @stderr:"1 success" @stderr!:"ignored" @stderr!:"failure" @-- $newCmdt1 @report=test_ignore5_sub


>&2 echo "## Test suite ignore"
$cmdtIn @init=suite_ignore1 
$cmdtIn @test=suite_ignore1/init_suite @-- $newCmdt1 @init=suite_ignore1_sub @ignore
$cmdtIn @test=suite_ignore1/ @stderr= @-- $newCmdt1 @test=suite_ignore1_sub/ true
$cmdtIn @test=suite_ignore1/ @stderr= @-- $newCmdt1 @test=suite_ignore1_sub/ true
$cmdtIn @test=suite_ignore1/report_suite @stderr:"Ignored not ran" @stderr:"suite_ignore1_sub" @-- $newCmdt1 @report=suite_ignore1_sub

newCmdt2="$newCmdt @isol=ignore $params0 $params1"

$cmdtIn @init=suite_ignore2 
$cmdtIn @test=suite_ignore2/init_suite @-- $newCmdt2 @init=suite_ignore2_sub @ignore
$cmdtIn @test=suite_ignore2/ @stderr= @-- $newCmdt2 @test=suite_ignore2_sub/ true
$cmdtIn @test=suite_ignore2/ @stderr= @-- $newCmdt2 @test=suite_ignore2_sub/ true
$cmdtIn @test=suite_ignore2/reportall_suite @stderr:"Ignored not ran" @stderr:"suite_ignore2_sub" @stderr!:"suite_ignore1_sub" @-- $newCmdt2 @report

$cmdtIn @init=suite_ignore3
$cmdtIn @test=suite_ignore3/init_suite @-- $newCmdt2 @init=suite_ignore3_sub @ignore
$cmdtIn @test=suite_ignore3/ @stderr= @-- $newCmdt2 @test=suite_ignore3_sub/ true
$cmdtIn @test=suite_ignore3/ @stderr= @-- $newCmdt2 @test=suite_ignore3_sub/ true
$cmdtIn @test=suite_ignore3/reportall_suite @stderr:"Ignored not ran" @stderr:"suite_ignore3_sub" @stderr!:"suite_ignore1_sub" @stderr!:"suite_ignore2_sub" @-- $newCmdt2 @report


>&2 echo "## Test test timeout"
$cmdtIn @init=test_timeout
$cmdtIn @test=test_timeout/init_suite @-- $newCmdt1 @init=test_timeout1_sub
$cmdtIn @test=test_timeout/test_sleep @stderr:TIMEOUT @-- $newCmdt1 @test=test_timeout1_sub/tSleep @timeout=0.1s sleep 1
$cmdtIn @test=test_timeout/report_suite @fail @stderr:"Failures running" @stderr:"0 success" @stderr:"1 timeout" @stderr!:"error" @stderr!:"failure" @-- $newCmdt1 @report=test_timeout1_sub

$cmdtIn @test=test_timeout/init_suite @-- $newCmdt1 @init=test_timeout2_sub
$cmdtIn @test=test_timeout/test_sleep @stderr:TIMEOUT @-- $newCmdt1 @test=test_timeout2_sub/tSleep @timeout=0.1s sleep 1
$cmdtIn @test=test_timeout/test_true @stderr:PASSED @-- $newCmdt1 @test=test_timeout2_sub/ true
$cmdtIn @test=test_timeout/report_suite @fail @stderr:"Failures running" @stderr:"1 success" @stderr:"1 timeout" @stderr!:"error" @stderr!:"failure" @-- $newCmdt1 @report=test_timeout2_sub

$cmdtIn @test=test_timeout/init_suite @-- $newCmdt1 @init=test_timeout3_sub
$cmdtIn @test=test_timeout/test_sleep @stderr:TIMEOUT @-- $newCmdt1 @test=test_timeout3_sub/tSleep @timeout=0.1s sleep 1
$cmdtIn @test=test_timeout/test_true @stderr:PASSED @-- $newCmdt1 @test=test_timeout3_sub/ true
$cmdtIn @test=test_timeout/test_false @stderr:FAILED @-- $newCmdt1 @test=test_timeout3_sub/ false
$cmdtIn @test=test_timeout/report_suite @fail @stderr:"Failures running" @stderr:"1 success" @stderr:"1 timeout" @stderr:"1 failure" @stderr!:"error" @-- $newCmdt1 @report=test_timeout3_sub


>&2 echo "## Test suite timeout"
$cmdtIn @init=suite_timeout @ignore
$cmdtIn @test=suite_timeout/init_suite @-- $newCmdt1 @init=suite_timeout_sub @suiteTimeout=0.1s
$cmdtIn @test=suite_timeout/test_sleep @fail @stderr:TIMEOUTED @-- $newCmdt1 @test=suite_timeout_sub/tSleep sleep 1
$cmdtIn @test=suite_timeout/report_suite @fail @stderr:Timeouted @-- $newCmdt1 @report=suite_timeout_sub


>&2 echo "## Test @mock"
mockCfg1="@mock=ls foo,stdin=,stdout=baz,exit=41"
mockCfg2="@mock=ls foo,cmd=sh -c 'echo -n baz; exit 42'"
mockCfg3="@mock=ls foo *,cmd=sh -c 'echo -n baz; exit 43'"
mockCfg4="@mock=ls foo,stdin=baz,exit=44"
mockCfg5="@mock=ls foo,stdin:baz,exit=44"
mockCfg6="@mock:ls foo bar,stdout=baz,exit=46"
mockCfg7="@mock:ls foo bar *,stdout=baz,exit=47"
rm -f @-- foo bar baz 2> /dev/null || true
#expectedFooErrMsg="cannot access 'foo'"
#expectedBarErrMsg="cannot access 'bar'"
#expectedBazErrMsg="cannot access 'baz'"
expectedFooErrMsg="$( 2>&1 ls foo || true )"
expectedBarErrMsg="$( 2>&1 ls bar || true )"
expectedBazErrMsg="$( 2>&1 ls baz || true )"
echo foo > /tmp/fooFileContent
echo baz > /tmp/bazFileContent

$cmdtIn @init=cmd_mock #@verbose=4
$cmdtIn @test=cmd_mock/ @fail @stdout= @stderr:"shell builtin" @-- $newCmdt1 echo foo @mock="echo" # cannot mock shell builtin
$cmdtIn @test=cmd_mock/ @fail @stdout= @stderr:"not found" @-- $newCmdt1 echo foo @mock="fooNotExists" # cannot mock not found command

$cmdtIn @test=cmd_mock/ @stderr:PASSED @-- $newCmdt1 ls @mock="ls,stdout=foo" @stdout=foo @stderr=
$cmdtIn @test=cmd_mock/ @stderr:PASSED @-- $newCmdt1 ls @mock="ls,stdout@=/tmp/fooFileContent" @stdout@=/tmp/fooFileContent @stderr=
$cmdtIn @test=cmd_mock/ @stderr:PASSED @-- $newCmdt1 ls @mock="ls,stderr=foo" @stdout= @stderr=foo
$cmdtIn @test=cmd_mock/ @stderr:PASSED @-- $newCmdt1 ls @mock="ls,stderr@=/tmp/fooFileContent" @stdout= @stderr@=/tmp/fooFileContent
$cmdtIn @test=cmd_mock/ @stderr:PASSED @-- $newCmdt1 ls @mock="ls,stdin@=/tmp/fooFileContent,stdout=foo" @stdout!:foo @stderr=
$cmdtIn @test=cmd_mock/ @stderr:PASSED @-- $newCmdt1 sh -c "echo foo | ls" @mock="ls,stdin@=/tmp/fooFileContent,stdout=foo" @stdout=foo @stderr=
$cmdtIn @test=cmd_mock/ @stderr:PASSED @-- $newCmdt1 sh -c "cat /tmp/fooFileContent | ls" @mock="ls,stdin@=/tmp/fooFileContent,stdout=foo" @stdout=foo @stderr=
$cmdtIn @test=cmd_mock/ @stderr:PASSED @-- $newCmdt1 sh -c "echo fooo | ls" @mock="ls,stdin@=/tmp/fooFileContent,stdout=foo" @stdout!:foo @stderr=
$cmdtIn @test=cmd_mock/ @stderr:PASSED @-- $newCmdt1 sh -c "cat /tmp/fooFileContent | ls" @mock="ls,stdin@:/tmp/fooFileContent,stdout=foo" @stdout=foo @stderr=
$cmdtIn @test=cmd_mock/ @stderr:PASSED @-- $newCmdt1 sh -c "echo fooo | ls" @mock="ls,stdin@:/tmp/fooFileContent,stdout=foo" @stdout=foo @stderr=
$cmdtIn @test=cmd_mock/ @stderr:PASSED @-- $newCmdt1 ls @mock="ls"
$cmdtIn @test=cmd_mock/ @stderr:PASSED @-- $newCmdt1 ls @mock="ls,exit=1" @fail
$cmdtIn @test=cmd_mock/ @fail @stderr:"absolute path" @-- $newCmdt1 ls @mock="/bin/ls" # cannot mock absolute path outside container

$cmdtIn @test=cmd_mock/ @stderr:PASSED @-- $newCmdt1 ls @mock="ls,exit=42" @mock="ls foo,exit=43" @exit=42
$cmdtIn @test=cmd_mock/ @stderr:PASSED @-- $newCmdt1 ls @mock="ls foo,exit=43" @mock="ls,exit=42" @exit=42
$cmdtIn @test=cmd_mock/ @stderr:PASSED @-- $newCmdt1 ls foo @mock="ls,exit=42" @mock="ls foo,exit=43" @exit=43
$cmdtIn @test=cmd_mock/ @stderr:PASSED @-- $newCmdt1 ls foo @mock="ls foo,exit=43" @mock="ls,exit=42" @exit=43

$cmdtIn @test=cmd_mock/ @stderr:PASSED @-- $newCmdt1 ls @mock:"ls,exit=42" @mock:"ls foo,exit=43" @exit=42
$cmdtIn @test=cmd_mock/ @stderr:PASSED @-- $newCmdt1 ls @mock:"ls foo,exit=43" @mock:"ls,exit=42" @exit=42
$cmdtIn @test=cmd_mock/ @stderr:PASSED @-- $newCmdt1 ls foo @mock:"ls,exit=42" @mock:"ls foo,exit=43" @exit=43
$cmdtIn @test=cmd_mock/ @stderr:PASSED @-- $newCmdt1 ls foo @mock:"ls foo,exit=43" @mock:"ls,exit=42" @exit=43

$cmdtIn @test=cmd_mock/ @stderr:PASSED @-- $newCmdt1 ls @mock="ls *,exit=42" @mock="ls foo *,exit=43" @exit=42
$cmdtIn @test=cmd_mock/ @stderr:PASSED @-- $newCmdt1 ls @mock="ls foo *,exit=43" @mock="ls *,exit=42" @exit=42
$cmdtIn @test=cmd_mock/ @stderr:PASSED @-- $newCmdt1 ls foo @mock="ls *,exit=42" @mock="ls foo *,exit=43" @exit=42
$cmdtIn @test=cmd_mock/ @stderr:PASSED @-- $newCmdt1 ls foo @mock="ls foo *,exit=43" @mock="ls *,exit=42" @exit=43

$cmdtIn @test=cmd_mock/ @stderr:PASSED @-- $newCmdt1 ls foo bar @mock:"ls,exit=42" @mock:"ls foo,exit=43" @mock:"ls foo bar,exit=44" @exit=44
$cmdtIn @test=cmd_mock/ @stderr:PASSED @-- $newCmdt1 ls bar foo @mock:"ls,exit=42" @mock:"ls foo,exit=43" @mock:"ls foo bar,exit=44" @exit=44
$cmdtIn @test=cmd_mock/ @stderr:PASSED @-- $newCmdt1 ls bar foo @mock:"ls *,exit=42" @mock:"ls foo,exit=43" @mock:"ls foo bar,exit=44" @exit=42
$cmdtIn @test=cmd_mock/ @stderr:PASSED @-- $newCmdt1 ls foo bar @mock:"ls,exit=42" @mock:"ls foo *,exit=43" @mock:"ls foo bar,exit=44" @exit=43
$cmdtIn @test=cmd_mock/ @stderr:PASSED @-- $newCmdt1 ls bar foo @mock:"ls,exit=42" @mock:"ls foo *,exit=43" @mock:"ls foo bar,exit=44" @exit=43

$cmdtIn @test=cmd_mock/ @stderr:PASSED @-- $newCmdt1 sh -c "echo \${PATH}" "$mockCfg1" "@stdout~/__mock_\d+:/"
$cmdtIn @test=cmd_mock/ @stderr:PASSED @-- $newCmdt1 which ls "$mockCfg1" @stderr= "@stdout~|__mock_\d+/ls|"
$cmdtIn @test=cmd_mock/ @stderr:PASSED @-- $newCmdt1 sh -c "ls foo" "$mockCfg1" @stdout=baz @exit=41 @keepOutputs
$cmdtIn @test=cmd_mock/ @stderr:PASSED @-- $newCmdt1 ls foo "$mockCfg1" @stdout=baz @exit=41
$cmdtIn @test=cmd_mock/ @stderr:PASSED @-- $newCmdt1 sh -c "echo foo | ls foo" "$mockCfg1" @fail @stdout= @stderr:"$expectedFooErrMsg"
$cmdtIn @test=cmd_mock/ @stderr:PASSED @-- $newCmdt1 ls bar "$mockCfg1" @fail @stderr:"$expectedBarErrMsg"
$cmdtIn @test=cmd_mock/ @stderr:PASSED @-- $newCmdt1 ls bar foo "$mockCfg1" @fail @stderr:"$expectedBarErrMsg" @stderr:"$expectedFooErrMsg"
$cmdtIn @test=cmd_mock/ @stderr:PASSED @-- $newCmdt1 ls foo bar "$mockCfg1" @fail @stderr:"$expectedBarErrMsg" @stderr:"$expectedFooErrMsg"

$cmdtIn @test=cmd_mock/ @stderr:PASSED @-- $newCmdt1 ls foo "$mockCfg2" @stdout=baz @exit=42

$cmdtIn @test=cmd_mock/ @stderr:PASSED @-- $newCmdt1 ls foo "$mockCfg3" @stdout=baz @exit=43
$cmdtIn @test=cmd_mock/ @stderr:PASSED @-- $newCmdt1 ls bar "$mockCfg3" @fail @stderr:"$expectedBarErrMsg"
$cmdtIn @test=cmd_mock/ @stderr:PASSED @-- $newCmdt1 ls bar foo "$mockCfg3" @fail @stderr:"$expectedBarErrMsg"
$cmdtIn @test=cmd_mock/ @stderr:PASSED @-- $newCmdt1 ls foo bar "$mockCfg3" @stdout=baz @exit=43

$cmdtIn @test=cmd_mock/ @stderr:PASSED @-- $newCmdt1 ls foo "$mockCfg4" @fail @stderr:"$expectedFooErrMsg"
$cmdtIn @test=cmd_mock/ @stderr:PASSED @-- $newCmdt1 sh -c "echo bazo | ls foo" "$mockCfg4" @exit=2
$cmdtIn @test=cmd_mock/ @stderr:PASSED @-- $newCmdt1 sh -c "echo baz | ls foo" "$mockCfg4" @stderr= @exit=44
$cmdtIn @test=cmd_mock/ @stderr:PASSED @-- $newCmdt1 sh -c "echo bazo | ls foo" "$mockCfg5" @stderr= @exit=44
$cmdtIn @test=cmd_mock/ @stderr:PASSED @-- $newCmdt1 sh -c "echo baz | ls foo" "$mockCfg5" @stderr= @exit=44

$cmdtIn @test=cmd_mock/ @stderr:PASSED @-- $newCmdt1 ls foo "$mockCfg6" @fail @stderr:"$expectedFooErrMsg"
$cmdtIn @test=cmd_mock/ @stderr:PASSED @-- $newCmdt1 ls bar "$mockCfg6" @fail @stderr:"$expectedBarErrMsg"
$cmdtIn @test=cmd_mock/ @stderr:PASSED @-- $newCmdt1 ls bar foo "$mockCfg6" @stdout=baz @exit=46
$cmdtIn @test=cmd_mock/ @stderr:PASSED @-- $newCmdt1 ls foo bar "$mockCfg6" @stdout=baz @exit=46
$cmdtIn @test=cmd_mock/ @stderr:PASSED @-- $newCmdt1 ls foo bar baz "$mockCfg6" @fail @stderr:"$expectedFooErrMsg" @stderr:"$expectedBarErrMsg" @stderr:"$expectedBazErrMsg"

$cmdtIn @test=cmd_mock/ @stderr:PASSED @-- $newCmdt1 ls foo "$mockCfg7" @fail @stderr:"$expectedFooErrMsg"
$cmdtIn @test=cmd_mock/ @stderr:PASSED @-- $newCmdt1 ls bar "$mockCfg7" @fail @stderr:"$expectedBarErrMsg"
$cmdtIn @test=cmd_mock/ @stderr:PASSED @-- $newCmdt1 ls bar foo "$mockCfg7" @stdout=baz @exit=47
$cmdtIn @test=cmd_mock/ @stderr:PASSED @-- $newCmdt1 ls foo bar "$mockCfg7" @stdout=baz @exit=47
$cmdtIn @test=cmd_mock/ @stderr:PASSED @-- $newCmdt1 ls foo bar baz "$mockCfg7" @stdout=baz @exit=47

$cmdtIn @test=cmd_mock/ @fail @-- $newCmdt1 @report=main


>&2 echo "## Test @before & @after"
testFile="/tmp/thisFileDoesNotExistsYet.txt"
testFile2="/tmp/thisFileDoesNotExistsYet2.txt"
rm -f @-- "$testFile" "$testFile2" 2> /dev/null || true
$cmdtIn @init=before_after
$cmdtIn @test=before_after/ @stderr:PASSED @-- $newCmdt1 ls "$testFile" @fail
$cmdtIn @test=before_after/ @stderr:PASSED @-- $newCmdt1 ls "$testFile" @before="touch $testFile"
$cmdtIn @test=before_after/ @stderr:PASSED @-- $newCmdt1 ls "$testFile"
$cmdtIn @test=before_after/ @stderr:PASSED @-- $newCmdt1 ls "$testFile" @after="rm -f -- $testFile"
$cmdtIn @test=before_after/ @stderr:PASSED @-- $newCmdt1 ls "$testFile" @fail
$cmdtIn @test=before_after/ @stderr:PASSED @-- $newCmdt1 ls "$testFile" "$testFile2" @before="touch $testFile" @before="touch $testFile2"

$cmdtIn @test=before_after/ @-- $newCmdt1 @report=main


>&2 echo "## Test @container"
$cmdtIn @init=container #@keepOutputs #@debug=4 #@ignore #@keepOutputs
$cmdtIn @test=container/run_off_container @stderr:PASSED @-- $newCmdt1 sh -c "cat --help 2>&1 | head -1" @stdout!:BusyBox
$cmdtIn @test=container/run_in_container @stderr:PASSED @-- $newCmdt1 @container sh -c "cat --help 2>&1 | head -1" @stdout:BusyBox
$cmdtIn @test=container/ @stderr:PASSED @-- $newCmdt1 @container true
$cmdtIn @test=container/ @stderr:FAILED @-- $newCmdt1 @container false
$cmdtIn @test=container/ @stderr:PASSED @-- $newCmdt1 ls /etc/alpine-release @fail
$cmdtIn @test=container/ @stderr:PASSED @-- $newCmdt1 @container=alpine ls /etc/alpine-release
$cmdtIn @test=container/ @stderr:PASSED @-- $newCmdt1 @container=alpine true
$cmdtIn @test=container/ @stderr:PASSED @-- $newCmdt1 @container=alpine ls @mock=ls,exit=0
$cmdtIn @test=container/ @stderr:FAILED @-- $newCmdt1 @container=alpine ls @mock=ls,exit=1
#$cmdtIn @test=container/ @stderr:PASSED @-- $newCmdt1 @container=alpine sh -c "ls -l /bin/cat*" @mock=/bin/cat,exit=42 @stdout= @debug=4
$cmdtIn @test=container/ @stderr:PASSED @-- $newCmdt1 @container=alpine /bin/cat @mock=/bin/cat,exit=42 @exit=42 @debug=0
$cmdtIn @test=container/ @stderr:PASSED @-- $newCmdt1 @container=alpine cat @mock=/bin/cat,exit=42 @exit=42

$cmdtIn @test=container/ @fail @-- $newCmdt1 @report=main


>&2 echo "## Test @container without exported token"
token="$__CMDT_TOKEN"
export -n __CMDT_TOKEN

$cmdtIn @init=container_wo_token #@keepOutputs #@ignore #@keepOutputs
$cmdtIn @test=container_wo_token/run_in_container @stderr:PASSED @-- $newCmdt1 @container sh -c "cat --help 2>&1 | head -1" @stdout:BusyBox
$cmdtIn @test=container_wo_token/ @stderr:PASSED @-- $newCmdt1 @container true
$cmdtIn @test=container_wo_token/ @stderr:PASSED @-- $newCmdt1 @container @fail false

#$cmdtIn @test=container_wo_token/ @fail @stderr:"$nothingToReportExpectedStderrMsg" @-- $newCmdt1 @report=main
$cmdtIn @test=container_wo_token/ @stderr:"3 success" @-- $newCmdt1 @report=main

export __CMDT_TOKEN="$token"


>&2 echo "## Test @dirtyContainer"
testFile="/tmp/thisFileDoesNotExistsYet.txt"
hostFile="/tmp/thisFileExistsOnHost.txt"
rm -f @-- "$testFile" 2> /dev/null || true
touch "$hostFile"
$cmdtIn @init=ephemeralContainer #@keepOutputs #@ignore #@keepOutputs
$cmdtIn @test=ephemeralContainer/run_in_container @stderr:PASSED @-- $newCmdt1 @container sh -c "cat --help 2>&1 | head -1" @stdout:BusyBox #check run inside container
$cmdtIn @test=ephemeralContainer/ @stderr:PASSED @-- $newCmdt1 ls "$hostFile" @stdout:"$hostFile" # file exists on host
$cmdtIn @test=ephemeralContainer/ @stderr:PASSED @-- $newCmdt1 ls "$testFile" @fail @stdout= @stderr:"$testFile" # file should not exist on host
$cmdtIn @test=ephemeralContainer/ @stderr:PASSED @-- $newCmdt1 @container ls "$testFile" @fail @stdout= @stderr:"$testFile" # file should not exist in container
$cmdtIn @test=ephemeralContainer/ @stderr:PASSED @-- $newCmdt1 @container ls "$hostFile" @fail @stdout= @stderr:"$hostFile" # file should not exist in container
$cmdtIn @test=ephemeralContainer/ @stderr:PASSED @-- $newCmdt1 @container touch "$testFile" # create file in ephemeral container
$cmdtIn @test=ephemeralContainer/ @stderr:PASSED @-- $newCmdt1 @container ls "$testFile" @fail @stdout= @stderr:"$testFile" # file should not exist in container
$cmdtIn @test=ephemeralContainer/ @stderr:PASSED @-- $newCmdt1 @container ls "$hostFile" @fail @stdout= @stderr:"$hostFile" # file should not exist in container
$cmdtIn @test=ephemeralContainer/ @stderr:PASSED @-- $newCmdt1 ls "$testFile" @fail @stdout= @stderr:"$testFile" # file should not exist on host
$cmdtIn @test=ephemeralContainer/ @-- $newCmdt1 @report=main

$cmdtIn @init=suiteContainer #@verbose=4 #@keepOutputs
$cmdtIn @test=suiteContainer/ @-- $newCmdt1 @init=sub @container # container should live the test suite
$cmdtIn @test=suiteContainer/run_in_container @stderr:PASSED @-- $newCmdt1 @test=sub/ sh -c "cat --help 2>&1 | head -1" @stdout:BusyBox #check run inside container
$cmdtIn @test=suiteContainer/ @stderr:PASSED @-- $newCmdt1 @test=sub/ ls "$testFile" @fail @stdout= @stderr:"$testFile" # file should not exist in suite container
$cmdtIn @test=suiteContainer/ @stderr:PASSED @-- $newCmdt1 @test=sub/ ls "$hostFile" @fail @stdout= @stderr:"$hostFile" # file should not exist in suite container
$cmdtIn @test=suiteContainer/ @stderr:PASSED @-- $newCmdt1 @test=sub/ touch "$testFile" # create file in suite container
$cmdtIn @test=suiteContainer/ @stderr:PASSED @-- $newCmdt1 @test=sub/ ls "$testFile" @stdout:"$testFile" # file should exist in suite container
$cmdtIn @test=suiteContainer/ @stderr:PASSED @-- $newCmdt1 @test=sub/ ls "$hostFile" @fail @stdout= @stderr:"$hostFile" # file should not exist in suite container
$cmdtIn @test=suiteContainer/ @stderr:PASSED @-- $newCmdt1 @test=sub/ ls "$testFile" @container @fail @stdout= @stderr:"$testFile" # file should not exists in ephemeral container
$cmdtIn @test=suiteContainer/ @stderr:PASSED @-- $newCmdt1 @test=sub/ ls "$testFile" @stdout:"$testFile" @debug=0 # file should exist in suite container
$cmdtIn @test=suiteContainer/ @-- $newCmdt1 @report=sub

$cmdtIn @init=dirtyContainer #@keepOutputs
$cmdtIn @test=dirtyContainer/ @-- $newCmdt1 @init=sub @container # container should live the test suite
$cmdtIn @test=dirtyContainer/run_in_container @stderr:PASSED @-- $newCmdt1 @test=sub/ sh -c "cat --help 2>&1 | head -1" @stdout:BusyBox #check run inside container
$cmdtIn @test=dirtyContainer/ @stderr:PASSED @-- $newCmdt1 @test=sub/ ls "$testFile" @fail @stderr:"$testFile" # file should not exist in container
$cmdtIn @test=dirtyContainer/ @stderr:PASSED @-- $newCmdt1 @test=sub/ ls "$hostFile" @fail @stderr:"$hostFile" # file should not exist in container
$cmdtIn @test=dirtyContainer/ @stderr:PASSED @-- $newCmdt1 @test=sub/ touch "$testFile" # create file in suite container
$cmdtIn @test=dirtyContainer/ @stderr:PASSED @-- $newCmdt1 @test=sub/ ls "$testFile" # file should exist in suite container
$cmdtIn @test=dirtyContainer/ @stderr:PASSED @-- $newCmdt1 @test=sub/ ls "$hostFile" @fail @stderr:"$hostFile" # file should not exist in container
$cmdtIn @test=dirtyContainer/ @stderr:PASSED @-- $newCmdt1 @test=sub/ ls "$testFile" @dirtyContainer=afterTest # file should exist in container
$cmdtIn @test=dirtyContainer/ @stderr:PASSED @-- $newCmdt1 @test=sub/ ls "$testFile" @fail @stderr:"$testFile" # file should not exist in fresh container
$cmdtIn @test=dirtyContainer/ @stderr:PASSED @-- $newCmdt1 @test=sub/ ls "$hostFile" @fail @stderr:"$hostFile" # file should not exist in container
$cmdtIn @test=dirtyContainer/ @stderr:PASSED @-- $newCmdt1 @test=sub/ touch "$testFile" # create file in suite container
$cmdtIn @test=dirtyContainer/ @stderr:PASSED @-- $newCmdt1 @test=sub/ ls "$testFile" # file should exist in container
$cmdtIn @test=dirtyContainer/ @stderr:PASSED @-- $newCmdt1 @test=sub/ ls "$hostFile" @fail @stderr:"$hostFile" # file should not exist in container
$cmdtIn @test=dirtyContainer/ @stderr:PASSED @-- $newCmdt1 @test=sub/ ls "$testFile" @fail @dirtyContainer=beforeTest # file should not exist in fresh container
$cmdtIn @test=dirtyContainer/ @-- $newCmdt1 @report=sub

$cmdtIn @init=testContainer #@keepOutputs
$cmdtIn @test=testContainer/ @-- $newCmdt1 @init=sub @container @dirtyContainer=beforeTest # container should live for each test
$cmdtIn @test=testContainer/run_in_container @stderr:PASSED @-- $newCmdt1 @test=sub/ sh -c "cat --help 2>&1 | head -1" @stdout:BusyBox #check run inside container
$cmdtIn @test=testContainer/ @stderr:PASSED @-- $newCmdt1 @test=sub/ ls "$testFile" @fail @stderr:"$testFile" # file should not exist in container
$cmdtIn @test=testContainer/ @stderr:PASSED @-- $newCmdt1 @test=sub/ ls "$hostFile" @fail @stderr:"$hostFile" # file should not exist in container
$cmdtIn @test=testContainer/ @stderr:PASSED @-- $newCmdt1 @test=sub/ touch "$testFile" # create file in test container
$cmdtIn @test=testContainer/ @stderr:PASSED @-- $newCmdt1 @test=sub/ ls "$testFile" @fail @stderr:"$testFile" # file should not exist in container
$cmdtIn @test=testContainer/ @stderr:PASSED @-- $newCmdt1 @test=sub/ ls "$hostFile" @fail @stderr:"$hostFile" # file should not exist in container
$cmdtIn @test=testContainer/ @-- $newCmdt1 @report=sub

>&2 echo "## Reporting all"
$cmdt @report= ; >&2 echo SUCCESS ; exit 0


