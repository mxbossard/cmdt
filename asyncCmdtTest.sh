#! /bin/bash
set -e -o pipefail
scriptDir=$( dirname $( readlink -f $0 ) )

. $scriptDir/buildCmdt.sh
newCmdt="$BUILT_CMDT_BIN"
ls -lh "$newCmdt"

# Trusted cmdt to works
cmdt="cmdt"
cmdt="$newCmdt"

# Cmdt used to test
#cmdtIn="cmdt"
cmdtIn="$cmdt $@"

# Tested cmdt
newCmdt="$newCmdt"
cmdt0="$newCmdt @isol=tested"
cmdt1="$newCmdt @isol=tested @verbose=4 @failuresLimit=-1" # Default verbose show passed test + perform all test beyond failures limit
#cmdt2="$newCmdt @verbose @failuresLimit=-1 @async @wait"

# Isolate tester & tested cmdt with tokens
testerTk=$( $cmdt @init @printToken )
cmdt="$cmdt @token=$testerTk"
cmdtIn="$cmdtIn @token=$testerTk"
newTk=$( $newCmdt @isol=tested @init @printToken @debug )
newCmdt0="$newCmdt0 @token=$newTk"
newCmdt1="$newCmdt1 @token=$newTk"

die() {
	>&2 echo "$1"
	exit 1
}

#$cmdt @global @silent

rm -rf -- /tmp/cmdt* /tmp/cmdt.log /tmp/daemon.log 2> /dev/null || true

# Clear context
export -n __CMDT_TOKEN

$cmdtIn @init="async success" 
$cmdtIn @test=async success/should init @-- $cmdt1 @init=main1 @async @verbose=4
$cmdtIn @test=async success/"should pass 1" @stderr= @-- $cmdt1 @test=main1/t1 true @verbose=4
$cmdtIn @test=async success/"should pass 2" @stderr= @-- $cmdt1 @test=main1/t2 sleep 0.2 @verbose=4
$cmdtIn @test=async success/"should pass 3" @stderr= @-- $cmdt1 @test=main1/t3 true @verbose=4
$cmdtIn @test=async success/should report @exit=0 @stderr:"#01" @stderr:"#02" @stderr!:"#04" @stderr:"PASSED" @stderr!:"FAILED" @stderr:"3 success" @stderr!:"failure" @stderr!:"error" @-- $cmdt0 @verbose @report=main1 @debug=6
$cmdtIn @report 2>&1 | grep -v "Failures"

$cmdtIn @init="async failure"
$cmdtIn @test=async failure/should init @-- $cmdt1 @init=main2 @async @verbose=4
$cmdtIn @test=async failure/"should pass" @stderr= @-- $cmdt1 @test=main2/t1 true
$cmdtIn @test=async failure/"should fail" @stderr= @-- $cmdt1 @test=main2/t2 false
$cmdtIn @test=async failure/should report @exit=1 @stderr:"#01" @stderr:"#02" @stderr!:"#03" @stderr:"PASSED" @stderr:"FAILED" @stderr:"1 success" @stderr:"1 failure" @stderr:"0 error" @-- $cmdt0 @verbose @report=main2 @debug=6
$cmdtIn @report 2>&1 | grep -v "Failures"

$cmdtIn @init="sync error" #@verbose=4
$cmdtIn @test=sync error/should init @-- $cmdt1 @init=main3 @async=false @verbose=5
$cmdtIn @test=sync error/should pass @stderr:"#01" @stderr:"PASSED" @-- $cmdt1 @test=main3/t1 true
$cmdtIn @test=sync error/should error 1 @fail @stderr:"#02" @stderr:"ERRORED" @stderr:'badRule does not exists' @-- $cmdt1 @test=main3/t2 true @badRule
$cmdtIn @test=sync error/should error 2 @fail @stderr:"#03" @stderr:"ERRORED" @-- $cmdt1 @test=main3/t3 true @before=badCmd
$cmdtIn @test=sync error/should report @exit=1 @stderr:"1 success" @stderr:"0 failure" @stderr:"2 error" @stderr:"3 test" @-- $cmdt0 @verbose @report=main3 @debug=6
$cmdtIn @report 2>&1 | grep -v "Failures"

$cmdtIn @init="async error" #@verbose=4
$cmdtIn @test=async error/should init @-- $cmdt1 @init=main4 @async @verbose=4
$cmdtIn @test=async error/should pass @stderr= @-- $cmdt1 @test=main4/t1 true
$cmdtIn @test=async error/should error 1 @fail @stderr:'badRule does not exists' @-- $cmdt1 @test=main4/t2 true @badRule
$cmdtIn @test=async error/should error 2 @stderr= @-- $cmdt1 @test=main4/t3 true @before=badCmd
$cmdtIn @test=async error/should report @exit=1 @stderr:"#01" @stderr!:"#02" @stderr:"#03" @stderr!:"#04" @stderr:"PASSED" @stderr!:"FAILED" @stderr:"ERRORED" @stderr:"1 success" @stderr:"0 failure" @stderr:"2 error" @-- $cmdt0 @verbose @report=main4 @debug=6
$cmdtIn @report 2>&1 | grep -v "Failures"


# FIXME: async report should not fail if no test exist yet. It should fail after a short timeout if no test to report.
nothingToReportExpectedStderrMsg="no test to report"
>&2 echo "## Test @report without test"
$cmdtIn @init=meta0 #@verbose=4
$cmdtIn @test=meta0/ @fail @stderr:"$nothingToReportExpectedStderrMsg" @-- $cmdt0 @report=foo @async #@debug=4
$cmdtIn @test=meta0/ @stderr= @-- $cmdt0 @init=foo @async #@debug=4
$cmdtIn @test=meta0/ @fail @stderr:"$nothingToReportExpectedStderrMsg" @-- $cmdt0 @report=foo @async #@debug=4
$cmdtIn @test=meta0/ @fail @stderr:"$nothingToReportExpectedStderrMsg" @-- $cmdt0 @report @async #@debug=4

>&2 echo "## Meta1 test context not shared without token"
$cmdtIn @init=meta1 #@verbose=4
$cmdtIn @test=meta1/init @stderr= @-- $cmdt1 @init @async @verbose=4
$cmdtIn @test=meta1/"without token one" @stderr= @-- $cmdt1 true #@debug
$cmdtIn @test=meta1/"without token two" @stderr= @-- $cmdt1 true #@debug
$cmdtIn @test=meta1/"command before rule stop" @fail @stderr:"ERRORED" @stderr:"before rule parsing stopper" @-- $cmdt1 true @-- @success
$cmdtIn @test=meta1/"rule on 2 args" @stderr= @-- $cmdt1 @stdout:foo bar @-- echo foo bar
$cmdtIn @test=meta1/ @exit=1 @stderr:"3 success" @stderr:"0 failure" @stderr:"1 error" @stderr:"PASSED" @stderr!:"ERRORED" @stderr:"#01" @stderr:"#02" @stderr!:"#03" @stderr:"#04" @stderr!:"#05" @-- $cmdt0 @report=main

exit 0

>&2 echo "## Test printed token"
tk0=$( $cmdt @init @printToken 2> /dev/null )
>&2 echo "token: $tk0"
$cmdtIn @init=meta2 #@verbose=4
$cmdtIn @test=meta2/init @-- $cmdt1 @init @async
$cmdtIn @test=meta2/ @stderr:"PASSED" @stderr:"#01" @-- $cmdt1 true @token=$tk0
$cmdtIn @test=meta2/ @stderr:"PASSED" @stderr:"#02" @-- $cmdt1 true @token=$tk0
$cmdtIn @test=meta2/ @fail @stderr:"$nothingToReportExpectedStderrMsg" @-- $cmdt1 @report=main
$cmdtIn @test=meta2/ @stderr:"2 success" @stderr!:"failure" @stderr!:"error" @-- $cmdt1 @report @token=$tk0
$cmdt @report 2>&1 | grep -v "Failures"

>&2 echo "## Test exported token"
eval $( $cmdt @init @exportToken 2> /dev/null )
>&2 echo "token: $__CMDT_TOKEN"
$cmdtIn @init=meta3 
$cmdtIn @test=meta3/init @-- $cmdt1 @init @async
$cmdtIn @test=meta3/ @stderr:"PASSED" @stderr:"#01" @-- $cmdt1 true
$cmdtIn @test=meta3/ @stderr:"PASSED" @stderr:"#02" @-- $cmdt1 true
$cmdtIn @test=meta3/ @stderr:"Successfuly ran" @-- $cmdt1 @report=main
$cmdtIn @test=meta3/ @fail @stderr:"$nothingToReportExpectedStderrMsg" @-- $cmdt1 @report=main @token=$tk0

$cmdtIn @init=meta4
$cmdtIn @test=meta4/init @-- $cmdt1 @init @async
$cmdtIn @test=meta4/ @stderr:"PASSED" @stderr:"#01" @-- $cmdt1 @test=sub4/ true
$cmdtIn @test=meta4/ @stderr:"PASSED" @stderr:"#02" @-- $cmdt1 @test=sub4/ true
$cmdtIn @test=meta4/ @stderr:"Successfuly ran" @-- $cmdt1 @report=sub4
$cmdtIn @test=meta4/ @fail @stderr:"$nothingToReportExpectedStderrMsg" @-- $cmdt1 @report=sub4 @token=$tk0
$cmdt @report 2>&1 | grep -v "Failures"

export -n __CMDT_TOKEN

exit 0
