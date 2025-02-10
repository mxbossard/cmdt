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

rm -rf -- /tmp/cmdt* /tmp/cmdt.log /tmp/daemon.log 2> /dev/null || true

# Mandatory assertions
"$scriptDir/assertCmdt.sh" "$cmdt"


# Clear context
export -n __CMDT_TOKEN

>&2 echo "## Visual test"
$newCmdt1 @init=longer_sync_visual @verbose=5 @suiteTimeout=7s
>&2 echo "Launching tests ..."
for i in $( seq 1 10 ); do
	>&2 echo -n "$i "
	time=$( echo "scale=1;$i/10" | bc )
	$newCmdt1 @test=longer_sync_visual/t$i @stdout:"end$i" @-- sh -c "sleep $time; echo end$i"
done
>&2 echo "done"
$newCmdt1 @report=longer_sync_visual

$newCmdt1 @init=longer_async_visual @async @verbose=5 @suiteTimeout=7s
>&2 echo "Launching tests ..."
for i in $( seq 1 10 ); do
	>&2 echo -n "$i "
	time=$( echo "scale=1;$i/10" | bc )
	$newCmdt1 @test=longer_async_visual/t$i @stdout:"end$i" @-- sh -c "sleep $time; echo end$i"
done
>&2 echo "done"
$newCmdt1 @report=longer_async_visual

