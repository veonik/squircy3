# Makefile for squircy, a proper IRC bot.
# https://code.dopame.me/veonik/squircy3

SUBPACKAGES := cli config event irc plugin vm

PLUGINS ?= $(patsubst plugins/%,%,$(wildcard plugins/*))

SOURCES := $(wildcard cmd/*/*.go) $(wildcard $(patsubst %,%/*.go,$(SUBPACKAGES)))
GENERATOR_SOURCES := $(wildcard cmd/squircy/defconf/*)

OUTPUT_BASE := out

RACE      ?= -race
TEST_ARGS ?= -count 1

# Include PLUGIN_TYPE=linked in the command-line when invoking make to link
# extra plugins directly in the main binary rather than generating shared object files.
PLUGIN_TYPE ?= shared

GOOS   ?= $(shell go env GOOS)
GOARCH ?= $(shell go env GOARCH)
GOARM  ?= $(shell go env GOARM)
CC     ?= $(shell go env CC)
PACKR  ?= $(shell which packr2)

SQUIRCY_TARGET := $(OUTPUT_BASE)/squircy
SQUIRCY_DIST   := $(SQUIRCY_TARGET)_$(GOOS)_$(GOARCH)$(GOARM)
PLUGIN_DIST    := $(patsubst %,$(OUTPUT_BASE)/%_$(GOOS)_$(GOARCH)$(GOARM).so,$(PLUGINS))

ifeq ($(PLUGIN_TYPE),linked)
	PLUGIN_TARGETS       :=
	EXTRA_TAGS           := -tags linked_plugins
	DIST_TARGETS         := $(SQUIRCY_DIST)
	LINKED_PLUGINS_FILE  := cmd/squircy/linked_plugins.go
else
	PLUGIN_TARGETS       := $(patsubst %,$(OUTPUT_BASE)/%.so,$(PLUGINS))
	EXTRA_TAGS           :=
	DIST_TARGETS         := $(SQUIRCY_DIST) $(PLUGIN_DIST)
	LINKED_PLUGINS_FILE  := cmd/squircy/shared_plugins.go
endif

TESTDATA_NODEMODS_TARGET := testdata/node_modules

SQUIRCY3_VERSION := $(if $(shell test -d .git && echo "1"),$(shell git describe --always --tags),SNAPSHOT)

.PHONY: all build run plugins clean test dist

all: build plugins

clean:
	cd cmd/squircy && \
		$(PACKR) clean
	rm -rf $(OUTPUT_BASE)

build: $(SQUIRCY_TARGET)

plugins: $(PLUGIN_TARGETS)

dist: $(DIST_TARGETS)

run: build
	$(SQUIRCY_TARGET)

test: $(TESTDATA_NODEMODS_TARGET)
	go test -tags netgo $(RACE) $(TEST_ARGS) ./...

$(TESTDATA_NODEMODS_TARGET):
	cd testdata && \
		yarn install

.SECONDEXPANSION:
$(PLUGIN_TARGETS): $(OUTPUT_BASE)/%.so: $$(wildcard plugins/%/*) $(SOURCES)
	go build -tags netgo $(RACE) -o $@ -buildmode=plugin plugins/$*/plugin/*.go

.SECONDEXPANSION:
$(PLUGIN_DIST): $(OUTPUT_BASE)/%_$(GOOS)_$(GOARCH)$(GOARM).so: $$(wildcard plugins/%/*) $(SOURCES)
	GOOS=$(GOOS) GOARCH=$(GOARCH) GOARM=$(GOARM) CC=$(CC) CGO_ENABLED=1 \
		go build -tags netgo $(EXTRA_TAGS) \
			-ldflags "-s -w -X main.Version=$(SQUIRCY3_VERSION)" \
			-o $@ -buildmode=plugin plugins/$*/plugin/*.go

$(SQUIRCY_TARGET): $(SOURCES)
	go build -tags netgo $(EXTRA_TAGS) $(RACE) -ldflags "-X main.Version=$(SQUIRCY3_VERSION)-dev" \
		-o $@ cmd/squircy/main*.go cmd/squircy/repl.go $(LINKED_PLUGINS_FILE)

$(SQUIRCY_DIST): $(OUTPUT_BASE) $(SOURCES)
	cd cmd/squircy/defconf && \
		yarn install
	cd cmd/squircy && \
		$(PACKR)
	GOOS=$(GOOS) GOARCH=$(GOARCH) GOARM=$(GOARM) CC=$(CC) CGO_ENABLED=1 \
		go build -tags netgo $(EXTRA_TAGS) \
			-ldflags "-s -w -X main.Version=$(SQUIRCY3_VERSION)" \
			-o $@ cmd/squircy/main*.go cmd/squircy/repl.go $(LINKED_PLUGINS_FILE)
	upx $@

$(OUTPUT_BASE):
	mkdir -p $(OUTPUT_BASE)

$(SOURCES): $(OUTPUT_BASE)
