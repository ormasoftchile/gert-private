PYTHON ?= python

.PHONY: help verify-corpus

help:
	@$(MAKE) --no-print-directory -f design/gert/Makefile help PYTHON="$(PYTHON)"

verify-corpus:
	@$(MAKE) --no-print-directory -f design/gert/Makefile verify-corpus PYTHON="$(PYTHON)"
