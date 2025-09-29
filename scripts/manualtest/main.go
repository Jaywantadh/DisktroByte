package main

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/jaywantadh/DisktroByte/config"
	"github.com/jaywantadh/DisktroByte/internal/chunker"
	"github.com/jaywantadh/DisktroByte/internal/metadata"
	"github.com/jaywantadh/DisktroByte/internal/storage"
)

func sha256File(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func main() {
	password := "testpass"
	inputPath := filepath.Join("samples", "ABC.pdf")
	if _, err := os.Stat(inputPath); err != nil {
		fmt.Printf("❌ Sample file not found: %v\n", err)
		return
	}

	config.LoadConfig("./config")

	origHash, err := sha256File(inputPath)
	if err != nil {
		fmt.Printf("❌ Failed hashing original: %v\n", err)
		return
	}
	fmt.Printf("📄 Original file: %s\n", inputPath)
	fmt.Printf("🔑 Original SHA256: %s\n", origHash)

	// Init storage and metadata
	_ = os.MkdirAll("output_chunks", 0755)
	store, err := storage.NewLocalStorage("output_chunks")
	if err != nil {
		fmt.Printf("❌ Storage init failed: %v\n", err)
		return
	}
	_ = os.RemoveAll("metadata_db_client_manual")
	ms, err := metadata.OpenMetadataStore("metadata_db_client_manual")
	if err != nil {
		fmt.Printf("❌ Metadata store init failed: %v\n", err)
		return
	}
	defer ms.Close()

	// Chunk and store
	metaList, err := chunker.ChunkAndStore(inputPath, password, ms, store)
	if err != nil {
		fmt.Printf("❌ ChunkAndStore failed: %v\n", err)
		return
	}
	if len(metaList) == 0 {
		fmt.Println("❌ No chunks produced")
		return
	}
	fileID := metaList[0].FileID
	fmt.Printf("🧩 Chunks created: %d | FileID: %s\n", len(metaList), fileID)

	// Reassemble
	outDir := "reassembled_manual"
	_ = os.MkdirAll(outDir, 0755)
	outPath := filepath.Join(outDir, "ABC_reassembled.pdf")
	if err := chunker.ReassembleFile(fileID, outPath, password, ms, store); err != nil {
		fmt.Printf("❌ Reassemble failed: %v\n", err)
		return
	}

	reHash, err := sha256File(outPath)
	if err != nil {
		fmt.Printf("❌ Failed hashing reassembled: %v\n", err)
		return
	}
	fmt.Printf("📦 Reassembled file: %s\n", outPath)
	fmt.Printf("🔑 Reassembled SHA256: %s\n", reHash)

	if reHash == origHash {
		fmt.Println("✅ SUCCESS: Reassembled file matches original")
	} else {
		fmt.Println("❌ MISMATCH: Reassembled file differs from original")
	}
}
