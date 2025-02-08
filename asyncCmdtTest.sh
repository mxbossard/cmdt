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


# Clear context
export -n __CMDT_TOKEN

$cmdtIn @init="async success" 
$cmdtIn @test=async success/should init @-- $newCmdt1 @init=main1 @async @verbose=4
$cmdtIn @test=async success/"should pass 1" @stderr= @-- $newCmdt1 @test=main1/t1 true @verbose=4
$cmdtIn @test=async success/"should pass 2" @stderr= @-- $newCmdt1 @test=main1/t2 sleep 0.2 @verbose=4
$cmdtIn @test=async success/"should pass 3" @stderr= @-- $newCmdt1 @test=main1/t3 true @verbose=4
$cmdtIn @test=async success/should report @exit=0 @stderr:"#01" @stderr:"#02" @stderr!:"#04" @stderr:"PASSED" @stderr!:"FAILED" @stderr:"3 success" @stderr!:"failure" @stderr!:"error" @-- $newCmdt0 @verbose @report=main1 @debug=6
$cmdtIn @report 2>&1 | grep -v "Failures"

$cmdtIn @init="async failure"
$cmdtIn @test=async failure/should init @-- $newCmdt1 @init=main2 @async @verbose=4
$cmdtIn @test=async failure/"should pass" @stderr= @-- $newCmdt1 @test=main2/t1 true
$cmdtIn @test=async failure/"should fail" @stderr= @-- $newCmdt1 @test=main2/t2 false
$cmdtIn @test=async failure/should report @exit=1 @stderr:"#01" @stderr:"#02" @stderr!:"#03" @stderr:"PASSED" @stderr:"FAILED" @stderr:"1 success" @stderr:"1 failure" @stderr:"0 error" @-- $newCmdt0 @verbose @report=main2 @debug=6
$cmdtIn @report 2>&1 | grep -v "Failures"

$cmdtIn @init="sync error" #@verbose=4
$cmdtIn @test=sync error/should init @-- $newCmdt1 @init=main3 @async=false @verbose=5
$cmdtIn @test=sync error/should pass @stderr:"#01" @stderr:"PASSED" @-- $newCmdt1 @test=main3/t1 true
$cmdtIn @test=sync error/should error 1 @fail @stderr:"#02" @stderr:"ERRORED" @stderr:'badRule does not exists' @-- $newCmdt1 @test=main3/t2 true @badRule
$cmdtIn @test=sync error/should error 2 @fail @stderr:"#03" @stderr:"ERRORED" @-- $newCmdt1 @test=main3/t3 true @before=badCmd
$cmdtIn @test=sync error/should report @exit=1 @stderr:"1 success" @stderr:"0 failure" @stderr:"2 error" @stderr:"3 test" @-- $newCmdt0 @verbose @report=main3 @debug=6
$cmdtIn @report 2>&1 | grep -v "Failures"

$cmdtIn @init="async error" #@verbose=4
$cmdtIn @test=async error/should init @-- $newCmdt1 @init=main4 @async @verbose=4
$cmdtIn @test=async error/should pass @stderr= @-- $newCmdt1 @test=main4/t1 true
$cmdtIn @test=async error/should error 1 @fail @stderr:'badRule does not exists' @-- $newCmdt1 @test=main4/t2 true @badRule
$cmdtIn @test=async error/should error 2 @stderr= @-- $newCmdt1 @test=main4/t3 true @before=badCmd
$cmdtIn @test=async error/should report @exit=1 @stderr:"#01" @stderr!:"#02" @stderr:"#03" @stderr!:"#04" @stderr:"PASSED" @stderr!:"FAILED" @stderr:"ERRORED" @stderr:"1 success" @stderr:"0 failure" @stderr:"2 error" @-- $newCmdt0 @verbose @report=main4 @debug=6
$cmdtIn @report 2>&1 | grep -v "Failures"


# FIXME: async report should not fail if no test exist yet. It should fail after a short timeout if no test to report.
nothingToReportExpectedStderrMsg="no test to report"
>&2 echo "## Test @report without test"
$cmdtIn @init=meta0 #@verbose=4
$cmdtIn @test=meta0/ @fail @stderr:"$nothingToReportExpectedStderrMsg" @-- $newCmdt0 @report=foo @async #@debug=4
$cmdtIn @test=meta0/ @stderr= @-- $newCmdt0 @init=foo @async #@debug=4
$cmdtIn @test=meta0/ @fail @stderr:"$nothingToReportExpectedStderrMsg" @-- $newCmdt0 @report=foo @async #@debug=4
$cmdtIn @test=meta0/ @fail @stderr:"$nothingToReportExpectedStderrMsg" @-- $newCmdt0 @report @async #@debug=4

>&2 echo "## Meta1 test context not shared without token"
$cmdtIn @init=meta1 #@verbose=4
$cmdtIn @test=meta1/init @stderr= @-- $newCmdt1 @init @async @verbose=4
$cmdtIn @test=meta1/"without token one" @stderr= @-- $newCmdt1 true #@debug
$cmdtIn @test=meta1/"without token two" @stderr= @-- $newCmdt1 true #@debug
$cmdtIn @test=meta1/"command before rule stop" @fail @stderr:"ERRORED" @stderr:"before rule parsing stopper" @-- $newCmdt1 true @-- @success
$cmdtIn @test=meta1/"rule value splited in 2 args" @stderr= @-- $newCmdt1 @stdout:foo bar @-- echo foo bar
$cmdtIn @test=meta1/"report without token" @exit=1 @stderr:"3 success" @stderr:"0 failure" @stderr:"1 error" @stderr:"PASSED" @stderr!:"ERRORED" @stderr:"#01" @stderr:"#02" @stderr!:"#03" @stderr:"#04" @stderr!:"#05" @-- $newCmdt0 @report=main @async

>&2 echo "## Test printed token"
tk0=$( $newCmdt0 @init @printToken )
>&2 echo "token: $tk0"
$cmdtIn @init=meta2 #@verbose=4
$cmdtIn @test=meta2/init @-- $newCmdt1 @token=$tk0 @init @async
$cmdtIn @test=meta2/"with token one" @stderr= @-- $newCmdt1 @token=$tk0 true
$cmdtIn @test=meta2/"with token two" @stderr= @-- $newCmdt1 @token=$tk0 true
$cmdtIn @test=meta2/"report without token" @fail @stderr:"$nothingToReportExpectedStderrMsg" @-- $newCmdt1 @report=main @async
$cmdtIn @test=meta2/"report all with token" @stderr:"2 success" @stderr!:"failure" @stderr!:"error" @stderr:"#01" @stderr:"#02" @stderr!:"#03" @-- $newCmdt1 @token=$tk0 @report @async
$cmdt @report 2>&1 | grep -v "Failures"

>&2 echo "## Test exported token"
eval $( $cmdt @init @exportToken )
>&2 echo "token: $__CMDT_TOKEN"
$cmdtIn @init=meta3 
$cmdtIn @test=meta3/init @-- $newCmdt1 @init @async
$cmdtIn @test=meta3/ @stderr= @-- $newCmdt1 true
$cmdtIn @test=meta3/ @stderr= @-- $newCmdt1 true
$cmdtIn @test=meta3/ @stderr:"Successfuly ran" @stderr!:"error" @stderr:"#01" @stderr:"#02" @stderr!:"#03" @-- $newCmdt1 @report=main @async
$cmdtIn @test=meta3/ @fail @stderr:"$nothingToReportExpectedStderrMsg" @-- $newCmdt1 @token=$tk0 @report=main @async

$cmdtIn @init=meta4
$cmdtIn @test=meta4/init @-- $newCmdt1 @init=sub4 @async
$cmdtIn @test=meta4/ @stderr= @-- $newCmdt1 @test=sub4/ true
$cmdtIn @test=meta4/ @stderr= @-- $newCmdt1 @test=sub4/ true
$cmdtIn @test=meta4/ @stderr:"Successfuly ran" @stderr!:"error" @stderr:"#01" @stderr:"#02" @stderr!:"#03" @-- $newCmdt1 @report=sub4 @async
$cmdtIn @test=meta4/ @fail @stderr:"$nothingToReportExpectedStderrMsg" @-- $newCmdt1 @token=$tk0 @report=sub4 @async
$cmdt @report 2>&1 | grep -v "Failures"

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

$cmdtIn @init=failure_async #@verbose
$cmdtIn @test=failure_async/init $noPanic @-- $newCmdt1 @init=failure_async_sub @async=true @verbose=5
$cmdtIn @test=failure_async/success @stderr= @-- $newCmdt1 @test=failure_async_sub/success true
$cmdtIn @test=failure_async/failure @stderr= @-- $newCmdt1 @test=failure_async_sub/failure false
$cmdtIn @test=failure_async/report $noPanic @exit=1 @stderr:"PASSED" @stderr!:"IGNORED" @stderr:"FAILED" @stderr!:"ERRORED" @stderr!:"TIMEOUT" @stderr:"1 success" @stderr:"1 failure" @-- $newCmdt1 @report=failure_async_sub

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


## Launch a longer suite async

$cmdtIn @init=longer_sync0 #@verbose
$cmdtIn @test=longer_sync0/init $noPanic @stderr:"[longer_sync0_sub]" @-- $newCmdt1 @init=longer_sync0_sub @async=false @verbose=5
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
$cmdtIn @test=longer_async0/report $noPanic @timeout=2s @stderr:"[longer_async0_sub]" @stderr:"#01" @stderr:"#02" @stderr:"#03" @stderr:"#04" @stderr:"#05" @stderr!:"#06" @-- $newCmdt1 @report=longer_async0_sub

$cmdtIn @init=longer_async1 #@verbose
$cmdtIn @test=longer_async1/init @stderr= @-- $newCmdt1 @init=longer_async1_sub @async @verbose=5
for i in $( seq 1 4 ); do
    $cmdtIn @test=longer_async1/ @stderr= @-- $newCmdt1 @test=longer_async1_sub/t$i @stdout:"end$i" @-- sh -c "sleep $i; echo end$i"
done
$cmdtIn @test=longer_async1/report $noPanic @timeout=10s @stderr:"[longer_async1_sub]" @stderr:"#01" @stderr:"#02" @stderr:"#03" @stderr:"#04" @stderr:"#05" @stderr!:"#06" @-- $newCmdt1 @report=longer_async1_sub

$cmdtIn @init=longer_async2 #@verbose
$cmdtIn @test=longer_async2/init @stderr= @-- $newCmdt1 @init=longer_async2_sub @async @verbose=5
for i in $( seq 1 4 ); do
    $cmdtIn @test=longer_async2/ @stderr= @-- $newCmdt1 @test=longer_async2_sub/t$i @stdout:"end$i" @-- sh -c "sleep $i; echo end$i"
done
$cmdtIn @test=longer_async2/report_all $noPanic @timeout=10s @stderr:"[longer_async2_sub]" @stderr:"#01" @stderr:"#02" @stderr:"#03" @stderr:"#04" @stderr:"#05" @stderr!:"#06" @-- $newCmdt1 @report

$cmdtIn @report

