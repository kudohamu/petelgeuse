TARGET_FILES=$(shell go list ./... 2> /dev/null)

test:
	@go test -v ./...

bench:
	@go test -bench="." ./...
