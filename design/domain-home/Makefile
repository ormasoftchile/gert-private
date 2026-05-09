PYTHON      ?= python3
MAIN        ?= home.tex
BUILD_DIR   ?= build
ENGINE      ?= auto
PDF_NAME    ?= home-manual

# Path to the gert design folder (sibling in this repo) — used by `make init`
# to copy MastersThesis.cls.
GERT_DESIGN ?= ../gert

# Path to the shared latex.py build script
SCRIPT_PATH ?= $(GERT_DESIGN)/scripts/latex.py

.PHONY: help check-tools build watch clean distclean rebuild release init

help:
	@echo "Home Domain Kit Manual"
	@echo ""
	@echo "Targets:"
	@echo "  make init          - One-time setup: copy MastersThesis.cls and substitute placeholders"
	@echo "  make check-tools   - Verify required tools are available"
	@echo "  make build         - Build PDF using selected engine"
	@echo "  make watch         - Continuous preview (requires latexmk)"
	@echo "  make clean         - Remove generated build artifacts"
	@echo "  make distclean     - Alias for clean"
	@echo "  make rebuild       - Clean then build"
	@echo "  make release       - Build and package design docs as zip"
	@echo ""
	@echo "Variables:"
	@echo "  PYTHON=<python executable>"
	@echo "  MAIN=<main tex file>"
	@echo "  BUILD_DIR=<build output directory>"
	@echo "  ENGINE=<auto|tectonic|latexmk|pdflatex>"
	@echo "  GERT_DESIGN=<path to sibling gert design folder>"
	@echo "  PDF_NAME=<output PDF base name>"
	@echo ""
	@echo "Init usage:"
	@echo "  make init DOMAIN_NAME='My Kit' DOMAIN_SLUG=my-kit"

# One-time initialisation: copy class file and substitute template placeholders.
# Usage: make init DOMAIN_NAME="DRI Domain Kit" DOMAIN_SLUG=dri
init:
	@if [ -z "$(DOMAIN_NAME)" ] || [ -z "$(DOMAIN_SLUG)" ]; then \
		echo "Error: DOMAIN_NAME and DOMAIN_SLUG must be set."; \
		echo "  make init DOMAIN_NAME='My Kit' DOMAIN_SLUG=my-kit"; \
		exit 1; \
	fi
	@echo "Copying MastersThesis.cls from $(GERT_DESIGN) ..."
	cp $(GERT_DESIGN)/MastersThesis.cls .
	@echo "Renaming home.tex to $(DOMAIN_SLUG).tex ..."
	mv 'home.tex' '$(DOMAIN_SLUG).tex'
	@echo "Substituting placeholders in $(DOMAIN_SLUG).tex and Makefile ..."
	sed -i.bak \
		-e 's/Home Domain Kit/$(DOMAIN_NAME)/g' \
		-e 's/home/$(DOMAIN_SLUG)/g' \
		'$(DOMAIN_SLUG).tex' Makefile
	rm -f '$(DOMAIN_SLUG).tex.bak' Makefile.bak
	mkdir -p $(BUILD_DIR)
	@echo "Done. Review main.tex, then run: make build"

check-tools:
	$(PYTHON) $(SCRIPT_PATH) check --main $(MAIN) --build-dir $(BUILD_DIR) --engine $(ENGINE)

build:
	$(PYTHON) $(SCRIPT_PATH) build --main $(MAIN) --build-dir $(BUILD_DIR) --engine $(ENGINE) --pdf-name $(PDF_NAME)

watch:
	latexmk -pdf -pvc -interaction=nonstopmode -shell-escape \
		-output-directory=$(BUILD_DIR) \
		-jobname=$(PDF_NAME) \
		$(MAIN)

clean:
	$(PYTHON) $(SCRIPT_PATH) clean --build-dir $(BUILD_DIR)

distclean: clean

rebuild: clean build

release: build
	$(PYTHON) $(SCRIPT_PATH) release --main $(MAIN) --build-dir $(BUILD_DIR)
