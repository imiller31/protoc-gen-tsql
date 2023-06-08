# the name of this package
PKG  := $(shell go list .)
PROTOC_VER := $(shell protoc --version | cut -d' ' -f2)

.PHONY: bootstrap
bootstrap: testdata # set up the project for development

.PHONY: testdata
testdata: testdata/generated # generate all testdata

generate_protos:
	go install google.golang.org/protobuf/cmd/protoc-gen-go

testdata/generated: protoc-gen-tsql
	go install google.golang.org/protobuf/cmd/protoc-gen-go
	rm -rf ./testdata/generated && mkdir -p ./testdata/generated

	protoc -I ./testdata/protos \
		--plugin=protoc-gen-tsql=./bin/protoc-gen-tsql \
		--tsql_out="paths=source_relative:./testdata/generated" \
		./testdata/protos/example/*.proto; \

.PHONY: protoc-gen-tsql
protoc-gen-tsql:
	go install google.golang.org/protobuf/cmd/protoc-gen-go
	rm -rf ./proto-ext-tsql
	rm -rf ./bin/protoc-gen-tsql
	protoc -I ./testdata/protos --go_out="." ./testdata/protos/tsql_options/tsql_options.proto
	go build -o ./bin/protoc-gen-tsql .

bin/protoc-gen-debug: # creates the protoc-gen-debug protoc plugin for output ProtoGeneratorRequest messages
	go build -o ./bin/protoc-gen-debug ./protoc-gen-debug

.PHONY: clean
clean:
	rm -rf bin
	rm -rf testdata/generated
	set -e; for f in `find . -name *.pb.bin`; do \
		rm $$f; \
	done
	set -e; for f in `find . -name *.pb.go`; do \
		rm $$f; \
	done