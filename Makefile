SHELL=/bin/bash

.PHONY: deps
deps:
	go mod vendor

.PHONY: build
build:
	GOOS="linux" CGO_ENABLED=0 GOARCH="amd64" go build -i cmd/aggregator
	GOOS="linux" CGO_ENABLED=0 GOARCH="amd64" go build -i cmd/mini-collector

writer/influxdb/api.proto.influxdb_formatter.go \
writer/datadog/api.proto.datadog_formatter.go \
publisher/api.proto.publisher_formatter.go: \
api/api.proto .codegen/emit.py \
.codegen/influxdb_formatter.go.jinja2 \
.codegen/datadog_formatter.go.jinja2 \
.codegen/publisher_formatter.go.jinja2
	protoc -I api api/api.proto --plugin=protoc-gen-custom=./.codegen/emit.py --custom_out=.
	find . -name "api.proto.*_formatter.go" | xargs gofmt -l -w

api/api.pb.go: api/api.proto
	protoc -I api/ api/api.proto --go_out=plugins=grpc:api

.PHONY: test
test:
	go test ./...
	go vet ./...

.PHONY: fmt
fmt:
	gofmt -l -w ./...

.DEFAULT_GOAL := test
