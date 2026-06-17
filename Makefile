# Gyver — universal command layer
BINARY      := gyver
PKG         := ./cmd/gyver
BUILD_DIR   := bin
INSTALL_DIR := $(HOME)/.local/bin

.PHONY: all build run test tidy fmt vet clean install-local

all: build

## build: compile the gyver binary into ./bin
build:
	@mkdir -p $(BUILD_DIR)
	go build -o $(BUILD_DIR)/$(BINARY) $(PKG)
	@echo "built $(BUILD_DIR)/$(BINARY)"

## run: run gyver directly (pass args with ARGS=…, e.g. make run ARGS="list --plain")
run:
	go run $(PKG) $(ARGS)

## test: run the test suite
test:
	go test ./...

## tidy: resolve and pin dependencies (run this first after cloning)
tidy:
	go mod tidy

## fmt: format all Go source
fmt:
	go fmt ./...

## vet: run go vet
vet:
	go vet ./...

## install-local: build and copy the binary to ~/.local/bin
install-local: build
	@mkdir -p $(INSTALL_DIR)
	cp $(BUILD_DIR)/$(BINARY) $(INSTALL_DIR)/$(BINARY)
	@echo "installed to $(INSTALL_DIR)/$(BINARY) (ensure it is on your PATH)"

## clean: remove build artifacts
clean:
	rm -rf $(BUILD_DIR)
