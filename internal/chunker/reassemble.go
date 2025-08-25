package chunker

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"

	"github.com/jaywantadh/DisktroByte/internal/compressor"
	"github.com/jaywantadh/DisktroByte/internal/encryptor"
	"github.com/jaywantadh/DisktroByte/internal/metadata"
	"github.com/jaywantadh/DisktroByte/internal/storage"
)

// ReassembleFile reconstructs a file from its chunks using metadata.
// - fileName: the original file name (as stored in metadata)
// - outputPath: where to write the reassembled file
// - password: for decryption
// - metaStore: the metadata store instance
// - store: the storage backend
func ReassembleFile(
	fileName string,
	outputPath string,
	password string,
	metaStore *metadata.MetadataStore,
	store storage.Storage,
) error {
	// Fetch file metadata
	fileMeta, err := metaStore.GetFileMetadata(fileName)
	if err != nil {
		return fmt.Errorf("failed to get file metadata for %s: %v", fileName, err)
	}

	// Create output file
	outputFile, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create output file %s: %v", outputPath, err)
	}
	defer outputFile.Close()

	enc := encryptor.NewEncryptor()

	// Process each chunk in order
	for i, chunkHash := range fileMeta.ChunkHashes {
		// Fetch chunk metadata
		chunkMeta, err := metaStore.GetChunkMetadata(chunkHash)
		if err != nil {
			return fmt.Errorf("failed to get chunk metadata for hash %s: %v", chunkHash, err)
		}

		// Read chunk file
		chunkReader, err := store.Get(chunkMeta.Path)
		if err != nil {
			return fmt.Errorf("failed to read chunk %s: %v", chunkMeta.Path, err)
		}
		defer chunkReader.Close()

		chunkData, err := io.ReadAll(chunkReader)
		if err != nil {
			return fmt.Errorf("failed to read chunk data %s: %v", chunkMeta.Path, err)
		}

		// Decrypt chunk
		decrypted, err := enc.Decrypt(chunkData, password)
		if err != nil {
			return fmt.Errorf("failed to decrypt chunk %d: %v", i, err)
		}

		// Decompress chunk (if needed)
		var decompressed []byte
		if compressor.ShouldSkipCompression(fileName) {
			decompressed = decrypted
		} else {
			decompressed, err = compressor.DecompressData(decrypted)
			if err != nil {
				return fmt.Errorf("failed to decompress chunk %d: %v", i, err)
			}
		}

		// Validate chunk hash
		hash := sha256.Sum256(decompressed)
		calculatedHash := hex.EncodeToString(hash[:])
		if calculatedHash != chunkHash {
			return fmt.Errorf("hash mismatch for chunk %d: expected %s, got %s", i, chunkHash, calculatedHash)
		}

		// Write chunk data to output file
		_, err = outputFile.Write(decompressed)
		if err != nil {
			return fmt.Errorf("failed to write chunk %d to output file: %v", i, err)
		}
	}

	// Validate final file size
	outputInfo, err := outputFile.Stat()
	if err != nil {
		return fmt.Errorf("failed to stat output file: %v", err)
	}

	if outputInfo.Size() != fileMeta.FileSize {
		return fmt.Errorf("file size mismatch: expected %d, got %d", fileMeta.FileSize, outputInfo.Size())
	}

	return nil
}
