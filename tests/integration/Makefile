SHELL := /bin/bash

DEBUG ?=
PARALLEL ?= 2
RUN ?=

.PHONY: test
test: test-kind

.PHONY: test-kind
test-kind:
	$(MAKE) -C ./kind test
