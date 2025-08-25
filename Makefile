APP_NAME=disktrobyte
BUILD_DIR=bin

.PHONY: run
run: 
	go run ./cmd/cli/main.go

.PHONY: build
build:
	mkdir -p $(BUILD_DIR)
	go build -o $(BUILD_DIR)/$(APP_NAME) ./cmd/cli/main.go

.PHONY: test
test: 
	go.test ./... -v

.PHONY: clean
clean:
	rm -rf $(BUILD_DIR)

.PHONY: help
help:
	@echo "Useful make commands:"
	@echo "	make run	- Run the application"
	@echo "	make build	- Build binary to ./bin"
	@echo "	make test	- Run all unit tests"
	@echo "	make clean	- Delete ./bin folder"


