#! /bin/sh
set -e

file="/tmp/demo_init_file"

if ! [ -f "$file" ] ; then
	cat <<- EOF > "$file"
foobarbaz
EOF
	>&2 echo "inited $file"
fi

cat "$file"

