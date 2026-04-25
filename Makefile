.PHONY: build test lint ci

build:
	go build -o vibedeploy .

test:
	go test -v -race -coverprofile=coverage.out ./...

lint:
	go vet ./...

ci: lint build test
