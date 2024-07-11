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
cmdt1="$newCmdt @isol=tested @verbose @failuresLimit=-1 @async" # Default verbose show passed test + perform all test beyond failures limit
#cmdt2="$newCmdt @verbose @failuresLimit=-1 @async @wait"

die() {
	>&2 echo "$1"
	exit 1
}

#$cmdt @global @silent

# Clear context
export -n __CMDT_TOKEN

nothingToReportExpectedStderrMsg="you must perform some test prior to report"
>&2 echo "## Meta1 test context not shared without token"
$cmdtIn @init=meta1 #@verbose=4
$cmdtIn @test=meta1/"without token one" @stderr= @-- $cmdt1 true
$cmdtIn @test=meta1/"without token two" @stderr= @-- $cmdt1 true
#$cmdtIn @test=meta1/"command before rule stop" @stderr= @-- $cmdt1 true @-- @success
$cmdtIn @test=meta1/"rule on 2 args" @stderr= @-- $cmdt1 @stdout:foo bar @-- echo foo bar
$cmdtIn @test=meta1/ @exit=0 @stderr:"#01" @stderr:"#02" @stderr:"before rule parsing stopper" @stderr:"PASSED" @stderr:"3 success" @stderr:"0 failure" @stderr:"0 error" @-- $cmdt0 @verbose @report=main

>&2 echo "## Test printed token"
tk0=$( $cmdt @init @printToken 2> /dev/null )
>&2 echo "token: $tk0"
$cmdtIn @init=meta2 #@verbose=4
$cmdtIn @test=meta2/ @stderr:"PASSED" @stderr:"#01" @-- $cmdt1 true @token=$tk0
$cmdtIn @test=meta2/ @stderr:"PASSED" @stderr:"#02" @-- $cmdt1 true @token=$tk0
$cmdtIn @test=meta2/ @fail @-- $cmdt1 @report=main
$cmdtIn @test=meta2/ @stderr:"2 success" @stderr!:"failure" @stderr!:"error" @-- $cmdt1 @report @token=$tk0
$cmdt @report 2>&1 | grep -v "Failures"

>&2 echo "## Test exported token"
eval $( $cmdt @init @exportToken 2> /dev/null )
>&2 echo "token: $__CMDT_TOKEN"
$cmdtIn @init=meta3
$cmdtIn @test=meta3/ @stderr:"PASSED" @stderr:"#01" @-- $cmdt1 true
$cmdtIn @test=meta3/ @stderr:"PASSED" @stderr:"#02" @-- $cmdt1 true
$cmdtIn @test=meta3/ @stderr:"Successfuly ran" @-- $cmdt1 @report=main
$cmdtIn @test=meta3/ @fail @stderr:"$nothingToReportExpectedStderrMsg" @-- $cmdt1 @report=main @token=$tk0

$cmdtIn @init=meta4
$cmdtIn @test=meta4/ @stderr:"PASSED" @stderr:"#01" @-- $cmdt1 @test=sub4/ true
$cmdtIn @test=meta4/ @stderr:"PASSED" @stderr:"#02" @-- $cmdt1 @test=sub4/ true
$cmdtIn @test=meta4/ @stderr:"Successfuly ran" @-- $cmdt1 @report=sub4
$cmdtIn @test=meta4/ @fail @stderr:"$nothingToReportExpectedStderrMsg" @-- $cmdt1 @report=sub4 @token=$tk0
$cmdt @report 2>&1 | grep -v "Failures"

export -n __CMDT_TOKEN

exit 0
