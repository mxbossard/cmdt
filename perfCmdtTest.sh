#! /bin/bash
set -e -o pipefail
scriptDir=$( dirname $( readlink -f $0 ) )

workspaceDir="/tmp/cmdtWorkspace"

>&2 echo "##### Building cmdtest binary ..."
export GOBIN="$scriptDir/bin"
cd "$scriptDir"
#go install
#CGO_ENABLED=0 GOOS=linux go install -a -ldflags '-extldflags "-static"'
#CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go install -ldflags '-extldflags "-static"'
#CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go install -a -tags netgo -ldflags '-w'
#CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go install -a -tags netgo -ldflags '-w -extldflags "-static"'

CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go install -tags netgo -ldflags '-w'

cd - > /dev/null

rm -rf -- "$workspaceDir"

cmd="TO_REPLACE"
newCmdt="$GOBIN/cmdtest"
ls -lh "$newCmdt"

# Trusted cmdt to works
cmdt="cmdt"
#cmdt="$newCmdt"

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

>&2 echo "## Test cmdt basic assertions should passed"
$cmdt1 @init=perf @verbose=4 @debug=5

$cmdt1 @test=perf/echo_foo echo foo

$cmdt1 @test=perf/sleep_1 sleep 1

$cmdt1 @report

