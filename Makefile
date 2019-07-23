# go option
GO         ?= go
PKG        := go mod
LDFLAGS    := -w -s
PACKAGE    := github.com/tritonmedia/sync
GOFLAGS    :=
TAGS       := 
BINDIR     := $(CURDIR)/bin

# Required for globs to work correctly
SHELL=/bin/bash

.PHONY: build
build:
	@echo " ===> building releases in ./bin/... <=== "
	GOBIN=$(BINDIR) $(GO) build -o $(BINDIR)/sync -v $(GOFLAGS) -tags '$(TAGS)' -ldflags '$(LDFLAGS)' $(PACKAGE)/cmd/...