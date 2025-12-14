.PHONY: test check preflight

test:
	go test ./...

check:
	go fmt ./...
	go vet ./...

preflight: check test