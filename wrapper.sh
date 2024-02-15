#! /bin/sh

>&2 echo "$@"

if [ "$1" = "echo" ] && [ "$2" = "foo" ]; then
	echo "baz"
	exit 0
fi


"$@"

