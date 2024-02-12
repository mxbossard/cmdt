#! /bin/bash
set -e -o pipefail
scriptDir=$( dirname $( readlink -f $0 ) )

workspaceDir="/tmp/cmdtWorkspace"

>&2 echo "##### Building cmdtest binary ..."
export GOBIN="$scriptDir/bin"
cd "$scriptDir"
go install
cd - > /dev/null

rm -rf -- "$workspaceDir"

cmd="TO_REPLACE"
cmdt="$GOBIN/cmdtest"
ls -lh "$cmdt"

cmdt0="$cmdt @silent"
cmdt1="$cmdt"

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

>&2 echo "## Test cmdt basic assertions should passed"
$cmdt0 @init=should_succeed @stopOnFailure=false

$cmdt0 @test=should_succeed/ true
$cmdt0 @test=should_succeed/ true @success
$cmdt0 @test=should_succeed/ false @fail
$cmdt0 @test=should_succeed/ true @exit=0
$cmdt0 @test=should_succeed/ false @exit=1

$cmdt0 @test=should_succeed/ echo foo bar @stdout:foo @stderr=
$cmdt0 @test=should_succeed/ echo foo bar @stdout:bar
$cmdt0 @test=should_succeed/ echo foo bar @stdout!:baz
$cmdt0 @test=should_succeed/ echo foo bar @stdout!=baz
$cmdt0 @test=should_succeed/ echo foo bar @stdout~/^foo/ @stderr=
$cmdt0 @test=should_succeed/ echo foo bar @stdout~/BaR/i
$cmdt0 @test=should_succeed/ echo foo bar @stdout~"/^foo bar\n$/"
$cmdt0 @test=should_succeed/ echo foo bar @stdout~"/^foo bar$/m"
$cmdt0 @test=should_succeed/ echo foo bar @stdout!~/bar$/
$cmdt0 @test=should_succeed/ echo foo\nbar\nbaz @stdout!~/^bar$/
$cmdt0 @test=should_succeed/ echo foo\nbar\nbaz @stdout!~/^bar$/m
$cmdt0 @test=should_succeed/ echo foo bar @stdout:foo @stdout:bar @stderr=
$cmdt0 @test=should_succeed/ echo foo bar @stdout="foo bar\n" @stderr=

$cmdt0 @test=should_succeed/ sh -c ">&2 echo foo bar" @stderr:foo @stdout=
$cmdt0 @test=should_succeed/ sh -c ">&2 echo foo bar" @stderr:bar
$cmdt0 @test=should_succeed/ sh -c ">&2 echo foo bar" @stderr!:baz
$cmdt0 @test=should_succeed/ sh -c ">&2 echo foo bar" @stderr!=baz
$cmdt0 @test=should_succeed/ sh -c ">&2 echo foo bar" @stderr~/^foo/ @stdout=
$cmdt0 @test=should_succeed/ sh -c ">&2 echo foo bar" @stderr~/BaR/i
$cmdt0 @test=should_succeed/ sh -c ">&2 echo foo bar" @stderr~"/^foo bar\n$/"
$cmdt0 @test=should_succeed/ sh -c ">&2 echo foo bar" @stderr!~/bar$/
$cmdt0 @test=should_succeed/ sh -c ">&2 echo foo bar" @stderr:foo @stderr:bar
$cmdt0 @test=should_succeed/ sh -c ">&2 echo foo bar" @stderr="foo bar\n" @stdout=

>&2 echo "## Test cmdt basic assertions should failed"
$cmdt0 @init=should_fail @stopOnFailure=false

$cmdt0 @test=should_fail/ false
$cmdt0 @test=should_fail/ true @fail
$cmdt0 @test=should_fail/ false @success
$cmdt0 @test=should_fail/ true @exit=1
$cmdt0 @test=should_fail/ false @exit=0

$cmdt0 @test=should_fail/ echo foo bar @stdout=
$cmdt0 @test=should_fail/ echo foo bar @stdout=foo
$cmdt0 @test=should_fail/ echo foo bar @stdout=foo bar
$cmdt0 @test=should_fail/ echo foo bar @stdout:baz
$cmdt0 @test=should_fail/ echo foo bar @stdout:foo @stdout:baz
$cmdt0 @test=should_fail/ echo foo bar @stderr:foo

$cmdt0 @test=should_fail/ sh -c ">&2 echo foo bar" @stderr=
$cmdt0 @test=should_fail/ sh -c ">&2 echo foo bar" @stderr=foo
$cmdt0 @test=should_fail/ sh -c ">&2 echo foo bar" @stderr=foo bar
$cmdt0 @test=should_fail/ sh -c ">&2 echo foo bar" @stderr:baz
$cmdt0 @test=should_fail/ sh -c ">&2 echo foo bar" @stderr:foo @stderr:baz
$cmdt0 @test=should_fail/ sh -c ">&2 echo foo bar" @stdout:foo

$cmdt0 @report=should_succeed
! $cmdt0 @report=should_fail 2>&1 | grep "0 success" || die "should_fail test suite should have no success"


>&2 echo "## Rules parsing stopper --"
$cmdt0 @stdout="foo @success @fail\n" -- echo foo @success @fail


>&2 echo "## Test @report without test"
expectedNothingToReportStderr="you must perform some test prior to report"
$cmdt0 @init=meta1
$cmdt0 @test=meta1/ @fail @stderr:"$expectedNothingToReportStderr" -- $cmdt @report=foo
$cmdt0 @test=meta1/ @fail @stderr:"$expectedNothingToReportStderr" -- $cmdt @report=foo

>&2 echo "## Met1a test context not shared without token"
$cmdt0 @test=meta1/ @stderr:"PASSED" @stderr:"#01" -- $cmdt true
$cmdt0 @test=meta1/ @stderr:"PASSED" @stderr:"#01" -- $cmdt true
$cmdt0 @test=meta1/ @fail -- $cmdt @report

>&2 echo "## Test printed token"
tk0=$( $cmdt @init @printToken )
>&2 echo "token: $tk0"
$cmdt0 @init=meta2
$cmdt0 @test=meta2/ @stderr:"PASSED" @stderr:"#01" -- $cmdt true @token=$tk0
$cmdt0 @test=meta2/ @stderr:"PASSED" @stderr:"#02" -- $cmdt true @token=$tk0
$cmdt0 @test=meta2/ @fail -- $cmdt @report
$cmdt0 @test=meta2/ @stderr:"Successfuly ran" -- $cmdt @report @token=$tk0
$cmdt0 @report 2>&1 | grep -v "Failures"

>&2 echo "## Test exported token"
eval $( $cmdt @init @exportToken )
>&2 echo "token: $__CMDT_TOKEN"
$cmdt0 @init=meta3
$cmdt0 @test=meta3/ @stderr:"PASSED" @stderr:"#01" -- $cmdt true
$cmdt0 @test=meta3/ @stderr:"PASSED" @stderr:"#02" -- $cmdt true
$cmdt0 @test=meta3/ @stderr:"Successfuly ran" -- $cmdt @report=main
$cmdt0 @test=meta3/ @fail @stderr:"$expectedNothingToReportStderr" -- $cmdt @report @token=$tk0

$cmdt0 @init=meta4
$cmdt0 @test=meta4/ @stderr:"PASSED" @stderr:"#01" -- $cmdt @test=sub4/ true
$cmdt0 @test=meta4/ @stderr:"PASSED" @stderr:"#02" -- $cmdt @test=sub4/ true
$cmdt0 @test=meta4/ @stderr:"Successfuly ran" -- $cmdt @report=sub4
$cmdt0 @test=meta4/ @fail @stderr:"$expectedNothingToReportStderr" -- $cmdt @report=sub4 @token=$tk0
$cmdt0 @report 2>&1 | grep -v "Failures"
export -n __CMDT_TOKEN

>&2 echo "## Test usage"
$cmdt0 @init=meta
$cmdt0 @test=meta/ @fail @stderr:"usage:" -- $cmdt

>&2 echo "## Test cmdt basic assertions"
$cmdt0 @test=meta/ @stderr:"PASSED" -- $cmdt true
$cmdt0 @test=meta/ @stderr:"FAILED" -- $cmdt false
$cmdt0 @test=meta/ @stderr:"PASSED" -- $cmdt true @success
$cmdt0 @test=meta/ @stderr:"FAILED" -- $cmdt false @success
$cmdt0 @test=meta/ @stderr:"FAILED" -- $cmdt true @fail
$cmdt0 @test=meta/ @stderr:"PASSED" -- $cmdt false @fail
$cmdt0 @test=meta/ @stderr:"PASSED" -- $cmdt true @exit=0
$cmdt0 @test=meta/ @stderr:"FAILED" -- $cmdt false @exit=0
$cmdt0 @test=meta/ @stderr:"FAILED" -- $cmdt true @exit=1
$cmdt0 @test=meta/ @stderr:"PASSED" -- $cmdt false @exit=1

$cmdt0 @test=meta/ @stderr:"PASSED" -- $cmdt echo foo bar @stdout:foo @stderr=
$cmdt0 @test=meta/ @stderr:"FAILED" -- $cmdt echo foo bar @stdout:baz @stderr=
$cmdt0 @test=meta/ @stderr:"PASSED" -- $cmdt echo foo bar @stdout:foo @stdout:bar @stderr=
$cmdt0 @test=meta/ @stderr:"FAILED" -- $cmdt echo foo bar @stdout:baz @stdout:bar @stderr=
$cmdt0 @test=meta/ @stderr:"FAILED" -- $cmdt echo foo bar @stdout:foo @stdout:baz @stderr=
$cmdt0 @test=meta/ @stderr:"FAILED" -- $cmdt echo foo bar @stdout!:foo
$cmdt0 @test=meta/ @stderr:"PASSED" -- $cmdt echo foo bar @stdout!=foo @stdout:bar @stdout!:baz

$cmdt0 @test=meta/ @stderr:"PASSED" -- $cmdt sh -c ">&2 echo foo bar" @stderr:foo @stdout=
$cmdt0 @test=meta/ @stderr:"FAILED" -- $cmdt sh -c ">&2 echo foo bar" @stderr:baz @stdout=
$cmdt0 @test=meta/ @stderr:"PASSED" -- $cmdt sh -c ">&2 echo foo bar" @stderr:foo @stderr:bar @stdout=
$cmdt0 @test=meta/ @stderr:"FAILED" -- $cmdt sh -c ">&2 echo foo bar" @stderr:baz @stderr:bar @stdout=
$cmdt0 @test=meta/ @stderr:"FAILED" -- $cmdt sh -c ">&2 echo foo bar" @stderr:foo @stderr:baz @stdout=
$cmdt0 @test=meta/ @stderr:"FAILED" -- $cmdt sh -c ">&2 echo foo bar" @stderr!:foo
$cmdt0 @test=meta/ @stderr:"PASSED" -- $cmdt sh -c ">&2 echo foo bar" @stderr!=foo @stderr:bar @stderr!:baz


## Forge a token for remaining tests
eval $( $cmdt @test=meta/ @keepStdout -- $cmdt @init=t1 @exportToken)

>&2 echo "## Test assertions outputs"
# Init the context used in t1 test suite
$cmdt0 @init=meta
$cmdt0 @test=meta/ @stderr:"#01..." @stderr:"PASSED" -- $cmdt true @test=t1/
$cmdt0 @test=meta/ @stderr:"#02..." @stderr:"PASSED" -- $cmdt true @test=t1/
$cmdt0 @test=meta/ @stderr:"#03..." @stderr:"FAILED" -- $cmdt false @test=t1/
$cmdt0 @test=meta/ @stderr:"#04..." @stderr:"PASSED" -- $cmdt false @fail @test=t1/
$cmdt0 @test=meta/ @fail @stderr:"Failures in [t1] test suite (3 success, 1 failures, 4 tests in" -- $cmdt @report=t1


>&2 echo "## Test namings"
$cmdt0 @init=naming
$cmdt0 @test=naming/ @stderr:"Test [main]/name1 #01..." @stderr:"PASSED" -- $cmdt true @test=name1
$cmdt0 @test=naming/ @stderr:"Test [main]/name2 #02..." @stderr:"PASSED" -- $cmdt true @test=name2
$cmdt0 @test=naming/ @stderr:"Test [main]/" @stderr:"true" @stderr:"#03..." @stderr:"PASSED" -- $cmdt true
$cmdt0 @test=naming/ @stderr:"Test [main]/" @stderr:"true" @stderr:"#04..." @stderr:"PASSED" -- $cmdt true
$cmdt0 @test=naming/ @stderr:"Test [suite1]/name1 #01..." @stderr:"PASSED" -- $cmdt true @test=suite1/name1
$cmdt0 @test=naming/ @stderr:"Test [suite1]/name2 #02..." @stderr:"PASSED" -- $cmdt true @test=suite1/name2
$cmdt0 @test=naming/ @stderr:"Test [suite2]/" @stderr:"#01..." @stderr:"PASSED" -- $cmdt true @test=suite2/
$cmdt0 @test=naming/ @stderr:"Test [suite2]/" @stderr:"#02..." @stderr:"PASSED" -- $cmdt true @test=suite2/
$cmdt0 @test=naming/ @stderr:"Successfuly ran [suite1] test suite" -- $cmdt  @report=suite1
$cmdt0 @test=naming/ @stderr:"Successfuly ran [suite2] test suite" -- $cmdt @report=suite2
$cmdt0 @test=naming/ @stderr:"Successfuly ran [main] test suite" -- $cmdt @report=main


>&2 echo "## Test rules missusage"
$cmdt0 @init=failing_rule_missusage
$cmdt0 @test=failing_rule_missusage/ @fail -- $cmdt @init true
$cmdt0 @test=failing_rule_missusage/ @fail -- $cmdt @init @test
$cmdt0 @test=failing_rule_missusage/ @fail -- $cmdt @init @report
$cmdt0 @test=failing_rule_missusage/ @fail -- $cmdt @test @report true
$cmdt0 @test=failing_rule_missusage/ @fail -- $cmdt @init @test true
$cmdt0 @test=failing_rule_missusage/ @fail -- $cmdt @init @test true
$cmdt0 @test=failing_rule_missusage/ @fail -- $cmdt @init @fail
$cmdt0 @test=failing_rule_missusage/ @fail -- $cmdt @init @success
$cmdt0 @test=failing_rule_missusage/ @fail -- $cmdt @init @exit=0
$cmdt0 @test=failing_rule_missusage/ @fail -- $cmdt true @fail @success
$cmdt0 @test=failing_rule_missusage/ @fail -- $cmdt true @fail @exit=0
$cmdt0 @test=failing_rule_missusage/ @fail -- $cmdt true @success @exit=0
$cmdt0 @test=failing_rule_missusage/ @fail -- $cmdt @report @fail
$cmdt0 @test=failing_rule_missusage/ @fail -- $cmdt @report @success
$cmdt0 @test=failing_rule_missusage/ @fail -- $cmdt @report @exit=0
$cmdt0 @test=failing_rule_missusage/ @fail @stderr:donotexist -- $cmdt true @donotexist
$cmdt0 @test=failing_rule_missusage/ @fail @stderr:donotexist -- $cmdt true @donotexist


>&2 echo "## Test config"
$cmdt0 @init=test_config

$cmdt0 @test=test_config/ @stderr:Ignore @stderr!:FAILED @stderr!:PASSED -- $cmdt1 true @ignore
$cmdt0 @test=test_config/ @stderr!:Ignore @stderr:PASSED -- $cmdt1 true
$cmdt0 @test=test_config/ @stderr:Ignore @stderr!:FAILED @stderr!:PASSED -- $cmdt1 true @ignore
$cmdt0 @test=test_config/ @stderr:Ignore @stderr!:FAILED @stderr!:PASSED -- $cmdt1 false @ignore
$cmdt0 @test=test_config/ @stderr!:Ignore @stderr:PASSED -- $cmdt1 true

$cmdt0 @test=test_config/ @stdout!:foo @stderr!:bar @stderr:PASSED -- $cmdt1 @test=test_keepouts sh -c "echo foo; >&2 echo bar"
$cmdt0 @test=test_config/ @stdout~/^foo$/m @stderr!:bar @stderr:PASSED -- $cmdt1 @test=test_keepouts sh -c "echo foo; >&2 echo bar" @keepStdout
$cmdt0 @test=test_config/ @stdout!:foo @stderr:bar @stderr:PASSED -- $cmdt1 @test=test_keepouts sh -c "echo foo; >&2 echo bar" @keepStderr
$cmdt0 @test=test_config/ @stdout~/^foo$/m @stderr:bar @stderr:PASSED -- $cmdt1 @test=test_keepouts sh -c "echo foo; >&2 echo bar" @keepOutputs

$cmdt0 @test=test_config/ @stderr:FAILED -- $cmdt1 sleep 0.01 @timeout=5ms
$cmdt0 @test=test_config/ @stderr:PASSED -- $cmdt1 sleep 0.01 @timeout=30ms

$cmdt0 @test=test_config/ @stderr:FAILED @success -- $cmdt1 false
$cmdt0 @test=test_config/ @stderr:FAILED @fail -- $cmdt1 false @stopOnFailure
$cmdt0 @test=test_config/ @stderr:FAILED @success -- $cmdt1 false

$cmdt0 @test=test_config/ @stderr:FAILED -- $cmdt1 @prefix=% %fail true
$cmdt0 @test=test_config/ @stderr:PASSED -- $cmdt1 @prefix=% %fail false
$cmdt0 @test=test_config/ @stderr:PASSED -- $cmdt1 @prefix=% %success true

$cmdt0 @test=test_config/ @fail -- $cmdt1 @report=main 


>&2 echo "## Test suite config"
$cmdt0 @init=suite_config
$cmdt0 @init=suite_config_silent @silent
$cmdt0 @test=suite_config/ @stdout= @stderr= -- $cmdt1 @test=suite_config_silent/ echo foo
$cmdt0 @test=suite_config/ @stderr:PASSED -- $cmdt1 @test=suite_config_silent/ echo foo @silent=false
$cmdt0 @test=suite_config/ @stdout="foo\n" @stderr= -- $cmdt1 @test=suite_config_silent/ echo foo @keepOutputs
$cmdt0 @test=suite_config/ @stdout= @stderr="bar\n" -- $cmdt1 @test=suite_config_silent/ sh -c ">&2 echo bar" @keepOutputs
$cmdt0 @test=suite_config/ -- $cmdt1 @report=suite_config_silent

>&2 echo "## Test global config"


>&2 echo "## Test assertions"
$cmdt0 @init=assertion
$cmdt0 @test=assertion/ @stderr:PASSED -- $cmdt true
$cmdt0 @test=assertion/ @stderr:PASSED -- $cmdt true @success
$cmdt0 @test=assertion/ @stderr:FAILED -- $cmdt true @fail

$cmdt0 @test=assertion/ @stderr:FAILED -- $cmdt false
$cmdt0 @test=assertion/ @stderr:FAILED -- $cmdt false @success
$cmdt0 @test=assertion/ @stderr:PASSED -- $cmdt false @fail

$cmdt0 @test=assertion/ @stderr:PASSED -- $cmdt true @exit=0
$cmdt0 @test=assertion/ @stderr:FAILED -- $cmdt true @exit=1

$cmdt0 @test=assertion/ @stderr:FAILED -- $cmdt false @exit=0
$cmdt0 @test=assertion/ @stderr:PASSED -- $cmdt false @exit=1

$cmdt0 @test=assertion/ @stderr:PASSED -- $cmdt echo foo bar @stdout="foo bar\n"
$cmdt0 @test=assertion/ @stderr:PASSED -- $cmdt echo foo bar @stdout:foo
$cmdt0 @test=assertion/ @stderr:PASSED -- $cmdt echo foo bar @stdout:bar
$cmdt0 @test=assertion/ @stderr:PASSED -- $cmdt echo foo bar @stderr=
$cmdt0 @test=assertion/ @stderr:FAILED -- $cmdt echo foo bar @stdout="foo bar"
$cmdt0 @test=assertion/ @stderr:FAILED -- $cmdt echo foo bar @stdout=
$cmdt0 @test=assertion/ @fail -- $cmdt echo foo bar @stdout:
$cmdt0 @test=assertion/ @stderr:FAILED -- $cmdt echo foo bar @stdout:baz
$cmdt0 @test=assertion/ @fail -- $cmdt echo foo bar @stderr:

$cmdt0 @test=assertion/ @stderr:PASSED -- $cmdt sh -c ">&2 echo foo bar" @stderr="foo bar\n"
$cmdt0 @test=assertion/ @stderr:PASSED -- $cmdt sh -c ">&2 echo foo bar" @stderr:foo
$cmdt0 @test=assertion/ @stderr:PASSED -- $cmdt sh -c ">&2 echo foo bar" @stderr:bar
$cmdt0 @test=assertion/ @stderr:PASSED -- $cmdt sh -c ">&2 echo foo bar" @stdout=
$cmdt0 @test=assertion/ @stderr:FAILED -- $cmdt sh -c ">&2 echo foo bar" @stderr="foo bar"
$cmdt0 @test=assertion/ @stderr:FAILED -- $cmdt sh -c ">&2 echo foo bar" @stderr=
$cmdt0 @test=assertion/ @fail -- $cmdt sh -c ">&2 echo foo bar" @stderr:
$cmdt0 @test=assertion/ @stderr:FAILED -- $cmdt sh -c ">&2 echo foo bar" @stderr:baz
$cmdt0 @test=assertion/ @fail -- $cmdt sh -c ">&2 echo foo bar" @stdout:

$cmdt0 @test=assertion/ @stderr:FAILED -- $cmdt sh -c "rm /tmp/donotexists || true" @exists=/tmp/donotexists
$cmdt0 @test=assertion/ @stderr:PASSED -- $cmdt sh -c "touch /tmp/doexists" @exists=/tmp/doexists
$cmdt0 @test=assertion/ @stderr:PASSED -- $cmdt sh -c "chmod 640 /tmp/doexists" @exists=/tmp/doexists,-rw-r-----
$cmdt0 @test=assertion/ @stderr:FAILED -- $cmdt sh -c "chmod 640 /tmp/doexists" @exists=/tmp/doexists,-rwxr-----

$cmdt0 @test=assertion/ @stderr:PASSED -- $cmdt false @fail @cmd=true
$cmdt0 @test=assertion/ @stderr:FAILED -- $cmdt true @cmd=false
touch /tmp/doexists
$cmdt0 @test=assertion/ @stderr:PASSED -- $cmdt false @fail @cmd="ls /tmp/doexists"

$cmdt0 @test=assertion/ @fail -- $cmdt @report=main 


>&2 echo "## Test stdin"
echo foo | $cmdt0 @test=stdin/ @stdout="foo\n" cat


>&2 echo "## Test export"
export foo=bar
$cmdt0 @test=export/ @stdout~"/foo='bar'/m" sh -c "export"


>&2 echo "## Interlaced tests"
eval $( $cmdt0 @init="testA" @keepStdout )
eval $( $cmdt0 @init="testB" @keepOutputs )

$cmdt0 echo not ignored @test="testA/"
$cmdt0 echo ignored @ignore @test="testA/"

$cmdt0 echo another test @test="testB/"
$cmdt0 echo interlaced test @test="testA/"

>&2 echo should have 2 success and 1 ignored
$cmd0 @stderr:"2 success" @stderr:"1 ignored" @stderr!:failure -- $cmdt1 @report="testA"

>&2 echo should have 1 success
$cmd0 @stderr:"1 success", @sdterr!:"ignored" @stderr!:failure -- $cmdt1 @report="testB"


>&2 echo "## Mutually exclusive rules"
# TODO


$cmdt @report= ; >&2 echo SUCCESS ; exit 0




>&2 echo "## Mutually exclusive rules"
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

