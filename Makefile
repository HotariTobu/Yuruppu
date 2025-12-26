.PHONY: fix check compile-all test test-integration preflight

fix:
	golangci-lint run --fix ./...

check:
	golangci-lint run ./...
	govulncheck ./...

# Compile all files including integration-tagged test files without running tests.
# Uses 'go test -run=^$' because 'go build' skips _test.go files.
compile-all:
	go test -tags=integration -run='^$$' ./...

test:
	go test ./...

test-integration:
	go test -tags=integration -run='Integration' ./...

preflight: check compile-all test
