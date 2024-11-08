COMMONENVVAR=GOOS=$(shell uname -s | tr A-Z a-z)
CPP = $(shell which cpp)

## Location to install dependencies to
LOCALBIN ?= $(shell pwd)/bin
$(LOCALBIN):
	mkdir -p $(LOCALBIN)

.PHONY: all
all: build

.PHONY: build
build: 
	mkdir -p ./bin
	go build -o ./bin/fs-gen cmd/fs/fs.go
	go build -o ./bin/compat-gen cmd/gen/gen.go
	go build -o ./bin/compat-server cmd/server/server.go
	go build -o ./bin/compat-cli cmd/client/client.go
	go build -o ./bin/fs-record cmd/record/record.go

.PHONY: protoc
protoc: $(LOCALBIN)
	GOBIN=$(LOCALBIN) go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.28
	GOBIN=$(LOCALBIN) go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.2

# You can use make protoc to download proto
# This will generate the protos in "protos"
.PHONY: proto
proto: protoc
	PATH=$(LOCALBIN):${PATH} protoc --go_out=. --go_opt=paths=source_relative --go-grpc_out=. --go-grpc_opt=paths=source_relative protos/compatibility.proto

.PHONY: test
test:
	bats -t test/bats/cli.bats
