#/bin/sh
 CGO_ENABLED=1 go build -a -ldflags="-linkmode=internal" .
