# Makefile for squircy, a proper IRC bot.
# https://code.dopame.me/veonik/squircy3

SUBPACKAGES := config event irc plugin script vm $(wildcard plugins/*)

PLUGINS := $(patsubst plugins_shared/%,%,$(wildcard plugins_shared/*))

SOURCES := $(wildcard cmd/*/*.go) $(wildcard $(patsubst %,%/*.go,$(SUBPACKAGES)))
GENERATOR_SOURCES := $(wildcard web/views/*.twig) $(wildcard web/views/*/*.twig) $(wildcard web/public/css/*.css)

OUTPUT_BASE := out

PLUGIN_TARGETS := $(patsubst %,$(OUTPUT_BASE)/%.so,$(PLUGINS))
SQUIRCY_TARGET := $(OUTPUT_BASE)/squircy

.PHONY: all build generate run squircy plugins clean

all: build

clean:
	rm -rf $(OUTPUT_BASE)

build: plugins squircy

generate: $(OUTPUT_BASE)/.generated

squircy: $(SQUIRCY_TARGET)

plugins: $(PLUGIN_TARGETS)

run: build
	$(SQUIRCY_TARGET)

.SECONDEXPANSION:
$(PLUGIN_TARGETS): $(OUTPUT_BASE)/%.so: $$(wildcard plugins_shared/%/*) $(SOURCES)
	go build -tags shared -race -o $@ -buildmode=plugin plugins_shared/$*/*.go

$(SQUIRCY_TARGET): $(SOURCES)
	go build -tags shared -race -o $@ cmd/squircy/*.go

$(OUTPUT_BASE)/.generated: $(GENERATOR_SOURCES)
	go generate
	touch $@

$(OUTPUT_BASE):
	mkdir -p $(OUTPUT_BASE)

$(SOURCES): $(OUTPUT_BASE)

$(GENERATOR_SOURCES): $(OUTPUT_BASE)
