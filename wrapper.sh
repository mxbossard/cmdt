#! /bin/sh

>&2 echo "$@"

#if [ "$1" = "echo" ] && [ "$2" = "foo" ]; then
if [ "$1" = "echo" ] && echo "$@" | grep "foo" > /dev/null; then
	echo "baz"
	exit 0
fi


"$@"

