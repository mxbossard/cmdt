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
cmdt0="$newCmdt @isol=tested"
cmdt1="$newCmdt @isol=tested @verbose @failuresLimit=-1" # Default verbose show passed test + perform all test beyond failures limit

mkdir -p "$scriptDir/.tmp"
reportFile="$( mktemp "$scriptDir/.tmp/XXXXXX.log" )"
rm -- "$scriptDir/.tmp/"*.log || true

RED_COLOR="\e[41m\e[30m"
GREEN_COLOR="\e[42m\e[37m"
CYAN_COLOR="\e[46m\e[30m"
RESET_COLOR="\e[0m"

die() {
	>&2 echo "$1"
	exit 1
}

#cd "$GOBIN"
#test -e cmdt || ln -s cmduest cmdt
#export PATH="$PATH:."
#cmd="cmdt"

### NOTES
# - Il est facile de sortir un test de la bonne test suite, et ce test ne sera jamais report !
# => Should @report report all opened tests suites by default ?

#$cmdt @global @silent

# Clear context
export -n __CMDT_TOKEN
#$cmdt @init=main

$cmdt1 @init=perf @verbose=4

$cmdt1 @test=perf/echo_foo echo foo

$cmdt1 @test=perf/sleep_0.1 sleep 0.1

$cmdt1 @report


>&2 echo "## test @async=false @fork=1"
$cmdt1 @init=fork1_sync @verbose=3 @async=false @fork=1

for i in $( seq 0 29 ); do
	$cmdt1 @test=fork1_sync/sleep_0.1_$i sleep 0.01
done
$cmdt1 @report

>&2 echo "## test @async @fork=1"
$cmdt1 @init=fork1_async @verbose=3 @async @fork=1

for i in $( seq 0 29 ); do
	$cmdt1 @test=fork1_async/sleep_0.1_$i sleep 0.01
done
$cmdt1 @report

>&2 echo "## test @fork=2"
$cmdt1 @init=fork2 @verbose=3 @fork=2

for i in $( seq 0 29 ); do
	$cmdt1 @test=fork2/sleep_0.1_$i sleep 0.01
done
$cmdt1 @report

>&2 echo "## test @fork=5"
$cmdt1 @init=fork5 @verbose=3 @fork=5

#$cmdt1 @test=fork/echo_foo echo foo
for i in $( seq 0 29 ); do
	$cmdt1 @test=fork5/sleep_0.1_$i sleep 0.01
done
$cmdt1 @report

>&2 echo "## test @fork=20"
$cmdt1 @init=fork20 @verbose=3 @fork=20

for i in $( seq 0 29 ); do
	$cmdt1 @test=fork20/sleep_0.1_$i sleep 0.01
done
$cmdt1 @report

$cmdt1 @report @all
