SHELL := /bin/bash

PREFIX = hpa-controller

PACKAGES = $(shell go list ./... | grep -v /vendor/)
TESTARGS ?= -race

CURRENTDIR = $(shell pwd)
SOURCEDIR = $(CURRENTDIR)
APP_SOURCES := $(shell find $(SOURCEDIR) -name '*.go' -not -path '$(SOURCEDIR)/vendor/*')

PATH := $(CURRENTDIR)/bin:$(PATH)

VERSION?=$(shell git describe --tags)

LD_FLAGS = -ldflags "-X main.VERSION=$(VERSION) -s -w"

all: build

.PHONY: clean build docker check
default: build
build: dist/hpa-controller


dist/hpa-controller:
	mkdir -p $(@D)
	CGO_ENABLED=0 GOOS=linux go build $(LD_FLAGS) -v -o dist/hpa-controller

docker:
	docker build -t $(PREFIX):$(VERSION) .

$(PACKAGES): check-deps format
	go test $(TESTARGS) $@
	cd $(GOPATH)/src/$@; gometalinter --deadline  720s --vendor -D gotype -D dupl -D gocyclo -D gas -D errcheck

check-deps:
	@which gometalinter > /dev/null || \
	(go get github.com/alecthomas/gometalinter && gometalinter --install)
