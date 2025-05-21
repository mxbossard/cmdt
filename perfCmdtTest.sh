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
$cmdt @global @suiteTimeout=10s

# Clear context
export -n __CMDT_TOKEN

$cmdt1 @init=perf @verbose=4

$cmdt1 @test=perf/echo_foo echo foo

$cmdt1 @test=perf/sleep_0.1 sleep 0.1

$cmdt1 @report


testSleepTime="0.02"

>&2 echo "## test @async=false @fork=1"
$cmdt1 @init=fork1_sync @verbose=3 @async=false @fork=1

for i in $( seq 0 29 ); do
	$cmdt1 @test=fork1_sync/sleep_$i sleep $testSleepTime
done
$cmdt1 @report

>&2 echo "## test @async @fork=1"
$cmdt1 @init=fork1_async @verbose=3 @async @fork=1

for i in $( seq 0 29 ); do
	$cmdt1 @test=fork1_async/sleep_$i sleep $testSleepTime
done
$cmdt1 @report

>&2 echo "## test @fork=2"
$cmdt1 @init=fork2 @verbose=3 @fork=2

for i in $( seq 0 29 ); do
	$cmdt1 @test=fork2/sleep_$i sleep $testSleepTime
done
$cmdt1 @report

>&2 echo "## test @fork=5"
$cmdt1 @init=fork5 @verbose=3 @fork=5

#$cmdt1 @test=fork/echo_foo echo foo
for i in $( seq 0 29 ); do
	$cmdt1 @test=fork5/sleep_$i sleep $testSleepTime
done
$cmdt1 @report

>&2 echo "## test @fork=20"
$cmdt1 @init=fork20 @verbose=3 @fork=20

for i in $( seq 0 29 ); do
	$cmdt1 @test=fork20/sleep_$i sleep $testSleepTime
done
$cmdt1 @report

$cmdt1 @report @all

testSleepTime="0.1"

for p in $( seq 1 $COUNT ); do
	>&2 echo
	>&2 echo "---------------------"
	>&2 echo "Stress test running 19 suites of 30 tests sleeping $testSleepTime sec #$p ..."
	for k in $( seq 2 20 ); do
		>&2 echo -n "## seq test @fork=$k "
		start=$( date +%s%3N )
		$cmdt1 @init=seq_fork$k @fork=$k @verbose=1 2>&1 | tr '\n' ' '

		for i in $( seq 0 29 ); do
			$cmdt1 @test=seq_fork$k/sleep_$i sleep $testSleepTime
		done
		end=$( date +%s%3N )
		>&2 echo " last $(( end - start )) ms"
		#$cmdt1 @report
	done
	$cmdt1 @report 
	$cmdt1 @report @all
done
