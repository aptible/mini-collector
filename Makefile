GOFILES_NOVENDOR = $(shell find . -type f -name '*.go' -not -path "./vendor/*")
SHELL=/bin/bash

.PHONY: deps
deps:
	dep ensure

.PHONY: build
build: $(GOFILES_NOVENDOR)
	go list ./...  | grep cmd | xargs -P $$(nproc) -n 1 -- go build -i

.PHONY: unit
unit: $(GOFILES_NOVENDOR)
	go test $$(go list ./... | grep -v /vendor/)
	go vet $$(go list ./... | grep -v /vendor/)

.PHONY: test
test: unit
	@true
