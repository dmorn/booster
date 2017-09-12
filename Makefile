PACKAGE  = booster
APIMAIN  = cmd/api/*
DATE     ?= $(shell date +%FT%T%z)
VERSION  ?= $(shell git describe --tags --always --dirty --match=v* 2> /dev/null || \
			cat $(CURDIR)/.version 2> /dev/null || echo v0)

GO      = go
GODOC   = godoc
GOFMT   = gofmt
GODEP   = dep

V = 0 # Stays for Verbose: binary

Q = $(if $(filter 1,$V),,@)
M = $(shell printf "\033[34;1m▶\033[0m") # Found this on the internet.. too ugly in my opition ;)

.PHONY: all
all: $(APIMAIN) vendor ; $(info $(M) building executable…)
	$(Q) $(GO) build \
		-tags release \
		-ldflags '-X $(PACKAGE)/cmd.Version=$(VERSION) -X $(PACKAGE)/cmd.BuildDate=$(DATE)' \
		-o bin/$(PACKAGE) $(APIMAIN)

.PHONY: fast
fast: $(APIBIN) ; $(info $(M) building executable…)
	$(Q) $(GO) build \
		-o bin/$(PACKAGE) $(APIMAIN)

.PHONY: run
run: $(APIBIN) ; $(info $(M) running $(APIMAIN)…)
	$(Q) $(GO) run $(APIMAIN)

.PHONY: vendor
vendor: Gopkg.toml Gopkg.lock ; $(info $(M) retrieving dependencies…)
	$Q $(GODEP) ensure
	@touch $@

.PHONY: vendor-update
vendor-update: Gopkg.toml Gopkg.lock ; $(info $(M) updating all dependencies…)
	$(info $(M) updating all dependencies…)
	$Q cd $(BASE) && $(GODEP) ensure -update

# Misc

.PHONY: clean
clean: ; $(info $(M) cleaning…)
	@ rm -rf bin

.PHONY: version
version:
	@echo $(VERSION)