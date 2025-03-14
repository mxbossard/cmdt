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
"$scriptDir/assertCmdt.sh" "$cmdt"


# Clear context
export -n __CMDT_TOKEN

count=4

>&2 echo "## Visual test in sync"
$newCmdt1 @init=sync_visual @verbose=5 @suiteTimeout=$((count/2+2))s
>&2 echo "Launching tests ..."
for i in $( seq 1 $count ); do
	>&2 echo -n "$i "
	time=$( echo "scale=2;$i/20 + 0.1" | bc )
	$newCmdt1 @test=sync_visual/t$i @stdout:"end$i" @-- sh -c "sleep $time; echo end$i"
done
>&2 echo "done tests"
$newCmdt1 @report=sync_visual
>&2 echo "done report"

rm -rf -- /tmp/cmdt* /tmp/cmdt*.log /tmp/daemon*.log 2> /dev/null || true

>&2 echo
>&2 echo "## Visual test async"
$newCmdt1 @init=async_visual @async @verbose=5 @suiteTimeout=$((count/2+2))s
>&2 echo "Launching tests ..."
for i in $( seq 1 $count ); do
	>&2 echo -n "$i "
	time=$( echo "scale=2;$i/20 + 0.1" | bc )
	$newCmdt1 @test=async_visual/t$i @stdout:"end$i" @-- sh -c "sleep $time; echo end$i"
done
>&2 echo "done tests"
$newCmdt1 @report=async_visual
>&2 echo "done report"

>&2 echo
>&2 echo "## Visual test async report all"
$newCmdt1 @init=async_reportall_visual @async @verbose=5 @suiteTimeout=$((count/2+2))s
>&2 echo "Launching tests ..."
for i in $( seq 1 $count ); do
	>&2 echo -n "$i "
	time=$( echo "scale=2;$i/20 + 0.1" | bc )
	$newCmdt1 @test=async_reportall_visual/t$i @stdout:"end$i" @-- sh -c "sleep $time; echo end$i"
done
>&2 echo "done tests"
$newCmdt1 @report
>&2 echo "done report"

