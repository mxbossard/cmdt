#! /bin/bash
set -e -o pipefail
scriptDir=$( dirname $( readlink -f $0 ) )

# Trusted cmdt to works
cmdt="cmdt"
if [ -n "$1" ]; then
	cmdt="$( which $1 )"
fi

>&2 echo "## Asserting [$cmdt] cmdt binary mandatory features ..."

errorCount=0
addError() {
	errorCount=$(( errorCount + 1 ))
	>&2 echo
	>&2 echo "ERROR: $1"
	>&2 echo
}

rm -rf -- /tmp/cmdt* /tmp/cmdt.log /tmp/daemon.log 2> /dev/null || true

# Clear context
export -n __CMDT_TOKEN
#$cmdt @init=main

assertions() {
	>&2 echo "## Test cmdt basic assertions should passed"
	$cmdt @init=should_succeed @stopOnFailure=false

	$cmdt @test=should_succeed/ true
	$cmdt @test=should_succeed/ true @success
	$cmdt @test=should_succeed/ false @fail
	$cmdt @test=should_succeed/ true @exit=0
	$cmdt @test=should_succeed/ false @exit=1

	$cmdt @test=should_succeed/ echo foo bar @stdout:foo @stderr=
	$cmdt @test=should_succeed/ echo foo bar @stdout:bar
	$cmdt @test=should_succeed/ echo foo bar @stdout!:baz
	$cmdt @test=should_succeed/ echo foo bar @stdout!=baz
	$cmdt @test=should_succeed/ echo foo bar @stdout~/^foo/ @stderr=
	$cmdt @test=should_succeed/ echo foo bar @stdout~/BaR/i
	$cmdt @test=should_succeed/ echo foo bar @stdout~"/^foo bar\n$/"
	$cmdt @test=should_succeed/ echo foo bar @stdout~"/^foo bar$/m"
	$cmdt @test=should_succeed/ echo foo bar @stdout!~/bar$/
	$cmdt @test=should_succeed/ echo foo\nbar\nbaz @stdout!~/^bar$/
	$cmdt @test=should_succeed/ echo foo\nbar\nbaz @stdout!~/^bar$/m
	$cmdt @test=should_succeed/ echo foo bar @stdout:foo @stdout:bar @stderr=
	$cmdt @test=should_succeed/ echo foo bar @stdout="foo bar\n" @stderr=

	$cmdt @test=should_succeed/ sh -c ">&2 echo foo bar" @stderr:foo @stdout=
	$cmdt @test=should_succeed/ sh -c ">&2 echo foo bar" @stderr:bar
	$cmdt @test=should_succeed/ sh -c ">&2 echo foo bar" @stderr!:baz
	$cmdt @test=should_succeed/ sh -c ">&2 echo foo bar" @stderr!=baz
	$cmdt @test=should_succeed/ sh -c ">&2 echo foo bar" @stderr~/^foo/ @stdout=
	$cmdt @test=should_succeed/ sh -c ">&2 echo foo bar" @stderr~/BaR/i
	$cmdt @test=should_succeed/ sh -c ">&2 echo foo bar" @stderr~"/^foo bar\n$/"
	$cmdt @test=should_succeed/ sh -c ">&2 echo foo bar" @stderr!~/bar$/
	$cmdt @test=should_succeed/ sh -c ">&2 echo foo bar" @stderr:foo @stderr:bar
	$cmdt @test=should_succeed/ sh -c ">&2 echo foo bar" @stderr="foo bar\n" @stdout=

	>&2 echo "## Test cmdt basic assertions should ignore"
	$cmdt @init=should_ignore @failuresLimit=-1 @verbose=0

	$cmdt 2> /dev/null @test=should_ignore/ true
	$cmdt 2> /dev/null @test=should_ignore/ @ignore false
	$cmdt 2> /dev/null @test=should_ignore/ true

	>&2 echo "## Test cmdt basic assertions should failed"
	$cmdt @init=should_fail @failuresLimit=-1 @verbose=0

	$cmdt 2> /dev/null @test=should_fail/ false
	$cmdt 2> /dev/null @test=should_fail/ true @fail
	$cmdt 2> /dev/null @test=should_fail/ false @success
	$cmdt 2> /dev/null @test=should_fail/ true @exit=1
	$cmdt 2> /dev/null @test=should_fail/ false @exit=0

	$cmdt 2> /dev/null @test=should_fail/ echo foo bar @stdout=
	$cmdt 2> /dev/null @test=should_fail/ echo foo bar @stdout=foo
	$cmdt 2> /dev/null @test=should_fail/ echo foo bar @stdout=foo bar
	$cmdt 2> /dev/null @test=should_fail/ echo foo bar @stdout:baz
	$cmdt 2> /dev/null @test=should_fail/ echo foo bar @stdout:foo @stdout:baz
	$cmdt 2> /dev/null @test=should_fail/ echo foo bar @stderr:foo

	$cmdt 2> /dev/null @test=should_fail/ sh -c ">&2 echo foo bar" @stderr=
	$cmdt 2> /dev/null @test=should_fail/ sh -c ">&2 echo foo bar" @stderr=foo
	$cmdt 2> /dev/null @test=should_fail/ sh -c ">&2 echo foo bar" @stderr=foo bar
	$cmdt 2> /dev/null @test=should_fail/ sh -c ">&2 echo foo bar" @stderr:baz
	$cmdt 2> /dev/null @test=should_fail/ sh -c ">&2 echo foo bar" @stderr:foo @stderr:baz
	$cmdt 2> /dev/null @test=should_fail/ sh -c ">&2 echo foo bar" @stdout:foo

	>&2 echo "## Test cmdt basic assertions should error"
	$cmdt @init=should_error @failuresLimit=-1 @verbose=0

	! $cmdt @test=should_error/ true @stdout:"" 2> /dev/null || addError "should error because empty contains"
	! $cmdt @test=should_error/ true @stdout~"" 2> /dev/null || addError "should error because empty regex"
	! $cmdt @test=should_error/ true @stderr:"" 2> /dev/null || addError "should error because empty contains"
	! $cmdt @test=should_error/ true @stderr~"" 2> /dev/null || addError "should error because empty regex"

	>&2 echo "## Test cmdt basic assertions should timeout"
	$cmdt @init=should_timeout @failuresLimit=-1 @verbose=0

	$cmdt 2> /dev/null @test=should_timeout/ true
	$cmdt 2> /dev/null @test=should_timeout/ @timeout=0.1s sleep 1
	$cmdt 2> /dev/null @test=should_timeout/ true


	of="/tmp/cmdtReportContent.txt"

	>&2 echo "## reporting should_succeed"
	rc=0
	$cmdt @report=should_succeed > "$of" 2>&1 || rc=$?
	test "$rc" -eq 0 || addError "reporting should_succeed should exit=0"
	grep "28 success" "$of" || addError "reporting should_succeed bad success count"

	>&2 echo "## re reporting should succeed"
	rc=0
	$cmdt @report=should_succeed > "$of" 2>&1 || rc=$?
	test "$rc" -eq 0 || addError "re reporting should_succeed should exit=0"
	#grep "you must perform" "$of" || addError "reporting no test should log a message"
	grep "28 success" "$of" || addError "reporting should_succeed bad success count"

	>&2 echo "## reporting should_ignore"
	rc=0
	$cmdt @report=should_ignore > "$of" 2>&1 || rc=$?
	test "$rc" -eq 0 || addError "reporting should_ignore shoud exit=1"
	grep "2 success" "$of" || addError "reporting should_ignore bad success count"
	grep "1 ignored" "$of" > /dev/null || addError "reporting should_ignore bad ignore count"

	>&2 echo "## reporting should_fail"
	rc=0
	$cmdt @report=should_fail > "$of" 2>&1 || rc=$?
	test "$rc" -eq 1 || addError "reporting should_fail shoud exit=1"
	grep "17 failures" "$of" || addError "reporting should_fail bad failures count"

	>&2 echo "## reporting should_error"
	rc=0
	$cmdt @report=should_error > "$of" 2>&1 || rc=$?
	test "$rc" -eq 1 || addError "reporting should_error should exit=1"
	grep "4 errors" "$of" || addError "reporting should_error bad errors count"

	>&2 echo "## reporting should_timeout"
	rc=0
	$cmdt @report=should_timeout > "$of" 2>&1 || rc=$?
	test "$rc" -eq 1 || addError "reporting should_timeout shoud exit=1"
	grep "2 success" "$of" || addError "reporting should_timeout bad success count"
	grep "1 timeout" "$of" || addError "reporting should_timeout bad timeout count"

	>&2 echo "## reporting all"
	rc=0
	$cmdt @report > "$of" 2>&1 || rc=$?
	test "$rc" -eq 1 || addError "reporting all without test shoud exit=1"
	grep "you must perform" "$of" || addError "reporting all with no test should log a message"
}

logFile="$( mktemp /tmp/cmdt_assertions_XXXXXX.log )"
assertionsRc=0
assertions > "$logFile" 2>&1 || assertionsRc=$?
if [ "$errorCount" -ne 0 ] || [ "$assertionsRc" -ne 0 ]; then
	>&2 echo "---------------------------------------------"
	>&2 echo "!!! Some cmdt mandatory assertions failed !!!"
	>&2 echo "---------------------------------------------"
	>&2 echo "detected $errorCount error(s) ; assertions RC=$assertionsRc"
	>&2 echo
	>&2 cat "$logFile"
	>&2 echo
	>&2 echo "---------------------------------------------"
	>&2 echo "!!! Some cmdt mandatory assertions failed !!!"
	>&2 echo "---------------------------------------------"
	>&2 echo
	exit 1
else
	>&2 echo "Assertions OK"
	>&2 echo
fi

