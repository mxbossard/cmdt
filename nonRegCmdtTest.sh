#! /bin/bash
set -e -o pipefail
scriptDir=$( dirname $( readlink -f $0 ) )

SCRIPTS=("$scriptDir/cmdtTest.sh" "$scriptDir/asyncCmdtTest.sh" "$scriptDir/perfCmdtTest.sh" "$scriptDir/daemonReliabilityTest.sh" "$scriptDir/visualCmdtTest.sh")

for script in ${SCRIPTS[@]}; do
	>&2 echo "-------------------- Running test script: $script ..."
	bash -e -c "$script"
	>&2 echo

	if pgrep -f -a "cmdt ._daemon" | grep ""; then
		>&2 echo "killing daemon ..."
		pkill -f "cmdt ._daemon" || true
		>&2 echo
	fi
done

>&2 echo
>&2 echo "Non regression test done with success."
