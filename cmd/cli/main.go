package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/jaywantadh/DisktroByte/config"
	"github.com/jaywantadh/DisktroByte/internal/chunker"
	"github.com/jaywantadh/DisktroByte/internal/metadata"
	"github.com/jaywantadh/DisktroByte/internal/storage"
	"github.com/jaywantadh/DisktroByte/internal/transfer"
	"golang.org/x/term"
)

var metaStore *metadata.MetadataStore

func main() {
	config.LoadConfig("./config")
	if len(os.Args) < 2 {
		fmt.Println("Usage:")
		fmt.Println("  Chunk files: go run main.go chunk <file1> [<file2> ...]")
		fmt.Println("  Reassemble file: go run main.go reassemble <filename> <output_path>")
		fmt.Println("  Upload file: go run main.go upload <file_path> <peer_address>")
		fmt.Println("  Start server: go run main.go server")
		return
	}

	command := os.Args[1]

	// Create storage backend
	store, err := storage.NewLocalStorage("./output_chunks")
	if err != nil {
		fmt.Println("Failed to create storage:", err)
		return
	}

	// Use separate metadata DBs for server and client
	var metaDBPath string
	if command == "server" {
		metaDBPath = "./metadata_db_server"
	} else {
		metaDBPath = "./metadata_db_client"
	}

	// Open metadata store
	metaStore, err = metadata.OpenMetadataStore(metaDBPath)
	if err != nil {
		fmt.Println("Failed to open metadata store:", err)
		return
	}
	defer metaStore.Close()

	// Prompt for password (hidden input)
	fmt.Print("Enter password for encryption/decryption: ")
	passwordBytes, err := term.ReadPassword(int(os.Stdin.Fd()))
	fmt.Println() // newline after password input
	if err != nil {
		fmt.Println("Failed to read password:", err)
		return
	}
	password := strings.TrimSpace(string(passwordBytes))

	switch command {
	case "chunk":
		// Process each file for chunking
		for _, filePath := range os.Args[2:] {
			fmt.Printf("\nProcessing: %s\n", filePath)
			metadataList, err := chunker.ChunkAndStore(filePath, password, metaStore, store)
			if err != nil {
				fmt.Printf("Failed to chunk file %s: %v\n", filePath, err)
				continue
			}

			fmt.Printf("Chunking completed for: %s\n", filePath)
			fmt.Printf("Total Chunks: %d\n", len(metadataList))
			fmt.Printf("Output Dir: ./output_chunks/\n")
		}

	case "upload":
		if len(os.Args) != 4 {
			fmt.Println("Usage for upload: go run main.go upload <file_path> <peer_address>")
			return
		}
		filePath := os.Args[2]
		peerAddress := os.Args[3]

		// Chunk the file before uploading
		fmt.Printf("\nChunking file: %s\n", filePath)
		_, err = chunker.ChunkAndStore(filePath, password, metaStore, store)
		if err != nil {
			fmt.Printf("Failed to chunk file %s: %v\n", filePath, err)
			return
		}
		fmt.Println("File chunked successfully.")

		client := transfer.NewClient(peerAddress, metaStore, store)
		err = client.SendFile(filePath, password)
		if err != nil {
			fmt.Printf("Failed to upload file: %v\n", err)
			return
		}

	case "server":
		server := transfer.NewServer(metaStore, store, config.Config.Port)
		err = server.Start()
		if err != nil {
			fmt.Printf("Failed to start server: %v\n", err)
			return
		}

	case "reassemble":
		if len(os.Args) != 4 {
			fmt.Println("Usage for reassemble: go run main.go reassemble <filename> <output_path>")
			return
		}

		fileName := os.Args[2]
		outputPath := os.Args[3]

		fmt.Printf("\nReassembling: %s\n", fileName)
		fmt.Printf("Output: %s\n", outputPath)

		err = chunker.ReassembleFile(fileName, outputPath, password, metaStore, store)
		if err != nil {
			fmt.Printf("Failed to reassemble file %s: %v\n", fileName, err)
			return
		}

		fmt.Printf("Reassembly completed successfully!\n")
		fmt.Printf("File saved to: %s\n", outputPath)

	default:
		fmt.Printf("Unknown command: %s\n", command)
		fmt.Println("Available commands: chunk, reassemble, upload, server")
	}
}

// Disktrobyte Rock:- Cli based gui. 
// Disktrobyte Paper:- Web application.
// Disktorbyte Scissor:- Desktop Setup.