.PHONY: build build-helper install uninstall test clean release

BINARY := break-reminder
HELPER := break-screen
BUILD_DIR := bin
INSTALL_DIR := $(HOME)/.local/bin

VERSION := $(shell cat VERSION 2>/dev/null || echo "dev")
LDFLAGS := -ldflags "-s -w -X main.version=$(VERSION)"

build-helper:
	@mkdir -p $(BUILD_DIR)
	cd helpers && swift build -c release
	cp helpers/.build/release/BreakScreenApp $(BUILD_DIR)/$(HELPER)
	cp helpers/.build/release/DashboardApp $(BUILD_DIR)/break-dashboard

build: build-helper
	go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY) ./cmd/break-reminder

install: build
	@mkdir -p $(INSTALL_DIR)
	cp $(BUILD_DIR)/$(BINARY) $(INSTALL_DIR)/
	cp $(BUILD_DIR)/$(HELPER) $(INSTALL_DIR)/
	cp $(BUILD_DIR)/break-dashboard $(INSTALL_DIR)/
	$(INSTALL_DIR)/$(BINARY) service install
	@echo "Installed to $(INSTALL_DIR)/$(BINARY)"

uninstall:
	$(INSTALL_DIR)/$(BINARY) service uninstall || true
	rm -f $(INSTALL_DIR)/$(BINARY)
	@echo "Uninstalled"

test:
	go test ./...
	cd helpers && swift test

release: build
	@echo "Creating release archive $(VERSION)..."
	cd $(BUILD_DIR) && tar czf $(BINARY)-$(VERSION)-darwin-arm64.tar.gz $(BINARY) $(HELPER) break-dashboard
	cd $(BUILD_DIR) && shasum -a 256 $(BINARY)-$(VERSION)-darwin-arm64.tar.gz > $(BINARY)-$(VERSION)-darwin-arm64.tar.gz.sha256
	@echo "Archive: $(BUILD_DIR)/$(BINARY)-$(VERSION)-darwin-arm64.tar.gz"
	@cat $(BUILD_DIR)/$(BINARY)-$(VERSION)-darwin-arm64.tar.gz.sha256

clean:
	rm -rf $(BUILD_DIR)
