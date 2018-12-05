APPNAME := htcontrol
VERSION ?= 0.1.0
GOPATH  ?= ${HOME}/.gopath
GOBIN   ?= $(GOPATH)/bin
DEPENDENCIES := $(shell go list -f '{{ join .Imports "\n" }}')

# tools
GOX           := $(GOBIN)/gox
GOX_TMPL      := -output="build/$(APPNAME)_$(VERSION)_{{.OS}}_{{.Arch}}"

export GOPATH

all: install-prereqs build

#
# Install the pre-requisites
#

install-prereqs: install-dependencies

$(GOX):
	@export GOPATH=$(GOPATH)
	@export GOBIN=$(GOBIN)
	go get -u github.com/mitchellh/gox

install-dependencies:
	for package in $(DEPENDENCIES); do \
		go get $$package; \
	done

install-dependencies-latest:
	for package in $(DEPENDENCIES); do \
		go get -u $$package; \
	done

#
# Write the version.go
#
define generate_version_file
    @echo "package main" > $(1)
    @echo "const VERSION = \"$(VERSION)\"" >> $(1)
endef

version.go:
	$(call generate_version_file, $@)


# find me some source go files in this directory, sorry about this
gofiles = version.go $(shell find . \( -path ./build -o -path ./examples \) -prune -o -type f -name \*.go -print)

build/$(APPNAME)_$(VERSION)_linux_arm: $(gofiles)  | $(GOX)
	$(GOX) $(GOX_TMPL) -osarch="linux/arm" .

build/$(APPNAME)_$(VERSION)_linux_amd64: $(gofiles) | $(GOX)
	$(GOX) $(GOX_TMPL) -osarch="linux/amd64" .

build: build/$(APPNAME)_$(VERSION)_linux_arm build/$(APPNAME)_$(VERSION)_linux_amd64

clean:
	go clean
	@rm -rf build version.go
	@mkdir build

.PHONY: all install-prereqs clean
