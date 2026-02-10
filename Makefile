.PHONY: 
	test-coverage
	test-integration
	lint
	format
	check-all

test-coverage:
	go test -covermode=count -coverprofile=cover.out ./...
	go tool cover -html=cover.out -o=cover.html
	go tool cover -func=cover.out

test-integration:
	go test -tags=integration_test ./tests/integration/... -p 3

lint: format
	golangci-lint run

format:
	go fmt ./...

check-all: lint test-coverage test-integration