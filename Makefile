.PHONY: test test-go test-ts

test: test-go test-ts

test-go:
	go test ./...

test-ts:
	npm install
	npm test
