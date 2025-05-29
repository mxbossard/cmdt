#! /bin/bash
set -e -o pipefail
scriptDir=$( dirname $( readlink -f $0 ) )

export GOBIN="$scriptDir/bin"
cd "$scriptDir"

export BUILT_CMDT_BIN="$GOBIN/cmdt"

if [ "$SKIP_CMDT_BUILD" == "true" ]; then
	>&2 echo "Skipped $BUILT_CMDT_BIN cmdt binary build."
else
	>&2 echo "## Building cmdtest binary ..."
	#go install
	#CGO_ENABLED=0 GOOS=linux go install -a -ldflags '-extldflags "-static"'
	#CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go install -ldflags '-extldflags "-static"'
	#CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go install -a -tags netgo -ldflags '-w'
	#CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go install -a -tags netgo -ldflags '-w -extldflags "-static"'
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o "$BUILT_CMDT_BIN" -tags netgo -ldflags '-w'
	>&2 echo "Built $BUILT_CMDT_BIN cmdt binary."
fi

cd - > /dev/null

ls -lh "$BUILT_CMDT_BIN"
>&2 echo

