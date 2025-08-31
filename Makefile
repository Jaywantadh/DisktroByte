APP_NAME=disktrobyte
BUILD_DIR=bin

.PHONY: run
run: 
	go run ./cmd/cli/main.go

.PHONY: gui
gui:
	go run ./cmd/cli/main.go

.PHONY: build
build:
	mkdir -p $(BUILD_DIR)
	go build -o $(BUILD_DIR)/$(APP_NAME) ./cmd/cli/main.go

.PHONY: build-gui
build-gui:
	mkdir -p $(BUILD_DIR)
	go build -o $(BUILD_DIR)/$(APP_NAME)-gui ./cmd/cli/main.go

.PHONY: test
test: 
	go.test ./... -v

.PHONY: clean
clean:
	rm -rf $(BUILD_DIR)

.PHONY: help
help:
	@echo "Useful make commands:"
	@echo "	make run	- Run the CLI application"
	@echo "	make gui	- Run the GUI application (web-based)"
	@echo "	make build	- Build CLI binary to ./bin"
	@echo "	make build-gui	- Build GUI binary to ./bin"
	@echo "	make test	- Run all unit tests"
	@echo "	make clean	- Delete ./bin folder"


