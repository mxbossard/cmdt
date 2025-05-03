#! /bin/bash
set -e -o pipefail
scriptDir=$( dirname $( readlink -f $0 ) )

SCRIPTS=("$scriptDir/cmdtTest.sh" "$scriptDir/asyncCmdtTest.sh" "$scriptDir/perfCmdtTest.sh")
#SCRIPTS=("false")

sleepRandom() {
	min=$1
	max=$2

	time=$(( min + RANDOM % (max - min) ))
	>&2 echo
	>&2 echo -n "sleeping $time sec "
	for i in $( seq 1 $time ); do
		sleep 1
		>&2 echo -n "."
	done
	>&2 echo
}

n=1
runRandomScript() {
	k=$(( RANDOM % ${#SCRIPTS[@]} ))
	randomScript=${SCRIPTS[k]}
	>&2 echo
	>&2 echo "---------- Running script ($k): [$randomScript] (#$n started since $SECONDS sec) ... ----------"
	bash -e -c "$randomScript"
	rc=$?
	>&2 echo "> RC=$rc"
	n=$(( n + 1 ))
	return $rc
}

SECONDS=0
while runRandomScript; do
	sleepRandom 5 10	
done

