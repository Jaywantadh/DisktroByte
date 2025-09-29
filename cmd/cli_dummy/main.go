package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

type Response struct {
	Success bool        `json:"success"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

type DummyFile struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Size      int64     `json:"size"`
	CreatedAt time.Time `json:"created_at"`
}

var (
	originalFileCache = make(map[string]string)     // fileID -> path
	dummyFiles        = make(map[string]*DummyFile) // fileID -> info
)

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("/", handleHome)
	mux.HandleFunc("/api/chunk", handleChunk)
	mux.HandleFunc("/api/reassemble", handleReassemble)
	mux.HandleFunc("/api/files", handleGetFiles)

	addr := ":8090"
	fmt.Printf("🚀 Dummy DisktroByte CLI starting on http://localhost%s\n", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		fmt.Printf("❌ Server error: %v\n", err)
	}
}

func handleHome(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain")
	w.Write([]byte("DisktroByte Dummy CLI running. Use /api/chunk and /api/reassemble."))
}

func handleChunk(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if err := r.ParseMultipartForm(64 << 20); err != nil {
		sendJSON(w, false, "Failed to parse form: "+err.Error(), nil)
		return
	}
	file, header, err := r.FormFile("file")
	if err != nil {
		sendJSON(w, false, "No file provided: "+err.Error(), nil)
		return
	}
	defer file.Close()

	_ = os.MkdirAll("./dummy_cache", 0755)
	id := fmt.Sprintf("dummy-%d", time.Now().UnixNano())
	path := filepath.Join("./dummy_cache", id+"_"+header.Filename)
	out, err := os.Create(path)
	if err != nil {
		sendJSON(w, false, "Failed to create file: "+err.Error(), nil)
		return
	}
	if _, err := io.Copy(out, file); err != nil {
		out.Close()
		sendJSON(w, false, "Failed to save file: "+err.Error(), nil)
		return
	}
	out.Close()
	st, _ := os.Stat(path)

	originalFileCache[id] = path
	dummyFiles[id] = &DummyFile{ID: id, Name: header.Filename, Size: st.Size(), CreatedAt: time.Now()}

	// Simulate chunking log
	fmt.Printf("📦 [DUMMY] Chunked '%s' into %d chunks (simulated)\n", header.Filename, 4)
	sendJSON(w, true, "File chunked and distributed (simulated)", dummyFiles[id])
}

func handleReassemble(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		FileID     string `json:"fileId"`
		OutputPath string `json:"outputPath"`
		Password   string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendJSON(w, false, "Failed to decode request: "+err.Error(), nil)
		return
	}
	if req.FileID == "" || req.OutputPath == "" {
		sendJSON(w, false, "File ID and output path are required", nil)
		return
	}
	cachePath, ok := originalFileCache[req.FileID]
	if !ok {
		sendJSON(w, false, "File not found", nil)
		return
	}
	_ = os.MkdirAll(filepath.Dir(req.OutputPath), 0755)
	if err := copyFile(cachePath, req.OutputPath); err != nil {
		sendJSON(w, false, "Failed to write output: "+err.Error(), nil)
		return
	}
	fmt.Printf("🔧 [DUMMY] Reassembled '%s' to '%s' (simulated)\n", cachePath, req.OutputPath)
	sendJSON(w, true, "File reassembled successfully (simulated)", map[string]string{"output": req.OutputPath})
}

func handleGetFiles(w http.ResponseWriter, r *http.Request) {
	files := make([]*DummyFile, 0, len(dummyFiles))
	for _, f := range dummyFiles {
		files = append(files, f)
	}
	sendJSON(w, true, "Files retrieved", files)
}

func sendJSON(w http.ResponseWriter, ok bool, msg string, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(Response{Success: ok, Message: msg, Data: data})
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer func() { _ = out.Close() }()
	_, err = io.Copy(out, in)
	return err
}

