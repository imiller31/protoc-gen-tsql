# the name of this package
PKG  := $(shell go list .)
PROTOC_VER := $(shell protoc --version | cut -d' ' -f2)

.PHONY: bootstrap
bootstrap: testdata # set up the project for development

.PHONY: testdata
testdata: testdata-graph testdata/generated # generate all testdata

.PHONY: testdata-graph
testdata-graph: bin/protoc-gen-debug # parses the proto file sets in testdata/graph and renders binary CodeGeneratorRequest
	set -e; for subdir in `find ./testdata/graph -mindepth 1 -maxdepth 1 -type d`; do \
		protoc -I ./testdata/graph \
			--plugin=protoc-gen-debug=./bin/protoc-gen-debug \
			--debug_out="$$subdir:$$subdir" \
			`find $$subdir -name "example.proto"`; \
	done

testdata/generated: protoc-gen-go bin/protoc-gen-tsql
	go install google.golang.org/protobuf/cmd/protoc-gen-go
	rm -rf ./testdata/generated && mkdir -p ./testdata/generated
	go install google.golang.org/protobuf/cmd/protoc-gen-go
	rm -rf ./testdata/generated && mkdir -p ./testdata/generated

	protoc -I ./testdata/protos \
		--plugin=protoc-gen-tsql=./bin/protoc-gen-tsql \
		--tsql_out="paths=source_relative:./testdata/generated" \
		./testdata/protos/example/example.proto; \


testdata/fdset.bin:
	@protoc -I ./testdata/protos \
		-o ./testdata/fdset.bin \
		--include_imports \
		testdata/protos/**/*.proto

.PHONY: protoc-gen-go
protoc-gen-go:
	go install google.golang.org/protobuf/cmd/protoc-gen-go

bin/protoc-gen-tsql:
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