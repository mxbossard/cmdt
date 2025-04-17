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
"$scriptDir/checkCmdt.sh" "$cmdt"


# Clear context
export -n __CMDT_TOKEN

count=4

>&2 echo "## Visual test in sync"
$newCmdt1 @init=sync_visual @verbose=5 @suiteTimeout=$((count/2+2))s
>&2 echo "Launching tests ..."
for i in $( seq 1 $count ); do
	>&2 echo -n "$i "
	time=$( echo "scale=1;$i/20 + 0.11" | bc )
	$newCmdt1 @test=sync_visual/tA$i @stdout:"endA$i" @-- sh -c "sleep $time; echo endA$i"
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
	time=$( echo "scale=1;$i/20 + 0.12" | bc )
	$newCmdt1 @test=async_visual/tB$i @stdout:"endB$i" @-- sh -c "sleep $time; echo endB$i"
done
>&2 echo "done tests"
$newCmdt1 @report=async_visual
>&2 echo "done report"

>&2 echo
>&2 echo "## Visual test in sync global report"
$newCmdt1 @init=sync_global_report_visual @verbose=5 @suiteTimeout=$((count/2+2))s
>&2 echo "Launching tests ..."
for i in $( seq 1 $count ); do
	>&2 echo -n "$i "
	time=$( echo "scale=1;$i/20 + 0.13" | bc )
	$newCmdt1 @test=sync_global_report_visual/tC$i @stdout:"endC$i" @-- sh -c "sleep $time; echo endC$i"
done
>&2 echo "done tests"
$newCmdt1 @report
>&2 echo "done report"

#rm -rf -- /tmp/cmdt*.log 2> /dev/null || true
>&2 echo
>&2 echo "## Visual test async global report"
$newCmdt1 @init=async_global_report_visual @async @verbose=5 @suiteTimeout=$((count/2+2))s
>&2 echo "Launching tests ..."
for i in $( seq 1 $count ); do
	>&2 echo -n "$i "
	time=$( echo "scale=1;$i/20 + 0.14" | bc )
	$newCmdt1 @test=async_global_report_visual/tD$i @stdout:"endD$i" @-- sh -c "sleep $time; echo endD$i"
done
>&2 echo "done tests"
$newCmdt1 @report
>&2 echo "done report"

>&2 echo
>&2 echo "## Visual test async & sync global report"
$newCmdt1 @init=async_global_report_async_and_sync_visual @async @verbose=5 @suiteTimeout=$((count/2+2))s
>&2 echo "Launching async tests ..."
for i in $( seq 1 $count ); do
	>&2 echo -n "$i "
	time=$( echo "scale=1;$i/20 + 0.15" | bc )
	$newCmdt1 @test=async_global_report_async_and_sync_visual/tE$i @stdout:"endD$i" @-- sh -c "sleep $time; echo endD$i"
done
>&2 echo
>&2 echo "done async tests"
$newCmdt1 @init=sync_global_report_async_and_sync_visual @async=false @verbose=5 @suiteTimeout=$((count/2+2))s
>&2 echo "Launching sync tests ..."
for i in $( seq 1 $count ); do
	>&2 echo -n "$i "
	time=$( echo "scale=1;$i/20 + 0.16" | bc )
	$newCmdt1 @test=sync_global_report_async_and_sync_visual/tF$i @stdout:"endD$i" @-- sh -c "sleep $time; echo endD$i"
done
>&2 echo
>&2 echo "done sync tests"
$newCmdt1 @report
>&2 echo "done report"

