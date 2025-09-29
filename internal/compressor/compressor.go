package compressor

import (
	"bytes"
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"github.com/pierrec/lz4/v4"
)

var skipExtensions = map[string]bool{
	".mp4": true, ".mov": true, ".avi": true,
	".jpg": true, ".jpeg": true, ".png": true, ".gif": true, ".webp": true,
	".zip": true, ".rar": true, ".7z": true,
	".mp3": true, ".flac": true, ".aac": true,
	".apk": true, ".iso": true,
}

func ShouldSkipCompression(filePath string) bool {
	ext := strings.ToLower(filepath.Ext(filePath))
	return skipExtensions[ext]
}

func CompressChunk(chunkData []byte) ([]byte, error) {
	var out bytes.Buffer
	writer := lz4.NewWriter(&out)
	// Optionally set defaults or leave as-is
	if _, err := writer.Write(chunkData); err != nil {
		return nil, fmt.Errorf("compression failed: %v", err)
	}
	if err := writer.Close(); err != nil {
		return nil, fmt.Errorf("compression close failed: %v", err)
	}
	return out.Bytes(), nil
}

func DecompressData(data []byte) ([]byte, error) {
	reader := lz4.NewReader(bytes.NewReader(data))
	var decompressed bytes.Buffer
	if _, err := io.Copy(&decompressed, reader); err != nil {
		return nil, fmt.Errorf("decompression failed: %v", err)
	}
	return decompressed.Bytes(), nil
}
