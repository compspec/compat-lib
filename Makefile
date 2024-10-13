COMMONENVVAR=GOOS=$(shell uname -s | tr A-Z a-z)
CPP = $(shell which cpp)

.PHONY: all
all: build

.PHONY: build
build: 
	mkdir -p ./bin
	go build -o ./bin/clib-gen cmd/gen/gen.go
	

.PHONY: test
test:
	bats -t test/bats/cli.bats
