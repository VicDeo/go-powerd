BINARY_NAME=go-powerd
VERSION=0.3.0
PREFIX ?= /usr/local
BINDIR=$(PREFIX)/bin
SERVICE_NAME=go-powerd.service

# Get the latest git commit hash
COMMIT_HASH=$(shell git rev-parse --short HEAD 2>/dev/null || echo "none")
DIRTY := $(shell git diff --quiet || echo "-dirty")

# LDFLAGS: -s -w for size, and injecting version + commit info
LDFLAGS=-trimpath -ldflags "-s -w -X main.version=$(VERSION) -X main.commit=$(COMMIT_HASH)$(DIRTY)"

.PHONY: all build install uninstall clean test help

all: build

## help: Show available commands
help:
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@sed -n 's/^##//p' ${MAKEFILE_LIST} | column -t -s ':' |  sed -e 's/^/ /'

## test: Run unit tests
test:
	go test ./...

## build: Compile the binary
build: test
	CGO_ENABLED=0 go build $(LDFLAGS) -o $(BINARY_NAME) ./cmd/go-powerd

## install: Install binary to the system (requires sudo for /usr/local)
install:
	install -Dm755 $(BINARY_NAME) $(DESTDIR)$(BINDIR)/$(BINARY_NAME)
	@echo "Installed to $(DESTDIR)$(BINDIR)/$(BINARY_NAME)"

## install-service: Setup and enable systemd user service
install-service:
	@echo "Installing systemd user service..."
	mkdir -p $(HOME)/.config/systemd/user
	cp configs/$(SERVICE_NAME) $(HOME)/.config/systemd/user/
	systemctl --user daemon-reload
	systemctl --user enable --now $(SERVICE_NAME)
	@echo "Service installed and started."

## uninstall: Remove only the binary from the system (requires sudo)
uninstall:
	@echo "Removing binary from $(DESTDIR)$(BINDIR)..."
	rm -f $(DESTDIR)$(BINDIR)/$(BINARY_NAME)

## uninstall-service: Stop and remove the systemd user service (run as user)
uninstall-service:
	@echo "Disabling and removing user service..."
	systemctl --user disable --now $(SERVICE_NAME) 2>/dev/null || true
	rm -f $(HOME)/.config/systemd/user/$(SERVICE_NAME)
	systemctl --user daemon-reload
	@echo "User service removed."

## reload: Build, install and restart (detects systemd vs standalone)
reload: build
	@echo "Updating binary (requires sudo)..."
	sudo install -Dm755 $(BINARY_NAME) $(BINDIR)/$(BINARY_NAME)
	@if systemctl --user is-active --quiet $(SERVICE_NAME); then \
		echo "Detected active systemd service. Restarting..."; \
		systemctl --user restart $(SERVICE_NAME); \
	else \
		echo "Service not active. Restarting standalone process..."; \
		pkill $(BINARY_NAME) || true; \
		$(BINDIR)/$(BINARY_NAME) -t > /dev/null 2>&1 & \
	fi
	@echo "Reload complete."

## clean: Remove build artifacts
clean:
	rm -f $(BINARY_NAME)