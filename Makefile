.PHONY: 
	test-coverage
	lint
	format
	check-all

test-coverage:
	go test -covermode=count -coverprofile=cover.out ./...
	go tool cover -html=cover.out -o=cover.html

lint: format
	golangci-lint run

format:
	go fmt ./...

check-all: lint test-coverage