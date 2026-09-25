.PHONY: test test-go test-ts

test: test-go test-ts

test-go:
	cd go && go test ./...

test-ts:
	cd typescript && npm install && npm test
