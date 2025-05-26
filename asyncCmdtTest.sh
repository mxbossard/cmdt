#! /bin/bash
set -e -o pipefail
scriptDir=$( dirname $( readlink -f $0 ) )

. $scriptDir/buildCmdt.sh
newCmdt="$BUILT_CMDT_BIN"

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

rm -rf -- /tmp/cmdt* /tmp/cmdt*.log /tmp/daemon*.log 2> /dev/null || true

# Mandatory assertions
#"$scriptDir/checkCmdt.sh" "$cmdt"


cannotReinitMsg="cannot erase test suite"
nothingToReportExpectedStderrMsg="you must perform some test prior to report"

# Clear context
export -n __CMDT_TOKEN

$cmdtIn @init="async success" 
$cmdtIn @test=async success/should init @-- $newCmdt1 @init=main1 @async @verbose=4
$cmdtIn @test=async success/"should pass 1" @stderr= @-- $newCmdt1 @test=main1/t1 true @verbose=4
$cmdtIn @test=async success/"should pass 2" @stderr= @-- $newCmdt1 @test=main1/t2 sleep 0.2 @verbose=4
$cmdtIn @test=async success/"should pass 3" @stderr= @-- $newCmdt1 @test=main1/t3 true @verbose=4
$cmdtIn @test=async success/should report @exit=0 @stderr:"#01" @stderr:"#02" @stderr!:"#04" @stderr:"PASSED" @stderr!:"FAILED" @stderr:"3 success" @stderr!:"failure" @stderr!:"error" @-- $newCmdt0 @verbose @report=main1 @debug=5
$cmdtIn @report

$cmdtIn @init="async failure"
$cmdtIn @test=async failure/should init @-- $newCmdt1 @init=main2 @async @verbose=4
$cmdtIn @test=async failure/"should pass" @stderr= @-- $newCmdt1 @test=main2/t1 true
$cmdtIn @test=async failure/"should fail" @stderr= @-- $newCmdt1 @test=main2/t2 false
$cmdtIn @test=async failure/should report @exit=1 @stderr:"#01" @stderr:"#02" @stderr!:"#03" @stderr:"PASSED" @stderr:"FAILED" @stderr:"1 success" @stderr:"1 failure" @stderr!:"error" @-- $newCmdt0 @verbose @report=main2 @debug=5
#$cmdtIn @test=async failure/should init @-- $newCmdt1 @init=main2b @async @verbose=4
#$cmdtIn @test=async failure/"should fail 2" @stderr= @-- $newCmdt1 @test=main2b/t2 false
#$cmdtIn @test=async failure/should global report a failure @exit=1 @stderr:"1 failure" @stderr!:"error" @-- $newCmdt0 @verbose @report @debug=5
$cmdtIn @report

$cmdtIn @init="sync error" #@verbose=4
$cmdtIn @test=sync error/should init @-- $newCmdt1 @init=main3 @async=false @verbose=5
$cmdtIn @test=sync error/should pass @stderr:"#01" @stderr:"PASSED" @-- $newCmdt1 @test=main3/t1 true
$cmdtIn @test=sync error/should error 1 @fail @stderr:"#02" @stderr:"ERRORED" @stderr:'badRule does not exists' @-- $newCmdt1 @test=main3/t2 true @badRule
$cmdtIn @test=sync error/should error 2 @fail @stderr:"#03" @stderr:"ERRORED" @-- $newCmdt1 @test=main3/t3 true @before=badCmd
$cmdtIn @test=sync error/should report @exit=1 @stderr:"1 success" @stderr!:"failure" @stderr:"2 error" @stderr:"3 test" @-- $newCmdt0 @verbose @report=main3 @debug=5
$cmdtIn @report

$cmdtIn @init="async error" #@verbose=4
$cmdtIn @test=async error/should init @-- $newCmdt1 @init=main4 @async @verbose=4
$cmdtIn @test=async error/should pass @stderr= @-- $newCmdt1 @test=main4/t1 true
$cmdtIn @test=async error/should error 1 @fail @stderr:'badRule does not exists' @-- $newCmdt1 @test=main4/t2 true @badRule
$cmdtIn @test=async error/should error 2 @stderr= @-- $newCmdt1 @test=main4/t3 true @before=badCmd
$cmdtIn @test=async error/should report @exit=1 @stderr:"#01" @stderr!:"#02" @stderr:"#03" @stderr!:"#04" @stderr:"PASSED" @stderr!:"FAILED" @stderr:"ERRORED" @stderr:"1 success" @stderr!:"failure" @stderr:"2 error" @-- $newCmdt0 @verbose @report=main4 @debug=5
$cmdtIn @report


$cmdtIn @init="sync empty" #@verbose=4
# reporting no test should always report an error
$cmdtIn @test=sync empty/should init1 @-- $newCmdt @isol="cleared_sync_empty_sub1" @init=sync_empty_sub1 @async=false @verbose=5
$cmdtIn @test=sync empty/should not report1 @exit=1 @stderr:"$nothingToReportExpectedStderrMsg" @-- $newCmdt @isol="cleared_sync_empty_sub1" @verbose @report=sync_empty_sub1
$cmdtIn @test=sync empty/should init2 @-- $newCmdt @isol="cleared_sync_empty_sub2" @init=sync_empty_sub2 @async=false @verbose=5
$cmdtIn @test=sync empty/should not global report2 @exit=1 @stderr:"$nothingToReportExpectedStderrMsg" @stderr:"sync_empty_sub2" @-- $newCmdt @isol="cleared_sync_empty_sub2" @verbose @report
$cmdtIn @test=sync empty/should not global report2 all @exit=0 @stderr:"Empty not ran" @stderr:"sync_empty_sub" @stderr!:"$nothingToReportExpectedStderrMsg" @-- $newCmdt @isol="cleared_sync_empty_sub2" @verbose @report @all
$cmdtIn @test=sync empty/should init3 @-- $newCmdt @isol="cleared_sync_empty_sub3" @init=sync_empty_sub3 @async=false @verbose=5
$cmdtIn @test=sync empty/should not global report3 all @exit=0 @stderr:"Empty not ran" @stderr:"sync_empty_sub" @stderr!:"$nothingToReportExpectedStderrMsg" @-- $newCmdt @isol="cleared_sync_empty_sub3" @verbose @report @all
# reporting 2 suites 1 empty should not error but warn for emptyness
$cmdtIn @test=sync empty/should init4a @-- $newCmdt @isol="cleared_sync_empty_sub4" @init=sync_empty_sub4a @async=false @verbose=5
$cmdtIn @test=sync empty/should init4b @-- $newCmdt @isol="cleared_sync_empty_sub4" @init=sync_empty_sub4b @async=false @verbose=5
$cmdtIn @test=sync empty/should test4b @-- $newCmdt @isol="cleared_sync_empty_sub4" @test=sync_empty_sub4b/test true
$cmdtIn @test=sync empty/should global report4 @exit=0 @stderr:"Empty not ran" @stderr:"sync_empty_sub4a" @stderr:"sync_empty_sub4b" @stderr:"Empty not ran" @stderr:"Successfully ran" @stderr!:"Ignored" @stderr!:"$nothingToReportExpectedStderrMsg" @-- $newCmdt @isol="cleared_sync_empty_sub4" @verbose @report
$cmdtIn @test=sync empty/should global report4 all @exit=0 @stderr:"Empty not ran" @stderr:"sync_empty_sub4a" @stderr:"sync_empty_sub4b" @stderr:"Empty not ran" @stderr:"Successfully ran" @stderr!:"Ignored" @stderr!:"$nothingToReportExpectedStderrMsg" @-- $newCmdt @isol="cleared_sync_empty_sub4" @verbose @report @all
# reporting suites, if one suite contain no tests must warn the user
$cmdtIn @test=sync empty/should init5 @-- $newCmdt1 @init=sync_empty_sub5 @async=false @verbose=5
$cmdtIn @test=sync empty/should not global report5 @exit=1 @stderr:"$nothingToReportExpectedStderrMsg" @stderr:"sync_empty_sub5" @-- $newCmdt0 @verbose @report
$cmdtIn @test=sync empty/should not global report5 all @exit=1 @stderr:"Empty not ran" @stderr:"sync_empty_sub" @stderr!:"$nothingToReportExpectedStderrMsg" @-- $newCmdt0 @verbose @report @all
$cmdtIn @report

$cmdtIn @init="async empty" #@verbose=4
# reporting no test should always report an error
$cmdtIn @test=async empty/should init1 @-- $newCmdt @isol="cleared_async_empty_sub1" @init=async_empty_sub1 @async=true @verbose=5
$cmdtIn @test=async empty/should not report1 @exit=1 @stderr:"$nothingToReportExpectedStderrMsg" @-- $newCmdt @isol="cleared_async_empty_sub1" @verbose @report=async_empty_sub1
$cmdtIn @test=async empty/should init2 @-- $newCmdt @isol="cleared_async_empty_sub2" @init=async_empty_sub2 @async=true @verbose=5
$cmdtIn @test=async empty/should not global report2 @exit=1 @stderr:"$nothingToReportExpectedStderrMsg" @stderr:"async_empty_sub2" @-- $newCmdt @isol="cleared_async_empty_sub2" @verbose @report
$cmdtIn @test=async empty/should not global report2 all @exit=0 @stderr:"Empty not ran" @stderr:"async_empty_sub" @stderr!:"$nothingToReportExpectedStderrMsg" @-- $newCmdt @isol="cleared_async_empty_sub2" @verbose @report @all
$cmdtIn @test=async empty/should init3 @-- $newCmdt @isol="cleared_async_empty_sub3" @init=async_empty_sub3 @async=true @verbose=5
$cmdtIn @test=async empty/should not global report3 all @exit=0 @stderr:"Empty not ran" @stderr:"async_empty_sub" @stderr!:"$nothingToReportExpectedStderrMsg" @-- $newCmdt @isol="cleared_async_empty_sub3" @verbose @report @all
# reporting 2 suites 1 empty should not error but warn for emptyness
$cmdtIn @test=async empty/should init4a @-- $newCmdt @isol="cleared_async_empty_sub4" @init=async_empty_sub4a @async=true @verbose=5
$cmdtIn @test=async empty/should init4b @-- $newCmdt @isol="cleared_async_empty_sub4" @init=async_empty_sub4b @async=true @verbose=5
$cmdtIn @test=async empty/should test4b @-- $newCmdt @isol="cleared_async_empty_sub4" @test=async_empty_sub4b/test true
$cmdtIn @test=async empty/should global report4 @exit=0 @stderr:"Empty not ran" @stderr:"async_empty_sub4a" @stderr:"async_empty_sub4b" @stderr:"Empty not ran" @stderr:"Successfully ran" @stderr!:"Ignored" @stderr!:"$nothingToReportExpectedStderrMsg" @-- $newCmdt @isol="cleared_async_empty_sub4" @verbose @report
$cmdtIn @test=async empty/should global report4 all @exit=0 @stderr:"Empty not ran" @stderr:"async_empty_sub4a" @stderr:"async_empty_sub4b" @stderr:"Empty not ran" @stderr:"Successfully ran" @stderr!:"Ignored" @stderr!:"$nothingToReportExpectedStderrMsg" @-- $newCmdt @isol="cleared_async_empty_sub4" @verbose @report @all
# reporting suites, if one suite contain no tests must warn the user
$cmdtIn @test=async empty/should init5 @-- $newCmdt1 @init=async_empty_sub5 @async=true @verbose=5
$cmdtIn @test=async empty/should not global report5 @exit=1 @stderr:"$nothingToReportExpectedStderrMsg" @stderr:"async_empty_sub5" @-- $newCmdt0 @verbose @report
$cmdtIn @test=async empty/should not global report5 all @exit=1 @stderr:"Empty not ran" @stderr:"async_empty_sub" @stderr!:"$nothingToReportExpectedStderrMsg" @-- $newCmdt0 @verbose @report @all
$cmdtIn @report


# FIXME: async report should not fail if no test exist yet. It should fail after a short timeout if no test to report.
>&2 echo "## Test @report without test"
$cmdtIn @init=report_wo_test #@verbose=4
$cmdtIn @test=report_wo_test/ @fail @stderr:"$nothingToReportExpectedStderrMsg" @-- $newCmdt0 @report=foo @async #@debug=4
$cmdtIn @test=report_wo_test/ @stderr= @-- $newCmdt0 @init=foo @async #@debug=4
$cmdtIn @test=report_wo_test/ @fail @stderr:"$nothingToReportExpectedStderrMsg" @-- $newCmdt0 @report=foo @async #@debug=4
$cmdtIn @test=report_wo_test/ @fail @stderr:"$nothingToReportExpectedStderrMsg" @-- $newCmdt0 @report @async #@debug=4

>&2 echo "## Meta1 test context not shared without token"
$cmdtIn @init=isolation #@verbose=4
$cmdtIn @test=isolation/init @stderr= @-- $newCmdt1 @init @async @verbose=4
$cmdtIn @test=isolation/"without token one" @stderr= @-- $newCmdt1 true #@debug
$cmdtIn @test=isolation/"without token two" @stderr= @-- $newCmdt1 true #@debug
$cmdtIn @test=isolation/"command before rule stop" @fail @stderr:"ERRORED" @stderr:"before rule parsing stopper" @-- $newCmdt1 true @-- @success
$cmdtIn @test=isolation/"rule value splited on 2 args" @stderr= @-- $newCmdt1 @stdout:foo bar @-- echo foo bar
$cmdtIn @test=isolation/"report without token" @exit=1 @stderr:"3 success" @stderr!:"failure" @stderr:"1 error" @stderr:"PASSED" @stderr!:"ERRORED" @stderr:"#01" @stderr:"#02" @stderr!:"#03" @stderr:"#04" @stderr!:"#05" @-- $newCmdt0 @report=main @async

>&2 echo "## Test printed token"
tk0=$( $newCmdt0 @init @printToken )
>&2 echo "printed token: $tk0"
$cmdtIn @init=printed_token #@verbose=4
$cmdtIn @test=printed_token/init1 @stderr= @-- $newCmdt1 @token=$tk0 @init @async @verbose=5
$cmdtIn @test=printed_token/"with token 1" @stderr= @-- $newCmdt1 @token=$tk0 @test=printed_token_sub_test1 true
$cmdtIn @test=printed_token/"with token 2" @stderr= @-- $newCmdt1 @token=$tk0 @test=printed_token_sub_test2 true
$cmdtIn @test=printed_token/"report wo token" @fail @stderr:"$nothingToReportExpectedStderrMsg" @-- $newCmdt1 @report=main 
$cmdtIn @test=printed_token/"report with token" @stderr:"2 success" @stderr!:"failure" @stderr!:"error" @stderr:"#01" @stderr:"#02" @stderr!:"#03" @-- $newCmdt1 @token=$tk0 @report=main
$cmdtIn @test=printed_token/init2 @stderr= @-- $newCmdt1 @token=$tk0 @init=master @async @verbose=5
$cmdtIn @test=printed_token/"with token 3" @stderr= @-- $newCmdt1 @token=$tk0 @test=master/printed_token_sub2_test3 true
$cmdtIn @test=printed_token/"with token 4" @stderr= @-- $newCmdt1 @token=$tk0 @test=master/printed_token_sub2_test4 true
$cmdtIn @test=printed_token/"global report with token" @stderr:"2 success" @stderr!:"failure" @stderr!:"error" @stderr:"#01" @stderr:"#02" @stderr!:"#03" @-- $newCmdt1 @token=$tk0 @report
$cmdt @report

>&2 echo "## Test exported token"
eval $( $cmdt @init @exportToken )
>&2 echo "exported token: $__CMDT_TOKEN"
$cmdtIn @init=exported_token #@ignore
$cmdtIn @test=exported_token/init @-- $newCmdt1 @init @async
$cmdtIn @test=exported_token/test1 @stderr= @-- $newCmdt1 true
$cmdtIn @test=exported_token/test2 @stderr= @-- $newCmdt1 true
$cmdtIn @test=exported_token/report1 @stderr:"Successfully ran" @stderr!:"error" @stderr:"#01" @stderr:"#02" @stderr!:"#03" @-- $newCmdt1 @report=main
$cmdtIn @test=exported_token/report2 @stderr:"Successfully ran" @stderr!:"error" @stderr!:"#01" @stderr!:"#02" @stderr!:"#03" @-- $newCmdt1 @report=main
$cmdtIn @test=exported_token/report_other_token @fail @stderr:"$nothingToReportExpectedStderrMsg" @-- $newCmdt1 @report=main @token=empty_token

$cmdtIn @init=exported_token_alt #@ignore
$cmdtIn @test=exported_token_alt/init @-- $newCmdt1 @init=sub4 @async
$cmdtIn @test=exported_token_alt/test1 @stderr= @-- $newCmdt1 @test=sub4/ true
$cmdtIn @test=exported_token_alt/test2 @stderr= @-- $newCmdt1 @test=sub4/ true
$cmdtIn @test=exported_token_alt/report1 @stderr:"Successfully ran" @stderr!:"error" @stderr:"#01" @stderr:"#02" @stderr!:"#03" @-- $newCmdt1 @report=sub4
$cmdtIn @test=exported_token_alt/report2 @stderr:"Successfully ran" @stderr!:"error" @stderr!:"#01" @stderr!:"#02" @stderr!:"#03" @-- $newCmdt1 @report=sub4
$cmdtIn @test=exported_token_alt/global_report_all @stderr:"Successfully ran" @stderr!:"error" @stderr!:"#01" @stderr!:"#02" @stderr!:"#03" @stderr:main @stderr:sub4 @-- $newCmdt1 @report @all
$cmdtIn @test=exported_token_alt/report_other_token @fail @stderr:"$nothingToReportExpectedStderrMsg" @-- $newCmdt1 @token=$tk0 @report=sub4
$cmdt @report 

export -n __CMDT_TOKEN


# Isolate tester & tested cmdt with tokens
testerTk=$( $cmdt @init @printToken )
cmdt="$cmdt @token=$testerTk"
cmdtIn="$cmdtIn @token=$testerTk"
newTk=$( $newCmdt @isol=tested @init @printToken @debug )
newCmdt0="$newCmdt0 @token=$newTk"
newCmdt1="$newCmdt1 @token=$newTk"

noPanic="@stderr!:'panic'"

# Test success
$cmdtIn @init=success_sync #@verbose
$cmdtIn @test=success_sync/init $noPanic @-- $newCmdt1 @init=success_sync_sub @async=false @verbose=5
$cmdtIn @test=success_sync/success1 $noPanic @stderr:"PASSED" @-- $newCmdt1 @test=success_sync_sub/success1 true
$cmdtIn @test=success_sync/success2 $noPanic @stderr:"PASSED" @-- $newCmdt1 @test=success_sync_sub/success2 true
$cmdtIn @test=success_sync/report $noPanic @exit=0 @stderr:"2 success" @-- $newCmdt1 @report=success_sync_sub

$cmdtIn @init=success_async #@verbose
$cmdtIn @test=success_async/init $noPanic @-- $newCmdt1 @init=success_async_sub @async=true @verbose=5
$cmdtIn @test=success_async/success1 @stderr= @-- $newCmdt1 @test=success_async_sub/success1 true
$cmdtIn @test=success_async/success1 @stderr= @-- $newCmdt1 @test=success_async_sub/success2 true
$cmdtIn @test=success_async/report $noPanic @exit=0 @stderr:"PASSED" @stderr!:"IGNORED" @stderr!:"FAILED" @stderr!:"ERRORED"  @stderr!:"TIMEOUT" @stderr:"2 success" @-- $newCmdt1 @report=success_async_sub

# Test ignore
$cmdtIn @init=ignore_sync #@verbose
$cmdtIn @test=ignore_sync/init $noPanic @-- $newCmdt1 @init=ignore_sync_sub @async=false @verbose=5
$cmdtIn @test=ignore_sync/success $noPanic @stderr:"PASSED" @-- $newCmdt1 @test=ignore_sync_sub/success true
$cmdtIn @test=ignore_sync/ignored $noPanic @stderr:"IGNORED" @-- $newCmdt1 @test=ignore_sync_sub/ignored @ignore false
$cmdtIn @test=ignore_sync/report $noPanic @exit=0 @stderr:"1 success" @stderr:"1 ignored" @-- $newCmdt1 @report=ignore_sync_sub

$cmdtIn @init=ignore_async #@verbose
$cmdtIn @test=ignore_async/init $noPanic @-- $newCmdt1 @init=ignore_async_sub @async=true @verbose=5
$cmdtIn @test=ignore_async/success @stderr= @-- $newCmdt1 @test=ignore_async_sub/success true
$cmdtIn @test=ignore_async/ignored @stderr= @-- $newCmdt1 @test=ignore_async_sub/ignored @ignore false
$cmdtIn @test=ignore_async/report $noPanic @exit=0 @stderr:"PASSED" @stderr:"IGNORED" @stderr!:"FAILED" @stderr!:"ERRORED"  @stderr!:"TIMEOUT" @stderr:"1 success" @stderr:"1 ignored" @-- $newCmdt1 @report=ignore_async_sub

# Test failure
$cmdtIn @init=failure_sync #@verbose
$cmdtIn @test=failure_sync/init $noPanic @-- $newCmdt1 @init=failure_sync_sub @async=false @verbose=5
$cmdtIn @test=failure_sync/success $noPanic @stderr:"PASSED" @-- $newCmdt1 @test=failure_sync_sub/success true
$cmdtIn @test=failure_sync/failure $noPanic @stderr:"FAILED" @-- $newCmdt1 @test=failure_sync_sub/failure false
$cmdtIn @test=failure_sync/report $noPanic @exit=1 @stderr:"1 success" @stderr:"1 failure" @-- $newCmdt1 @report=failure_sync_sub

$cmdtIn @init=failure_global_sync #@verbose
$cmdtIn @test=failure_global_sync/init $noPanic @-- $newCmdt1 @init=failure_global_sync_sub @async=false @verbose=5
$cmdtIn @test=failure_global_sync/success $noPanic @stderr:"PASSED" @-- $newCmdt1 @test=failure_global_sync_sub/success true
$cmdtIn @test=failure_global_sync/failure $noPanic @stderr:"FAILED" @-- $newCmdt1 @test=failure_global_sync_sub/failure false
$cmdtIn @test=failure_global_sync/"global report" $noPanic @exit=1 @stderr:"1 success" @stderr:"1 failure" @-- $newCmdt1 @report

$cmdtIn @init=failure_async #@verbose
$cmdtIn @test=failure_async/init $noPanic @-- $newCmdt1 @init=failure_async_sub @async=true @verbose=5
$cmdtIn @test=failure_async/success @stderr= @-- $newCmdt1 @test=failure_async_sub/success true
$cmdtIn @test=failure_async/failure @stderr= @-- $newCmdt1 @test=failure_async_sub/failure false
$cmdtIn @test=failure_async/report $noPanic @exit=1 @stderr:"PASSED" @stderr!:"IGNORED" @stderr:"FAILED" @stderr!:"ERRORED" @stderr!:"TIMEOUT" @stderr:"1 success" @stderr:"1 failure" @-- $newCmdt1 @report=failure_async_sub

$cmdtIn @init=failure_global_async #@verbose
$cmdtIn @test=failure_global_async/init $noPanic @-- $newCmdt1 @init=failure_global_async_sub @async=true @verbose=5
$cmdtIn @test=failure_global_async/success @stderr= @-- $newCmdt1 @test=failure_global_async_sub/success true
$cmdtIn @test=failure_global_async/failure @stderr= @-- $newCmdt1 @test=failure_global_async_sub/failure false
$cmdtIn @test=failure_global_async/"global report" $noPanic @exit=1 @stderr:"PASSED" @stderr!:"IGNORED" @stderr:"FAILED" @stderr!:"ERRORED" @stderr!:"TIMEOUT" @stderr:"1 success" @stderr:"1 failure" @-- $newCmdt1 @report

# Test error
$cmdtIn @init=error_sync #@verbose
$cmdtIn @test=error_sync/init $noPanic @-- $newCmdt1 @init=error_sync_sub @async=false @verbose=5
$cmdtIn @test=error_sync/success $noPanic @stderr:"PASSED" @-- $newCmdt1 @test=error_sync_sub/success true
$cmdtIn @test=error_sync/error $noPanic @stderr:"ERRORED" @-- $newCmdt1 @test=error_sync_sub/error doNotExists
$cmdtIn @test=error_sync/report $noPanic @exit=1 @stderr:"1 success" @stderr:"1 error" @-- $newCmdt1 @report=error_sync_sub

$cmdtIn @init=error_async #@verbose
$cmdtIn @test=error_async/init $noPanic @-- $newCmdt1 @init=error_async_sub @async=true @verbose=5
$cmdtIn @test=error_async/success @stderr= @-- $newCmdt1 @test=error_async_sub/success true
$cmdtIn @test=error_async/error @stderr= @-- $newCmdt1 @test=error_async_sub/timeout doNotExists
$cmdtIn @test=error_async/report $noPanic @exit=1 @stderr:"PASSED" @stderr!:"IGNORED" @stderr!:"FAILED" @stderr:"ERRORED" @stderr!:"TIMEOUT" @stderr:"1 success" @stderr:"1 error" @-- $newCmdt1 @report=error_async_sub

# Test timeout
$cmdtIn @init=timeout_sync #@verbose
$cmdtIn @test=timeout_sync/init $noPanic @-- $newCmdt1 @init=timeout_sync_sub @async=false @verbose=5
$cmdtIn @test=timeout_sync/success $noPanic @stderr:"PASSED" @-- $newCmdt1 @test=timeout_sync_sub/success true
$cmdtIn @test=timeout_sync/timeout $noPanic @stderr:"TIMEOUT" @-- $newCmdt1 @test=timeout_sync_sub/timeout @timeout=0.1s sleep 1
$cmdtIn @test=timeout_sync/report $noPanic @exit=1 @stderr:"1 success" @stderr:"1 timeout" @-- $newCmdt1 @report=timeout_sync_sub

$cmdtIn @init=timeout_async #@verbose
$cmdtIn @test=timeout_async/init $noPanic @-- $newCmdt1 @init=timeout_async_sub @async=true @verbose=5
$cmdtIn @test=timeout_async/success @stderr= @-- $newCmdt1 @test=timeout_async_sub/success true
$cmdtIn @test=timeout_async/timeout @stderr= @-- $newCmdt1 @test=timeout_async_sub/timeout @timeout=0.1s sleep 1
$cmdtIn @test=timeout_async/report $noPanic @exit=1 @stderr:"PASSED" @stderr!:"IGNORED" @stderr:"TIMEOUT" @stderr:"1 success" @stderr:"1 timeout" @-- $newCmdt1 @report=timeout_async_sub


# Test a double suite init
$cmdtIn @init=double_suite_init_sync
$cmdtIn @test=double_suite_init_sync/open1 @-- $newCmdt1 @init=double_suite_init_sync_sub @async=false @verbose=5 
$cmdtIn @test=double_suite_init_sync/open2_without_test @-- $newCmdt1 @init=double_suite_init_sync_sub @async=false @verbose=5 
$cmdtIn @test=double_suite_init_sync/test @-- $newCmdt1 @test=double_suite_init_sync_sub/test true
$cmdtIn @test=double_suite_init_sync/open3_after_test @fail @stderr:"$cannotReinitMsg" @-- $newCmdt1 @init=double_suite_init_sync_sub @async=false @verbose=5 

$cmdtIn @init=double_suite_init_async
$cmdtIn @test=double_suite_init_async/open1 @stderr= @-- $newCmdt1 @init=double_suite_init_async_sub @async=true @verbose=5 
$cmdtIn @test=double_suite_init_async/open2_without_test @stderr= @-- $newCmdt1 @init=double_suite_init_async_sub @async=true @verbose=5 
$cmdtIn @test=double_suite_init_async/test @stderr= @-- $newCmdt1 @test=double_suite_init_async_sub/test true
$cmdtIn @test=double_suite_init_async/open3_after_test @fail @stderr:"$cannotReinitMsg" @-- $newCmdt1 @init=double_suite_init_async_sub @async=true @verbose=5 



## Display verbosity tests
# SPEC:
# verbose=0: display reports only (no test displayed)
# verbose=1: display only not successing tests
# verbose=2: display not successing tests outs
# verbose=3: display successing tests
# verbose=4: display successing test outs
# verbose=5: display all
#
$cmdtIn @init=verbosity_sync
$cmdtIn @test=verbosity_sync/open @stderr= @-- $newCmdt0 @init=verbosity_sync_sub @failuresLimit=-1 @verbose=0 @async=false
$cmdtIn @test=verbosity_sync/verbose0_success @stderr= @-- $newCmdt0 @test=verbosity_sync_sub/verbose0_success @verbose=0 echo foo0_success
$cmdtIn @test=verbosity_sync/verbose0_failure @stderr= @-- $newCmdt0 @test=verbosity_sync_sub/verbose0_failure @verbose=0 sh -c "echo foo0_failure; exit 1"
$cmdtIn @test=verbosity_sync/verbose1_success @stderr= @-- $newCmdt0 @test=verbosity_sync_sub/verbose1_success @verbose=1 echo foo1_success
$cmdtIn @test=verbosity_sync/verbose1_failure @stderr:FAILED @stderr!:">foo1" @-- $newCmdt0 @test=verbosity_sync_sub/verbose1_failure @verbose=1 sh -c "echo foo1_failure; exit 1"
$cmdtIn @test=verbosity_sync/verbose2_success @stderr= @-- $newCmdt0 @test=verbosity_sync_sub/verbose2_success @verbose=2 echo foo2_success
$cmdtIn @test=verbosity_sync/verbose2_failure @stderr:FAILED @stderr:">foo2" @-- $newCmdt0 @test=verbosity_sync_sub/verbose2_failure @verbose=2 sh -c "echo foo2_failure; exit 1"
$cmdtIn @test=verbosity_sync/verbose3_success @stderr:PASSED @stderr!:">foo3" @-- $newCmdt0 @test=verbosity_sync_sub/verbose3_success @verbose=3 echo foo3_success
$cmdtIn @test=verbosity_sync/verbose3_failure @stderr:FAILED @stderr:">foo3" @-- $newCmdt0 @test=verbosity_sync_sub/verbose3_failure @verbose=3 sh -c "echo foo3_failure; exit 1"
$cmdtIn @test=verbosity_sync/verbose4_success @stderr:PASSED @stderr:">foo4" @-- $newCmdt0 @test=verbosity_sync_sub/verbose4_success @verbose=4 echo foo4_success
$cmdtIn @test=verbosity_sync/verbose4_failure @stderr:FAILED @stderr:">foo4" @-- $newCmdt0 @test=verbosity_sync_sub/verbose4_failure @verbose=4 sh -c "echo foo4_failure; exit 1"
$cmdtIn @test=verbosity_sync/verbose5_success @stderr:PASSED @stderr:">foo5" @-- $newCmdt0 @test=verbosity_sync_sub/verbose5_success @verbose=5 echo foo5_success
$cmdtIn @test=verbosity_sync/verbose5_failure @stderr:FAILED @stderr:">foo5" @-- $newCmdt0 @test=verbosity_sync_sub/verbose5_failure @verbose=5 sh -c "echo foo5_failure; exit 1"
$cmdtIn @test=verbosity_sync/report @fail @stderr:verbosity_sync_sub @stderr!:verbose0 @stderr!:verbose1 @stderr!:verbose2 @stderr!:verbose3 @stderr!:verbose4 @stderr!:verbose5 @-- $newCmdt0 @report=verbosity_sync_sub

$cmdtIn @test=verbosity_sync/suite_verbose0_open @stderr= @-- $newCmdt0 @init=suite_verbose0_sub @async=false @verbose=0
$cmdtIn @test=verbosity_sync/suite_verbose0_success @stderr= @-- $newCmdt0 @test=suite_verbose0_sub/success echo bar0_success
$cmdtIn @test=verbosity_sync/suite_verbose0_failure @stderr= @-- $newCmdt0 @test=suite_verbose0_sub/failure sh -c "echo bar0_failure; exit 1"
$cmdtIn @test=verbosity_sync/suite_verbose0_report @fail @stderr:suite_verbose0_sub @-- $newCmdt1 @report=suite_verbose0_sub

$cmdtIn @test=verbosity_sync/suite_verbose1_open @stderr= @-- $newCmdt0 @init=suite_verbose1_sub @async=false @verbose=1
$cmdtIn @test=verbosity_sync/suite_verbose1_success @stderr= @-- $newCmdt0 @test=suite_verbose1_sub/success echo bar1_success
$cmdtIn @test=verbosity_sync/suite_verbose1_failure @stderr:"FAILED" @stderr!=">bar1" @-- $newCmdt0 @test=suite_verbose1_sub/failure sh -c "echo bar1_failure; exit 1"
$cmdtIn @test=verbosity_sync/suite_verbose1_report @fail @stderr:suite_verbose1_sub @-- $newCmdt1 @report=suite_verbose1_sub

$cmdtIn @test=verbosity_sync/suite_verbose2_open @stderr= @-- $newCmdt0 @init=suite_verbose2_sub @async=false @verbose=2
$cmdtIn @test=verbosity_sync/suite_verbose2_success @stderr= @-- $newCmdt0 @test=suite_verbose2_sub/success echo bar2_success
$cmdtIn @test=verbosity_sync/suite_verbose2_failure @stderr:"FAILED" @stderr:">bar2" @-- $newCmdt0 @test=suite_verbose2_sub/failure sh -c "echo bar2_failure; exit 1"
$cmdtIn @test=verbosity_sync/suite_verbose2_report @fail @stderr:suite_verbose2_sub @-- $newCmdt1 @report=suite_verbose2_sub

$cmdtIn @test=verbosity_sync/suite_verbose3_open @stderr:"suite_verbose3_sub" @-- $newCmdt0 @init=suite_verbose3_sub @async=false @verbose=3
$cmdtIn @test=verbosity_sync/suite_verbose3_success @stderr:PASSED @stderr!:">bar3" @-- $newCmdt0 @test=suite_verbose3_sub/success echo bar3_success
$cmdtIn @test=verbosity_sync/suite_verbose3_failure @stderr:"FAILED" @stderr:">bar3" @-- $newCmdt0 @test=suite_verbose3_sub/failure sh -c "echo bar3_failure; exit 1"
$cmdtIn @test=verbosity_sync/suite_verbose3_report @fail @stderr:suite_verbose3_sub @-- $newCmdt1 @report=suite_verbose3_sub

$cmdtIn @test=verbosity_sync/suite_verbose4_open @stderr:"suite_verbose4_sub" @-- $newCmdt0 @init=suite_verbose4_sub @async=false @verbose=4
$cmdtIn @test=verbosity_sync/suite_verbose4_success @stderr:PASSED @stderr:">bar4" @-- $newCmdt0 @test=suite_verbose4_sub/success echo bar4_success
$cmdtIn @test=verbosity_sync/suite_verbose4_failure @stderr:"FAILED" @stderr:">bar4" @-- $newCmdt0 @test=suite_verbose4_sub/failure sh -c "echo bar4_failure; exit 1"
$cmdtIn @test=verbosity_sync/suite_verbose4_report @fail @stderr:suite_verbose4_sub @-- $newCmdt1 @report=suite_verbose4_sub

$cmdtIn @test=verbosity_sync/suite_verbose5_open @stderr:"suite_verbose5_sub" @-- $newCmdt0 @init=suite_verbose5_sub @async=false @verbose=5
$cmdtIn @test=verbosity_sync/suite_verbose5_success @stderr:PASSED @stderr:">bar5" @-- $newCmdt0 @test=suite_verbose5_sub/success echo bar5_success
$cmdtIn @test=verbosity_sync/suite_verbose5_failure @stderr:"FAILED" @stderr:">bar5" @-- $newCmdt0 @test=suite_verbose5_sub/failure sh -c "echo bar5_failure; exit 1"
$cmdtIn @test=verbosity_sync/suite_verbose5_report @fail @stderr:suite_verbose5_sub @-- $newCmdt1 @report=suite_verbose5_sub


$cmdtIn @init=verbosity_async @failuresLimit=-1
$cmdtIn @test=verbosity_async/open @stderr= @-- $newCmdt0 @init=verbosity_async_sub @failuresLimit=-1 @async=true @fork=1 @verbose=0
$cmdtIn @test=verbosity_async/verbose0_success @stderr= @-- $newCmdt0 @test=verbosity_async_sub/verbose0_success @verbose=0 echo foo0_success
$cmdtIn @test=verbosity_async/verbose0_failure @stderr= @-- $newCmdt0 @test=verbosity_async_sub/verbose0_failure @verbose=0 sh -c "echo foo0_failure; exit 1"
$cmdtIn @test=verbosity_async/verbose1_success @stderr= @-- $newCmdt0 @test=verbosity_async_sub/verbose1_success @verbose=1 echo foo1_success
$cmdtIn @test=verbosity_async/verbose1_failure @stderr= @-- $newCmdt0 @test=verbosity_async_sub/verbose1_failure @verbose=1 sh -c "echo foo1_failure; exit 1"
$cmdtIn @test=verbosity_async/verbose2_success @stderr= @-- $newCmdt0 @test=verbosity_async_sub/verbose2_success @verbose=2 echo foo2_success
$cmdtIn @test=verbosity_async/verbose2_failure @stderr= @-- $newCmdt0 @test=verbosity_async_sub/verbose2_failure @verbose=2 sh -c "echo foo2_failure; exit 1"
$cmdtIn @test=verbosity_async/verbose3_success @stderr= @-- $newCmdt0 @test=verbosity_async_sub/verbose3_success @verbose=3 echo foo3_success
$cmdtIn @test=verbosity_async/verbose3_failure @stderr= @-- $newCmdt0 @test=verbosity_async_sub/verbose3_failure @verbose=3 sh -c "echo foo3_failure; exit 1"
$cmdtIn @test=verbosity_async/verbose4_success @stderr= @-- $newCmdt0 @test=verbosity_async_sub/verbose4_success @verbose=4 echo foo4_success
$cmdtIn @test=verbosity_async/verbose4_failure @stderr= @-- $newCmdt0 @test=verbosity_async_sub/verbose4_failure @verbose=4 sh -c "echo foo4_failure; exit 1"
$cmdtIn @test=verbosity_async/verbose5_success @stderr= @-- $newCmdt0 @test=verbosity_async_sub/verbose5_success @verbose=5 echo foo5_success
$cmdtIn @test=verbosity_async/verbose5_failure @stderr= @-- $newCmdt0 @test=verbosity_async_sub/verbose5_failure @verbose=5 sh -c "echo foo5_failure; exit 1"
$cmdtIn @test=verbosity_async/report @fail @stderr:verbosity_async_sub @stderr!:verbose0 @stderr!:">foo0" @stderr!:verbose1_success @stderr:verbose1_failure @stderr!:">foo1" @stderr!:verbose2_success @stderr!:">foo2_succ" @stderr:verbose2_failure @stderr:">foo2_fail" @stderr:verbose3_success @stderr!:">foo3_succ" @stderr:verbose3_failure @stderr:">foo3_fail" @stderr:verbose4_success @stderr:">foo4_succ" @stderr:verbose4_failure @stderr:">foo4_fail" @stderr:verbose5_success @stderr:">foo5_succ" @stderr:verbose5_failure @stderr:">foo5_fail" @-- $newCmdt0 @report=verbosity_async_sub

$cmdtIn @test=verbosity_async/suite_verbose0_open @stderr= @-- $newCmdt0 @init=async_suite_verbose0_sub @async=true @verbose=0
$cmdtIn @test=verbosity_async/suite_verbose0_success @stderr= @-- $newCmdt0 @test=async_suite_verbose0_sub/success echo bar0_success
$cmdtIn @test=verbosity_async/suite_verbose0_failure @stderr= @-- $newCmdt0 @test=async_suite_verbose0_sub/failure sh -c "echo bar0_failure; exit 1"
$cmdtIn @test=verbosity_async/suite_verbose0_report @fail @stderr:suite_verbose0_sub @stderr!:bar0 @-- $newCmdt1 @report=async_suite_verbose0_sub

$cmdtIn @test=verbosity_async/suite_verbose1_open @stderr= @-- $newCmdt0 @init=async_suite_verbose1_sub @async=true @verbose=1
$cmdtIn @test=verbosity_async/suite_verbose1_success @stderr= @-- $newCmdt0 @test=async_suite_verbose1_sub/success echo bar1_success
$cmdtIn @test=verbosity_async/suite_verbose1_failure @stderr= @-- $newCmdt0 @test=async_suite_verbose1_sub/failure sh -c "echo bar1_failure; exit 1"
$cmdtIn @test=verbosity_async/suite_verbose1_report @fail @stderr:suite_verbose1_sub @stderr:FAILED @stderr!:bar2 @-- $newCmdt1 @report=async_suite_verbose1_sub

$cmdtIn @test=verbosity_async/suite_verbose2_open @stderr= @-- $newCmdt0 @init=async_suite_verbose2_sub @async=true @verbose=2
$cmdtIn @test=verbosity_async/suite_verbose2_success @stderr= @-- $newCmdt0 @test=async_suite_verbose2_sub/success echo bar2_success
$cmdtIn @test=verbosity_async/suite_verbose2_failure @stderr= @-- $newCmdt0 @test=async_suite_verbose2_sub/failure sh -c "echo bar2_failure; exit 1"
$cmdtIn @test=verbosity_async/suite_verbose2_report @fail @stderr:suite_verbose2_sub @stderr!:">bar2_succ" @stderr:FAILED @stderr:">bar2_fail" @-- $newCmdt1 @report=async_suite_verbose2_sub

$cmdtIn @test=verbosity_async/suite_verbose3_open @stderr= @-- $newCmdt0 @init=async_suite_verbose3_sub @async=true @verbose=3
$cmdtIn @test=verbosity_async/suite_verbose3_success @stderr= @-- $newCmdt0 @test=async_suite_verbose3_sub/success echo bar3_success
$cmdtIn @test=verbosity_async/suite_verbose3_failure @stderr= @-- $newCmdt0 @test=async_suite_verbose3_sub/failure sh -c "echo bar3_failure; exit 1"
$cmdtIn @test=verbosity_async/suite_verbose3_report @fail @stderr:suite_verbose3_sub @stderr:PASSED @stderr!:">bar3_succ" @stderr:FAILED @stderr:">bar3_fail" @-- $newCmdt1 @report=async_suite_verbose3_sub

$cmdtIn @test=verbosity_async/suite_verbose4_open @stderr= @-- $newCmdt0 @init=async_suite_verbose4_sub @async=true @verbose=4
$cmdtIn @test=verbosity_async/suite_verbose4_success @stderr= @-- $newCmdt0 @test=async_suite_verbose4_sub/success echo bar4_success
$cmdtIn @test=verbosity_async/suite_verbose4_failure @stderr= @-- $newCmdt0 @test=async_suite_verbose4_sub/failure sh -c "echo bar4_failure; exit 1"
$cmdtIn @test=verbosity_async/suite_verbose4_report @fail @stderr:suite_verbose4_sub @stderr:PASSED @stderr:">bar4_succ" @stderr:FAILED @stderr:">bar4_fail" @-- $newCmdt1 @report=async_suite_verbose4_sub

$cmdtIn @test=verbosity_async/suite_verbose5_open @stderr= @-- $newCmdt0 @init=async_suite_verbose5_sub @async=true @verbose=5
$cmdtIn @test=verbosity_async/suite_verbose5_success @stderr= @-- $newCmdt0 @test=async_suite_verbose5_sub/success echo bar5_success
$cmdtIn @test=verbosity_async/suite_verbose5_failure @stderr= @-- $newCmdt0 @test=async_suite_verbose5_sub/failure sh -c "echo bar5_failure; exit 1"
$cmdtIn @test=verbosity_async/suite_verbose5_report @fail @stderr:suite_verbose5_sub @stderr:PASSED @stderr:">bar5_succ" @stderr:FAILED @stderr:">bar5_fail" @-- $newCmdt1 @report=async_suite_verbose5_sub


## Flow test

syncOpenExpected="$noPanic @stderr:Test suite ["
syncReOpenExpected="$noPanic @stderr:Cleared suite:"
syncTestExpected="$noPanic @stderr:#01 @stderr!:#02"
syncReportExpected="$noPanic @stderr:1 success"
asyncOpenExpected="@stderr="
asyncReOpenExpected="@stderr:Cleared suite:"
#asyncTestExpected="$noPanic @stderr!:#01 @stderr!:#02"
asyncTestExpected="@stderr="
asyncReportExpected="$noPanic @stderr:Test suite [ @stderr:#01 @stderr!:#02 @stderr:1 success"
asyncReportAllExpected="$noPanic @stderr!:Test suite [ @stderr!:#01 @stderr!:#02 @stderr:1 success"

$cmdtIn @init=suite_flow_sync
$cmdtIn @test=suite_flow_sync/A_open $syncOpenExpected @-- $newCmdt1 @init=suite_flow_sync_sub @async=false @verbose=5 @suiteTimeout=2s
$cmdtIn @test=suite_flow_sync/A_test $syncTestExpected @stderr:tA @-- $newCmdt1 @test=suite_flow_sync_sub/tA true
$cmdtIn @test=suite_flow_sync/A_report_suite $syncReportExpected @-- $newCmdt1 @report=suite_flow_sync_sub
$cmdtIn @test=suite_flow_sync/B0_reopen $syncOpenExpected @-- $newCmdt1 @init=suite_flow_sync_sub @async=false @verbose=5 @suiteTimeout=2s
$cmdtIn @test=suite_flow_sync/B0_test $syncTestExpected @stderr:tB0z @stderr!:tA @-- $newCmdt1 @test=suite_flow_sync_sub/tB0z true
$cmdtIn @test=suite_flow_sync/B0_report_suite $syncReportExpected @stderr!:tB0z @stderr!:tA @-- $newCmdt1 @report=suite_flow_sync_sub

for i in $( seq 1 5 ); do
	$cmdtIn @test=suite_flow_sync/B${i}a_reopen $syncReOpenExpected @-- $newCmdt1 @init=suite_flow_sync_sub @async=false @verbose=5 @suiteTimeout=2s
	$cmdtIn @test=suite_flow_sync/B${i}a_test $syncTestExpected @stderr:tB${i}z @-- $newCmdt1 @test=suite_flow_sync_sub/tB${i}z true
	$cmdtIn @test=suite_flow_sync/B${i}a_report_suite $syncReportExpected @stderr!:tB${i}z @-- $newCmdt1 @report=suite_flow_sync_sub
done

for i in $( seq 1 5 ); do
	$cmdtIn @test=suite_flow_sync/B${i}b_reopen $syncReOpenExpected @-- $newCmdt1 @init=suite_flow_sync_sub @async=false @verbose=5 @suiteTimeout=2s
	$cmdtIn @test=suite_flow_sync/B${i}b_test $syncTestExpected @stderr:tB${i}z @-- $newCmdt1 @test=suite_flow_sync_sub/tB${i}z true
	$cmdtIn @test=suite_flow_sync/B${i}b_global_report $syncReportExpected @stderr!:tB${i}z @stderr:"Session duration:" @-- $newCmdt1 @report
	$cmdtIn @test=suite_flow_sync/B${i}b_global_report_all @fail $syncReportExpected @stderr:"suite_flow_sync_sub" @stderr!:tB${i}z @stderr:"Session duration:" @-- $newCmdt1 @report @all
done

$cmdtIn @test=suite_flow_sync/C_reopen $syncReOpenExpected @-- $newCmdt1 @init=suite_flow_sync_sub @async=false @verbose=5 @suiteTimeout=2s
$cmdtIn @test=suite_flow_sync/C_test $syncTestExpected @stderr:tC @-- $newCmdt1 @test=suite_flow_sync_sub/tC true
$cmdtIn @test=suite_flow_sync/C_global_report $syncReportExpected @stderr!:tA @stderr!:tB @stderr!:tC @-- $newCmdt1 @report
$cmdtIn @test=suite_flow_sync/C_global_report_all $syncReportExpected @stderr:"suite_flow_sync_sub" @fail @stderr!:tA @stderr!:tB @stderr!:tC @-- $newCmdt1 @report @all
$cmdtIn @test=suite_flow_sync/D_reopen $syncReOpenExpected @-- $newCmdt1 @init=suite_flow_sync_sub @async=false @verbose=5 @suiteTimeout=2s
$cmdtIn @test=suite_flow_sync/D_test $syncTestExpected @stderr:tD @-- $newCmdt1 @test=suite_flow_sync_sub/tD true
$cmdtIn @test=suite_flow_sync/D_report_suite $syncReportExpected @stderr!:tA @stderr!:tB @stderr!:tC @stderr!:tD @-- $newCmdt1 @report=suite_flow_sync_sub
$cmdtIn @test=suite_flow_sync/E_reopen $syncReOpenExpected @-- $newCmdt1 @init=suite_flow_sync_sub @async=false @verbose=5 @suiteTimeout=2s
$cmdtIn @test=suite_flow_sync/E_test $syncTestExpected @stderr:tE @-- $newCmdt1 @test=suite_flow_sync_sub/tE true
$cmdtIn @test=suite_flow_sync/E_global_report $syncReportExpected @stderr!:tA @stderr!:tB @stderr!:tC @stderr!:tD @stderr!:tE @-- $newCmdt1 @report

$cmdtIn @init=suite_flow_async
$cmdtIn @test=suite_flow_async/A_open $asyncOpenExpected @-- $newCmdt1 @init=suite_flow_async_sub @async=true @verbose=5 @suiteTimeout=2s
$cmdtIn @test=suite_flow_async/A_test $asyncTestExpected @-- $newCmdt1 @test=suite_flow_async_sub/tA true
$cmdtIn @test=suite_flow_async/A_report_suite $asyncReportExpected @stderr:tA @-- $newCmdt1 @report=suite_flow_async_sub

sleepTime=0
sleep $sleepTime
$cmdtIn @test=suite_flow_async/B0_reopen $asyncReOpenExpected @-- $newCmdt1 @init=suite_flow_async_sub @async=true @verbose=5 @suiteTimeout=2s
$cmdtIn @test=suite_flow_async/B0_test $asyncTestExpected @-- $newCmdt1 @test=suite_flow_async_sub/tB0z true
$cmdtIn @test=suite_flow_async/B0_report_suite $asyncReportExpected @stderr:tB0z @stderr!:tA @-- $newCmdt1 @report=suite_flow_async_sub

for i in $( seq 1 5 ); do
	sleep $sleepTime
	$cmdtIn @test=suite_flow_async/B${i}a_reopen $asyncReOpenExpected @-- $newCmdt1 @init=suite_flow_async_sub @async=true @verbose=5 @suiteTimeout=2s
	$cmdtIn @test=suite_flow_async/B${i}a_test $asyncTestExpected @-- $newCmdt1 @test=suite_flow_async_sub/tB${i}z true
	$cmdtIn @test=suite_flow_async/B${i}a_report_suite $asyncReportExpected @stderr:tB${i}z @stderr!:t1 @stderr!=t2 @-- $newCmdt1 @report=suite_flow_async_sub
done

for i in $( seq 1 5 ); do
	sleep $sleepTime
	$cmdtIn @test=suite_flow_async/B${i}b_reopen $asyncReOpenExpected @-- $newCmdt1 @init=suite_flow_async_sub @async=true @verbose=5 @suiteTimeout=2s
	$cmdtIn @test=suite_flow_async/B${i}b_test $asyncTestExpected @-- $newCmdt1 @test=suite_flow_async_sub/tB${i}z true
	$cmdtIn @test=suite_flow_async/B${i}b_global_report $asyncReportExpected @stderr:"Session duration:" @stderr:tB${i}z @stderr!:t1 @stderr!=t2 @-- $newCmdt1 @report
	$cmdtIn @test=suite_flow_async/B${i}b_global_report_all $asyncReportAllExpected @stderr:"suite_flow_async_sub" @fail @stderr:"Session duration:" @stderr!:tB${i}z @stderr!:t1 @stderr!=t2 @-- $newCmdt1 @report @all
done

$cmdtIn @test=suite_flow_async/C_reopen $asyncReOpenExpected @-- $newCmdt1 @init=suite_flow_async_sub @async=true @verbose=5 @suiteTimeout=2s
$cmdtIn @test=suite_flow_async/C_test $asyncTestExpected @-- $newCmdt1 @test=suite_flow_async_sub/tC true
$cmdtIn @test=suite_flow_async/C_global_report $asyncReportExpected @stderr:tC @stderr!:tA @stderr!:tB @-- $newCmdt1 @report
$cmdtIn @test=suite_flow_async/C_global_report_all $asyncReportAllExpected @stderr:"suite_flow_async_sub" @fail @stderr!:tC @stderr!:tA @stderr!:tB @-- $newCmdt1 @report @all

$cmdtIn @test=suite_flow_async/D_reopen $asyncReOpenExpected @-- $newCmdt1 @init=suite_flow_async_sub @async=true @verbose=5 @suiteTimeout=2s
$cmdtIn @test=suite_flow_async/D_test $asyncTestExpected @-- $newCmdt1 @test=suite_flow_async_sub/tD true
$cmdtIn @test=suite_flow_async/D_report_suite $asyncReportExpected @stderr:tD @stderr!:tA @stderr!:tB @stderr!:tC @-- $newCmdt1 @report=suite_flow_async_sub
$cmdtIn @test=suite_flow_async/E_reopen $asyncReOpenExpected @-- $newCmdt1 @init=suite_flow_async_sub @async=true @verbose=5 @suiteTimeout=2s
$cmdtIn @test=suite_flow_async/E_test $asyncTestExpected @-- $newCmdt1 @test=suite_flow_async_sub/tE true
$cmdtIn @test=suite_flow_async/E_global_report $asyncReportExpected @stderr:tE @stderr!:tA @stderr!:tB @stderr!:tC @stderr!:tD @-- $newCmdt1 @report

$cmdtIn @report

# Test a timeouting suite (tests too long)
$cmdtIn @init=suite_timeout_sync @ignore
$cmdtIn @test=suite_timeout_sync/open @stderr:suite_timeout_sync_sub @-- $newCmdt1 @init=suite_timeout_sync_sub @async=false @verbose=5 @suiteTimeout=0.1s
$cmdtIn @test=suite_timeout_sync/test_sleep @fail @stderr:timeout @-- $newCmdt1 @test=suite_timeout_sync_sub/sleep sleep 1
$cmdtIn @test=suite_timeout_sync/report_suite @fail @stderr:timeout @-- $newCmdt1 @report=suite_timeout_sync_sub

$cmdtIn @init=suite_timeout_async @ignore
$cmdtIn @test=suite_timeout_async/open @stderr= @-- $newCmdt1 @init=suite_timeout_async_sub @async=true @verbose=5 @suiteTimeout=0.1s
$cmdtIn @test=suite_timeout_async/test_sleep @stderr= @-- $newCmdt1 @test=suite_timeout_async_sub/sleep sleep 1
$cmdtIn @test=suite_timeout_async/report_suite @fail @stderr:timeout @-- $newCmdt1 @report=suite_timeout_async_sub

$cmdtIn @init=suite_flow_sync_then_async
# sync
$cmdtIn @test=suite_flow_sync_then_async/A1_open_sync $syncOpenExpected @-- $newCmdt1 @init=suite_flow_sync_then_async_sub @async=false @verbose=5 @suiteTimeout=2s
$cmdtIn @test=suite_flow_sync_then_async/A1_test $syncTestExpected @stderr:tA1 @-- $newCmdt1 @test=suite_flow_sync_then_async_sub/tA1 true
$cmdtIn @test=suite_flow_sync_then_async/A1_report_suite $syncReportExpected @stderr:suite_flow_sync_then_async_sub @stderr!:tA1 @-- $newCmdt1 @report=suite_flow_sync_then_async_sub
$cmdtIn @test=suite_flow_sync_then_async/A2_reopen_sync $syncReOpenExpected @-- $newCmdt1 @init=suite_flow_sync_then_async_sub @async=false @verbose=5 @suiteTimeout=2s
$cmdtIn @test=suite_flow_sync_then_async/A2_test $syncTestExpected @stderr:tA2 @-- $newCmdt1 @test=suite_flow_sync_then_async_sub/tA2 true
$cmdtIn @test=suite_flow_sync_then_async/A2_report_suite $syncReportExpected @stderr:suite_flow_sync_then_async_sub @stderr!:tA1 @stderr!:tA2 @-- $newCmdt1 @report=suite_flow_sync_then_async_sub
# async
$cmdtIn @test=suite_flow_sync_then_async/B_reopen_async $asyncReOpenExpected @-- $newCmdt1 @init=suite_flow_sync_then_async_sub @async=true @verbose=5 @suiteTimeout=2s
$cmdtIn @test=suite_flow_sync_then_async/B_test $asyncTestExpected @-- $newCmdt1 @test=suite_flow_sync_then_async_sub/t2 true
$cmdtIn @test=suite_flow_sync_then_async/B_report_suite $asyncReportExpected @stderr:t2 @stderr!:t1 @-- $newCmdt1 @report=suite_flow_sync_then_async_sub
# sync
$cmdtIn @test=suite_flow_sync_then_async/C_reopen_sync $syncReOpenExpected @-- $newCmdt1 @init=suite_flow_sync_then_async_sub @async=false @verbose=5 @suiteTimeout=2s
$cmdtIn @test=suite_flow_sync_then_async/C_test $syncTestExpected @stderr:tC @-- $newCmdt1 @test=suite_flow_sync_then_async_sub/tC true
$cmdtIn @test=suite_flow_sync_then_async/C_report_suite $syncReportExpected @stderr!:tA @stderr!:tB @stderr!:tC @-- $newCmdt1 @report=suite_flow_sync_then_async_sub
# Reporting all
# sync
$cmdtIn @test=suite_flow_sync_then_async/D_open_sync $syncOpenExpected @-- $newCmdt1 @init=suite_flow_sync_then_async_sub @async=false @verbose=5 @suiteTimeout=2s
$cmdtIn @test=suite_flow_sync_then_async/D_test $syncTestExpected @stderr:tD @-- $newCmdt1 @test=suite_flow_sync_then_async_sub/tD true
$cmdtIn @test=suite_flow_sync_then_async/D_global_report $syncReportExpected @stderr!:tD @stderr!:tA @stderr!:tB @stderr!:tC @-- $newCmdt1 @report
# async
$cmdtIn @test=suite_flow_sync_then_async/E_reopen_async $asyncReOpenExpected @-- $newCmdt1 @init=suite_flow_sync_then_async_sub @async=true @verbose=5 @suiteTimeout=2s
$cmdtIn @test=suite_flow_sync_then_async/E_test $asyncTestExpected @-- $newCmdt1 @test=suite_flow_sync_then_async_sub/tE true
$cmdtIn @test=suite_flow_sync_then_async/E_global_report $asyncReportExpected @stderr:tE @stderr!:tA @stderr!:tB @stderr!:tC @stderr!:tD @-- $newCmdt1 @report
# sync
$cmdtIn @test=suite_flow_sync_then_async/F_reopen_sync $syncReOpenExpected @-- $newCmdt1 @init=suite_flow_sync_then_async_sub @async=false @verbose=5 @suiteTimeout=2s
$cmdtIn @test=suite_flow_sync_then_async/F_test $syncTestExpected @stderr:tF @-- $newCmdt1 @test=suite_flow_sync_then_async_sub/tF true
$cmdtIn @test=suite_flow_sync_then_async/F_global_report $syncReportExpected @stderr!:tF @stderr!:tA @stderr!:tB @stderr!:tC @stderr!:tD @stderr!:tE @-- $newCmdt1 @report

$cmdtIn @report

newCmdt2="$newCmdt @isol=syncAndAsync $params0 $params1"

$cmdtIn @init=report_all_sync_and_async 
$cmdtIn @test=report_all_sync_and_async/ssync_open $syncOpenExpected @-- $newCmdt2 @init=report_all_ssync_sub1 @async=false @verbose=5
$cmdtIn @test=report_all_sync_and_async/ssync_test $syncTestExpected @-- $newCmdt2 @test=report_all_ssync_sub1/tsync true
$cmdtIn @test=report_all_sync_and_async/async_open $asyncOpenExpected @-- $newCmdt2 @init=report_all_async_sub1 @async=true @verbose=5
$cmdtIn @test=report_all_sync_and_async/async_test $asyncTestExpected @-- $newCmdt2 @test=report_all_async_sub1/tasync true
$cmdtIn @test=report_all_sync_and_async/global_report @stderr:"1 success" @stderr!:tsync @stderr:tasync @stderr:report_all_ssync_sub1 @stderr:report_all_async_sub1 @-- $newCmdt2 @report
$cmdtIn @test=report_all_sync_and_async/global_re-report @fail @stderr!:tsync @stderr!:tasync @stderr!:report_all_ssync_sub1 @stderr!:report_all_async_sub1 @stderr:"$nothingToReportExpectedStderrMsg" @-- $newCmdt2 @report
$cmdtIn @test=report_all_sync_and_async/global_report_all @stderr:"1 success" @stderr!:tsync @stderr!:tasync @stderr:report_all_ssync_sub1 @stderr:report_all_async_sub1 @-- $newCmdt2 @report @all
# Add more tests to already reported suite
$cmdtIn @test=report_all_sync_and_async/ssync_test2 @stderr:"tsync2" @-- $newCmdt2 @test=report_all_ssync_sub1/tsync2 true
$cmdtIn @test=report_all_sync_and_async/async_test2 @-- $newCmdt2 @test=report_all_async_sub1/tasync2 true
$cmdtIn @test=report_all_sync_and_async/global_report_2 @stderr!:"1 success" @stderr:"2 success" @stderr!:tsync2 @stderr:tasync2 @stderr:report_all_ssync_sub1 @stderr:report_all_async_sub1 @-- $newCmdt2 @report
$cmdtIn @test=report_all_sync_and_async/global_report_all_2 @stderr!:"1 success" @stderr:"2 success" @stderr!:tsync @stderr!:tasync @stderr:report_all_ssync_sub1 @stderr:report_all_async_sub1 @-- $newCmdt2 @report @all
# Add 2 new suites with tests
$cmdtIn @test=report_all_sync_and_async/ssync_open2 $syncOpenExpected @-- $newCmdt2 @init=report_all_ssync_sub2 @async=false @verbose=5
$cmdtIn @test=report_all_sync_and_async/ssync_test3 @stderr:tsync3 @-- $newCmdt2 @test=report_all_ssync_sub2/tsync3 true
$cmdtIn @test=report_all_sync_and_async/async_open2 $asyncOpenExpected @-- $newCmdt2 @init=report_all_async_sub2 @async=true @verbose=5
$cmdtIn @test=report_all_sync_and_async/async_test3 @-- $newCmdt2 @test=report_all_async_sub2/tasync3 true
$cmdtIn @test=report_all_sync_and_async/global_report_3 @stderr:"1 success" @stderr!:tsync3 @stderr:tasync3 @stderr!:report_all_ssync_sub1 @stderr!:report_all_async_sub1 @stderr:report_all_ssync_sub2 @stderr:report_all_async_sub2 @-- $newCmdt2 @report
$cmdtIn @test=report_all_sync_and_async/global_report_all_3 @stderr:"1 success" @stderr:"2 success" @stderr!:tsync @stderr!:tasync @stderr:report_all_ssync_sub1 @stderr:report_all_async_sub1 @-- $newCmdt2 @report @all
# Add 1 more test in already reported sync suite
$cmdtIn @test=report_all_sync_and_async/sync_test4 @stderr:tsync4 @-- $newCmdt2 @test=report_all_ssync_sub1/tsync4 true
$cmdtIn @test=report_all_sync_and_async/global_report_4 @stderr:"3 success" @stderr!:tasync4 @stderr:report_all_ssync_sub1 @stderr!:report_all_async_sub1 @-- $newCmdt2 @report
$cmdtIn @test=report_all_sync_and_async/global_report_all_4 @stderr:"1 success" @stderr:"2 success" @stderr:"3 success" @stderr!:tsync @stderr!:tasync @stderr:report_all_ssync_sub1 @stderr:report_all_async_sub1 @stderr:report_all_ssync_sub2 @stderr:report_all_async_sub2 @-- $newCmdt2 @report @all
# Add 1 more test in already reported async suite
$cmdtIn @test=report_all_sync_and_async/async_test4 @-- $newCmdt2 @test=report_all_async_sub1/tasync4 true
$cmdtIn @test=report_all_sync_and_async/global_report_5 @stderr:"3 success" @stderr:tasync4 @stderr!:report_all_ssync_sub1 @stderr:report_all_async_sub1 @-- $newCmdt2 @report
$cmdtIn @test=report_all_sync_and_async/global_report_all_5 @stderr:"1 success" @stderr!:"2 success" @stderr:"3 success" @stderr!:tsync @stderr!:tasync @stderr:report_all_ssync_sub1 @stderr:report_all_async_sub1 @stderr:report_all_ssync_sub2 @stderr:report_all_async_sub2 @-- $newCmdt2 @report @all
# Reinit a sync suite
$cmdtIn @test=report_all_sync_and_async/ssync_open3 @-- $newCmdt2 @init=report_all_ssync_sub1 @async=false @verbose=5
$cmdtIn @test=report_all_sync_and_async/ssync_test5 @stderr:"tsync5" @-- $newCmdt2 @test=report_all_ssync_sub1/tsync5 true
$cmdtIn @test=report_all_sync_and_async/global_report_6 @stderr:"1 success" @stderr:report_all_ssync_sub1 @stderr!:report_all_async_sub1 @-- $newCmdt2 @report
$cmdtIn @test=report_all_sync_and_async/global_report_all_6 @stderr:"1 success" @stderr:"3 success" @stderr!:tsync @stderr!:tasync @stderr:report_all_ssync_sub1 @stderr:report_all_async_sub1 @stderr:report_all_ssync_sub2 @stderr:report_all_async_sub2 @-- $newCmdt2 @report @all
# Reinit an async suite
$cmdtIn @test=report_all_sync_and_async/async_open3 @-- $newCmdt2 @init=report_all_async_sub1 @async=true @verbose=5
$cmdtIn @test=report_all_sync_and_async/async_test5 @-- $newCmdt2 @test=report_all_async_sub1/tasync5 true
$cmdtIn @test=report_all_sync_and_async/global_report_7 @stderr:"1 success" @stderr:tasync5 @stderr!:report_all_ssync_sub1 @stderr:report_all_async_sub1 @-- $newCmdt2 @report
$cmdtIn @test=report_all_sync_and_async/global_report_all_7 @stderr:"1 success" @stderr!:"2 success" @stderr!:tsync @stderr!:tasync @stderr:report_all_ssync_sub1 @stderr:report_all_async_sub1 @stderr:report_all_ssync_sub2 @stderr:report_all_async_sub2 @-- $newCmdt2 @report @all
# Add new forked async suite with tests
$cmdtIn @test=report_all_sync_and_async/fork_open $asyncOpenExpected @-- $newCmdt2 @init=report_all_fork_sub1 @fork=2 @verbose=5
$cmdtIn @test=report_all_sync_and_async/fork_test @-- $newCmdt2 @test=report_all_fork_sub1/tfork1 true
$cmdtIn @test=report_all_sync_and_async/global_report_8 @stderr:"1 success" @stderr:tfork1 @stderr!:report_all_ssync_sub1 @stderr!:report_all_async_sub1 @stderr!:report_all_ssync_sub2 @stderr!:report_all_async_sub2 @stderr:report_all_fork_sub1 @-- $newCmdt2 @report
$cmdtIn @test=report_all_sync_and_async/global_report_all_8 @stderr:"1 success" @stderr!:"2 success" @stderr!:tsync @stderr!:tasync @stderr:report_all_ssync_sub1 @stderr:report_all_async_sub1 @stderr:report_all_ssync_sub2 @stderr:report_all_async_sub2 @stderr:report_all_fork_sub1 @-- $newCmdt2 @report @all


## Launch a longer suite async

$cmdtIn @init=longer_sync0 #@verbose
$cmdtIn @test=longer_sync0/init $noPanic @stderr:"longer_sync0_sub" @-- $newCmdt1 @init=longer_sync0_sub @async=false @verbose=5
for i in $( seq 1 4 ); do
    $cmdtIn @test=longer_sync0/ $noPanic @stderr:"#0$i" @-- $newCmdt1 @test=longer_sync0_sub/t$i @stdout:"end$i" @-- sh -c "echo end$i"
done
# should report nearly instantly
$cmdtIn @test=longer_sync0/report $noPanic @timeout=2s @-- $newCmdt1 @report=longer_sync0_sub

$cmdtIn @init=longer_async0 #@verbose
$cmdtIn @test=longer_async0/init @stderr= @-- $newCmdt1 @init=longer_async0_sub @async @verbose=5
for i in $( seq 1 4 ); do
    $cmdtIn @test=longer_async0/ @stderr= @-- $newCmdt1 @test=longer_async0_sub/t$i @stdout:"end$i" @-- sh -c "echo end$i"
done
# should report nearly instantly
$cmdtIn @test=longer_async0/report $noPanic @timeout=2s @stderr:"longer_async0_sub" @stderr:"#01" @stderr:"#02" @stderr:"#03" @stderr:"#04" @stderr!:"#05" @-- $newCmdt1 @report=longer_async0_sub

$cmdtIn @init=longer_async1 #@verbose
$cmdtIn @test=longer_async1/init @stderr= @-- $newCmdt1 @init=longer_async1_sub @async @verbose=5
for i in $( seq 1 4 ); do
	time=$( echo "scale=1;$i/10" | bc )
	$cmdtIn @test=longer_async1/ @stderr= @-- $newCmdt1 @test=longer_async1_sub/t$i @stdout:"end$i" @-- sh -c "sleep $time; echo end$i"
done
$cmdtIn @test=longer_async1/report $noPanic @timeout=2s @stderr:"longer_async1_sub" @stderr:"#01" @stderr:"#02" @stderr:"#03" @stderr:"#04" @stderr!:"#05" @-- $newCmdt1 @report=longer_async1_sub

$cmdtIn @init=longer_async2 #@verbose
$cmdtIn @test=longer_async2/init @stderr= @-- $newCmdt1 @init=longer_async2_sub @async @verbose=5
for i in $( seq 1 4 ); do
	time=$( echo "scale=1;$i/10" | bc )
	$cmdtIn @test=longer_async2/ @stderr= @-- $newCmdt1 @test=longer_async2_sub/t$i @stdout:"end$i" @-- sh -c "sleep $time; echo end$i"
done
$cmdtIn @test=longer_async2/report_all $noPanic @timeout=2s @stderr:"longer_async2_sub" @stderr:"#01" @stderr:"#02" @stderr:"#03" @stderr:"#04" @stderr!:"#05" @-- $newCmdt1 @report

$cmdtIn @report 

