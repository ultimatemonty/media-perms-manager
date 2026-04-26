SHELL := /bin/bash

APP_NAME := media-perms-manager
RELEASE_NUMBER ?= dev
DIST_DIR := dist
BUILD_DIR := $(DIST_DIR)/build
GO ?= go
SHASUM ?= shasum -a 256
XZ ?= xz
REMOTE ?= origin
TAG_PREFIX ?= v
VERSION ?=
TAG := $(TAG_PREFIX)$(VERSION)

TARGETS := darwin-arm64 linux-amd64 linux-arm64
ARCHIVES := $(addprefix $(DIST_DIR)/,$(addsuffix .tar.xz,$(foreach target,$(TARGETS),$(APP_NAME)-$(RELEASE_NUMBER)-$(target))))
BUILD_STAMPS := $(addprefix $(BUILD_DIR)/,$(addsuffix /.built,$(foreach target,$(TARGETS),$(APP_NAME)-$(RELEASE_NUMBER)-$(target))))

.PHONY: all build package release check-version check-tag-not-exists tag push-tag tag-and-push clean

all: release

build: $(BUILD_STAMPS)

package: $(ARCHIVES)

release: build package

check-version:
	@if [ -z "$(VERSION)" ]; then \
		echo "VERSION is required (example: make tag-and-push VERSION=1.2.3)"; \
		exit 1; \
	fi

check-tag-not-exists: check-version
	@if git rev-parse -q --verify "refs/tags/$(TAG)" >/dev/null; then \
		echo "Tag $(TAG) already exists"; \
		exit 1; \
	fi

tag: check-tag-not-exists
	git tag "$(TAG)"

push-tag: check-version
	git push "$(REMOTE)" "refs/tags/$(TAG)"

tag-and-push: tag
	git push "$(REMOTE)" "refs/tags/$(TAG)"

$(BUILD_DIR)/$(APP_NAME)-$(RELEASE_NUMBER)-%/.built:
	@set -euo pipefail; \
	goos="$$(printf '%s' "$*" | cut -d- -f1)"; \
	goarch="$$(printf '%s' "$*" | cut -d- -f2)"; \
	binary_path="$(@D)/$(APP_NAME)"; \
	rm -rf "$(@D)"; \
	mkdir -p "$(@D)"; \
	CGO_ENABLED=0 GOOS="$$goos" GOARCH="$$goarch" $(GO) build -ldflags "-X main.version=$(RELEASE_NUMBER)" -o "$$binary_path" ./; \
	chmod 755 "$$binary_path"; \
	(cd "$(@D)" && $(SHASUM) "$(APP_NAME)" > "$(APP_NAME).sha256"); \
	touch "$@"

$(DIST_DIR)/$(APP_NAME)-$(RELEASE_NUMBER)-%.tar.xz: $(BUILD_DIR)/$(APP_NAME)-$(RELEASE_NUMBER)-%/.built
	@set -euo pipefail; \
	package_name="$(APP_NAME)-$(RELEASE_NUMBER)-$*"; \
	mkdir -p "$(DIST_DIR)"; \
	tar -C "$(BUILD_DIR)" -cf - "$$package_name" | $(XZ) -z -c > "$@"

clean:
	rm -rf "$(DIST_DIR)"