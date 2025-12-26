.PHONY: fix test test-integration check compile-all preflight

fix:
	golangci-lint run --fix ./...

test:
	go test ./...

test-integration:
	go test -tags=integration ./...

check:
	golangci-lint run ./...
	go run golang.org/x/vuln/cmd/govulncheck@latest ./...

# Compile all files including integration-tagged test files without running tests.
# Uses 'go test -run=^$' because 'go build' skips _test.go files.
compile-all:
	go test -tags=integration -run='^$$' ./...

preflight: check compile-all test