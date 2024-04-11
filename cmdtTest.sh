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
cmdt0="$newCmdt"
cmdt1="$newCmdt @verbose @failuresLimit=-1" # Default verbose show passed test + perform all test beyond failures limit

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
$cmdtIn @init=should_succeed @stopOnFailure=false

$cmdtIn @test=should_succeed/ true
$cmdtIn @test=should_succeed/ true @success
$cmdtIn @test=should_succeed/ false @fail
$cmdtIn @test=should_succeed/ true @exit=0
$cmdtIn @test=should_succeed/ false @exit=1

$cmdtIn @test=should_succeed/ echo foo bar @stdout:foo @stderr=
$cmdtIn @test=should_succeed/ echo foo bar @stdout:bar
$cmdtIn @test=should_succeed/ echo foo bar @stdout!:baz
$cmdtIn @test=should_succeed/ echo foo bar @stdout!=baz
$cmdtIn @test=should_succeed/ echo foo bar @stdout~/^foo/ @stderr=
$cmdtIn @test=should_succeed/ echo foo bar @stdout~/BaR/i
$cmdtIn @test=should_succeed/ echo foo bar @stdout~"/^foo bar\n$/"
$cmdtIn @test=should_succeed/ echo foo bar @stdout~"/^foo bar$/m"
$cmdtIn @test=should_succeed/ echo foo bar @stdout!~/bar$/
$cmdtIn @test=should_succeed/ echo foo\nbar\nbaz @stdout!~/^bar$/
$cmdtIn @test=should_succeed/ echo foo\nbar\nbaz @stdout!~/^bar$/m
$cmdtIn @test=should_succeed/ echo foo bar @stdout:foo @stdout:bar @stderr=
$cmdtIn @test=should_succeed/ echo foo bar @stdout="foo bar\n" @stderr=

$cmdtIn @test=should_succeed/ sh -c ">&2 echo foo bar" @stderr:foo @stdout=
$cmdtIn @test=should_succeed/ sh -c ">&2 echo foo bar" @stderr:bar
$cmdtIn @test=should_succeed/ sh -c ">&2 echo foo bar" @stderr!:baz
$cmdtIn @test=should_succeed/ sh -c ">&2 echo foo bar" @stderr!=baz
$cmdtIn @test=should_succeed/ sh -c ">&2 echo foo bar" @stderr~/^foo/ @stdout=
$cmdtIn @test=should_succeed/ sh -c ">&2 echo foo bar" @stderr~/BaR/i
$cmdtIn @test=should_succeed/ sh -c ">&2 echo foo bar" @stderr~"/^foo bar\n$/"
$cmdtIn @test=should_succeed/ sh -c ">&2 echo foo bar" @stderr!~/bar$/
$cmdtIn @test=should_succeed/ sh -c ">&2 echo foo bar" @stderr:foo @stderr:bar
$cmdtIn @test=should_succeed/ sh -c ">&2 echo foo bar" @stderr="foo bar\n" @stdout=

>&2 echo "## Test cmdt basic assertions should failed"
$cmdtIn @init=should_fail @failuresLimit=-1 @verbose=0

$cmdtIn 2> /dev/null @test=should_fail/ false
$cmdtIn 2> /dev/null @test=should_fail/ true @fail
$cmdtIn 2> /dev/null @test=should_fail/ false @success
$cmdtIn 2> /dev/null @test=should_fail/ true @exit=1
$cmdtIn 2> /dev/null @test=should_fail/ false @exit=0

$cmdtIn 2> /dev/null @test=should_fail/ echo foo bar @stdout=
$cmdtIn 2> /dev/null @test=should_fail/ echo foo bar @stdout=foo
$cmdtIn 2> /dev/null @test=should_fail/ echo foo bar @stdout=foo bar
$cmdtIn 2> /dev/null @test=should_fail/ echo foo bar @stdout:baz
$cmdtIn 2> /dev/null @test=should_fail/ echo foo bar @stdout:foo @stdout:baz
$cmdtIn 2> /dev/null @test=should_fail/ echo foo bar @stderr:foo

$cmdtIn 2> /dev/null @test=should_fail/ sh -c ">&2 echo foo bar" @stderr=
$cmdtIn 2> /dev/null @test=should_fail/ sh -c ">&2 echo foo bar" @stderr=foo
$cmdtIn 2> /dev/null @test=should_fail/ sh -c ">&2 echo foo bar" @stderr=foo bar
$cmdtIn 2> /dev/null @test=should_fail/ sh -c ">&2 echo foo bar" @stderr:baz
$cmdtIn 2> /dev/null @test=should_fail/ sh -c ">&2 echo foo bar" @stderr:foo @stderr:baz
$cmdtIn 2> /dev/null @test=should_fail/ sh -c ">&2 echo foo bar" @stdout:foo

>&2 echo "## Test cmdt basic assertions should error"
$cmdtIn @init=should_error @failuresLimit=-1

! $cmdtIn @test=should_error/ true @stdout:"" || die "should error because empty contains"
! $cmdtIn @test=should_error/ true @stdout~"" || die "should error because empty regex"
! $cmdtIn @test=should_error/ true @stderr:"" || die "should error because empty contains"
! $cmdtIn @test=should_error/ true @stderr~"" || die "should error because empty regex"


of="/tmp/cmdtReportContent.txt"

>&2 echo "## reporting should_succeed"
rc=0
$cmdt @report=should_succeed > "$of" 2>&1 || rc=$?
test "$rc" -eq 0 || die "reporting should_succeed should exit=0"
grep "28 success" "$of" || die "reporting should_succeed bad success count"


>&2 echo "## reporting should_fail"
rc=0
$cmdt @report=should_fail > "$of" 2>&1 || rc=$?
test "$rc" -eq 1 || die "reporting should_fail shoud exit=1"
grep "17 failures" "$of" || die "reporting should_fail bad failures count"

>&2 echo "## reporting should_error"
rc=0
$cmdt @report=should_error > "$of" 2>&1 || rc=$?
test "$rc" -eq 1 || die "reporting should_error should exit=1"
grep "4 errors" "$of" || die "reporting should_error bad errors count"


>&2 echo "## reporting all"
! $cmdt @report >/dev/null 2>&1 || die "reporting all should exit=1"

nothingToReportExpectedStderrMsg="you must perform some test prior to report"
>&2 echo "## Test @report without test"
$cmdtIn @init=meta1
$cmdtIn @test=meta1/ @fail @stderr:"$nothingToReportExpectedStderrMsg" @-- $cmdt0 @report=foo
$cmdtIn @test=meta1/ @fail @stderr:"$nothingToReportExpectedStderrMsg" @-- $cmdt0 @report=foo

>&2 echo "## Meta1 test context not shared without token"
# Without token, cmdt run with different pid should run in differents workspaces
$cmdtIn @test=meta1/"without token one" @stderr:"PASSED" @stderr:"#01" @-- $cmdt1 true @debug
# FIXME: without token, should be first or second test ?
$cmdtIn @test=meta1/"without token two" @stderr:"PASSED" @stderr:"#02" @-- $cmdt1 true @debug
$cmdtIn @test=meta1/"command before rule stop" @fail @stderr:"before rule parsing stopper" @-- $cmdt1 true @-- @success
$cmdtIn @test=meta1/"rule on 2 args" @stderr:"PASSED" @-- $cmdt1 @stdout:foo bar @-- echo foo bar
#$cmdtIn @test=meta1/ @exit=1 @stderr:"$nothingToReportExpectedStderrMsg" @-- $cmdt1 @report
$cmdtIn @test=meta1/ @exit=1 @stderr:"Failures in" @-- $cmdt0 @report=main

>&2 echo "## Test printed token"
tk0=$( $cmdt @init @printToken 2> /dev/null )
>&2 echo "token: $tk0"
$cmdtIn @init=meta2
$cmdtIn @test=meta2/ @stderr:"PASSED" @stderr:"#01" @-- $cmdt0 true @token=$tk0
$cmdtIn @test=meta2/ @stderr:"PASSED" @stderr:"#02" @-- $cmdt0 true @token=$tk0
$cmdtIn @test=meta2/ @fail @-- $cmdt0 @report=main
$cmdtIn @test=meta2/ @stderr:"Successfuly ran" @-- $cmdt0 @report @token=$tk0
$cmdt @report 2>&1 | grep -v "Failures"

>&2 echo "## Test exported token"
eval $( $cmdt @init @exportToken 2> /dev/null )
>&2 echo "token: $__CMDT_TOKEN"
$cmdtIn @init=meta3
$cmdtIn @test=meta3/ @stderr:"PASSED" @stderr:"#01" @-- $cmdt0 true
$cmdtIn @test=meta3/ @stderr:"PASSED" @stderr:"#02" @-- $cmdt0 true
$cmdtIn @test=meta3/ @stderr:"Successfuly ran" @-- $cmdt0 @report=main
$cmdtIn @test=meta3/ @fail @stderr:"$nothingToReportExpectedStderrMsg" @-- $cmdt0 @report=main @token=$tk0

$cmdtIn @init=meta4
$cmdtIn @test=meta4/ @stderr:"PASSED" @stderr:"#01" @-- $cmdt0 @test=sub4/ true
$cmdtIn @test=meta4/ @stderr:"PASSED" @stderr:"#02" @-- $cmdt0 @test=sub4/ true
$cmdtIn @test=meta4/ @stderr:"Successfuly ran" @-- $cmdt0 @report=sub4
$cmdtIn @test=meta4/ @fail @stderr:"$nothingToReportExpectedStderrMsg" @-- $cmdt0 @report=sub4 @token=$tk0
$cmdt @report 2>&1 | grep -v "Failures"

export -n __CMDT_TOKEN


eval $( $cmdt @init @exportToken 2> /dev/null )

>&2 echo "## Rules parsing stopper @--"
$cmdtIn @init=parsing_stopper
$cmdtIn @test=parsing_stopper/ @stdout="foo @success @fail\n" @-- echo foo @success @fail
$cmdtIn @report=parsing_stopper

>&2 echo "## Test Suite re-init"
$cmdtIn @init=reinit
$cmdtIn @test=reinit/ @-- $cmdt1 @test=sub1/ true
$cmdtIn @test=reinit/ @-- $cmdt1 @init=sub1
$cmdtIn @test=reinit/ @fail @stderr:"$nothingToReportExpectedStderrMsg" @-- $cmdt1 @report=sub1

$cmdtIn @test=reinit/ @-- $cmdt1 @keepOutputs @test=sub2/ true
$cmdtIn @test=reinit/ @-- $cmdt1 @keepOutputs @init=sub2
$cmdtIn @test=reinit/ @-- $cmdt1 @keepOutputs @test=sub2/ true
$cmdtIn @test=reinit/ @-- $cmdt1 @keepOutputs @test=sub2/ true
$cmdtIn @test=reinit/ @stderr:"2 success" @-- $cmdt1 @report=sub2

$cmdtIn @test=reinit/ @fail @stderr:"$nothingToReportExpectedStderrMsg" @-- $cmdt1 @report=sub3
$cmdtIn @test=reinit/ @-- $cmdt1 @test=sub3/ true
$cmdtIn @test=reinit/ @-- $cmdt1 @init=sub3
$cmdtIn @test=reinit/ @-- $cmdt1 @test=sub3/ true
$cmdtIn @test=reinit/ @-- $cmdt1 @report=sub3
$cmdtIn @test=reinit/ @-- $cmdt1 @init=sub3
$cmdtIn @test=reinit/ @-- $cmdt1 @test=sub3/ true
$cmdtIn @test=reinit/ @stderr:"1 success" @-- $cmdt1 @report=sub3


>&2 echo "## Test usage"
$cmdtIn @init=meta
$cmdtIn @test=meta/ @fail @stderr:"usage:" @-- $cmdt0

>&2 echo "## Test cmdt basic assertions"
$cmdtIn @test=meta/ @stderr:"PASSED" @-- $cmdt0 true
$cmdtIn @test=meta/ @stderr:"FAILED" @-- $cmdt0 false
$cmdtIn @test=meta/ @stderr:"PASSED" @-- $cmdt0 true @success
$cmdtIn @test=meta/ @stderr:"FAILED" @-- $cmdt0 false @success
$cmdtIn @test=meta/ @stderr:"FAILED" @-- $cmdt0 true @fail
$cmdtIn @test=meta/ @stderr:"PASSED" @-- $cmdt0 false @fail
$cmdtIn @test=meta/ @stderr:"PASSED" @-- $cmdt0 true @exit=0
$cmdtIn @test=meta/ @stderr:"FAILED" @-- $cmdt0 false @exit=0
$cmdtIn @test=meta/ @stderr:"FAILED" @-- $cmdt0 true @exit=1
$cmdtIn @test=meta/ @stderr:"PASSED" @-- $cmdt0 false @exit=1

$cmdtIn @test=meta/ @stderr:"PASSED" @-- $cmdt0 echo foo bar @stdout:foo @stderr=
$cmdtIn @test=meta/ @stderr:"FAILED" @-- $cmdt0 echo foo bar @stdout:baz @stderr=
$cmdtIn @test=meta/ @stderr:"PASSED" @-- $cmdt0 echo foo bar @stdout:foo @stdout:bar @stderr=
$cmdtIn @test=meta/ @stderr:"FAILED" @-- $cmdt0 echo foo bar @stdout:baz @stdout:bar @stderr=
$cmdtIn @test=meta/ @stderr:"FAILED" @-- $cmdt0 echo foo bar @stdout:foo @stdout:baz @stderr=
$cmdtIn @test=meta/ @stderr:"FAILED" @-- $cmdt0 echo foo bar @stdout!:foo
$cmdtIn @test=meta/ @stderr:"PASSED" @-- $cmdt0 echo foo bar @stdout!=foo @stdout:bar @stdout!:baz

$cmdtIn @test=meta/ @stderr:"PASSED" @-- $cmdt0 sh -c ">&2 echo foo bar" @stderr:foo @stdout=
$cmdtIn @test=meta/ @stderr:"FAILED" @-- $cmdt0 sh -c ">&2 echo foo bar" @stderr:baz @stdout=
$cmdtIn @test=meta/ @stderr:"PASSED" @-- $cmdt0 sh -c ">&2 echo foo bar" @stderr:foo @stderr:bar @stdout=
$cmdtIn @test=meta/ @stderr:"FAILED" @-- $cmdt0 sh -c ">&2 echo foo bar" @stderr:baz @stderr:bar @stdout=
$cmdtIn @test=meta/ @stderr:"FAILED" @-- $cmdt0 sh -c ">&2 echo foo bar" @stderr:foo @stderr:baz @stdout=
$cmdtIn @test=meta/ @stderr:"FAILED" @-- $cmdt0 sh -c ">&2 echo foo bar" @stderr!:foo
$cmdtIn @test=meta/ @stderr:"PASSED" @-- $cmdt0 sh -c ">&2 echo foo bar" @stderr!=foo @stderr:bar @stderr!:baz


## Forge a token for remaining tests
#eval $( $cmdt @test=meta/ @keepStdout @-- $cmdt @init=t1 @exportToken)

>&2 echo "## Test assertions outputs"
# Init the context used in t1 test suite
$cmdtIn @init=outputs_assertions
$cmdtIn @test=outputs_assertions/ @stderr:"#01..." @stderr:"PASSED" @-- $cmdt0 true @test=t1/
$cmdtIn @test=outputs_assertions/ @stderr:"#02..." @stderr:"PASSED" @-- $cmdt0 true @test=t1/
$cmdtIn @test=outputs_assertions/ @stderr:"#03..." @stderr:"FAILED" @-- $cmdt0 false @test=t1/
$cmdtIn @test=outputs_assertions/ @stderr:"#04..." @stderr:"PASSED" @-- $cmdt0 false @fail @test=t1/
$cmdtIn @test=outputs_assertions/ @fail @stderr~"/Failures in \[.*t1.*\] test suite \(3 success, 1 failures, 0 errors on 4 tests in/" @-- $cmdt0 @report=t1


>&2 echo "## Test namings"
$cmdt @init=main 2> /dev/null
$cmdtIn @init=naming
$cmdtIn @test=naming/ @stderr~"/Test \[main\].*name1 #01.../" @stderr:"PASSED" @-- $cmdt1 true @test=name1
$cmdtIn @test=naming/ @stderr~"/Test \[main\].*name2 #02.../" @stderr:"PASSED" @-- $cmdt1 true @test=name2
$cmdtIn @test=naming/ @stderr~"/Test \[main\].*/" @stderr:"true" @stderr:"#03..." @stderr:"PASSED" @-- $cmdt1 true
$cmdtIn @test=naming/ @stderr~"/Test \[main\].*/" @stderr:"true" @stderr:"#04..." @stderr:"PASSED" @-- $cmdt1 true
$cmdtIn @test=naming/ @stderr~"/Test \[suite1\].*name1 #01.../" @stderr:"PASSED" @-- $cmdt1 true @test=suite1/name1
$cmdtIn @test=naming/ @stderr~"/Test \[suite1\].*name2 #02.../" @stderr:"PASSED" @-- $cmdt1 true @test=suite1/name2
$cmdtIn @test=naming/ @stderr~"/Test \[suite2\].*/" @stderr:"#01..." @stderr:"PASSED" @-- $cmdt1 true @test=suite2/
$cmdtIn @test=naming/ @stderr~"/Test \[suite2\].*/" @stderr:"#02..." @stderr:"PASSED" @-- $cmdt1 true @test=suite2/
$cmdtIn @test=naming/ @stderr~"/Successfuly ran \[.*suite1.*\] test suite/" @-- $cmdt1  @report=suite1
$cmdtIn @test=naming/ @stderr~"/Successfuly ran \[.*suite2.*\] test suite/" @-- $cmdt1 @report=suite2
$cmdtIn @test=naming/ @stderr~"/Successfuly ran \[.*main.*\] test suite/" @-- $cmdt1 @report=main


>&2 echo "## Test display verbosity"
$cmdtIn @init=display_verbosity
# verbose=SHOW_REPORTS_ONLY
$cmdtIn @test=display_verbosity/ @stdout= @stderr=                                                   @-- $cmdt0 @verbose=0 echo foo 
$cmdtIn @test=display_verbosity/ @stdout= @stderr=                                                   @-- $cmdt0 @verbose=0 echo foo @fail
$cmdtIn @test=display_verbosity/ @stdout= @stderr=                                                   @-- $cmdt0 @verbose=0 foo
# verbose=SHOW_FAILED_ONLY
$cmdtIn @test=display_verbosity/ @stdout= @stderr=                                                   @-- $cmdt0 @verbose=1 echo foo 
$cmdtIn @test=display_verbosity/ @stdout= @stderr:FAILED @stderr:"Executing cmd" @stderr!:">foo"     @-- $cmdt0 @verbose=1 echo foo @fail
$cmdtIn @test=display_verbosity/ @stdout= @stderr:ERRORED @stderr:"Executing cmd"                    @-- $cmdt0 @verbose=1 foo
# verbose=SHOW_FAILED_OUTS
$cmdtIn @test=display_verbosity/ @stdout= @stderr=                                                   @-- $cmdt0 @verbose=2 echo foo
$cmdtIn @test=display_verbosity/ @stdout= @stderr:FAILED @stderr:"Executing cmd" @stderr:">foo"      @-- $cmdt0 @verbose=2 echo foo @fail
$cmdtIn @test=display_verbosity/ @stdout= @stderr:ERRORED @stderr:"Executing cmd"                    @-- $cmdt0 @verbose=2 foo
# verbose=SHOW_PASSED
$cmdtIn @test=display_verbosity/ @stdout= @stderr:PASSED @stderr!:"Executing cmd" @stderr!:">foo"    @-- $cmdt0 @verbose=3 echo foo
$cmdtIn @test=display_verbosity/ @stdout= @stderr:FAILED @stderr:"Executing cmd" @stderr:">foo"      @-- $cmdt0 @verbose=3 echo foo @fail
$cmdtIn @test=display_verbosity/ @stdout= @stderr:ERRORED @stderr:"Executing cmd"                    @-- $cmdt0 @verbose=3 foo
# verbose=SHOW_PASSED_OUTS
$cmdtIn @test=display_verbosity/ @stdout= @stderr:PASSED @stderr:"Executing cmd" @stderr:">foo"      @-- $cmdt0 @verbose=4 echo foo
$cmdtIn @test=display_verbosity/ @stdout= @stderr:FAILED @stderr:"Executing cmd" @stderr:">foo"      @-- $cmdt0 @verbose=4 echo foo @fail
$cmdtIn @test=display_verbosity/ @stdout= @stderr:ERRORED @stderr:"Executing cmd"                    @-- $cmdt0 @verbose=4 foo


>&2 echo "## Test rules missusage"
$cmdtIn @init=failing_rule_missusage
$cmdtIn @test=failing_rule_missusage/ @fail @-- $cmdt1 @init true
$cmdtIn @test=failing_rule_missusage/ @fail @-- $cmdt1 @init @test
$cmdtIn @test=failing_rule_missusage/ @fail @-- $cmdt1 @init @report
$cmdtIn @test=failing_rule_missusage/ @fail @-- $cmdt1 @test @report true
$cmdtIn @test=failing_rule_missusage/ @fail @-- $cmdt1 @init @test true
$cmdtIn @test=failing_rule_missusage/ @fail @-- $cmdt1 @init @test true
$cmdtIn @test=failing_rule_missusage/ @fail @-- $cmdt1 @init @fail
$cmdtIn @test=failing_rule_missusage/ @fail @-- $cmdt1 @init @success
$cmdtIn @test=failing_rule_missusage/ @fail @-- $cmdt1 @init @exit=0
$cmdtIn @test=failing_rule_missusage/ @fail @-- $cmdt1 true @fail @success
$cmdtIn @test=failing_rule_missusage/ @fail @-- $cmdt1 true @fail @exit=0
$cmdtIn @test=failing_rule_missusage/ @fail @-- $cmdt1 true @success @exit=0
$cmdtIn @test=failing_rule_missusage/ @fail @-- $cmdt1 @report @fail
$cmdtIn @test=failing_rule_missusage/ @fail @-- $cmdt1 @report @success
$cmdtIn @test=failing_rule_missusage/ @fail @-- $cmdt1 @report @exit=0
$cmdtIn @test=failing_rule_missusage/ @fail @stderr:donotexist @-- $cmdt1 true @donotexist
$cmdtIn @test=failing_rule_missusage/ @fail @stderr:donotexist @-- $cmdt1 true @donotexist


>&2 echo "## Test config"
$cmdtIn @init=test_config

$cmdtIn @test=test_config/ @stderr:IGNORED @stderr!:FAILED @stderr!:PASSED @-- $cmdt1 true @ignore
$cmdtIn @test=test_config/ @stderr!:IGNORED @stderr:PASSED @-- $cmdt1 true
$cmdtIn @test=test_config/ @stderr:IGNORED @stderr!:FAILED @stderr!:PASSED @-- $cmdt1 true @ignore
$cmdtIn @test=test_config/ @stderr:IGNORED @stderr!:FAILED @stderr!:PASSED @-- $cmdt1 false @ignore
$cmdtIn @test=test_config/ @stderr!:IGNORED @stderr:PASSED @-- $cmdt1 true

$cmdtIn @test=test_config/ @stdout!:foo @stderr!:bar @stderr:PASSED @-- $cmdt1 @test=hide_name sh -c "echo foo; >&2 echo bar"
$cmdtIn @test=test_config/ @stdout~/^foo$/m @stderr!:bar @stderr:PASSED @-- $cmdt1 @test=hide_name sh -c "echo foo; >&2 echo bar" @keepStdout
$cmdtIn @test=test_config/ @stdout!:foo @stderr:bar @stderr:PASSED @-- $cmdt1 @test=hide_name sh -c "echo foo; >&2 echo bar" @keepStderr
$cmdtIn @test=test_config/ @stdout~/^foo$/m @stderr:bar @stderr:PASSED @-- $cmdt1 @test=hide_name sh -c "echo foo; >&2 echo bar" @keepOutputs

$cmdtIn @test=test_config/ @stderr:TIMEOUT @-- $cmdt1 sleep 0.01 @timeout=5ms
$cmdtIn @test=test_config/ @stderr:PASSED @-- $cmdt1 sleep 0.01 @timeout=30ms

$cmdtIn @test=test_config/ @stderr:FAILED @success @-- $cmdt1 false
$cmdtIn @test=test_config/ @stderr:FAILED @fail @-- $cmdt1 false @stopOnFailure
$cmdtIn @test=test_config/ @stderr:FAILED @success @-- $cmdt1 false

$cmdtIn @test=test_config/ @stderr:FAILED @-- $cmdt1 @prefix=% %fail true
$cmdtIn @test=test_config/ @stderr:PASSED @-- $cmdt1 @prefix=% %fail false
$cmdtIn @test=test_config/ @stderr:PASSED @-- $cmdt1 @prefix=% %success true

$cmdtIn @test=test_config/ @stderr= @-- $cmdt1 @quiet true
$cmdtIn @test=test_config/ @stderr= @-- $cmdt1 @quiet false
$cmdtIn @test=test_config/ @stdout="foo\n" @stderr= @-- $cmdt1 @quiet @keepOutputs echo foo
$cmdtIn @test=test_config/ @stdout= @stderr="foo\n" @-- $cmdt1 @quiet @keepOutputs sh -c ">&2 echo foo"

$cmdtIn @test=test_config/ @fail @-- $cmdt1 @report=main


>&2 echo "## Test suite config"
$cmdt @init=suite_config_quiet @quiet
$cmdtIn @init=suite_config
$cmdtIn @test=suite_config/ @stdout= @stderr= @-- $cmdt1 @test=suite_config_quiet/ echo foo
$cmdtIn @test=suite_config/ @stderr:PASSED @-- $cmdt1 @test=suite_config_quiet/ echo foo @quiet=false
$cmdtIn @test=suite_config/ @stdout="foo\n" @stderr= @-- $cmdt1 @test=suite_config_quiet/ echo foo @keepOutputs
$cmdtIn @test=suite_config/ @stdout= @stderr="bar\n" @-- $cmdt1 @test=suite_config_quiet/ sh -c ">&2 echo bar" @keepOutputs
$cmdtIn @test=suite_config/ @-- $cmdt1 @report=suite_config_quiet

>&2 echo "## Test global config"


>&2 echo "## Test assertions"
$cmdtIn @init=assertion
$cmdtIn @test=assertion/ @stderr:PASSED @-- $cmdt1 true
$cmdtIn @test=assertion/ @stderr:PASSED @-- $cmdt1 true @success
$cmdtIn @test=assertion/ @stderr:FAILED @-- $cmdt1 true @fail

$cmdtIn @test=assertion/ @stderr:FAILED @-- $cmdt1 false
$cmdtIn @test=assertion/ @stderr:FAILED @-- $cmdt1 false @success
$cmdtIn @test=assertion/ @stderr:PASSED @-- $cmdt1 false @fail

$cmdtIn @test=assertion/ @stderr:PASSED @-- $cmdt1 true @exit=0
$cmdtIn @test=assertion/ @stderr:FAILED @-- $cmdt1 true @exit=1

$cmdtIn @test=assertion/ @stderr:FAILED @-- $cmdt1 false @exit=0
$cmdtIn @test=assertion/ @stderr:PASSED @-- $cmdt1 false @exit=1

$cmdtIn @test=assertion/ @stderr:PASSED @-- $cmdt1 echo foo bar @stdout="foo bar\n"
$cmdtIn @test=assertion/ @stderr:PASSED @-- $cmdt1 echo foo bar @stdout:foo
$cmdtIn @test=assertion/ @stderr:PASSED @-- $cmdt1 echo foo bar @stdout:bar
$cmdtIn @test=assertion/ @stderr:PASSED @-- $cmdt1 echo foo bar @stderr=
$cmdtIn @test=assertion/ @stderr:FAILED @-- $cmdt1 echo foo bar @stdout="foo bar"
$cmdtIn @test=assertion/ @stderr:FAILED @-- $cmdt1 echo foo bar @stdout=
$cmdtIn @test=assertion/ @fail @-- $cmdt1 echo foo bar @stdout:
$cmdtIn @test=assertion/ @stderr:FAILED @-- $cmdt1 echo foo bar @stdout:baz
$cmdtIn @test=assertion/ @fail @-- $cmdt1 echo foo bar @stderr:

$cmdtIn @test=assertion/ @stderr:PASSED @-- $cmdt1 sh -c ">&2 echo foo bar" @stderr="foo bar\n"
$cmdtIn @test=assertion/ @stderr:PASSED @-- $cmdt1 sh -c ">&2 echo foo bar" @stderr:foo
$cmdtIn @test=assertion/ @stderr:PASSED @-- $cmdt1 sh -c ">&2 echo foo bar" @stderr:bar
$cmdtIn @test=assertion/ @stderr:PASSED @-- $cmdt1 sh -c ">&2 echo foo bar" @stdout=
$cmdtIn @test=assertion/ @stderr:FAILED @-- $cmdt1 sh -c ">&2 echo foo bar" @stderr="foo bar"
$cmdtIn @test=assertion/ @stderr:FAILED @-- $cmdt1 sh -c ">&2 echo foo bar" @stderr=
$cmdtIn @test=assertion/ @fail @-- $cmdt1 sh -c ">&2 echo foo bar" @stderr:
$cmdtIn @test=assertion/ @stderr:FAILED @-- $cmdt1 sh -c ">&2 echo foo bar" @stderr:baz
$cmdtIn @test=assertion/ @fail @-- $cmdt1 sh -c ">&2 echo foo bar" @stdout:

$cmdtIn @test=assertion/ @stderr:FAILED @-- $cmdt1 sh -c "rm /tmp/donotexists || true" @exists=/tmp/donotexists
$cmdtIn @test=assertion/ @stderr:PASSED @-- $cmdt1 sh -c "touch /tmp/doexists" @exists=/tmp/doexists
$cmdtIn @test=assertion/ @stderr:PASSED @-- $cmdt1 sh -c "chmod 640 /tmp/doexists" @exists=/tmp/doexists,-rw-r-----
$cmdtIn @test=assertion/ @stderr:FAILED @-- $cmdt1 sh -c "chmod 640 /tmp/doexists" @exists=/tmp/doexists,-rwxr-----

$cmdtIn @test=assertion/ @stderr:PASSED @-- $cmdt1 false @fail @cmd=true
$cmdtIn @test=assertion/ @stderr:FAILED @-- $cmdt1 true @cmd=false
touch /tmp/doexists
$cmdtIn @test=assertion/ @stderr:PASSED @-- $cmdt1 false @fail @cmd="ls /tmp/doexists"

$cmdtIn @test=assertion/ @fail @-- $cmdt1 @report=main


>&2 echo "## Test stdin"
$cmdtIn @init=stdin
echo foo | $cmdt @test=stdin/ @stdout="foo\n" cat
# TODO test error raised if stdin used with @async=true


>&2 echo "## Test file ref content in assertions"
$cmdtIn @init=file_ref_content
echo foo > /tmp/fileRefContent
$cmdtIn @test=file_ref_content/ @stderr:PASSED @-- $cmdt1 @stdout@=/tmp/fileRefContent echo foo
$cmdtIn @test=file_ref_content/ @stderr:FAILED @-- $cmdt1 @stdout@=/tmp/fileRefContent echo -n foo
echo -n foo > /tmp/fileRefContent
$cmdtIn @test=file_ref_content/ @stderr:PASSED @-- $cmdt1 @stdout@=/tmp/fileRefContent echo -n foo
$cmdtIn @test=file_ref_content/ @stderr:FAILED @-- $cmdt1 @stdout@=/tmp/fileRefContent echo foo
$cmdtIn @test=file_ref_content/ @stderr:PASSED @-- $cmdt1 @stdout@:/tmp/fileRefContent echo foo
$cmdtIn @test=file_ref_content/ @stderr:FAILED @-- $cmdt1 @stdout@:/tmp/fileRefContent echo bar

$cmdtIn @test=file_ref_content/ @fail @-- $cmdt1 @report=main

>&2 echo "## Test export"
export foo=bar
$cmdtIn @init=export
$cmdtIn @test=export/ @stdout~"/foo='bar'/m" sh -c "export"


>&2 echo "## Interlaced tests"
$cmdtIn @init="testA" @verbose=0
$cmdtIn @init="testB" @verbose=0

$cmdtIn echo ignored1 @ignore @test="testA/"
$cmdtIn echo ignored2 @ignore @test="testA/"
$cmdtIn false @test="testA/" 2> /dev/null

$cmdtIn echo another test @test="testB/"
$cmdtIn echo ignored3 @ignore @test="testA/"
$cmdtIn echo interlaced test @test="testA/"
$cmdtIn false @test="testB/" 2> /dev/null
$cmdtIn true @test="testB/"
$cmdtIn false @test="testA/" 2> /dev/null

$cmdtIn @init=interlaced
# should have 1 success 2 failures and 3 ignored
$cmdtIn @test="interlaced/" @fail @stderr:"1 success" @stderr:"2 failure" @stderr:"3 ignored" @-- $cmdt1 @report="testA"
# should have 2 success 1 failure
$cmdtIn @test="interlaced/" @fail @stderr:"2 success" @stderr:"1 failure" @stderr!:"ignored" @-- $cmdt1 @report="testB"


>&2 echo "## Mutually exclusive rules"
merExpectedMsgRule="@stderr~/mutually exclusives/"
# Actions are mutually exclusives
#eval $( $cmdtIn @init=mutually_exclusive_rules @exportToken )
$cmdtIn @init=mutually_exclusive_rules
$cmdtIn @test=mutually_exclusive_rules/ @fail "$merExpectedMsgRule" @-- $cmdt1 @global @init
$cmdtIn @test=mutually_exclusive_rules/ @fail "$merExpectedMsgRule" @-- $cmdt1 @init @global
$cmdtIn @test=mutually_exclusive_rules/ @fail "$merExpectedMsgRule" @-- $cmdt1 @init @test
$cmdtIn @test=mutually_exclusive_rules/ @fail "$merExpectedMsgRule" @-- $cmdt1 @init @report
$cmdtIn @test=mutually_exclusive_rules/ @fail "$merExpectedMsgRule" @-- $cmdt1 @global @test
$cmdtIn @test=mutually_exclusive_rules/ @fail "$merExpectedMsgRule" @-- $cmdt1 @global @report
$cmdtIn @test=mutually_exclusive_rules/ @fail "$merExpectedMsgRule" @-- $cmdt1 @test @report

# Assertions are only accepted on test actions
for action in @global @init @report; do
	for assertion in @success @fail @exit=0 @cmd=true @stdout= @stderr= @exists=foo; do
		$cmdtIn @test=mutually_exclusive_rules/ @fail "$merExpectedMsgRule" @-- $cmdt1 "$action" "$assertion"
	done
done

# Suite Config are only accepted on global and init actions
for action in @test @report; do
	cmd=""
	if [ "$action" = "@test" ]; then
		cmd="true"
	fi
	for assertion in @fork=5; do
		$cmdtIn @test=mutually_exclusive_rules/ @fail "$merExpectedMsgRule" @-- $cmdt1 "$action" "$assertion" $cmd
	done

done

# Test config are only accepted on global init and test actions
for action in @report; do
	for assertion in @quiet @keepStdout @keepStderr @keepOutputs @stopOnFailure @ignore @timeout=1s @runCount=2 @parallel=3; do
		$cmdtIn @test=mutually_exclusive_rules/ @fail "$merExpectedMsgRule" @-- $cmdt1 "$action" "$assertion"
	done
done


>&2 echo "## Test flow"
$cmdt @init=main 2> /dev/null # clear main test suite
$cmdtIn @init=test_flow
$cmdtIn @test=test_flow/ @stderr:"#01" @stderr:PASSED @-- $cmdt1 @fail false
$cmdtIn @test=test_flow/ @stderr:"#02" @stderr:PASSED @-- $cmdt1 true
$cmdtIn @test=test_flow/ @fail @stderr:"mutually exclusive" @-- $cmdt1 "@test" "@fork=5" # Should error because of bad param
$cmdtIn @test=test_flow/ @stderr:"#04" @stderr:ERROR @-- $cmdt1 doNotExists @stderr:"not executed" # Should error because of not executable
$cmdtIn @test=test_flow/ @stderr:"#05" @stderr:PASSED @-- $cmdt1 true
$cmdtIn @test=test_flow/ @fail @stderr:"3 success" @stderr:"2 error" @-- $cmdt1 @report=main
$cmdtIn @test=test_flow/ @stderr:"#01" @stderr:PASSED @-- $cmdt1 true
$cmdtIn @test=test_flow/ @stderr:"1 success" @-- $cmdt1 @report=main


>&2 echo "## Test @mock"
mockCfg1="@mock=ls foo,stdin=,stdout=baz,exit=41"
mockCfg2="@mock=ls foo,cmd=sh -c 'echo -n baz; exit 42'"
mockCfg3="@mock=ls foo *,cmd=sh -c 'echo -n baz; exit 43'"
mockCfg4="@mock=ls foo,stdin=baz,exit=44"
mockCfg5="@mock=ls foo,stdin:baz,exit=44"
mockCfg6="@mock:ls foo bar,stdout=baz,exit=46"
mockCfg7="@mock:ls foo bar *,stdout=baz,exit=47"
rm -f @-- foo bar baz 2> /dev/null || true
#expectedFooErrMsg="cannot access 'foo'"
#expectedBarErrMsg="cannot access 'bar'"
#expectedBazErrMsg="cannot access 'baz'"
expectedFooErrMsg="$( 2>&1 ls foo || true )"
expectedBarErrMsg="$( 2>&1 ls bar || true )"
expectedBazErrMsg="$( 2>&1 ls baz || true )"
echo foo > /tmp/fooFileContent
echo baz > /tmp/bazFileContent

$cmdtIn @init=cmd_mock #@verbose=4
$cmdtIn @test=cmd_mock/ @fail @stdout= @stderr:"shell builtin" @-- $cmdt1 echo foo @mock="echo" # cannot mock shell builtin
$cmdtIn @test=cmd_mock/ @fail @stdout= @stderr:"not found" @-- $cmdt1 echo foo @mock="fooNotExists" # cannot mock not found command

$cmdtIn @test=cmd_mock/ @stderr:PASSED @-- $cmdt1 ls @mock="ls,stdout=foo" @stdout=foo @stderr=
$cmdtIn @test=cmd_mock/ @stderr:PASSED @-- $cmdt1 ls @mock="ls,stdout@=/tmp/fooFileContent" @stdout@=/tmp/fooFileContent @stderr=
$cmdtIn @test=cmd_mock/ @stderr:PASSED @-- $cmdt1 ls @mock="ls,stderr=foo" @stdout= @stderr=foo
$cmdtIn @test=cmd_mock/ @stderr:PASSED @-- $cmdt1 ls @mock="ls,stderr@=/tmp/fooFileContent" @stdout= @stderr@=/tmp/fooFileContent
$cmdtIn @test=cmd_mock/ @stderr:PASSED @-- $cmdt1 ls @mock="ls,stdin@=/tmp/fooFileContent,stdout=foo" @stdout!:foo @stderr=
$cmdtIn @test=cmd_mock/ @stderr:PASSED @-- $cmdt1 sh -c "echo foo | ls" @mock="ls,stdin@=/tmp/fooFileContent,stdout=foo" @stdout=foo @stderr=
$cmdtIn @test=cmd_mock/ @stderr:PASSED @-- $cmdt1 sh -c "cat /tmp/fooFileContent | ls" @mock="ls,stdin@=/tmp/fooFileContent,stdout=foo" @stdout=foo @stderr=
$cmdtIn @test=cmd_mock/ @stderr:PASSED @-- $cmdt1 sh -c "echo fooo | ls" @mock="ls,stdin@=/tmp/fooFileContent,stdout=foo" @stdout!:foo @stderr=
$cmdtIn @test=cmd_mock/ @stderr:PASSED @-- $cmdt1 sh -c "cat /tmp/fooFileContent | ls" @mock="ls,stdin@:/tmp/fooFileContent,stdout=foo" @stdout=foo @stderr=
$cmdtIn @test=cmd_mock/ @stderr:PASSED @-- $cmdt1 sh -c "echo fooo | ls" @mock="ls,stdin@:/tmp/fooFileContent,stdout=foo" @stdout=foo @stderr=
$cmdtIn @test=cmd_mock/ @stderr:PASSED @-- $cmdt1 ls @mock="ls"
$cmdtIn @test=cmd_mock/ @stderr:PASSED @-- $cmdt1 ls @mock="ls,exit=1" @fail
$cmdtIn @test=cmd_mock/ @fail @stderr:"absolute path" @-- $cmdt1 ls @mock="/bin/ls" # cannot mock absolute path outside container

$cmdtIn @test=cmd_mock/ @stderr:PASSED @-- $cmdt1 ls @mock="ls,exit=42" @mock="ls foo,exit=43" @exit=42
$cmdtIn @test=cmd_mock/ @stderr:PASSED @-- $cmdt1 ls @mock="ls foo,exit=43" @mock="ls,exit=42" @exit=42
$cmdtIn @test=cmd_mock/ @stderr:PASSED @-- $cmdt1 ls foo @mock="ls,exit=42" @mock="ls foo,exit=43" @exit=43
$cmdtIn @test=cmd_mock/ @stderr:PASSED @-- $cmdt1 ls foo @mock="ls foo,exit=43" @mock="ls,exit=42" @exit=43

$cmdtIn @test=cmd_mock/ @stderr:PASSED @-- $cmdt1 ls @mock:"ls,exit=42" @mock:"ls foo,exit=43" @exit=42
$cmdtIn @test=cmd_mock/ @stderr:PASSED @-- $cmdt1 ls @mock:"ls foo,exit=43" @mock:"ls,exit=42" @exit=42
$cmdtIn @test=cmd_mock/ @stderr:PASSED @-- $cmdt1 ls foo @mock:"ls,exit=42" @mock:"ls foo,exit=43" @exit=43
$cmdtIn @test=cmd_mock/ @stderr:PASSED @-- $cmdt1 ls foo @mock:"ls foo,exit=43" @mock:"ls,exit=42" @exit=43

$cmdtIn @test=cmd_mock/ @stderr:PASSED @-- $cmdt1 ls @mock="ls *,exit=42" @mock="ls foo *,exit=43" @exit=42
$cmdtIn @test=cmd_mock/ @stderr:PASSED @-- $cmdt1 ls @mock="ls foo *,exit=43" @mock="ls *,exit=42" @exit=42
$cmdtIn @test=cmd_mock/ @stderr:PASSED @-- $cmdt1 ls foo @mock="ls *,exit=42" @mock="ls foo *,exit=43" @exit=42
$cmdtIn @test=cmd_mock/ @stderr:PASSED @-- $cmdt1 ls foo @mock="ls foo *,exit=43" @mock="ls *,exit=42" @exit=43

$cmdtIn @test=cmd_mock/ @stderr:PASSED @-- $cmdt1 ls foo bar @mock:"ls,exit=42" @mock:"ls foo,exit=43" @mock:"ls foo bar,exit=44" @exit=44
$cmdtIn @test=cmd_mock/ @stderr:PASSED @-- $cmdt1 ls bar foo @mock:"ls,exit=42" @mock:"ls foo,exit=43" @mock:"ls foo bar,exit=44" @exit=44
$cmdtIn @test=cmd_mock/ @stderr:PASSED @-- $cmdt1 ls bar foo @mock:"ls *,exit=42" @mock:"ls foo,exit=43" @mock:"ls foo bar,exit=44" @exit=42
$cmdtIn @test=cmd_mock/ @stderr:PASSED @-- $cmdt1 ls foo bar @mock:"ls,exit=42" @mock:"ls foo *,exit=43" @mock:"ls foo bar,exit=44" @exit=43
$cmdtIn @test=cmd_mock/ @stderr:PASSED @-- $cmdt1 ls bar foo @mock:"ls,exit=42" @mock:"ls foo *,exit=43" @mock:"ls foo bar,exit=44" @exit=43

$cmdtIn @test=cmd_mock/ @stderr:PASSED @-- $cmdt1 sh -c "echo \${PATH}" "$mockCfg1" "@stdout~/__mock_\d+:/"
$cmdtIn @test=cmd_mock/ @stderr:PASSED @-- $cmdt1 which ls "$mockCfg1" @stderr= "@stdout~|__mock_\d+/ls|"
$cmdtIn @test=cmd_mock/ @stderr:PASSED @-- $cmdt1 sh -c "ls foo" "$mockCfg1" @stdout=baz @exit=41 @keepOutputs
$cmdtIn @test=cmd_mock/ @stderr:PASSED @-- $cmdt1 ls foo "$mockCfg1" @stdout=baz @exit=41
$cmdtIn @test=cmd_mock/ @stderr:PASSED @-- $cmdt1 sh -c "echo foo | ls foo" "$mockCfg1" @fail @stdout= @stderr:"$expectedFooErrMsg"
$cmdtIn @test=cmd_mock/ @stderr:PASSED @-- $cmdt1 ls bar "$mockCfg1" @fail @stderr:"$expectedBarErrMsg"
$cmdtIn @test=cmd_mock/ @stderr:PASSED @-- $cmdt1 ls bar foo "$mockCfg1" @fail @stderr:"$expectedBarErrMsg" @stderr:"$expectedFooErrMsg"
$cmdtIn @test=cmd_mock/ @stderr:PASSED @-- $cmdt1 ls foo bar "$mockCfg1" @fail @stderr:"$expectedBarErrMsg" @stderr:"$expectedFooErrMsg"

$cmdtIn @test=cmd_mock/ @stderr:PASSED @-- $cmdt1 ls foo "$mockCfg2" @stdout=baz @exit=42

$cmdtIn @test=cmd_mock/ @stderr:PASSED @-- $cmdt1 ls foo "$mockCfg3" @stdout=baz @exit=43
$cmdtIn @test=cmd_mock/ @stderr:PASSED @-- $cmdt1 ls bar "$mockCfg3" @fail @stderr:"$expectedBarErrMsg"
$cmdtIn @test=cmd_mock/ @stderr:PASSED @-- $cmdt1 ls bar foo "$mockCfg3" @fail @stderr:"$expectedBarErrMsg"
$cmdtIn @test=cmd_mock/ @stderr:PASSED @-- $cmdt1 ls foo bar "$mockCfg3" @stdout=baz @exit=43

$cmdtIn @test=cmd_mock/ @stderr:PASSED @-- $cmdt1 ls foo "$mockCfg4" @fail @stderr:"$expectedFooErrMsg"
$cmdtIn @test=cmd_mock/ @stderr:PASSED @-- $cmdt1 sh -c "echo bazo | ls foo" "$mockCfg4" @exit=2
$cmdtIn @test=cmd_mock/ @stderr:PASSED @-- $cmdt1 sh -c "echo baz | ls foo" "$mockCfg4" @stderr= @exit=44
$cmdtIn @test=cmd_mock/ @stderr:PASSED @-- $cmdt1 sh -c "echo bazo | ls foo" "$mockCfg5" @stderr= @exit=44
$cmdtIn @test=cmd_mock/ @stderr:PASSED @-- $cmdt1 sh -c "echo baz | ls foo" "$mockCfg5" @stderr= @exit=44

$cmdtIn @test=cmd_mock/ @stderr:PASSED @-- $cmdt1 ls foo "$mockCfg6" @fail @stderr:"$expectedFooErrMsg"
$cmdtIn @test=cmd_mock/ @stderr:PASSED @-- $cmdt1 ls bar "$mockCfg6" @fail @stderr:"$expectedBarErrMsg"
$cmdtIn @test=cmd_mock/ @stderr:PASSED @-- $cmdt1 ls bar foo "$mockCfg6" @stdout=baz @exit=46
$cmdtIn @test=cmd_mock/ @stderr:PASSED @-- $cmdt1 ls foo bar "$mockCfg6" @stdout=baz @exit=46
$cmdtIn @test=cmd_mock/ @stderr:PASSED @-- $cmdt1 ls foo bar baz "$mockCfg6" @fail @stderr:"$expectedFooErrMsg" @stderr:"$expectedBarErrMsg" @stderr:"$expectedBazErrMsg"

$cmdtIn @test=cmd_mock/ @stderr:PASSED @-- $cmdt1 ls foo "$mockCfg7" @fail @stderr:"$expectedFooErrMsg"
$cmdtIn @test=cmd_mock/ @stderr:PASSED @-- $cmdt1 ls bar "$mockCfg7" @fail @stderr:"$expectedBarErrMsg"
$cmdtIn @test=cmd_mock/ @stderr:PASSED @-- $cmdt1 ls bar foo "$mockCfg7" @stdout=baz @exit=47
$cmdtIn @test=cmd_mock/ @stderr:PASSED @-- $cmdt1 ls foo bar "$mockCfg7" @stdout=baz @exit=47
$cmdtIn @test=cmd_mock/ @stderr:PASSED @-- $cmdt1 ls foo bar baz "$mockCfg7" @stdout=baz @exit=47

$cmdtIn @test=cmd_mock/ @fail @-- $cmdt1 @report=main


>&2 echo "## Test @before & @after"
testFile="/tmp/thisFileDoesNotExistsYet.txt"
testFile2="/tmp/thisFileDoesNotExistsYet2.txt"
rm -f @-- "$testFile" "$testFile2" 2> /dev/null || true
$cmdtIn @init=before_after
$cmdtIn @test=before_after/ @stderr:PASSED @-- $cmdt1 ls "$testFile" @fail
$cmdtIn @test=before_after/ @stderr:PASSED @-- $cmdt1 ls "$testFile" @before="touch $testFile"
$cmdtIn @test=before_after/ @stderr:PASSED @-- $cmdt1 ls "$testFile"
$cmdtIn @test=before_after/ @stderr:PASSED @-- $cmdt1 ls "$testFile" @after="rm -f -- $testFile"
$cmdtIn @test=before_after/ @stderr:PASSED @-- $cmdt1 ls "$testFile" @fail
$cmdtIn @test=before_after/ @stderr:PASSED @-- $cmdt1 ls "$testFile" "$testFile2" @before="touch $testFile" @before="touch $testFile2"

$cmdtIn @test=before_after/ @-- $cmdt1 @report=main


>&2 echo "## Test @container"
$cmdtIn @init=container #@keepOutputs #@debug=4 #@ignore #@keepOutputs
$cmdtIn @test=container/run_off_container @stderr:PASSED @-- $cmdt1 sh -c "cat --help 2>&1 | head -1" @stdout!:BusyBox
$cmdtIn @test=container/run_in_container @stderr:PASSED @-- $cmdt1 @container sh -c "cat --help 2>&1 | head -1" @stdout:BusyBox
$cmdtIn @test=container/ @stderr:PASSED @-- $cmdt1 @container true
$cmdtIn @test=container/ @stderr:FAILED @-- $cmdt1 @container false
$cmdtIn @test=container/ @stderr:PASSED @-- $cmdt1 ls /etc/alpine-release @fail
$cmdtIn @test=container/ @stderr:PASSED @-- $cmdt1 @container=alpine ls /etc/alpine-release
$cmdtIn @test=container/ @stderr:PASSED @-- $cmdt1 @container=alpine true
$cmdtIn @test=container/ @stderr:PASSED @-- $cmdt1 @container=alpine ls @mock=ls,exit=0
$cmdtIn @test=container/ @stderr:FAILED @-- $cmdt1 @container=alpine ls @mock=ls,exit=1
#$cmdtIn @test=container/ @stderr:PASSED @-- $cmdt1 @container=alpine sh -c "ls -l /bin/cat*" @mock=/bin/cat,exit=42 @stdout= @debug=4
$cmdtIn @test=container/ @stderr:PASSED @-- $cmdt1 @container=alpine /bin/cat @mock=/bin/cat,exit=42 @exit=42 @debug=0
$cmdtIn @test=container/ @stderr:PASSED @-- $cmdt1 @container=alpine cat @mock=/bin/cat,exit=42 @exit=42

$cmdtIn @test=container/ @fail @-- $cmdt1 @report=main


>&2 echo "## Test @container without exported token"
token="$__CMDT_TOKEN"
export -n __CMDT_TOKEN

$cmdtIn @init=container_wo_token #@keepOutputs #@ignore #@keepOutputs
$cmdtIn @test=container_wo_token/run_in_container @stderr:PASSED @-- $cmdt1 @container sh -c "cat --help 2>&1 | head -1" @stdout:BusyBox
$cmdtIn @test=container_wo_token/ @stderr:PASSED @-- $cmdt1 @container true
$cmdtIn @test=container_wo_token/ @stderr:PASSED @-- $cmdt1 @container @fail false

#$cmdtIn @test=container_wo_token/ @fail @stderr:"$nothingToReportExpectedStderrMsg" @-- $cmdt1 @report=main
$cmdtIn @test=container_wo_token/ @stderr:"3 success" @-- $cmdt1 @report=main

export __CMDT_TOKEN="$token"


>&2 echo "## Test @dirtyContainer"
testFile="/tmp/thisFileDoesNotExistsYet.txt"
hostFile="/tmp/thisFileExistsOnHost.txt"
rm -f @-- "$testFile" 2> /dev/null || true
touch "$hostFile"
$cmdtIn @init=ephemeralContainer #@keepOutputs #@ignore #@keepOutputs
$cmdtIn @test=ephemeralContainer/run_in_container @stderr:PASSED @-- $cmdt1 @container sh -c "cat --help 2>&1 | head -1" @stdout:BusyBox #check run inside container
$cmdtIn @test=ephemeralContainer/ @stderr:PASSED @-- $cmdt1 ls "$hostFile" @stdout:"$hostFile" # file exists on host
$cmdtIn @test=ephemeralContainer/ @stderr:PASSED @-- $cmdt1 ls "$testFile" @fail @stdout= @stderr:"$testFile" # file should not exist on host
$cmdtIn @test=ephemeralContainer/ @stderr:PASSED @-- $cmdt1 @container ls "$testFile" @fail @stdout= @stderr:"$testFile" # file should not exist in container
$cmdtIn @test=ephemeralContainer/ @stderr:PASSED @-- $cmdt1 @container ls "$hostFile" @fail @stdout= @stderr:"$hostFile" # file should not exist in container
$cmdtIn @test=ephemeralContainer/ @stderr:PASSED @-- $cmdt1 @container touch "$testFile" # create file in ephemeral container
$cmdtIn @test=ephemeralContainer/ @stderr:PASSED @-- $cmdt1 @container ls "$testFile" @fail @stdout= @stderr:"$testFile" # file should not exist in container
$cmdtIn @test=ephemeralContainer/ @stderr:PASSED @-- $cmdt1 @container ls "$hostFile" @fail @stdout= @stderr:"$hostFile" # file should not exist in container
$cmdtIn @test=ephemeralContainer/ @stderr:PASSED @-- $cmdt1 ls "$testFile" @fail @stdout= @stderr:"$testFile" # file should not exist on host
$cmdtIn @test=ephemeralContainer/ @-- $cmdt1 @report=main

$cmdtIn @init=suiteContainer #@keepOutputs
$cmdt @init=sub @container 2> /dev/null # container should live the test suite
$cmdtIn @test=suiteContainer/run_in_container @stderr:PASSED @-- $cmdt1 @test=sub/ sh -c "cat --help 2>&1 | head -1" @stdout:BusyBox #check run inside container
$cmdtIn @test=suiteContainer/ @stderr:PASSED @-- $cmdt1 @test=sub/ ls "$testFile" @fail @stdout= @stderr:"$testFile" # file should not exist in suite container
$cmdtIn @test=suiteContainer/ @stderr:PASSED @-- $cmdt1 @test=sub/ ls "$hostFile" @fail @stdout= @stderr:"$hostFile" # file should not exist in suite container
$cmdtIn @test=suiteContainer/ @stderr:PASSED @-- $cmdt1 @test=sub/ touch "$testFile" # create file in suite container
$cmdtIn @test=suiteContainer/ @stderr:PASSED @-- $cmdt1 @test=sub/ ls "$testFile" @stdout:"$testFile" # file should exist in suite container
$cmdtIn @test=suiteContainer/ @stderr:PASSED @-- $cmdt1 @test=sub/ ls "$hostFile" @fail @stdout= @stderr:"$hostFile" # file should not exist in suite container
$cmdtIn @test=suiteContainer/ @stderr:PASSED @-- $cmdt1 @test=sub/ ls "$testFile" @container @fail @stdout= @stderr:"$testFile" # file should not exists in ephemeral container
$cmdtIn @test=suiteContainer/ @stderr:PASSED @-- $cmdt1 @test=sub/ ls "$testFile" @stdout:"$testFile" @debug=0 # file should exist in suite container
$cmdtIn @test=suiteContainer/ @-- $cmdt1 @report=sub

$cmdtIn @init=dirtyContainer #@keepOutputs
$cmdt @init=sub @container 2> /dev/null # container should live the test suite
$cmdtIn @test=dirtyContainer/run_in_container @stderr:PASSED @-- $cmdt1 @test=sub/ sh -c "cat --help 2>&1 | head -1" @stdout:BusyBox #check run inside container
$cmdtIn @test=dirtyContainer/ @stderr:PASSED @-- $cmdt1 @test=sub/ ls "$testFile" @fail @stderr:"$testFile" # file should not exist in container
$cmdtIn @test=dirtyContainer/ @stderr:PASSED @-- $cmdt1 @test=sub/ ls "$hostFile" @fail @stderr:"$hostFile" # file should not exist in container
$cmdtIn @test=dirtyContainer/ @stderr:PASSED @-- $cmdt1 @test=sub/ touch "$testFile" # create file in suite container
$cmdtIn @test=dirtyContainer/ @stderr:PASSED @-- $cmdt1 @test=sub/ ls "$testFile" # file should exist in suite container
$cmdtIn @test=dirtyContainer/ @stderr:PASSED @-- $cmdt1 @test=sub/ ls "$hostFile" @fail @stderr:"$hostFile" # file should not exist in container
$cmdtIn @test=dirtyContainer/ @stderr:PASSED @-- $cmdt1 @test=sub/ ls "$testFile" @dirtyContainer=afterTest # file should exist in container
$cmdtIn @test=dirtyContainer/ @stderr:PASSED @-- $cmdt1 @test=sub/ ls "$testFile" @fail @stderr:"$testFile" # file should not exist in fresh container
$cmdtIn @test=dirtyContainer/ @stderr:PASSED @-- $cmdt1 @test=sub/ ls "$hostFile" @fail @stderr:"$hostFile" # file should not exist in container
$cmdtIn @test=dirtyContainer/ @stderr:PASSED @-- $cmdt1 @test=sub/ touch "$testFile" # create file in suite container
$cmdtIn @test=dirtyContainer/ @stderr:PASSED @-- $cmdt1 @test=sub/ ls "$testFile" # file should exist in container
$cmdtIn @test=dirtyContainer/ @stderr:PASSED @-- $cmdt1 @test=sub/ ls "$hostFile" @fail @stderr:"$hostFile" # file should not exist in container
$cmdtIn @test=dirtyContainer/ @stderr:PASSED @-- $cmdt1 @test=sub/ ls "$testFile" @fail @dirtyContainer=beforeTest # file should not exist in fresh container
$cmdtIn @test=dirtyContainer/ @-- $cmdt1 @report=sub

$cmdtIn @init=testContainer #@keepOutputs
$cmdt @init=sub @container @dirtyContainer=beforeTest 2> /dev/null # container should live for each test
$cmdtIn @test=testContainer/run_in_container @stderr:PASSED @-- $cmdt1 @test=sub/ sh -c "cat --help 2>&1 | head -1" @stdout:BusyBox #check run inside container
$cmdtIn @test=testContainer/ @stderr:PASSED @-- $cmdt1 @test=sub/ ls "$testFile" @fail @stderr:"$testFile" # file should not exist in container
$cmdtIn @test=testContainer/ @stderr:PASSED @-- $cmdt1 @test=sub/ ls "$hostFile" @fail @stderr:"$hostFile" # file should not exist in container
$cmdtIn @test=testContainer/ @stderr:PASSED @-- $cmdt1 @test=sub/ touch "$testFile" # create file in test container
$cmdtIn @test=testContainer/ @stderr:PASSED @-- $cmdt1 @test=sub/ ls "$testFile" @fail @stderr:"$testFile" # file should not exist in container
$cmdtIn @test=testContainer/ @stderr:PASSED @-- $cmdt1 @test=sub/ ls "$hostFile" @fail @stderr:"$hostFile" # file should not exist in container
$cmdtIn @test=testContainer/ @-- $cmdt1 @report=sub

>&2 echo "## Reporting all"
$cmdt @report= ; >&2 echo SUCCESS ; exit 0


