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
cmdt="$GOBIN/cmdtest"
ls -lh "$cmdt"

#cmdt0="$cmdt @silent"
cmdt0="$cmdt $@"

die() {
	>&2 echo "$1"
	exit 1
}

$cmdt0 @test=initless/passed true
$cmdt0 @test=initless/failed false


$cmdt0 @init=display

$cmdt0 @test=display/passed true
$cmdt0 @test=display/failed false
$cmdt0 @test=display/ignored true @ignore
$cmdt0 @test=display/timeouted sleep 1 @timeout=10ms
$cmdt0 @test=display/errored foo

$cmdt0 @test=display/ echo foo bar baz

$cmdt0 @test=display/compare_expect_not_empty_out true @stdout=foo
$cmdt0 @test=display/compare_expect_empty_out echo "foo bar" @stdout=
$cmdt0 @test=display/compare_bad_outs echo "foo bar" @stdout=foo @stderr=bar

$cmdt0 @report
