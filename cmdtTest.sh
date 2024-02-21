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

cmdt0="$cmdt @silent"
#cmdt0="$cmdt"
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

# Clear context
export -n __CMDT_TOKEN
#$cmdt @init=main

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

$cmdt0 @test=should_fail/ false 2> /dev/null
$cmdt0 @test=should_fail/ true @fail 2> /dev/null
$cmdt0 @test=should_fail/ false @success 2> /dev/null
$cmdt0 @test=should_fail/ true @exit=1 2> /dev/null
$cmdt0 @test=should_fail/ false @exit=0 2> /dev/null

$cmdt0 @test=should_fail/ echo foo bar @stdout= 2> /dev/null
$cmdt0 @test=should_fail/ echo foo bar @stdout=foo 2> /dev/null
$cmdt0 @test=should_fail/ echo foo bar @stdout=foo bar 2> /dev/null
$cmdt0 @test=should_fail/ echo foo bar @stdout:baz 2> /dev/null
$cmdt0 @test=should_fail/ echo foo bar @stdout:foo @stdout:baz 2> /dev/null
$cmdt0 @test=should_fail/ echo foo bar @stderr:foo 2> /dev/null

$cmdt0 @test=should_fail/ sh -c ">&2 echo foo bar" @stderr= 2> /dev/null
$cmdt0 @test=should_fail/ sh -c ">&2 echo foo bar" @stderr=foo 2> /dev/null
$cmdt0 @test=should_fail/ sh -c ">&2 echo foo bar" @stderr=foo bar 2> /dev/null
$cmdt0 @test=should_fail/ sh -c ">&2 echo foo bar" @stderr:baz 2> /dev/null
$cmdt0 @test=should_fail/ sh -c ">&2 echo foo bar" @stderr:foo @stderr:baz 2> /dev/null
$cmdt0 @test=should_fail/ sh -c ">&2 echo foo bar" @stdout:foo 2> /dev/null

>&2 echo "## Test cmdt basic assertions should error"
$cmdt0 @init=should_error @stopOnFailure=false

! $cmdt0 @test=should_error/ true @stdout:"" || die "should error because empty contains"
! $cmdt0 @test=should_error/ true @stdout~"" || die "should error because empty regex"
! $cmdt0 @test=should_error/ true @stderr:"" || die "should error because empty contains"
! $cmdt0 @test=should_error/ true @stderr~"" || die "should error because empty regex"

$cmdt @report=should_succeed
! $cmdt @report=should_fail 2>&1 | grep "0 success" | grep "0 error" || die "should_fail test suite should have no success nor error"
! $cmdt @report=should_error 2>&1 | grep "0 success" | grep "0 failure" || die "should_error test suite should have no success nor failure"

nothingToReportExpectedStderrMsg="you must perform some test prior to report"
>&2 echo "## Test @report without test"
$cmdt0 @init=meta1
$cmdt0 @test=meta1/ @fail @stderr:"$nothingToReportExpectedStderrMsg" -- $cmdt1 @report=foo
$cmdt0 @test=meta1/ @fail @stderr:"$nothingToReportExpectedStderrMsg" -- $cmdt1 @report=foo

>&2 echo "## Meta1 test context not shared without token"
# Without token, cmdt run with different pid should run in differents workspaces
$cmdt0 @test=meta1/"without token one" @stderr:"PASSED" @stderr:"#01" -- $cmdt1 true
$cmdt0 @test=meta1/"without token two" @stderr:"PASSED" @stderr:"#01" -- $cmdt1 true
$cmdt0 @test=meta1/ @fail -- $cmdt1 @report

>&2 echo "## Test printed token"
tk0=$( $cmdt @init @printToken 2> /dev/null )
>&2 echo "token: $tk0"
$cmdt0 @init=meta2
$cmdt0 @test=meta2/ @stderr:"PASSED" @stderr:"#01" -- $cmdt true @token=$tk0
$cmdt0 @test=meta2/ @stderr:"PASSED" @stderr:"#02" -- $cmdt true @token=$tk0
$cmdt0 @test=meta2/ @fail -- $cmdt @report
$cmdt0 @test=meta2/ @stderr:"Successfuly ran" -- $cmdt @report @token=$tk0
$cmdt @report 2>&1 | grep -v "Failures"

>&2 echo "## Test exported token"
eval $( $cmdt @init @exportToken 2> /dev/null )
>&2 echo "token: $__CMDT_TOKEN"
$cmdt0 @init=meta3
$cmdt0 @test=meta3/ @stderr:"PASSED" @stderr:"#01" -- $cmdt true
$cmdt0 @test=meta3/ @stderr:"PASSED" @stderr:"#02" -- $cmdt true
$cmdt0 @test=meta3/ @stderr:"Successfuly ran" -- $cmdt @report=main
$cmdt0 @test=meta3/ @fail @stderr:"$nothingToReportExpectedStderrMsg" -- $cmdt @report @token=$tk0

$cmdt0 @init=meta4
$cmdt0 @test=meta4/ @stderr:"PASSED" @stderr:"#01" -- $cmdt @test=sub4/ true
$cmdt0 @test=meta4/ @stderr:"PASSED" @stderr:"#02" -- $cmdt @test=sub4/ true
$cmdt0 @test=meta4/ @stderr:"Successfuly ran" -- $cmdt @report=sub4
$cmdt0 @test=meta4/ @fail @stderr:"$nothingToReportExpectedStderrMsg" -- $cmdt @report=sub4 @token=$tk0
$cmdt @report 2>&1 | grep -v "Failures"

export -n __CMDT_TOKEN


eval $( $cmdt @init @exportToken 2> /dev/null )

>&2 echo "## Rules parsing stopper --"
$cmdt0 @stdout="foo @success @fail\n" -- echo foo @success @fail

>&2 echo "## Test Suite re-init"
$cmdt0 @init=reinit
$cmdt0 @test=reinit/ -- $cmdt1 @test=sub1/ true
$cmdt0 @test=reinit/ -- $cmdt1 @init=sub1
$cmdt0 @test=reinit/ @stderr:"0 test" -- $cmdt1 @report=sub1

$cmdt0 @test=reinit/ -- $cmdt1 @keepOutputs @test=sub2/ true
$cmdt0 @test=reinit/ -- $cmdt1 @keepOutputs @init=sub2
$cmdt0 @test=reinit/ -- $cmdt1 @keepOutputs @test=sub2/ true
$cmdt0 @test=reinit/ -- $cmdt1 @keepOutputs @test=sub2/ true
$cmdt0 @test=reinit/ @stderr:"2 test" -- $cmdt1 @report=sub2

$cmdt0 @test=reinit/ @fail @stderr:"$nothingToReportExpectedStderrMsg" -- $cmdt1 @report=sub3
$cmdt0 @test=reinit/ -- $cmdt1 @test=sub3/ true
$cmdt0 @test=reinit/ -- $cmdt1 @init=sub3
$cmdt0 @test=reinit/ -- $cmdt1 @test=sub3/ true
$cmdt0 @test=reinit/ -- $cmdt1 @report=sub3
$cmdt0 @test=reinit/ -- $cmdt1 @init=sub3
$cmdt0 @test=reinit/ -- $cmdt1 @test=sub3/ true
$cmdt0 @test=reinit/ @stderr:"1 test" -- $cmdt1 @report=sub3


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
#eval $( $cmdt @test=meta/ @keepStdout -- $cmdt @init=t1 @exportToken)

>&2 echo "## Test assertions outputs"
# Init the context used in t1 test suite
$cmdt0 @init=meta
$cmdt0 @test=meta/ @stderr:"#01..." @stderr:"PASSED" -- $cmdt true @test=t1/
$cmdt0 @test=meta/ @stderr:"#02..." @stderr:"PASSED" -- $cmdt true @test=t1/
$cmdt0 @test=meta/ @stderr:"#03..." @stderr:"FAILED" -- $cmdt false @test=t1/
$cmdt0 @test=meta/ @stderr:"#04..." @stderr:"PASSED" -- $cmdt false @fail @test=t1/
$cmdt0 @test=meta/ @fail @stderr:"Failures in [t1] test suite (3 success, 1 failures, 0 errors on 4 tests in" -- $cmdt @report=t1


>&2 echo "## Test namings"
$cmdt @init=main 2> /dev/null
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
$cmdt @init=suite_config_silent @silent
$cmdt0 @init=suite_config
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
$cmdt0 @init=stdin
echo foo | $cmdt0 @test=stdin/ @stdout="foo\n" cat


>&2 echo "## Test export"
export foo=bar
$cmdt0 @init=export
$cmdt0 @test=export/ @stdout~"/foo='bar'/m" sh -c "export"


>&2 echo "## Interlaced tests"
$cmdt0 @init="testA"
$cmdt0 @init="testB"

$cmdt0 echo ignored1 @ignore @test="testA/"
$cmdt0 echo ignored2 @ignore @test="testA/"
$cmdt0 false @test="testA/" 2> /dev/null

$cmdt0 echo another test @test="testB/"
$cmdt0 echo ignored3 @ignore @test="testA/"
$cmdt0 echo interlaced test @test="testA/"
$cmdt0 false @test="testB/" 2> /dev/null
$cmdt0 true @test="testB/"
$cmdt0 false @test="testA/" 2> /dev/null

$cmdt0 @init=interlaced
# should have 1 success 2 failures and 3 ignored
$cmdt0 @test="interlaced/" @fail @stderr:"1 success" @stderr:"2 failure" @stderr:"3 ignored" -- $cmdt1 @report="testA"
# should have 2 success 1 failure
$cmdt0 @test="interlaced/" @fail @stderr:"2 success" @stderr:"1 failure" @stderr!:"ignored" -- $cmdt1 @report="testB"


>&2 echo "## Mutually exclusive rules"
merExpectedMsgRule="@stderr~/mutually exclusives/"
# Actions are mutually exclusives
#eval $( $cmdt0 @init=mutually_exclusive_rules @exportToken )
$cmdt0 @init=mutually_exclusive_rules
$cmdt0 @test=mutually_exclusive_rules/ @fail "$merExpectedMsgRule" -- $cmdt1 @global @init
$cmdt0 @test=mutually_exclusive_rules/ @fail "$merExpectedMsgRule" -- $cmdt1 @init @global
$cmdt0 @test=mutually_exclusive_rules/ @fail "$merExpectedMsgRule" -- $cmdt1 @init @test
$cmdt0 @test=mutually_exclusive_rules/ @fail "$merExpectedMsgRule" -- $cmdt1 @init @report
$cmdt0 @test=mutually_exclusive_rules/ @fail "$merExpectedMsgRule" -- $cmdt1 @global @test
$cmdt0 @test=mutually_exclusive_rules/ @fail "$merExpectedMsgRule" -- $cmdt1 @global @report
$cmdt0 @test=mutually_exclusive_rules/ @fail "$merExpectedMsgRule" -- $cmdt1 @test @report

# Assertions are only accepted on test actions
for action in @global @init @report; do
	for assertion in @success @fail @exit=0 @cmd=true @stdout= @stderr= @exists=foo; do
		$cmdt0 @test=mutually_exclusive_rules/ @fail "$merExpectedMsgRule" -- $cmdt1 "$action" "$assertion"
	done
done

# Suite Config are only accepted on global and init actions
for action in @test @report; do
	cmd=""
	if [ "$action" = "@test" ]; then
		cmd="true"
	fi
	for assertion in @fork=5; do
		$cmdt0 @test=mutually_exclusive_rules/ @fail "$merExpectedMsgRule" -- $cmdt1 "$action" "$assertion" $cmd
	done

done

# Test config are only accepted on global init and test actions
for action in @report; do
	for assertion in @silent @keepStdout @keepStderr @keepOutputs @stopOnFailure @ignore @timeout=1s @runCount=2 @parallel=3; do
		$cmdt0 @test=mutually_exclusive_rules/ @fail "$merExpectedMsgRule" -- $cmdt1 "$action" "$assertion"
	done
done


>&2 echo "## Test flow"
$cmdt @init=main 2> /dev/null # clear main test suite
$cmdt0 @init=test_flow
$cmdt0 @test=test_flow/ @stderr:"#01" @stderr:PASSED -- $cmdt1 @fail false
$cmdt0 @test=test_flow/ @stderr:"#02" @stderr:PASSED -- $cmdt1 true
$cmdt0 @test=test_flow/ @fail @stderr:"mutually exclusive" -- $cmdt1 "@test" "@fork=5" # Should error because of bad param
$cmdt0 @test=test_flow/ @stderr:"#03" @stderr:ERROR -- $cmdt1 doNotExists @stderr:"not executed" # Should error because of not executable
$cmdt0 @test=test_flow/ @stderr:"#04" @stderr:PASSED -- $cmdt1 true
$cmdt0 @test=test_flow/ @fail @stderr:"3 success" @stderr:"2 error" -- $cmdt1 @report=main
$cmdt0 @test=test_flow/ @stderr:"#01" @stderr:PASSED -- $cmdt1 true
$cmdt0 @test=test_flow/ @stderr:"1 test" -- $cmdt1 @report=main


>&2 echo "## Test @mock"
mockCfg1="@mock=ls foo,stdin=,stdout=baz,exit=41"
mockCfg2="@mock=ls foo,cmd=sh -c 'echo -n baz; exit 42'"
mockCfg3="@mock=ls foo *,cmd=sh -c 'echo -n baz; exit 43'"
mockCfg4="@mock=ls foo,stdin=baz,exit=44"
mockCfg6="@mock:ls foo bar,stdout=baz,exit=46"
mockCfg7="@mock:ls foo bar *,stdout=baz,exit=47"
rm -f -- foo bar baz 2> /dev/null || true
#expectedFooErrMsg="cannot access 'foo'"
#expectedBarErrMsg="cannot access 'bar'"
#expectedBazErrMsg="cannot access 'baz'"
expectedFooErrMsg="$( 2>&1 ls foo || true )"
expectedBarErrMsg="$( 2>&1 ls bar || true )"
expectedBazErrMsg="$( 2>&1 ls baz || true )"
$cmdt0 @init=cmd_mock
$cmdt0 @test=cmd_mock/ @stderr:PASSED -- $cmdt1 sh -c "echo \${PATH}" "$mockCfg1" @stdout:/mock:/
$cmdt0 @test=cmd_mock/ @stderr:PASSED -- $cmdt1 which ls "$mockCfg1" @stderr= @stdout:/mock/ls
$cmdt0 @test=cmd_mock/ @stderr:PASSED -- $cmdt1 sh -c "ls foo" "$mockCfg1" @stdout=baz @exit=41 @keepOutputs
$cmdt0 @test=cmd_mock/ @stderr:PASSED -- $cmdt1 ls foo "$mockCfg1" @stdout=baz @exit=41
$cmdt0 @test=cmd_mock/ @stderr:PASSED -- $cmdt1 sh -c "echo foo | ls foo" "$mockCfg1" @fail @stdout= @stderr:"$expectedFooErrMsg"
$cmdt0 @test=cmd_mock/ @stderr:PASSED -- $cmdt1 ls bar "$mockCfg1" @fail @stderr:"$expectedBarErrMsg"
$cmdt0 @test=cmd_mock/ @stderr:PASSED -- $cmdt1 ls bar foo "$mockCfg1" @fail @stderr:"$expectedBarErrMsg" @stderr:"$expectedFooErrMsg"
$cmdt0 @test=cmd_mock/ @stderr:PASSED -- $cmdt1 ls foo bar "$mockCfg1" @fail @stderr:"$expectedBarErrMsg" @stderr:"$expectedFooErrMsg"

$cmdt0 @test=cmd_mock/ @stderr:PASSED -- $cmdt1 ls foo "$mockCfg2" @stdout=baz @exit=42

$cmdt0 @test=cmd_mock/ @stderr:PASSED -- $cmdt1 ls foo "$mockCfg3" @stdout=baz @exit=43
$cmdt0 @test=cmd_mock/ @stderr:PASSED -- $cmdt1 ls bar "$mockCfg3" @fail @stderr:"$expectedBarErrMsg"
$cmdt0 @test=cmd_mock/ @stderr:PASSED -- $cmdt1 ls bar foo "$mockCfg3" @fail @stderr:"$expectedBarErrMsg"
$cmdt0 @test=cmd_mock/ @stderr:PASSED -- $cmdt1 ls foo bar "$mockCfg3" @stdout=baz @exit=43

$cmdt0 @test=cmd_mock/ @stderr:PASSED -- $cmdt1 ls foo "$mockCfg4" @fail @stderr:"$expectedFooErrMsg"
$cmdt0 @test=cmd_mock/ @stderr:PASSED -- $cmdt1 sh -c "echo baz | ls foo" "$mockCfg4" @stderr= @exit=44

$cmdt0 @test=cmd_mock/ @stderr:PASSED -- $cmdt1 ls foo "$mockCfg6" @fail @stderr:"$expectedFooErrMsg"
$cmdt0 @test=cmd_mock/ @stderr:PASSED -- $cmdt1 ls bar "$mockCfg6" @fail @stderr:"$expectedBarErrMsg"
$cmdt0 @test=cmd_mock/ @stderr:PASSED -- $cmdt1 ls bar foo "$mockCfg6" @stdout=baz @exit=46
$cmdt0 @test=cmd_mock/ @stderr:PASSED -- $cmdt1 ls foo bar "$mockCfg6" @stdout=baz @exit=46
$cmdt0 @test=cmd_mock/ @stderr:PASSED -- $cmdt1 ls foo bar baz "$mockCfg6" @fail @stderr:"$expectedFooErrMsg" @stderr:"$expectedBarErrMsg" @stderr:"$expectedBazErrMsg"

$cmdt0 @test=cmd_mock/ @stderr:PASSED -- $cmdt1 ls foo "$mockCfg7" @fail @stderr:"$expectedFooErrMsg"
$cmdt0 @test=cmd_mock/ @stderr:PASSED -- $cmdt1 ls bar "$mockCfg7" @fail @stderr:"$expectedBarErrMsg"
$cmdt0 @test=cmd_mock/ @stderr:PASSED -- $cmdt1 ls bar foo "$mockCfg7" @stdout=baz @exit=47
$cmdt0 @test=cmd_mock/ @stderr:PASSED -- $cmdt1 ls foo bar "$mockCfg7" @stdout=baz @exit=47
$cmdt0 @test=cmd_mock/ @stderr:PASSED -- $cmdt1 ls foo bar baz "$mockCfg7" @stdout=baz @exit=47

$cmdt0 @test=cmd_mock/ -- $cmdt1 @report=main


>&2 echo "## Test @before & @after"
testFile="/tmp/thisFileDoesNotExistsYet.txt"
testFile2="/tmp/thisFileDoesNotExistsYet2.txt"
rm -f -- "$testFile" "$testFile2" 2> /dev/null || true
$cmdt0 @init=before_after
$cmdt0 @test=before_after/ @stderr:PASSED -- $cmdt1 ls "$testFile" @fail
$cmdt0 @test=before_after/ @stderr:PASSED -- $cmdt1 ls "$testFile" @before="touch $testFile"
$cmdt0 @test=before_after/ @stderr:PASSED -- $cmdt1 ls "$testFile"
$cmdt0 @test=before_after/ @stderr:PASSED -- $cmdt1 ls "$testFile" @after="rm -f -- $testFile"
$cmdt0 @test=before_after/ @stderr:PASSED -- $cmdt1 ls "$testFile" @fail
$cmdt0 @test=before_after/ @stderr:PASSED -- $cmdt1 ls "$testFile" "$testFile2" @before="touch $testFile" @before="touch $testFile2"

$cmdt0 @test=before_after/ -- $cmdt1 @report=main


>&2 echo "## Test @container"
$cmdt @init=container @keepOutputs
$cmdt0 @test=container/ @stderr:PASSED -- $cmdt1 true @container
$cmdt0 @test=container/ @stderr:FAILED -- $cmdt1 false @container
$cmdt0 @test=container/ @stderr:PASSED -- $cmdt1 true @container=alpine

$cmdt0 @test=before_after/ @fail -- $cmdt1 @report=main


$cmdt @report= ; >&2 echo SUCCESS ; exit 0


