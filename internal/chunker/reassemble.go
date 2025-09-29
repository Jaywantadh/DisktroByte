package chunker

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"sort"

	"github.com/jaywantadh/DisktroByte/internal/compressor"
	"github.com/jaywantadh/DisktroByte/internal/encryptor"
	"github.com/jaywantadh/DisktroByte/internal/metadata"
	"github.com/jaywantadh/DisktroByte/internal/storage"
)

// ReassembleFile reconstructs a file from its chunks using enhanced metadata.
// - fileID: the unique file identifier (SHA-256 hash of original file)
// - outputPath: where to write the reassembled file
// - password: for decryption
// - metaStore: the metadata store instance
// - store: the storage backend
func ReassembleFile(
	fileID string,
	outputPath string,
	password string,
	metaStore *metadata.MetadataStore,
	store storage.Storage,
) error {
	// Fetch all chunks for the file using FileID
	chunks, err := metaStore.GetChunksByFileID(fileID)
	if err != nil {
		return fmt.Errorf("failed to get chunks for FileID %s: %v", fileID, err)
	}

	// Validate chunk chain integrity
	if err := metadata.ValidateChunkChain(chunks); err != nil {
		return fmt.Errorf("chunk chain validation failed: %v", err)
	}

	// Create output file
	outputFile, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create output file %s: %v", outputPath, err)
	}
	defer outputFile.Close()

	enc := encryptor.NewEncryptor()

	// Sort chunks by offset to ensure correct order (should already be sorted by validation)
	sortChunksByOffset(chunks)

	// Process each chunk in order
	for i, chunkMeta := range chunks {

		// Read chunk file using the chunk path (which is the hash)
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

		// Decompress chunk only if it was compressed during storage
		var decompressed []byte
		if chunkMeta.IsCompressed {
			// This chunk was compressed, so decompress it
			decompData, err := compressor.DecompressData(decrypted)
			if err != nil {
				return fmt.Errorf("failed to decompress chunk %d: %v", chunkMeta.Index, err)
			}
			decompressed = decompData
		} else {
			// This chunk wasn't compressed, use decrypted data directly
			decompressed = decrypted
		}

		// Validate chunk hash
		hash := sha256.Sum256(decompressed)
		calculatedHash := hex.EncodeToString(hash[:])
		if calculatedHash != chunkMeta.Hash {
			return fmt.Errorf("hash mismatch for chunk %d: expected %s, got %s", 
				chunkMeta.Index, chunkMeta.Hash, calculatedHash)
		}

		// Write chunk data to output file
		_, err = outputFile.Write(decompressed)
		if err != nil {
			return fmt.Errorf("failed to write chunk %d to output file: %v", i, err)
		}
	}

	// Calculate expected file size from chunk metadata (sum of original chunk data)
	// Note: This is the size of the uncompressed/decrypted data, not the stored chunk sizes
	// For now, we'll skip this validation and rely on chunk hash validation
	// TODO: Add proper file size validation using original file size metadata

	// Validate final file size (simplified - just check that we wrote something)
	outputInfo, err := outputFile.Stat()
	if err != nil {
		return fmt.Errorf("failed to stat output file: %v", err)
	}

	if outputInfo.Size() == 0 {
		return fmt.Errorf("output file is empty")
	}

	// TODO: Add proper file size validation using original file size metadata

	return nil
}

// sortChunksByOffset sorts chunks by their offset in the original file
func sortChunksByOffset(chunks []metadata.ChunkMetadata) {
	sort.Slice(chunks, func(i, j int) bool {
		return chunks[i].Offset < chunks[j].Offset
	})
}
