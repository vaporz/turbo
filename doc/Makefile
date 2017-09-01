# Makefile for Sphinx documentation
#

# You can set these variables from the command line.
SPHINXOPTS    =
SPHINXBUILD   = sphinx-build
PAPER         =
BUILDDIR      = build

# User-friendly check for sphinx-build
ifeq ($(shell which $(SPHINXBUILD) >/dev/null 2>&1; echo $$?), 1)
$(error The '$(SPHINXBUILD)' command was not found. Make sure you have Sphinx installed, then set the SPHINXBUILD environment variable to point to the full path of the '$(SPHINXBUILD)' executable. Alternatively you can add the directory with the executable to your PATH. If you don't have Sphinx installed, grab it from http://sphinx-doc.org/)
endif

# Internal variables.
PAPEROPT_a4     = -D latex_paper_size=a4
PAPEROPT_letter = -D latex_paper_size=letter
ALLSPHINXOPTS   = -d $(BUILDDIR)/doctrees $(PAPEROPT_$(PAPER)) $(SPHINXOPTS) source
# the i18n builder cannot share the environment and doctrees with the others
I18NSPHINXOPTS  = $(PAPEROPT_$(PAPER)) $(SPHINXOPTS) source

.PHONY: clean
clean:
	rm -rf $(BUILDDIR)/*

.PHONY: html
html: clean
	$(SPHINXBUILD) -b html $(ALLSPHINXOPTS) $(BUILDDIR)/html
	cd $(BUILDDIR)/html && touch .nojekyll
	cd source/0.1/en && make html VERSION=0.1 LANG=en
	cd source/0.1/zh && make html VERSION=0.1 LANG=zh
	cd source/0.2/en && make html VERSION=0.2 LANG=en
	cd source/0.2/zh && make html VERSION=0.2 LANG=zh
	cd source/master/en && make html VERSION=master LANG=en
	cd source/master/zh && make html VERSION=master LANG=zh
