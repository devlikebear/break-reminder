.PHONY: build build-helper install uninstall test clean

BINARY := break-reminder
HELPER := break-screen
BUILD_DIR := bin
INSTALL_DIR := $(HOME)/.local/bin

VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS := -ldflags "-s -w -X main.version=$(VERSION)"

build-helper:
	swiftc -O -o $(BUILD_DIR)/$(HELPER) helpers/break-screen/main.swift -framework AppKit

build: build-helper
	go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY) ./cmd/break-reminder

install: build
	@mkdir -p $(INSTALL_DIR)
	cp $(BUILD_DIR)/$(BINARY) $(INSTALL_DIR)/
	cp $(BUILD_DIR)/$(HELPER) $(INSTALL_DIR)/
	$(INSTALL_DIR)/$(BINARY) service install
	@echo "Installed to $(INSTALL_DIR)/$(BINARY)"

uninstall:
	$(INSTALL_DIR)/$(BINARY) service uninstall || true
	rm -f $(INSTALL_DIR)/$(BINARY)
	@echo "Uninstalled"

test:
	go test ./...

clean:
	rm -rf $(BUILD_DIR)
