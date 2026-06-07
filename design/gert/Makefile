PYTHON ?= python3
MAIN ?= main.tex
BUILD_DIR ?= build
ENGINE ?= auto
PDF_NAME ?= gert
GERT_DESIGN_DIR ?= $(if $(wildcard design/gert/Makefile),design/gert,.)

.PHONY: help check-tools build clean distclean rebuild release watch verify-corpus

help:
	@echo "Gert v2 design docs"
	@echo ""
	@echo "Targets:"
	@echo "  make check-tools  - Verify required tools are available"
	@echo "  make build        - Build PDF using selected engine"
	@echo "  make watch        - Continuous preview (requires latexmk)"
	@echo "  make verify-corpus - Validate conformance YAML and schema (requires PyYAML + jsonschema)"
	@echo "  make clean        - Remove generated build artifacts"
	@echo "  make distclean    - Alias for clean"
	@echo "  make rebuild      - Clean then build"
	@echo "  make release      - Build and package design docs as zip"
	@echo ""
	@echo "Variables:"
	@echo "  PYTHON=<python executable>"
	@echo "  MAIN=<main tex file>"
	@echo "  BUILD_DIR=<build output directory>"
	@echo "  ENGINE=<auto|tectonic|latexmk|pdflatex>"

check-tools:
	$(PYTHON) scripts/latex.py check --main $(MAIN) --build-dir $(BUILD_DIR) --engine $(ENGINE)

build:
	$(PYTHON) scripts/latex.py build --main $(MAIN) --build-dir $(BUILD_DIR) --engine $(ENGINE) --pdf-name $(PDF_NAME)

verify-corpus:
	@$(PYTHON) $(GERT_DESIGN_DIR)/scripts/verify_corpus.py

clean:
	$(PYTHON) scripts/latex.py clean --build-dir $(BUILD_DIR)

distclean: clean

rebuild: clean build

release: build
	$(PYTHON) scripts/latex.py release --main $(MAIN) --build-dir $(BUILD_DIR)

watch:
	latexmk -pdf -pvc -interaction=nonstopmode -shell-escape -output-directory=$(BUILD_DIR) $(MAIN)
