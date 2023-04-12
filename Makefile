MAKEFLAGS += --warn-undefined-variables
SHELL = /bin/bash -o nounset -o errexit -o pipefail
BIN_PATH = cmd/go2x3
.DEFAULT_GOAL = build

## display this help message
help:
	@echo -e "\033[32m"
	@echo "go2x3"
	@echo
	@awk '/^##.*$$/,/[a-zA-Z_-]+:/' $(MAKEFILE_LIST) | awk '!(NR%2){print $$0p}{p=$$0}' | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-16s\033[0m %s\n", $$1, $$2}' | sort

# ----------------------------------------
# build

GOFILES = $(shell find . -type f -name '*.go')
	
.PHONY: build clean

## build the go2x3 binary
build: $(GOFILES)
	cd ${BIN_PATH} && \
	CGO_ENABLED=0 \
	go build -trimpath

## generate "gold" output for py2x3 scripts
gold: clean $(GOFILES)
	cd ${BIN_PATH} && \
	go test -timeout 1h -run Golden
	
## same as gold but 2x3 catalog dbs not wiped
silver: ## Z, I love you from and to the ends of space and time
	cd ${BIN_PATH} && \
	go test -timeout 1h -run Golden
# ----------------------------------------
# tooling

## remove build artifacts & wipe ALL 2x3 catalog dbs
clean:
	touch  ${BIN_PATH}/main.go
	rm -rf ${BIN_PATH}/catalogs
	rm -rf ${BIN_PATH}/tmp
	rm -f  ${BIN_PATH}/go2x3
	rm -f  ${BIN_PATH}/go2x3.exe
		
	
## install req'd build tools
tools:
	go install github.com/gogo/protobuf/protoc-gen-gogoslick
	go get -d  github.com/gogo/protobuf/proto
#	go get -d  google.golang.org/grpc/cmd/protoc-gen-go-grpc
#	go get -d  google.golang.org/protobuf/cmd/protoc-gen-go
#	go get -d  github.com/gogo/protobuf/protoc-gen-gogo
#	go get -d  github.com/gogo/protobuf/jsonpb
#	go get -d  github.com/gogo/protobuf/gogoproto

## generate code from .proto files
protos: 
	protoc \
		-I='${GOPATH}/src' \
		--gogoslick_opt=paths=source_relative   \
		--gogoslick_out=plugins=grpc:.          \
		--proto_path=.  \
		lib2x3/graph/graph.proto
		