GOPATH:=$(shell go env GOPATH)
APP?=codeowners

.PHONY: build
## build: build the application
build:
	go build -o build/${APP} cmd/main.go

.PHONY: run
## run: run the application
run:
	go run -v -race cmd/main.go

.PHONY: format
## format: format files
format:
	@go get golang.org/x/tools/cmd/goimports
	goimports -local github.com/jungwinter -w .
	gofmt -s -w .
	go mod tidy

.PHONY: test
## test: run tests
test:
	@go get github.com/rakyll/gotest
	gotest -p 1 -race -cover -v ./...

.PHONY: coverage
## coverage: run tests with coverage
coverage:
	@go get github.com/rakyll/gotest
	gotest -p 1 -race -coverprofile=coverage.txt -covermode=atomic -v ./...

.PHONY: lint
## lint: check everything's okay
lint:
	golangci-lint run ./...
	go mod verify

.PHONY: generate
## generate: generate source code for mocking
generate:
	@go get golang.org/x/tools/cmd/stringer
	@go get github.com/golang/mock/gomock
	@go install github.com/golang/mock/mockgen
	go generate ./...
	${MAKE} format

.PHONY: help
## help: prints this help message
help:
	@echo "Usage: \n"
	@sed -n 's/^##//p' ${MAKEFILE_LIST} | column -t -s ':'
