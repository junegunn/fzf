#/bin/sh
export CGO_ENABLED=1
rm -rf /tmp/go-link-*
rm -rf /tmp/go-build*
go clean -r -cache -testcache -modcache
go build -a -ldflags="-linkmode=internal" .
