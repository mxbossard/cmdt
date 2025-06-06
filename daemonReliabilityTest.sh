#! /bin/bash
set -e -o pipefail
scriptDir=$( dirname $( readlink -f $0 ) )

COUNT="${1:-1}"

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
cmdt0="$newCmdt @isol=tested"
cmdt1="$newCmdt @isol=tested @failuresLimit=-1" # Default verbose show passed test + perform all test beyond failures limit

die() {
	>&2 echo "$1"
	exit 1
}

find /tmp -name "cmdt*.log" -o -name "daemon*.log" -exec rm {} \; 2> /dev/null || true
rm -rf -- /tmp/cmdt* 2> /dev/null || true

#$cmdt @global @silent
$cmdt @global @suiteTimeout=10s @verbose=5

# Clear context
export -n __CMDT_TOKEN


testSleepTime=0.1
testCount=5

>&2 echo
>&2 echo "## test @fork=1 long tests longer than daemon inactivity max period"
$cmdt1 @init=long_fork1 @verbose=5 @fork=1
>&2 echo ">> long_fork1/sleep_3sec ..."
$cmdt1 @test=long_fork1/sleep_3sec sleep 3
>&2 echo ">> daemon PID: $( pgrep -f "cmdt ._daemon" || true )"
>&2 echo ">> sleep 3sec ..."
sleep 3
>&2 echo ">> long_fork1/sleep_0.1sec ..."
$cmdt1 @test=long_fork1/sleep_0.1sec sleep 0.1

>&2 echo ">> reporting long_fork1 ..."
$cmdt @verbose=5 @test="report_long_fork1" @stderr:"Successfully ran" @stderr:"2 success" @-- $cmdt1 @report=long_fork1

>&2 echo ">> daemon PID: $( pgrep -f "cmdt ._daemon" || true )"

>&2 echo
>&2 echo "## test @fork=2"
$cmdt1 @init=fork2 @verbose=3 @fork=2
for i in $( seq -f "%02g" 1 $testCount ); do
	$cmdt1 @test=fork2/sleep_$i sleep $( echo "$testSleepTime * ($i % 3 + 1) ^ 2" | bc )
done
>&2 echo ">> launched $testCount tests"
# Kill Daemon
>&2 echo ">> Killing daemon ..."
pkill -1 -f "cmdt ._daemon" || true

>&2 echo ">> reporting fork2 ..."
$cmdt @verbose=5 @test="report_fork2" @stderr:"Successfully ran" @stderr:"$testCount success" @-- $cmdt1 @report=fork2

>&2 echo
>&2 echo "## test @fork=5"
$cmdt1 @init=fork5 @verbose=3 @fork=5

#$cmdt1 @test=fork/echo_foo echo foo
for i in $( seq -f "%02g" 1 $testCount ); do
	$cmdt1 @test=fork5/sleep_$i sleep $( echo "$testSleepTime * ($i % 3 + 1) ^ 2" | bc )
done
>&2 echo ">> launched $testCount tests"
# Kill Daemon
>&2 echo ">> Killing daemon ..."
pkill -f "cmdt ._daemon" || true

>&2 echo ">> reporting fork5 ..."
$cmdt @test="report_fork5" @stderr:"Successfully ran" @stderr:"$testCount success" @-- $cmdt1 @report=fork5

$cmdt @report @all

