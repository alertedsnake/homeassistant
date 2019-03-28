APPNAME := htcontrol
VERSION ?= 0.2.0
GOPATH  ?= ${HOME}/.gopath
GOBIN   ?= $(GOPATH)/bin

# tools
GOX           := $(GOBIN)/gox
GOX_TMPL      := -output="build/$(APPNAME)_$(VERSION)_{{.OS}}_{{.Arch}}"

export GOPATH
export GO111MODULES=on

all: build

#
# Install the pre-requisites
#

$(GOX):
	@export GOPATH=$(GOPATH)
	@export GOBIN=$(GOBIN)
	go get -u github.com/mitchellh/gox

# find me some source go files in this directory, sorry about this
gofiles = $(shell find . -type f -name \*.go -print)

build/$(APPNAME)_$(VERSION)_linux_arm: $(gofiles)  | $(GOX)
	$(GOX) $(GOX_TMPL) -osarch="linux/arm" -ldflags "$(LDFLAGS) -X main.VERSION=$(VERSION)".

build/$(APPNAME)_$(VERSION)_linux_amd64: $(gofiles) | $(GOX)
	$(GOX) $(GOX_TMPL) -osarch="linux/amd64" -ldflags "$(LDFLAGS) -X main.VERSION=$(VERSION)".

build: build/$(APPNAME)_$(VERSION)_linux_arm build/$(APPNAME)_$(VERSION)_linux_amd64

clean:
	go clean
	@rm -rf build version.go
	@mkdir build

.PHONY: all install-prereqs clean
