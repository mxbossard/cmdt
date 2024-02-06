#! /bin/bash
set -e
scriptDir=$( dirname $( readlink -f $0 ) )

workspaceDir="/tmp/cmdtWorkspace"

>&2 echo "##### Building cmdtest binary ..."
export GOBIN="$scriptDir/bin"
cd "$scriptDir"
go install

rm -rf -- "$workspaceDir"

cmd="$GOBIN/cmdtest"
ls -lh "$cmd"

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

cd "$workspace"

>&2 echo "## Test usage"
! $cmd || false

>&2 echo "## Test @report without test"
! $cmd @report || false
! $cmd @report || false

>&2 echo "## Test whithout @init"
$cmd "@test=without @init ok" true @success
$cmd "@test=without @init ko" false @fail
$cmd "@test=test suite/without @init ok" true
$cmd @report
! $cmd @report || true
$cmd @report="test suite"

>&2 $cmd @init
>&2 echo "## Test suite @init"
#eval $( $cmd @init @ignore )
eval $( $cmd @init @stopOnFailure=false)
eval $( $cmd @init @stopOnFailure=false)

>&2 echo "## Test without name"
$cmd true @success
! $cmd "" true @success @ignore=false || false
! $cmd "" true @success @ignore=false || false

>&2 echo "## Test not exising assertion"
! $cmd "@test=not exising assertion" true @foo || false

>&2 echo "## Test without assertion"
$cmd "@test=without assertion" true

>&2 echo "## Test suite @success"
$cmd "@test=succes ok" true @success
! $cmd "@test=success ko" false @success || false

>&2 echo "## Test suite @fail"
$cmd "@test=fail ok" false @fail
! $cmd "@test=fail ko" true @fail || false

>&2 echo "## Test suite @exit="
$cmd "@test=rc=0 ok" true @exit=0
! $cmd "@test=rc=0 ko" false @exit=0 || false

$cmd "@test=rc=1 ok" false @exit=1
! $cmd "@test=rc=1 ko" true @exit=1 || false

>&2 echo "## Test sleep"
$cmd sleep 0.3

>&2 echo "## Test suite @stdout="
$cmd "@test=empty stdout ok" true @stdout=
! $cmd "@test=empty stdout ko" echo "bar" @stdout= || false

$cmd "@test=stdout= ok" echo foo "bar baz" @stdout="foo bar baz\n"
! $cmd "@test=stdout= ko" echo foo "bar baz" @stdout="foo barbaz" || false

>&2 echo "## Test suite @stdout~"
$cmd "@test=stdout~ ok" echo foo "bar baz" @stdout~"bar"
! $cmd "@test=stdout~ ko" echo foo "bar baz" @stdout~"pif" || false

>&2 echo "## Test suite @stderr="
$cmd "@test=empty stderr" true @stderr=
! $cmd "@test=empty stderr" notExist @stderr= || false

$cmd "@test=stderr= ok" sh -c ">&2 echo foo" @stderr="foo\n"
! $cmd "@test=stderr= ko" sh foo @stderr="foo" || false

>&2 echo "## Test suite @stderr~"
$cmd "@test=stderr~ ok" ls /notExist @stderr~"otExi"
! $cmd "@test=stderr~ ko" ls /notExist @stderr~"foo" || false

>&2 echo "## Test stdin"
echo foo | $cmd "@test=stdin" cat @stdout="foo\n"

>&2 echo "## Test exported var"
export foo=bar
$cmd "@test=exported var" sh -c "export" @stdout~"foo='bar'\n"

#>&2 echo "## Test alias"
#alias foo=echo
#$cmd "@test=alias" sh -c "foo bar" @stdout~"bar\n"

>&2 echo "## Test @timeout"
! $cmd sleep 1 @timeout=50ms || false


>&2 echo "## Test @cmd"
$cmd false @cmd="true"
! $cmd true @cmd="false" || false
! $cmd true @cmd= || false
! $cmd true @cmd=notExist || false

>&2 echo "## Test @exists"
! $cmd echo "test non existance" @exists=/tmp/donotexistsmbd123 || false
$cmd touch /tmp/existsmbd123 @exists=/tmp/existsmbd123
! $cmd true @exists= || false

>&2 echo "## Test @report"
$cmd @report

# Should not report a second time
! $cmd @report || false

>&2 echo "## Interlaced tests"
eval $( $cmd @init="another" @keepStdout )
eval $( $cmd @init="another one" @keepOutputs )

$cmd echo not ignored @test="another/"
$cmd echo ignored @ignore @test="another/"

$cmd echo another test @test="another one/"
$cmd echo interlaced test @test="another/"

>&2 echo should have 2 success and 1 ignored
$cmd @report="another"

>&2 echo should have 1 success
$cmd @report="another one"

>&2 echo "## Mutually exclusive rules @init"
! $cmd @init @fail || false
! $cmd @init @test || false

>&2 echo "## Mutually exclusive rules @test"
! $cmd true @fail @success || false
! $cmd true @fail @success || false
! $cmd true @fail @exit=0 || false
! $cmd true @fail @suiteTimeout=1m || false

>&2 echo "## Mutually exclusive rules @report"
! $cmd @report @ignore || false
! $cmd @report @test || false
! $cmd @report @fail || false

>&2 echo SUCCESS
