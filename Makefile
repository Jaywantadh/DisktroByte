run:
	/tgo run ./cmd/cli/main.go

build:
	/tgo build -o disktrobyte ./cmd/cli

test:
	/tgo test ./...