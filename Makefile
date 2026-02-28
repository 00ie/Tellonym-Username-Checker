.PHONY: build run dev clean test deps lint fmt release build-windows build-macos build-linux

BINARY_NAME=tellonym-checker
BUILD_DIR=build
DIST_DIR=dist
GOCMD=go
GOTEST=$(GOCMD) test
GOMOD=$(GOCMD) mod
WAILS_CMD=wails
WAILS_BUILD=$(WAILS_CMD) build
WAILS_DEV=$(WAILS_CMD) dev

all: test build

build:
	$(WAILS_BUILD) -o $(BINARY_NAME)

build-windows:
	$(WAILS_BUILD) -platform windows/amd64 -o $(BINARY_NAME).exe

build-macos:
	$(WAILS_BUILD) -platform darwin/universal -o $(BINARY_NAME)

build-linux:
	$(WAILS_BUILD) -platform linux/amd64 -o $(BINARY_NAME)

dev:
	$(WAILS_DEV)

run:
	./$(BUILD_DIR)/$(BINARY_NAME)

clean:
	$(GOCMD) clean
	rm -rf $(BUILD_DIR)
	rm -rf $(DIST_DIR)
	rm -f $(BINARY_NAME)

test:
	$(GOTEST) -v ./...

deps:
	$(GOMOD) download
	$(GOMOD) tidy
	cd frontend && npm install

lint:
	cd frontend && npm run lint

fmt:
	$(GOCMD) fmt ./...
	cd frontend && npm run format

release: build-windows build-macos build-linux
	mkdir -p $(DIST_DIR)
	zip -j $(DIST_DIR)/$(BINARY_NAME)-windows-amd64.zip $(BUILD_DIR)/$(BINARY_NAME).exe
	zip -j $(DIST_DIR)/$(BINARY_NAME)-darwin-universal.zip $(BUILD_DIR)/$(BINARY_NAME)
	zip -j $(DIST_DIR)/$(BINARY_NAME)-linux-amd64.zip $(BUILD_DIR)/$(BINARY_NAME)
