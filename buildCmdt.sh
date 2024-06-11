#! /bin/bash
set -e -o pipefail
scriptDir=$( dirname $( readlink -f $0 ) )

>&2 echo "##### Building cmdtest binary ..."
export GOBIN="$scriptDir/bin"
cd "$scriptDir"

export BUILT_CMDT_BIN="$GOBIN/cmdt"

#go install
#CGO_ENABLED=0 GOOS=linux go install -a -ldflags '-extldflags "-static"'
#CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go install -ldflags '-extldflags "-static"'
#CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go install -a -tags netgo -ldflags '-w'
#CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go install -a -tags netgo -ldflags '-w -extldflags "-static"'
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o "$BUILT_CMDT_BIN" -tags netgo -ldflags '-w'

cd - > /dev/null

