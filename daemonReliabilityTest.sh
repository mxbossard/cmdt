#! /bin/bash
set -e -o pipefail
scriptDir=$( dirname $( readlink -f $0 ) )

COUNT="${1:-1}"

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
cmdt0="$newCmdt @isol=tested"
cmdt1="$newCmdt @isol=tested @failuresLimit=-1" # Default verbose show passed test + perform all test beyond failures limit

die() {
	>&2 echo "$1"
	exit 1
}

find /tmp -name "cmdt*.log" -o -name "daemon*.log" -exec rm {} \; 2> /dev/null || true
rm -rf -- /tmp/cmdt* 2> /dev/null || true

#$cmdt @global @silent
$cmdt @global @suiteTimeout=10s @verbose=5 #FIXME: @global @verbose=5 does not works !
$cmdt @init @suiteTimeout=10s @verbose=5

# Clear context
export -n __CMDT_TOKEN


testSleepTime=0.2
#testSleepTime=2
testCount=5

test1() {
	>&2 echo
	>&2 echo "## test @fork=1 long tests longer than daemon inactivity max period"
	$cmdt1 @init=long_fork1 @fork=1
	>&2 echo ">> long_fork1/sleep_3sec ..."
	$cmdt1 @test=long_fork1/sleep_3sec sleep 3
	>&2 echo ">> daemon PID: $( pgrep -f "cmdt ._daemon" || true )"
	>&2 echo ">> sleep 3sec ..."
	sleep 3
	>&2 echo ">> long_fork1/sleep_0.1sec ..."
	$cmdt1 @test=long_fork1/sleep_0.1sec sleep 0.1

	#tree -Ch /tmp/cmdtest-*_tested/zcreen
	>&2 echo ">> reporting long_fork1 ..."
	$cmdt @test="report_long_fork1" @stderr:"Successfully ran" @stderr:"2 success" @-- $cmdt1 @report=long_fork1

	>&2 echo ">> daemon PID: $( pgrep -f "cmdt ._daemon" || true )"
}
#test1

$cmdt1 @init @fork=1
$cmdt1 true

#tree -Ch /tmp/cmdtest-*_tested/zcreen
$cmdt1 @report

>&2 echo
>&2 echo "## test @fork=2"
$cmdt1 @init=fork2 @fork=2 @verbose=5
for i in $( seq -f "%02g" 1 $testCount ); do
	$cmdt1 @verbose=5 @test=fork2/sleep_$i sleep "$testSleepTime"
done
>&2 echo ">> launched $testCount tests"

sleep "$testSleepTime"
#tree -Ch /tmp/cmdtest-*_tested/zcreen

# Kill Daemon
>&2 echo ">> Killing daemon ..."
pkill -1 -f "cmdt ._daemon" || true

#exit 1

#tree -Ch /tmp/cmdtest-*_tested/zcreen
>&2 echo ">> reporting fork2 ..."
$cmdt @test="report_fork2" @stderr:"Successfully ran" @stderr:"$testCount success" @-- $cmdt1 @report=fork2
#$cmdt @test="report_fork2" @stderr:"Successfully ran" @stderr:"$testCount success" @-- $cmdt1 @report

#tree -Ch /tmp/cmdtest-*_tested/zcreen
#exit 1

>&2 echo
>&2 echo "## test @fork=5"
$cmdt1 @init=fork5 @fork=5 @verbose=5

#$cmdt1 @test=fork/echo_foo echo foo
for i in $( seq -f "%02g" 1 $testCount ); do
	$cmdt1 @test=fork5/sleep_$i sleep "$testSleepTime"
done
>&2 echo ">> launched $testCount tests"
# Kill Daemon
>&2 echo ">> Killing daemon ..."
pkill -f "cmdt ._daemon" || true

>&2 echo ">> reporting fork5 ..."
$cmdt @test="report_fork5" @stderr:"Successfully ran" @stderr:"$testCount success" @-- $cmdt1 @report=fork5

$cmdt @report @all

