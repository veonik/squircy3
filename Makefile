# Makefile for squircy, a proper IRC bot.
# https://code.dopame.me/veonik/squircy3

SUBPACKAGES := cli config event irc plugin vm

PLUGINS := $(patsubst plugins/%,%,$(wildcard plugins/*))

SOURCES := $(wildcard cmd/*/*.go) $(wildcard $(patsubst %,%/*.go,$(SUBPACKAGES)))
GENERATOR_SOURCES := $(wildcard cmd/squircy/defconf/*)

OUTPUT_BASE := out

PLUGIN_TARGETS := $(patsubst %,$(OUTPUT_BASE)/%.so,$(PLUGINS))
SQUIRCY_TARGET := $(OUTPUT_BASE)/squircy

RACE ?= -race
TEST_ARGS ?= -count 1

TESTDATA_NODEMODS_TARGET := testdata/node_modules

.PHONY: all build generate run plugins clean test

all: build plugins

clean:
	cd cmd/squircy && \
		packr2 clean
	rm -rf $(OUTPUT_BASE)

build: generate $(SQUIRCY_TARGET)

generate: $(OUTPUT_BASE)/.generated

plugins: $(PLUGIN_TARGETS)

run: build
	$(SQUIRCY_TARGET)

test: $(TESTDATA_NODEMODS_TARGET)
	go test -tags netgo $(RACE) $(TEST_ARGS) ./...

$(TESTDATA_NODEMODS_TARGET):
	cd testdata && \
		yarn install

.SECONDEXPANSION:
$(PLUGIN_TARGETS): $(OUTPUT_BASE)/%.so: $$(wildcard plugins/%/*) $(SOURCES)
	go build -tags netgo $(RACE) -o $@ -buildmode=plugin plugins/$*/*.go

$(SQUIRCY_TARGET): $(SOURCES)
	go build -tags netgo $(RACE) -o $@ cmd/squircy/*.go

$(OUTPUT_BASE)/.generated: $(GENERATOR_SOURCES)
	cd cmd/squircy && \
 		packr2
	touch $@

$(OUTPUT_BASE):
	mkdir -p $(OUTPUT_BASE)

$(SOURCES): $(OUTPUT_BASE)

$(GENERATOR_SOURCES): $(OUTPUT_BASE)
