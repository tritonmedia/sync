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

.PHONY: release
release:
	gox -output "release/sync-{{ .OS }}-{{ .Arch }}" \
		-osarch "windows/amd64 linux/amd64 darwin/amd64 windows/386 linux/386 darwin/386" \
		$(PACKAGE)/cmd/...