BINARY_NAME=go-powerd
INSTALL_PATH=/usr/local/bin/$(BINARY_NAME)
USER_BIN_DIR=$(HOME)/.local/bin

.PHONY: build install reload clean

test:
	go test ./...

build: test
	go build -o $(BINARY_NAME) ./cmd

install: build
	@echo "Installing to $(INSTALL_PATH)..."
	sudo cp $(BINARY_NAME) $(INSTALL_PATH)
	sudo chmod +x $(INSTALL_PATH)

# Reload the daemon without restarting all of Sway
reload: install
	@echo "Restarting $(BINARY_NAME)..."
	pkill $(BINARY_NAME) || true
	swaymsg exec $(INSTALL_PATH)


clean:
	rm -f $(BINARY_NAME)