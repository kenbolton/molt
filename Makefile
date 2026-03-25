BINARY := molt
SRC_DIR := ./src
BUILD_DIR := ./build

.PHONY: all build clean test install

all: build

build:
	@mkdir -p $(BUILD_DIR)
	go build -o $(BUILD_DIR)/$(BINARY) $(SRC_DIR)
	@echo "Built: $(BUILD_DIR)/$(BINARY)"

install: build
	cp $(BUILD_DIR)/$(BINARY) /usr/local/bin/$(BINARY)
	@echo "Installed: /usr/local/bin/$(BINARY)"

test:
	go test ./...

clean:
	rm -rf $(BUILD_DIR)

# Cross-compile for common targets
build-all:
	GOOS=darwin  GOARCH=arm64  go build -o $(BUILD_DIR)/$(BINARY)-darwin-arm64  $(SRC_DIR)
	GOOS=darwin  GOARCH=amd64  go build -o $(BUILD_DIR)/$(BINARY)-darwin-amd64  $(SRC_DIR)
	GOOS=linux   GOARCH=amd64  go build -o $(BUILD_DIR)/$(BINARY)-linux-amd64   $(SRC_DIR)
	GOOS=linux   GOARCH=arm64  go build -o $(BUILD_DIR)/$(BINARY)-linux-arm64   $(SRC_DIR)
	@echo "Cross-compiled binaries in $(BUILD_DIR)/"
