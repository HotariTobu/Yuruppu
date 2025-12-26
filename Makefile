.PHONY: test test-integration check compile-all preflight

test:
	go test ./...

test-integration:
	go test -tags=integration ./...

check:
	go fmt ./...
	go vet ./...

# Compile all files including integration-tagged test files without running tests.
# Uses 'go test -run=^$' because 'go build' skips _test.go files.
compile-all:
	go test -tags=integration -run='^$$' ./...

preflight: check compile-all test