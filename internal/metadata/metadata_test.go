package metadata

import (
	"os"
	"testing"
	"path/filepath"
)

func TestMetadataStoreCRUD(t *testing.T) {
	dbPath := filepath.Join(os.TempDir(), "disktrobyte_test_metadata_db")
	defer os.RemoveAll(dbPath)

	store, err := OpenMetadataStore(dbPath)
	if err != nil {
		t.Fatalf("failed to open metadata store: %v", err)
	}
	defer store.Close()

	// Test FileMetadata
	fileMeta := NewFileMetadata("testfile.txt", 12345, []string{"hash1", "hash2"})
	err = store.PutFileMetadata(fileMeta)
	if err != nil {
		t.Fatalf("failed to put file metadata: %v", err)
	}

	gotFileMeta, err := store.GetFileMetadata("testfile.txt")
	if err != nil {
		t.Fatalf("failed to get file metadata: %v", err)
	}
	if gotFileMeta.FileName != fileMeta.FileName || gotFileMeta.FileSize != fileMeta.FileSize || len(gotFileMeta.ChunkHashes) != len(fileMeta.ChunkHashes) {
		t.Errorf("retrieved file metadata does not match")
	}

	// Test ChunkMetadata
	chunkMeta := ChunkMetadata{
		FileName: "testfile.txt",
		Index:    0,
		Hash:     "hash1",
		Path:     "/tmp/chunk1.bin",
		Size:     4096,
	}
	err = store.PutChunkMetadata(chunkMeta)
	if err != nil {
		t.Fatalf("failed to put chunk metadata: %v", err)
	}

	gotChunkMeta, err := store.GetChunkMetadata("hash1")
	if err != nil {
		t.Fatalf("failed to get chunk metadata: %v", err)
	}
	if gotChunkMeta.Hash != chunkMeta.Hash || gotChunkMeta.Path != chunkMeta.Path {
		t.Errorf("retrieved chunk metadata does not match")
	}
} 