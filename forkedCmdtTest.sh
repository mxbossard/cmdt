#! /bin/bash
set -e -o pipefail
scriptDir=$( dirname $( readlink -f $0 ) )

# Use last cmdt bin built
export CMDT_BIN="$scriptDir/bin/cmdt"
export CMDT_FORK_CFG="@fork=2"

. $scriptDir/cmdtTest.sh

