TARGET_FILES=$(shell go list ./... 2> /dev/null)

test:
	@go test -v -race ./...

bench:
	@go test -bench="." ./...
