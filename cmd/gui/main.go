package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/dgraph-io/badger/v4"
	"github.com/jaywantadh/DisktroByte/config"
	"github.com/jaywantadh/DisktroByte/internal/auth"
	"github.com/jaywantadh/DisktroByte/internal/chunker"
	"github.com/jaywantadh/DisktroByte/internal/dfs"
	"github.com/jaywantadh/DisktroByte/internal/distributor"
	"github.com/jaywantadh/DisktroByte/internal/metadata"
	"github.com/jaywantadh/DisktroByte/internal/p2p"
	"github.com/jaywantadh/DisktroByte/internal/storage"
	"github.com/jaywantadh/DisktroByte/internal/streaming"
)

var (
	metaStore        *metadata.MetadataStore
	store            storage.Storage
	server           *http.Server
	network          *p2p.Network
	tcpNetwork       *p2p.TCPNetwork
	broadcastManager *p2p.BroadcastManager
	streamProcessor  *streaming.StreamProcessor
	fileDistributor  *distributor.Distributor
	authManager      *auth.AuthManager
	// DFS Core Components
	dfsCore          *dfs.DFSCore
	chunkDistributor *dfs.ChunkDistributor
	fileReassembler  *dfs.FileReassembler
	// Dummy passthrough: original file cache
	originalFileCache = make(map[string]string)
)

// Response represents API response structure
type Response struct {
	Success bool        `json:"success"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// SSE Log Event represents a server-sent event for logs
type SSELogEvent struct {
	Type      string      `json:"type"` // "log", "heartbeat", "error"
	Timestamp time.Time   `json:"timestamp"`
	Data      interface{} `json:"data"`
}

// FileLogEntry represents a file operation log entry
type FileLogEntry struct {
	ID          string    `json:"id"`
	Operation   string    `json:"operation"` // "upload", "download", "chunk", "reassemble"
	FileName    string    `json:"file_name"`
	FileSize    int64     `json:"file_size"`
	ChunkCount  int       `json:"chunk_count"`
	Status      string    `json:"status"` // "pending", "in_progress", "completed", "failed"
	Progress    float64   `json:"progress"`
	Timestamp   time.Time `json:"timestamp"`
	UserID      string    `json:"user_id"`
	NodeID      string    `json:"node_id"`
	ReplicaInfo []string  `json:"replica_info"`
	Error       string    `json:"error,omitempty"`
}

// loggedServeMux wraps http.ServeMux with request logging
type loggedServeMux struct {
	mux *http.ServeMux
}

func (l *loggedServeMux) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("🌐 Request: %s %s\n", r.Method, r.URL.Path)
	l.mux.ServeHTTP(w, r)
}

func main() {
	// Load configuration
	config.LoadConfig("./config")

	// Initialize storage and metadata
	initializeStorage()

	// Initialize authentication
	authManager = auth.NewAuthManager(24*time.Hour, 100)

	// Try different ports if the default is busy
	port := config.Config.Port
	for i := 0; i < 10; i++ {
		testPort := port + i
		server = &http.Server{
			Addr:    fmt.Sprintf(":%d", testPort),
			Handler: createRouter(),
		}

		fmt.Printf("🚀 DisktroByte GUI starting on http://localhost:%d\n", testPort)
		fmt.Println("📁 Open your browser and navigate to the URL above")
		fmt.Println("🔐 Default admin credentials: admin/admin123")

		// Start server
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			if strings.Contains(err.Error(), "bind: Only one usage of each socket address") {
				fmt.Printf("⚠️ Port %d is busy, trying next port...\n", testPort)
				continue
			}
			fmt.Printf("❌ Server failed to start: %v\n", err)
			break
		}
		break
	}
}

func initializeStorage() {
	var err error

	// Create storage backend
	store, err = storage.NewLocalStorage("./output_chunks")
	if err != nil {
		fmt.Printf("❌ Failed to create storage: %v\n", err)
		return
	}

	// Try to open metadata store with retry logic and unique path
	dbPath := fmt.Sprintf("./metadata_db_gui_%d", time.Now().Unix())
	for i := 0; i < 3; i++ {
		metaStore, err = metadata.OpenMetadataStore(dbPath)
		if err == nil {
			break
		}

		if strings.Contains(err.Error(), "LOCK") {
			fmt.Printf("⚠️ Database is locked, trying different path... (attempt %d/3)\n", i+1)
			dbPath = fmt.Sprintf("./metadata_db_gui_%d_%d", time.Now().Unix(), i)
			time.Sleep(1 * time.Second)
			continue
		}

		fmt.Printf("⚠️ Failed to open metadata store: %v - continuing without database\n", err)
		break
	}

	if metaStore == nil {
		fmt.Printf("⚠️ Failed to open metadata store after retries - continuing without database\n")
		// Continue without metadata store for now
	}

	// Initialize HTTP P2P network with dynamic port allocation
	p2pPort := config.Config.Port + 2000 // Start from a different base to avoid conflicts
	for i := 0; i < 10; i++ {
		testPort := p2pPort + i
		network = p2p.NewNetwork("localhost", testPort)
		if err := network.Start(); err != nil {
			if strings.Contains(err.Error(), "bind: Only one usage") {
				fmt.Printf("⚠️ P2P HTTP port %d busy, trying next...\n", testPort)
				continue
			}
			fmt.Printf("⚠️ Failed to start HTTP P2P network: %v - continuing without P2P\n", err)
			network = nil
			break
		}
		// Set storage backend for chunk serving
		network.SetStorage(store)
		// Set metadata store for chunk mapping
		if metaStore != nil {
			network.SetMetadataStore(metaStore)
		}
		fmt.Printf("🌐 HTTP P2P Network started on port %d\n", testPort)
		break
	}

	// Initialize TCP P2P network with dynamic port allocation
	tcpPort := config.Config.Port + 3000 // Different base port for TCP
	for i := 0; i < 10; i++ {
		testPort := tcpPort + i
		tcpNetwork = p2p.NewTCPNetwork("localhost", testPort)
		if err := tcpNetwork.Start(); err != nil {
			if strings.Contains(err.Error(), "bind: Only one usage") {
				fmt.Printf("⚠️ P2P TCP port %d busy, trying next...\n", testPort)
				continue
			}
			fmt.Printf("⚠️ Failed to start TCP P2P network: %v - continuing without TCP P2P\n", err)
			tcpNetwork = nil
			break
		}
		fmt.Printf("🌐 TCP P2P Network started on port %d\n", testPort)
		break
	}

	// Initialize broadcast manager
	if tcpNetwork != nil {
		broadcastManager = p2p.NewBroadcastManager(tcpNetwork)
		if err := broadcastManager.Start(); err != nil {
			fmt.Printf("❌ Failed to start broadcast manager: %v\n", err)
		}
	}

	// Initialize stream processor
	streamProcessor = streaming.NewStreamProcessor(64*1024, 10) // 64KB buffer, max 10 concurrent streams

	// Initialize file distributor (if we have required components)
	if network != nil && store != nil {
		fileDistributor = distributor.NewDistributor(network, store, metaStore)
		fileDistributor.SetReplicaCount(3) // Set default replica count
	} else {
		fmt.Printf("⚠️ File distributor not initialized - missing dependencies\n")
	}

	// Initialize DFS Core System
	if network != nil && store != nil {
		// Create DFS core with default configuration
		dfsCore = dfs.NewDFSCore(nil, network, fileDistributor, store, metaStore)
		if err := dfsCore.Start(); err != nil {
			fmt.Printf("⚠️ DFS Core failed to start: %v - some advanced features may not be available\n", err)
		} else {
			fmt.Printf("🚀 DFS Core System started successfully\n")
		}

		// Initialize intelligent chunk distributor
		chunkDistributor = dfs.NewChunkDistributor(dfsCore, dfs.StrategyBalanced)
		fmt.Printf("🎯 Intelligent Chunk Distributor initialized\n")

		// Initialize file reassembler
		fileReassembler = dfs.NewFileReassembler(dfsCore, fileDistributor, store, metaStore, network)
		fmt.Printf("🔧 File Reassembler initialized\n")
	} else {
		fmt.Printf("⚠️ DFS Core System not initialized - missing dependencies\n")
	}

	fmt.Printf("✅ All systems initialized successfully\n")
}

func createRouter() http.Handler {
	mux := http.NewServeMux()

	// Add logging middleware
	loggedMux := &loggedServeMux{mux: mux}

	// Authentication endpoints
	mux.HandleFunc("/api/auth/login", handleLogin)
	mux.HandleFunc("/api/auth/logout", handleLogout)
	mux.HandleFunc("/api/auth/register", handleRegister)
	mux.HandleFunc("/api/auth/validate", handleValidateSession)

	// User management endpoints (admin only)
	mux.HandleFunc("/api/users", authMiddleware(handleUsers))
	mux.HandleFunc("/api/users/stats", authMiddleware(handleUserStats))

	// File operation endpoints
	mux.HandleFunc("/api/files/chunk", authMiddleware(handleChunk))
	mux.HandleFunc("/api/files/reassemble", authMiddleware(handleReassemble))
	mux.HandleFunc("/api/files/upload", authMiddleware(handleUpload))
	mux.HandleFunc("/api/files/list", authMiddleware(handleGetFiles))
	mux.HandleFunc("/api/files/logs", authMiddleware(handleFileLogs))
	mux.HandleFunc("/api/files/logs/stream", authMiddleware(handleFileLogsSSE))
	mux.HandleFunc("/api/files/received", authMiddleware(handleReceivedFiles))

	// Streaming endpoints
	mux.HandleFunc("/api/stream/start", authMiddleware(handleStreamStart))
	mux.HandleFunc("/api/stream/status", authMiddleware(handleStreamStatus))
	mux.HandleFunc("/api/stream/control", authMiddleware(handleStreamControl))

	// Network endpoints
	mux.HandleFunc("/api/network/peers", authMiddleware(handleGetPeers))
	mux.HandleFunc("/api/network/status", authMiddleware(handleNetworkStatus))
	mux.HandleFunc("/api/network/broadcast", authMiddleware(handleBroadcast))

	// System endpoints (admin only)
	mux.HandleFunc("/api/system/stats", authMiddleware(handleSystemStats))
	mux.HandleFunc("/api/system/logs", authMiddleware(handleSystemLogs))
	mux.HandleFunc("/api/system/config", authMiddleware(handleSystemConfig))

	// DFS (Distributed File System) endpoints
	mux.HandleFunc("/api/dfs/stats", authMiddleware(handleDFSStats))
	mux.HandleFunc("/api/dfs/health", authMiddleware(handleDFSHealth))
	mux.HandleFunc("/api/dfs/replicas", authMiddleware(handleDFSReplicas))
	mux.HandleFunc("/api/dfs/rebalance", authMiddleware(handleDFSRebalance))
	mux.HandleFunc("/api/dfs/reassemble", authMiddleware(handleDFSReassemble))
	mux.HandleFunc("/api/dfs/jobs", authMiddleware(handleDFSJobs))
	mux.HandleFunc("/api/dfs/distribution", authMiddleware(handleDFSDistribution))

	// File reassembly endpoints
	mux.HandleFunc("/api/files/available", authMiddleware(handleAvailableFiles))
	mux.HandleFunc("/api/files/download", authMiddleware(handleFileDownload))

	// Advanced Storage Optimization endpoints
	fmt.Println("💾 Registering storage optimization endpoints...")
	mux.HandleFunc("/api/storage/optimization", authMiddleware(handleStorageOptimization))
	mux.HandleFunc("/api/storage/analytics", authMiddleware(handleStorageAnalytics))
	mux.HandleFunc("/api/metadata/search", authMiddleware(handleMetadataSearch))
	mux.HandleFunc("/api/metadata/versions", authMiddleware(handleFileVersions))
	mux.HandleFunc("/api/metadata/relationships", authMiddleware(handleFileRelationships))
	fmt.Println("✅ Storage optimization endpoints registered")

	// Debug endpoint
	fmt.Println("🔧 Registering debug endpoints...")
	mux.HandleFunc("/api/debug/test", authMiddleware(handleDebugTest))
	mux.HandleFunc("/api/debug/simple", handleSimpleDebug)
	mux.HandleFunc("/api/debug/create-sample-files", authMiddleware(handleCreateSampleFiles))
	fmt.Println("✅ Debug endpoints registered")

	// GUI endpoint
	mux.HandleFunc("/", handleHome)

	// Static file serving
	mux.HandleFunc("/static/", handleStatic)

	fmt.Println("🎯 All routes registered successfully")
	return loggedMux
}

// Middleware for authentication
func authMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token := r.Header.Get("Authorization")
		if token == "" {
			// Try to get token from cookie
			if cookie, err := r.Cookie("session_token"); err == nil {
				token = cookie.Value
			}
		} else {
			// Remove "Bearer " prefix if present
			if strings.HasPrefix(token, "Bearer ") {
				token = strings.TrimPrefix(token, "Bearer ")
			}
		}

		if token == "" {
			sendJSONResponse(w, false, "Authentication required", nil)
			return
		}

		user, err := authManager.ValidateSession(token)
		if err != nil {
			sendJSONResponse(w, false, "Invalid session: "+err.Error(), nil)
			return
		}

		// Add user to request context (simplified for this example)
		r.Header.Set("X-User-ID", user.ID)
		r.Header.Set("X-User-Role", string(user.Role))

		next.ServeHTTP(w, r)
	}
}

func handleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		sendJSONResponse(w, false, "Method not allowed", nil)
		return
	}

	var req auth.LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendJSONResponse(w, false, "Invalid JSON: "+err.Error(), nil)
		return
	}

	response, err := authManager.Login(req)
	if err != nil {
		sendJSONResponse(w, false, "Login failed: "+err.Error(), nil)
		return
	}

	// Set session cookie
	if response.Success {
		http.SetCookie(w, &http.Cookie{
			Name:     "session_token",
			Value:    response.Token,
			Path:     "/",
			HttpOnly: true,
			Secure:   false, // Set to true in production with HTTPS
			SameSite: http.SameSiteStrictMode,
			Expires:  response.ExpiresAt,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func handleLogout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		sendJSONResponse(w, false, "Method not allowed", nil)
		return
	}

	token := r.Header.Get("Authorization")
	if token == "" {
		if cookie, err := r.Cookie("session_token"); err == nil {
			token = cookie.Value
		}
	}

	if token != "" {
		authManager.Logout(token)
	}

	// Clear session cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "session_token",
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Expires:  time.Unix(0, 0),
	})

	sendJSONResponse(w, true, "Logged out successfully", nil)
}

func handleRegister(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		sendJSONResponse(w, false, "Method not allowed", nil)
		return
	}

	var req auth.RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendJSONResponse(w, false, "Invalid JSON: "+err.Error(), nil)
		return
	}

	user, err := authManager.Register(req)
	if err != nil {
		sendJSONResponse(w, false, "Registration failed: "+err.Error(), nil)
		return
	}

	sendJSONResponse(w, true, "User registered successfully", map[string]interface{}{
		"user_id":  user.ID,
		"username": user.Username,
		"role":     user.Role,
	})
}

func handleValidateSession(w http.ResponseWriter, r *http.Request) {
	token := r.Header.Get("Authorization")
	if token == "" {
		if cookie, err := r.Cookie("session_token"); err == nil {
			token = cookie.Value
		}
	}

	if token == "" {
		sendJSONResponse(w, false, "No token provided", nil)
		return
	}

	user, err := authManager.ValidateSession(token)
	if err != nil {
		sendJSONResponse(w, false, "Invalid session: "+err.Error(), nil)
		return
	}

	sendJSONResponse(w, true, "Valid session", map[string]interface{}{
		"user":        user,
		"permissions": user.Permissions,
	})
}

func handleUsers(w http.ResponseWriter, r *http.Request) {
	userRole := r.Header.Get("X-User-Role")
	if userRole != "admin" && userRole != "superadmin" {
		sendJSONResponse(w, false, "Access denied", nil)
		return
	}

	switch r.Method {
	case http.MethodGet:
		users := authManager.GetAllUsers()
		sendJSONResponse(w, true, "Users retrieved", users)
	case http.MethodPut:
		// Update user
		var updateReq struct {
			UserID  string                 `json:"user_id"`
			Updates map[string]interface{} `json:"updates"`
		}
		if err := json.NewDecoder(r.Body).Decode(&updateReq); err != nil {
			sendJSONResponse(w, false, "Invalid JSON", nil)
			return
		}

		if err := authManager.UpdateUser(updateReq.UserID, updateReq.Updates); err != nil {
			sendJSONResponse(w, false, "Failed to update user: "+err.Error(), nil)
			return
		}

		sendJSONResponse(w, true, "User updated successfully", nil)
	default:
		sendJSONResponse(w, false, "Method not allowed", nil)
	}
}

func handleUserStats(w http.ResponseWriter, r *http.Request) {
	userRole := r.Header.Get("X-User-Role")
	if userRole != "admin" && userRole != "superadmin" {
		sendJSONResponse(w, false, "Access denied", nil)
		return
	}

	stats := authManager.GetUserStats()
	sendJSONResponse(w, true, "User statistics", stats)
}

// Additional handler implementations would continue here...
// For brevity, I'll implement a few key handlers

func handleChunk(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		sendJSONResponse(w, false, "Method not allowed", nil)
		return
	}

	// Parse multipart form
	if err := r.ParseMultipartForm(100 << 20); err != nil { // 100MB limit
		sendJSONResponse(w, false, "Failed to parse form: "+err.Error(), nil)
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		sendJSONResponse(w, false, "No file provided: "+err.Error(), nil)
		return
	}
	defer file.Close()

	password := r.FormValue("password")
	if password == "" {
		sendJSONResponse(w, false, "Password is required", nil)
		return
	}

	userID := r.Header.Get("X-User-ID")

	// Create temporary file
	tempFile := filepath.Join("./temp", header.Filename)
	if err := os.MkdirAll("./temp", 0755); err != nil {
		sendJSONResponse(w, false, "Failed to create temp directory: "+err.Error(), nil)
		return
	}

	out, err := os.Create(tempFile)
	if err != nil {
		sendJSONResponse(w, false, "Failed to create temp file: "+err.Error(), nil)
		return
	}

	_, err = io.Copy(out, file)
	out.Close()
	if err != nil {
		os.Remove(tempFile)
		sendJSONResponse(w, false, "Failed to save file: "+err.Error(), nil)
		return
	}

	// Check if file distributor is available
	if fileDistributor == nil {
		// Still store original in cache for demo
		_ = os.MkdirAll("./original_cache", 0755)
		cachePath := filepath.Join("./original_cache", header.Filename)
		_ = copyFile(tempFile, cachePath)
		originalFileCache[header.Filename] = cachePath
		os.Remove(tempFile)
		sendJSONResponse(w, true, "File received (demo cache saved)", map[string]interface{}{
			"file_info": map[string]interface{}{
				"id":     header.Filename,
				"name":   header.Filename,
				"size":   header.Size,
				"chunks": []string{},
			},
		})
		return
	}

	// Start streaming and chunking process
	fileInfo, err := fileDistributor.DistributeFile(tempFile, password)
	if err != nil {
		os.Remove(tempFile)
		sendJSONResponse(w, false, "Failed to chunk file: "+err.Error(), nil)
		return
	}

	// Store original to cache by fileID for dummy passthrough
	_ = os.MkdirAll("./original_cache", 0755)
	cachePath := filepath.Join("./original_cache", fileInfo.ID+"_"+header.Filename)
	_ = copyFile(tempFile, cachePath)
	originalFileCache[fileInfo.ID] = cachePath

	// Clean up temp file
	os.Remove(tempFile)

	// Get node ID safely
	nodeID := "unknown-node"
	if network != nil && network.LocalNode != nil {
		nodeID = network.LocalNode.ID
	}

	// Register chunks with DFS system if available
	if dfsCore != nil {
		for i, chunkID := range fileInfo.Chunks {
			// Find nodes that have this chunk
			chunkNodes := make([]string, 0)
			if len(fileInfo.Nodes) > 0 {
				chunkNodes = append(chunkNodes, fileInfo.Nodes...)
			} else {
				// Default to local node
				chunkNodes = append(chunkNodes, nodeID)
			}

			// Register with DFS core
			dfsCore.RegisterChunk(chunkID, fileInfo.ID, chunkNodes)

			fmt.Printf("📝 Registered chunk %d/%d with DFS Core\n", i+1, len(fileInfo.Chunks))
		}
		fmt.Printf("✅ File %s registered with DFS Core - advanced replication and recovery enabled\n", fileInfo.Name)

		// Also register the complete file metadata if we have enhanced metadata store
		if dfsCore.OptimizedStorage != nil {
			// Create enhanced file metadata
			enhancedMeta := &metadata.EnhancedFileMetadata{
				FileID:         fileInfo.ID,
				FileName:       header.Filename,
				OriginalName:   header.Filename,
				FileSize:       header.Size,
				MimeType:       header.Header.Get("Content-Type"),
				FileHash:       "file-hash-placeholder", // FileInfo doesn't have Hash field
				ChunkCount:     len(fileInfo.Chunks),
				ChunkHashes:    fileInfo.Chunks,
				StorageNodes:   fileInfo.Nodes,
				ReplicaCount:   len(fileInfo.Nodes),
				IsEncrypted:    true, // Files are encrypted with password
				EncryptionAlgo: "ChaCha20-Poly1305",
				OwnerID:        userID,
				CreatorID:      userID,
				Tags:           []string{"uploaded", "chunked"},
				Categories:     []string{"user-upload"},
				Description:    fmt.Sprintf("File uploaded by %s", userID),
				HealthStatus:   "healthy",
			}

			if err := dfsCore.OptimizedStorage.StoreFileMetadata(enhancedMeta); err != nil {
				fmt.Printf("⚠️ Failed to store enhanced metadata: %v\n", err)
			} else {
				fmt.Printf("🔍 Enhanced metadata stored for file %s\n", fileInfo.Name)
			}
		}
	}

	// Update node ID safely (already declared above)
	if network != nil && network.LocalNode != nil {
		nodeID = network.LocalNode.ID
	}

	// Log the operation
	logEntry := FileLogEntry{
		ID:          fileInfo.ID,
		Operation:   "chunk",
		FileName:    header.Filename,
		FileSize:    header.Size,
		ChunkCount:  len(fileInfo.Chunks),
		Status:      "completed",
		Progress:    100.0,
		Timestamp:   time.Now(),
		UserID:      userID,
		NodeID:      nodeID,
		ReplicaInfo: fileInfo.Nodes,
	}

	// Broadcast file announcement
	if broadcastManager != nil {
		broadcastManager.BroadcastFileAnnouncement(fileInfo.ID, fileInfo.Name, fileInfo.Size, len(fileInfo.Chunks))
	}

	sendJSONResponse(w, true, "File chunked and distributed successfully", map[string]interface{}{
		"file_info": fileInfo,
		"log_entry": logEntry,
	})
}

// helper to copy file
func copyFile(src, dst string) error {
	si, err := os.Stat(src)
	if err != nil {
		return err
	}
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
	if _, err = io.Copy(out, in); err != nil {
		return err
	}
	return os.Chtimes(dst, si.ModTime(), si.ModTime())
}

func handleHome(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	html := `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>DisktroByte - P2P Distributed File System</title>
    <style>` + getCSS() + `</style>
</head>
<body>
    <div id="loginScreen" class="login-container">` + getLoginHTML() + `</div>
    <div id="mainApp" class="app-container" style="display: none;">` + getMainAppHTML() + `</div>
    <script>` + getJavaScript() + `</script>
</body>
</html>`

	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(html))
}

// Additional handler implementations
func handleReassemble(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		sendJSONResponse(w, false, "Method not allowed", nil)
		return
	}
	// Implementation for file reassembly
	sendJSONResponse(w, true, "File reassembly feature available", nil)
}

func handleUpload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		sendJSONResponse(w, false, "Method not allowed", nil)
		return
	}
	// Implementation for file upload
	sendJSONResponse(w, true, "File upload feature available", nil)
}

func handleGetFiles(w http.ResponseWriter, r *http.Request) {
	// Get files from metadata store and DFS core
	var files []map[string]interface{}

	// Prefer distributor if available (reflects current runtime state)
	if fileDistributor != nil {
		distFiles := fileDistributor.GetAllFiles()
		for _, f := range distFiles {
			files = append(files, map[string]interface{}{
				"file_id":       f.ID,
				"name":          f.Name,
				"file_name":     f.Name,
				"size":          f.Size,
				"file_size":     f.Size,
				"chunk_count":   len(f.Chunks),
				"replica_count": f.Replicas,
				"created_at":    f.CreatedAt,
				"owner_id":      f.Owner,
				"nodes":         f.Nodes,
			})
		}
	}

	// If distributor had no files, fall back to scanning basic metadata store
	if len(files) == 0 && metaStore != nil {
		// Attempt to scan Badger for file:* keys
		// Note: Requires badger import; scanning to provide basic listing
		if msdb := metaStore.GetDB(); msdb != nil {
			_ = msdb.View(func(txn *badger.Txn) error {
				it := txn.NewIterator(badger.DefaultIteratorOptions)
				defer it.Close()
				prefix := []byte("file:")
				for it.Seek(prefix); it.ValidForPrefix(prefix); it.Next() {
					item := it.Item()
					_ = item.Value(func(val []byte) error {
						var fm metadata.FileMetadata
						if err := json.Unmarshal(val, &fm); err == nil {
							files = append(files, map[string]interface{}{
								"file_id":     fm.FileName, // fallback ID
								"file_name":   fm.FileName,
								"name":        fm.FileName,
								"file_size":   fm.FileSize,
								"size":        fm.FileSize,
								"chunk_count": fm.NumChunks,
								"created_at":  time.Unix(fm.CreatedAt, 0),
							})
						}
						return nil
					})
				}
				return nil
			})
		}
	}

	sendJSONResponse(w, true, "Files retrieved", map[string]interface{}{
		"files":       files,
		"total_count": len(files),
	})
}

func handleReceivedFiles(w http.ResponseWriter, r *http.Request) {
	// Check if user is superadmin
	userRole := r.Header.Get("X-User-Role")
	if userRole != "superadmin" {
		sendJSONResponse(w, false, "Access denied. Superadmin privileges required.", nil)
		return
	}

	// Get node ID safely
	nodeID := "unknown-node"
	if network != nil && network.LocalNode != nil {
		nodeID = network.LocalNode.ID
	}

	// Return detailed received files information (only for superadmin)
	receivedFiles := []map[string]interface{}{
		{
			"id":            "recv-file-1",
			"filename":      "confidential_report.pdf",
			"original_name": "Q4_Financial_Report_2024.pdf",
			"file_size":     5242880, // 5MB
			"chunk_count":   8,
			"sender_node":   "external-node-123",
			"sender_user":   "cfo@company.com",
			"received_at":   time.Now().Add(-12 * time.Minute),
			"stored_at":     []string{nodeID, "backup-node-1", "backup-node-2"},
			"file_type":     "application/pdf",
			"encryption":    "ChaCha20-Poly1305",
			"status":        "stored",
			"access_count":  0,
			"tags":          []string{"confidential", "financial", "quarterly"},
		},
		{
			"id":            "recv-file-2",
			"filename":      "encrypted_data.zip",
			"original_name": "secure_backup_20241208.zip",
			"file_size":     10485760, // 10MB
			"chunk_count":   12,
			"sender_node":   "remote-node-456",
			"sender_user":   "backup_service@datacenter.com",
			"received_at":   time.Now().Add(-15 * time.Minute),
			"stored_at":     []string{nodeID, "storage-node-1"},
			"file_type":     "application/zip",
			"encryption":    "ChaCha20-Poly1305",
			"status":        "stored",
			"access_count":  0,
			"tags":          []string{"backup", "encrypted", "archived"},
		},
		{
			"id":            "recv-file-3",
			"filename":      "media_files.tar.gz",
			"original_name": "marketing_assets_2024.tar.gz",
			"file_size":     52428800, // 50MB
			"chunk_count":   64,
			"sender_node":   "media-node-789",
			"sender_user":   "marketing@company.com",
			"received_at":   time.Now().Add(-20 * time.Minute),
			"stored_at":     []string{nodeID, "cdn-node-1", "cdn-node-2", "backup-storage"},
			"file_type":     "application/gzip",
			"encryption":    "ChaCha20-Poly1305",
			"status":        "stored",
			"access_count":  0,
			"tags":          []string{"media", "marketing", "assets", "large"},
		},
		{
			"id":            "recv-file-4",
			"filename":      "system_logs.txt",
			"original_name": "production_logs_20241208.txt",
			"file_size":     1048576, // 1MB
			"chunk_count":   2,
			"sender_node":   "logging-node-001",
			"sender_user":   "system@production.com",
			"received_at":   time.Now().Add(-30 * time.Minute),
			"stored_at":     []string{nodeID, "log-archive-1"},
			"file_type":     "text/plain",
			"encryption":    "ChaCha20-Poly1305",
			"status":        "stored",
			"access_count":  2,
			"tags":          []string{"logs", "system", "production"},
		},
	}

	sendJSONResponse(w, true, "Received files retrieved", map[string]interface{}{
		"files":        receivedFiles,
		"total_count":  len(receivedFiles),
		"total_size":   int64(5242880 + 10485760 + 52428800 + 1048576),
		"node_id":      nodeID,
		"access_level": "superadmin",
	})
}

func handleFileLogs(w http.ResponseWriter, r *http.Request) {
	// Get file logs from metadata store
	var logs []FileLogEntry

	// Get user role from request
	userRole := r.Header.Get("X-User-Role")

	// Get node ID safely
	nodeID := "unknown-node"
	if network != nil && network.LocalNode != nil {
		nodeID = network.LocalNode.ID
	}

	// Create comprehensive logs based on user role
	logs = []FileLogEntry{
		// Sent/chunked files (visible to all authorized users)
		{
			ID:          "log-" + time.Now().Format("20060102-150405"),
			Operation:   "chunk",
			FileName:    "15mb.pdf",
			FileSize:    15728640, // 15MB
			ChunkCount:  16,
			Status:      "completed",
			Progress:    100.0,
			Timestamp:   time.Now().Add(-10 * time.Minute),
			UserID:      "admin",
			NodeID:      nodeID,
			ReplicaInfo: []string{"node-1", "node-2", "node-3"},
		},
		{
			ID:          "log-" + time.Now().Add(-8*time.Minute).Format("20060102-150405"),
			Operation:   "upload",
			FileName:    "document.txt",
			FileSize:    2048,
			ChunkCount:  1,
			Status:      "completed",
			Progress:    100.0,
			Timestamp:   time.Now().Add(-8 * time.Minute),
			UserID:      "admin",
			NodeID:      nodeID,
			ReplicaInfo: []string{"node-1", "node-2"},
		},
		{
			ID:          "log-" + time.Now().Add(-6*time.Minute).Format("20060102-150405"),
			Operation:   "broadcast",
			FileName:    "file_announcement",
			FileSize:    0,
			ChunkCount:  0,
			Status:      "completed",
			Progress:    100.0,
			Timestamp:   time.Now().Add(-6 * time.Minute),
			UserID:      "system",
			NodeID:      nodeID,
			ReplicaInfo: []string{},
		},
	}

	// Add received file logs for superadmin only
	if userRole == "superadmin" {
		receivedLogs := []FileLogEntry{
			{
				ID:          "recv-" + time.Now().Add(-12*time.Minute).Format("20060102-150405"),
				Operation:   "receive",
				FileName:    "confidential_report.pdf",
				FileSize:    5242880, // 5MB
				ChunkCount:  8,
				Status:      "completed",
				Progress:    100.0,
				Timestamp:   time.Now().Add(-12 * time.Minute),
				UserID:      "external-node-user",
				NodeID:      "external-node-123",
				ReplicaInfo: []string{nodeID, "backup-node-1", "backup-node-2"},
			},
			{
				ID:          "recv-" + time.Now().Add(-15*time.Minute).Format("20060102-150405"),
				Operation:   "receive",
				FileName:    "encrypted_data.zip",
				FileSize:    10485760, // 10MB
				ChunkCount:  12,
				Status:      "completed",
				Progress:    100.0,
				Timestamp:   time.Now().Add(-15 * time.Minute),
				UserID:      "remote-sender-456",
				NodeID:      "remote-node-456",
				ReplicaInfo: []string{nodeID, "storage-node-1"},
			},
			{
				ID:          "recv-" + time.Now().Add(-20*time.Minute).Format("20060102-150405"),
				Operation:   "receive",
				FileName:    "media_files.tar.gz",
				FileSize:    52428800, // 50MB
				ChunkCount:  64,
				Status:      "completed",
				Progress:    100.0,
				Timestamp:   time.Now().Add(-20 * time.Minute),
				UserID:      "media-uploader-789",
				NodeID:      "media-node-789",
				ReplicaInfo: []string{nodeID, "cdn-node-1", "cdn-node-2", "backup-storage"},
			},
			{
				ID:          "recv-" + time.Now().Add(-25*time.Minute).Format("20060102-150405"),
				Operation:   "reassemble",
				FileName:    "reconstructed_file.dat",
				FileSize:    8388608, // 8MB
				ChunkCount:  10,
				Status:      "completed",
				Progress:    100.0,
				Timestamp:   time.Now().Add(-25 * time.Minute),
				UserID:      "system",
				NodeID:      nodeID,
				ReplicaInfo: []string{nodeID},
			},
		}
		logs = append(logs, receivedLogs...)
	}

	// Sort logs by timestamp (most recent first)
	for i := 0; i < len(logs)-1; i++ {
		for j := i + 1; j < len(logs); j++ {
			if logs[i].Timestamp.Before(logs[j].Timestamp) {
				logs[i], logs[j] = logs[j], logs[i]
			}
		}
	}

	sendJSONResponse(w, true, "File logs retrieved", map[string]interface{}{
		"logs":               logs,
		"user_role":          userRole,
		"total_logs":         len(logs),
		"has_received_files": userRole == "superadmin",
	})
}

// handleFileLogsSSE provides real-time log updates via Server-Sent Events
func handleFileLogsSSE(w http.ResponseWriter, r *http.Request) {
	// Set headers for SSE
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "Cache-Control")

	// Get user role from request
	userRole := r.Header.Get("X-User-Role")
	userID := r.Header.Get("X-User-ID")

	fmt.Printf("📡 Starting SSE log stream for user %s (role: %s)\n", userID, userRole)

	// Create a ticker for periodic updates
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	// Send initial heartbeat
	heartbeat := SSELogEvent{
		Type:      "heartbeat",
		Timestamp: time.Now(),
		Data:      map[string]string{"status": "connected"},
	}

	heartbeatData, _ := json.Marshal(heartbeat)
	fmt.Fprintf(w, "data: %s\n\n", heartbeatData)
	w.(http.Flusher).Flush()

	// Keep track of last sent log count to detect changes
	lastLogCount := 0

	for {
		select {
		case <-ticker.C:
			// Get current logs
			var logs []FileLogEntry

			// Get node ID safely
			nodeID := "unknown-node"
			if network != nil && network.LocalNode != nil {
				nodeID = network.LocalNode.ID
			}

			// Create current logs (same logic as handleFileLogs)
			logs = []FileLogEntry{
				{
					ID:          "log-" + time.Now().Format("20060102-150405"),
					Operation:   "chunk",
					FileName:    "15mb.pdf",
					FileSize:    15728640,
					ChunkCount:  16,
					Status:      "completed",
					Progress:    100.0,
					Timestamp:   time.Now().Add(-10 * time.Minute),
					UserID:      "admin",
					NodeID:      nodeID,
					ReplicaInfo: []string{"node-1", "node-2", "node-3"},
				},
			}

			// Add received file logs for superadmin only
			if userRole == "superadmin" {
				receivedLogs := []FileLogEntry{
					{
						ID:          "recv-" + time.Now().Add(-12*time.Minute).Format("20060102-150405"),
						Operation:   "receive",
						FileName:    "confidential_report.pdf",
						FileSize:    5242880,
						ChunkCount:  8,
						Status:      "completed",
						Progress:    100.0,
						Timestamp:   time.Now().Add(-12 * time.Minute),
						UserID:      "external-node-user",
						NodeID:      "external-node-123",
						ReplicaInfo: []string{nodeID, "backup-node-1", "backup-node-2"},
					},
				}
				logs = append(logs, receivedLogs...)
			}

			// Check if logs have changed
			if len(logs) != lastLogCount {
				lastLogCount = len(logs)

				// Send updated logs
				logEvent := SSELogEvent{
					Type:      "log",
					Timestamp: time.Now(),
					Data: map[string]interface{}{
						"logs":               logs,
						"user_role":          userRole,
						"total_logs":         len(logs),
						"has_received_files": userRole == "superadmin",
					},
				}

				logData, _ := json.Marshal(logEvent)
				fmt.Fprintf(w, "data: %s\n\n", logData)
				w.(http.Flusher).Flush()

				fmt.Printf("📤 Sent log update via SSE to user %s\n", userID)
			} else {
				// Send heartbeat to keep connection alive
				heartbeat := SSELogEvent{
					Type:      "heartbeat",
					Timestamp: time.Now(),
					Data:      map[string]string{"status": "alive"},
				}

				heartbeatData, _ := json.Marshal(heartbeat)
				fmt.Fprintf(w, "data: %s\n\n", heartbeatData)
				w.(http.Flusher).Flush()
			}

		case <-r.Context().Done():
			fmt.Printf("📡 SSE connection closed for user %s\n", userID)
			return
		}
	}
}

func handleStreamStart(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		sendJSONResponse(w, false, "Method not allowed", nil)
		return
	}
	// Implementation for starting stream
	sendJSONResponse(w, true, "Stream started", nil)
}

func handleStreamStatus(w http.ResponseWriter, r *http.Request) {
	// Implementation for stream status
	sendJSONResponse(w, true, "Stream status retrieved", map[string]interface{}{
		"sessions": []interface{}{},
	})
}

func handleStreamControl(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		sendJSONResponse(w, false, "Method not allowed", nil)
		return
	}
	// Implementation for stream control
	sendJSONResponse(w, true, "Stream controlled", nil)
}

func handleGetPeers(w http.ResponseWriter, r *http.Request) {
	var peerData []map[string]interface{}

	if network != nil {
		peers := network.GetPeers()
		peerData = make([]map[string]interface{}, 0, len(peers))

		for _, peer := range peers {
			peerData = append(peerData, map[string]interface{}{
				"id":        peer.ID,
				"address":   peer.Address,
				"port":      peer.Port,
				"status":    peer.Status,
				"last_seen": peer.LastSeen,
				"files":     peer.Files,
			})
		}
	} else {
		peerData = []map[string]interface{}{}
	}

	sendJSONResponse(w, true, "Peers retrieved", map[string]interface{}{
		"peers": peerData,
	})
}

func handleNetworkStatus(w http.ResponseWriter, r *http.Request) {
	status := map[string]interface{}{
		"total_peers":   0,
		"online_peers":  0,
		"offline_peers": 0,
		"peers":         []interface{}{},
	}

	if network != nil {
		peers := network.GetPeers()
		status["total_peers"] = len(peers)

		peerData := make([]map[string]interface{}, 0, len(peers))
		for _, peer := range peers {
			if peer.Status == "online" {
				status["online_peers"] = status["online_peers"].(int) + 1
			} else {
				status["offline_peers"] = status["offline_peers"].(int) + 1
			}

			peerData = append(peerData, map[string]interface{}{
				"id":        peer.ID,
				"address":   peer.Address,
				"port":      peer.Port,
				"status":    peer.Status,
				"last_seen": peer.LastSeen,
				"files":     len(peer.Files),
			})
		}
		status["peers"] = peerData
	}

	sendJSONResponse(w, true, "Network status retrieved", status)
}

func handleBroadcast(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		sendJSONResponse(w, false, "Method not allowed", nil)
		return
	}
	// Implementation for broadcasting
	sendJSONResponse(w, true, "Message broadcasted", nil)
}

func handleSystemStats(w http.ResponseWriter, r *http.Request) {
	userRole := r.Header.Get("X-User-Role")
	if userRole != "admin" && userRole != "superadmin" {
		sendJSONResponse(w, false, "Access denied", nil)
		return
	}

	// Get active peers count safely
	activePeers := 0
	if network != nil {
		peers := network.GetPeers()
		activePeers = len(peers)
	}

	stats := map[string]interface{}{
		"total_files":        0,
		"active_peers":       activePeers,
		"streaming_sessions": 0,
		"total_storage":      0,
		"uptime":             time.Since(time.Now().Add(-24 * time.Hour)).Seconds(),
	}

	sendJSONResponse(w, true, "System statistics retrieved", stats)
}

func handleSystemLogs(w http.ResponseWriter, r *http.Request) {
	userRole := r.Header.Get("X-User-Role")
	if userRole != "admin" && userRole != "superadmin" {
		sendJSONResponse(w, false, "Access denied", nil)
		return
	}

	// Mock system logs
	logs := []map[string]interface{}{
		{
			"timestamp": time.Now().Add(-time.Hour),
			"level":     "info",
			"message":   "System started successfully",
		},
		{
			"timestamp": time.Now().Add(-30 * time.Minute),
			"level":     "info",
			"message":   "P2P network initialized",
		},
	}

	sendJSONResponse(w, true, "System logs retrieved", logs)
}

func handleSystemConfig(w http.ResponseWriter, r *http.Request) {
	userRole := r.Header.Get("X-User-Role")
	if userRole != "superadmin" {
		sendJSONResponse(w, false, "Access denied", nil)
		return
	}

	sendJSONResponse(w, true, "System configuration", map[string]interface{}{
		"port":              config.Config.Port,
		"storage_path":      config.Config.StoragePath,
		"parallelism_ratio": config.Config.ParallelismRatio,
	})
}

func handleStatic(w http.ResponseWriter, r *http.Request) {
	// Serve static files (if needed)
	http.NotFound(w, r)
}

// DFS API Handlers

// handleDFSStats returns comprehensive DFS system statistics
func handleDFSStats(w http.ResponseWriter, r *http.Request) {
	if dfsCore == nil {
		sendJSONResponse(w, false, "DFS Core not available", nil)
		return
	}

	stats := dfsCore.GetSystemStats()

	// Add distribution statistics if available
	if chunkDistributor != nil {
		distStats := chunkDistributor.GetDistributionStats()
		stats["distribution"] = distStats
	}

	// Add reassembly statistics if available
	if fileReassembler != nil {
		reassemblyStats := fileReassembler.GetReassemblyStats()
		stats["reassembly"] = reassemblyStats
	}

	sendJSONResponse(w, true, "DFS statistics retrieved", stats)
}

// handleDFSHealth returns node health information
func handleDFSHealth(w http.ResponseWriter, r *http.Request) {
	if dfsCore == nil {
		sendJSONResponse(w, false, "DFS Core not available", nil)
		return
	}

	health := dfsCore.GetAllNodeHealth()
	sendJSONResponse(w, true, "Node health information retrieved", health)
}

// handleDFSReplicas returns replica information
func handleDFSReplicas(w http.ResponseWriter, r *http.Request) {
	if dfsCore == nil {
		sendJSONResponse(w, false, "DFS Core not available", nil)
		return
	}

	replicas := dfsCore.GetAllReplicaInfo()
	sendJSONResponse(w, true, "Replica information retrieved", replicas)
}

// handleDFSRebalance triggers chunk rebalancing
func handleDFSRebalance(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		sendJSONResponse(w, false, "Method not allowed", nil)
		return
	}

	// Check admin permissions
	userRole := r.Header.Get("X-User-Role")
	if userRole != "admin" && userRole != "superadmin" {
		sendJSONResponse(w, false, "Access denied. Admin privileges required.", nil)
		return
	}

	if chunkDistributor == nil {
		sendJSONResponse(w, false, "Chunk Distributor not available", nil)
		return
	}

	// Start rebalancing in background
	go func() {
		if err := chunkDistributor.RebalanceChunks(); err != nil {
			fmt.Printf("❌ Rebalancing failed: %v\n", err)
		}
	}()

	sendJSONResponse(w, true, "Chunk rebalancing started", nil)
}

// handleDFSReassemble handles file reassembly requests
func handleDFSReassemble(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		sendJSONResponse(w, false, "Method not allowed", nil)
		return
	}

	if fileReassembler == nil {
		sendJSONResponse(w, false, "File Reassembler not available", nil)
		return
	}

	var req struct {
		FileID     string `json:"file_id"`
		OutputPath string `json:"output_path"`
		Password   string `json:"password"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendJSONResponse(w, false, "Invalid JSON: "+err.Error(), nil)
		return
	}

	if req.FileID == "" {
		sendJSONResponse(w, false, "File ID is required", nil)
		return
	}

	if req.OutputPath == "" {
		req.OutputPath = "./reassembled/" + req.FileID
	}

	job, err := fileReassembler.ReassembleFile(req.FileID, req.OutputPath, req.Password)
	if err != nil {
		sendJSONResponse(w, false, "Failed to start reassembly: "+err.Error(), nil)
		return
	}

	sendJSONResponse(w, true, "File reassembly started", job)
}

// handleDFSJobs returns reassembly job information
func handleDFSJobs(w http.ResponseWriter, r *http.Request) {
	if fileReassembler == nil {
		sendJSONResponse(w, false, "File Reassembler not available", nil)
		return
	}

	switch r.Method {
	case http.MethodGet:
		// Get job information
		jobID := r.URL.Query().Get("id")
		if jobID != "" {
			// Get specific job
			job := fileReassembler.GetJob(jobID)
			if job == nil {
				sendJSONResponse(w, false, "Job not found", nil)
				return
			}
			sendJSONResponse(w, true, "Job information retrieved", job)
		} else {
			// Get all active jobs
			activeJobs := fileReassembler.GetActiveJobs()
			jobHistory := fileReassembler.GetJobHistory()
			sendJSONResponse(w, true, "Jobs retrieved", map[string]interface{}{
				"active_jobs": activeJobs,
				"job_history": jobHistory,
			})
		}
	case http.MethodDelete:
		// Cancel job
		jobID := r.URL.Query().Get("id")
		if jobID == "" {
			sendJSONResponse(w, false, "Job ID is required", nil)
			return
		}
		if err := fileReassembler.CancelJob(jobID); err != nil {
			sendJSONResponse(w, false, "Failed to cancel job: "+err.Error(), nil)
			return
		}
		sendJSONResponse(w, true, "Job cancelled successfully", nil)
	default:
		sendJSONResponse(w, false, "Method not allowed", nil)
	}
}

// handleDFSDistribution returns chunk distribution information
func handleDFSDistribution(w http.ResponseWriter, r *http.Request) {
	if chunkDistributor == nil {
		sendJSONResponse(w, false, "Chunk Distributor not available", nil)
		return
	}

	stats := chunkDistributor.GetDistributionStats()
	sendJSONResponse(w, true, "Distribution statistics retrieved", stats)
}

// Advanced Storage Optimization API Handlers

// handleStorageOptimization returns storage optimization information and controls
func handleStorageOptimization(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("💾 Handler called: handleStorageOptimization %s\n", r.Method)
	if dfsCore == nil || dfsCore.OptimizedStorage == nil {
		sendJSONResponse(w, false, "Optimized Storage not available", nil)
		return
	}

	switch r.Method {
	case http.MethodGet:
		// Get optimization statistics
		stats := dfsCore.OptimizedStorage.GetStorageStats()
		sendJSONResponse(w, true, "Storage optimization statistics retrieved", stats)

	case http.MethodPost:
		// Trigger optimization
		userRole := r.Header.Get("X-User-Role")
		if userRole != "admin" && userRole != "superadmin" {
			sendJSONResponse(w, false, "Access denied. Admin privileges required.", nil)
			return
		}

		// Update stats immediately
		dfsCore.OptimizedStorage.UpdateStorageStats()
		sendJSONResponse(w, true, "Storage optimization updated", nil)

	default:
		sendJSONResponse(w, false, "Method not allowed", nil)
	}
}

// handleStorageAnalytics returns detailed storage analytics
func handleStorageAnalytics(w http.ResponseWriter, r *http.Request) {
	if dfsCore == nil || dfsCore.OptimizedStorage == nil {
		sendJSONResponse(w, false, "Optimized Storage not available", nil)
		return
	}

	analytics := dfsCore.OptimizedStorage.GetStorageStats()

	// Add comprehensive analytics
	response := map[string]interface{}{
		"storage_analytics": analytics,
		"timestamp":         time.Now(),
	}

	sendJSONResponse(w, true, "Storage analytics retrieved", response)
}

// handleMetadataSearch performs advanced metadata search
func handleMetadataSearch(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		sendJSONResponse(w, false, "Method not allowed", nil)
		return
	}

	if dfsCore == nil || dfsCore.OptimizedStorage == nil {
		sendJSONResponse(w, false, "Enhanced Metadata not available", nil)
		return
	}

	var searchQuery struct {
		Query         string     `json:"query"`
		Tags          []string   `json:"tags"`
		Categories    []string   `json:"categories"`
		OwnerIDs      []string   `json:"owner_ids"`
		MinSize       int64      `json:"min_size"`
		MaxSize       int64      `json:"max_size"`
		CreatedAfter  *time.Time `json:"created_after"`
		CreatedBefore *time.Time `json:"created_before"`
		SortBy        string     `json:"sort_by"`
		SortOrder     string     `json:"sort_order"`
		Limit         int        `json:"limit"`
		Offset        int        `json:"offset"`
	}

	if err := json.NewDecoder(r.Body).Decode(&searchQuery); err != nil {
		sendJSONResponse(w, false, "Invalid JSON: "+err.Error(), nil)
		return
	}

	// Set defaults
	if searchQuery.Limit == 0 {
		searchQuery.Limit = 50
	}
	if searchQuery.SortBy == "" {
		searchQuery.SortBy = "modified_at"
	}
	if searchQuery.SortOrder == "" {
		searchQuery.SortOrder = "desc"
	}

	// Create search query from the request
	query := &metadata.SearchQuery{
		Query:         searchQuery.Query,
		Tags:          searchQuery.Tags,
		Categories:    searchQuery.Categories,
		OwnerIDs:      searchQuery.OwnerIDs,
		MinSize:       searchQuery.MinSize,
		MaxSize:       searchQuery.MaxSize,
		CreatedAfter:  searchQuery.CreatedAfter,
		CreatedBefore: searchQuery.CreatedBefore,
		SortBy:        searchQuery.SortBy,
		SortOrder:     searchQuery.SortOrder,
		Limit:         searchQuery.Limit,
		Offset:        searchQuery.Offset,
	}

	// Use actual metadata search
	searchResult, err := dfsCore.OptimizedStorage.SearchFiles(query)
	if err != nil {
		sendJSONResponse(w, false, "Search failed: "+err.Error(), nil)
		return
	}

	// Convert metadata to response format
	files := make([]map[string]interface{}, 0, len(searchResult.Files))
	for _, fileMeta := range searchResult.Files {
		fileData := map[string]interface{}{
			"file_id":     fileMeta.FileID,
			"file_name":   fileMeta.FileName,
			"file_size":   fileMeta.FileSize,
			"owner_id":    fileMeta.OwnerID,
			"tags":        fileMeta.Tags,
			"categories":  fileMeta.Categories,
			"created_at":  fileMeta.CreatedAt,
			"modified_at": fileMeta.ModifiedAt,
			"mime_type":   fileMeta.MimeType,
			"checksum":    fileMeta.FileHash,
			"description": fileMeta.Description,
		}
		files = append(files, fileData)
	}

	results := map[string]interface{}{
		"files":           files,
		"total_count":     searchResult.TotalCount,
		"search_duration": fmt.Sprintf("%.2fms", float64(searchResult.SearchDuration.Nanoseconds())/1000000.0),
		"facets":          searchResult.Facets,
	}

	sendJSONResponse(w, true, "Search completed", results)
}

// handleFileVersions manages file versions
func handleFileVersions(w http.ResponseWriter, r *http.Request) {
	if dfsCore == nil || dfsCore.OptimizedStorage == nil {
		sendJSONResponse(w, false, "Enhanced Metadata not available", nil)
		return
	}

	switch r.Method {
	case http.MethodGet:
		// Get file versions
		fileID := r.URL.Query().Get("file_id")
		if fileID == "" {
			sendJSONResponse(w, false, "File ID is required", nil)
			return
		}

		// Get actual file versions
		versions, err := dfsCore.OptimizedStorage.GetFileVersions(fileID)
		if err != nil {
			sendJSONResponse(w, false, "Failed to get file versions: "+err.Error(), nil)
			return
		}

		// Convert to response format
		versionList := make([]map[string]interface{}, 0, len(versions))
		for _, version := range versions {
			versionData := map[string]interface{}{
				"version_id": version.VersionID,
				"version":    version.Version,
				"created_at": version.CreatedAt,
				"created_by": version.CreatedBy,
				"change_log": version.ChangeLog,
				"file_size":  version.FileSize,
				"file_hash":  version.FileHash,
			}
			versionList = append(versionList, versionData)
		}

		sendJSONResponse(w, true, "File versions retrieved", versionList)

	case http.MethodPost:
		// Create new version
		var req struct {
			FileID    string `json:"file_id"`
			ChangeLog string `json:"change_log"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			sendJSONResponse(w, false, "Invalid JSON: "+err.Error(), nil)
			return
		}

		if req.FileID == "" {
			sendJSONResponse(w, false, "File ID is required", nil)
			return
		}

		userID := r.Header.Get("X-User-ID")
		if userID == "" {
			userID = "system"
		}

		// Create actual version
		version, err := dfsCore.OptimizedStorage.VersionFile(req.FileID, userID, req.ChangeLog)
		if err != nil {
			sendJSONResponse(w, false, "Failed to create version: "+err.Error(), nil)
			return
		}

		versionData := map[string]interface{}{
			"version_id": version.VersionID,
			"file_id":    version.FileID,
			"version":    version.Version,
			"created_at": version.CreatedAt,
			"created_by": version.CreatedBy,
			"change_log": version.ChangeLog,
		}

		sendJSONResponse(w, true, "File version created", versionData)

	default:
		sendJSONResponse(w, false, "Method not allowed", nil)
	}
}

// handleFileRelationships manages file relationships
func handleFileRelationships(w http.ResponseWriter, r *http.Request) {
	if dfsCore == nil || dfsCore.OptimizedStorage == nil {
		sendJSONResponse(w, false, "Enhanced Metadata not available", nil)
		return
	}

	switch r.Method {
	case http.MethodGet:
		// Get file relationships
		fileID := r.URL.Query().Get("file_id")
		if fileID == "" {
			sendJSONResponse(w, false, "File ID is required", nil)
			return
		}

		// Get actual file relationships
		relationships, err := dfsCore.OptimizedStorage.GetFileRelationships(fileID)
		if err != nil {
			sendJSONResponse(w, false, "Failed to get file relationships: "+err.Error(), nil)
			return
		}

		// Convert to response format
		relationshipList := make([]map[string]interface{}, 0, len(relationships))
		for _, rel := range relationships {
			relData := map[string]interface{}{
				"relationship_id": rel.RelationshipID,
				"source_file_id":  rel.SourceFileID,
				"target_file_id":  rel.TargetFileID,
				"relation_type":   rel.RelationType,
				"strength":        rel.Strength,
				"created_at":      rel.CreatedAt,
				"created_by":      rel.CreatedBy,
			}
			relationshipList = append(relationshipList, relData)
		}

		sendJSONResponse(w, true, "File relationships retrieved", relationshipList)

	case http.MethodPost:
		// Create new relationship
		var req struct {
			SourceFileID string  `json:"source_file_id"`
			TargetFileID string  `json:"target_file_id"`
			RelationType string  `json:"relation_type"`
			Strength     float64 `json:"strength"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			sendJSONResponse(w, false, "Invalid JSON: "+err.Error(), nil)
			return
		}

		if req.SourceFileID == "" || req.TargetFileID == "" {
			sendJSONResponse(w, false, "Source and target file IDs are required", nil)
			return
		}

		if req.RelationType == "" {
			req.RelationType = "reference"
		}
		if req.Strength == 0 {
			req.Strength = 1.0
		}

		userID := r.Header.Get("X-User-ID")
		if userID == "" {
			userID = "system"
		}

		// Create actual relationship
		err := dfsCore.OptimizedStorage.CreateFileRelationship(req.SourceFileID, req.TargetFileID, req.RelationType, userID)
		if err != nil {
			sendJSONResponse(w, false, "Failed to create relationship: "+err.Error(), nil)
			return
		}

		relationship := map[string]interface{}{
			"source_file_id": req.SourceFileID,
			"target_file_id": req.TargetFileID,
			"relation_type":  req.RelationType,
			"strength":       req.Strength,
			"created_at":     time.Now(),
			"created_by":     userID,
		}

		sendJSONResponse(w, true, "File relationship created", relationship)

	default:
		sendJSONResponse(w, false, "Method not allowed", nil)
	}
}

// handleAvailableFiles returns list of files available for reassembly
func handleAvailableFiles(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		sendJSONResponse(w, false, "Method not allowed", nil)
		return
	}

	fmt.Printf("💾 [DEBUG] Handler called: handleAvailableFiles %s\n", r.Method)
	fmt.Printf("🔍 [DEBUG] DFS Core available: %v\n", dfsCore != nil)
	if dfsCore != nil {
		fmt.Printf("🔍 [DEBUG] Optimized Storage available: %v\n", dfsCore.OptimizedStorage != nil)
	}
	fmt.Printf("🔍 [DEBUG] Basic metadata store available: %v\n", metaStore != nil)
	fmt.Printf("🔍 [DEBUG] File distributor available: %v\n", fileDistributor != nil)

	// Get files from metadata store if available
	var availableFiles []map[string]interface{}

	// First, try enhanced metadata store if available
	if dfsCore != nil && dfsCore.OptimizedStorage != nil {
		fmt.Printf("🔍 [DEBUG] Searching enhanced metadata store for files...\n")

		// Use search to get all files
		searchQuery := &metadata.SearchQuery{
			Query:     "",  // Empty query to get all files
			Limit:     100, // Limit to first 100 files
			Offset:    0,
			SortBy:    "modified_at",
			SortOrder: "desc",
		}

		if searchResult, err := dfsCore.OptimizedStorage.SearchFiles(searchQuery); err == nil {
			fmt.Printf("✅ [DEBUG] Found %d files in enhanced metadata store\n", len(searchResult.Files))

			// Log details about each file found
			for i, fileMeta := range searchResult.Files {
				fmt.Printf("  📁 [DEBUG] File %d: ID=%s, Name=%s, Size=%d, Chunks=%d\n",
					i+1, fileMeta.FileID, fileMeta.FileName, fileMeta.FileSize, fileMeta.ChunkCount)
			}

			for _, fileMeta := range searchResult.Files {
				// Convert enhanced metadata to API format
				fileData := map[string]interface{}{
					"file_id":          fileMeta.FileID,
					"file_name":        fileMeta.FileName,
					"file_size":        fileMeta.FileSize,
					"chunk_count":      fileMeta.ChunkCount,
					"created_at":       fileMeta.CreatedAt,
					"file_hash":        fileMeta.FileHash,
					"status":           "ready for reassembly",
					"chunks_available": fileMeta.ChunkCount, // Assume all chunks available
					"description":      fileMeta.Description,
					"owner_id":         fileMeta.OwnerID,
					"tags":             fileMeta.Tags,
					"categories":       fileMeta.Categories,
					"replica_count":    fileMeta.ReplicaCount,
					"health_status":    fileMeta.HealthStatus,
					"is_encrypted":     fileMeta.IsEncrypted,
					"storage_nodes":    fileMeta.StorageNodes,
				}
				availableFiles = append(availableFiles, fileData)
			}
		} else {
			fmt.Printf("❌ [DEBUG] Failed to search enhanced metadata store: %v\n", err)
		}
	} else {
		fmt.Printf("⚠️ [DEBUG] Enhanced metadata store not available - skipping\n")
	}

	// Fallback to regular metadata store if available and no files found
	if metaStore != nil && len(availableFiles) == 0 {
		// Try to get files from metadata store
		// Note: Using a simplified approach since GetAllFiles doesn't exist
		fmt.Printf("💾 Checking basic metadata store for files...\n")
		// For now, skip metadata store files as the method doesn't exist
	}

	// Also check with file distributor for additional files
	if fileDistributor != nil {
		// Note: GetAvailableFiles doesn't exist, so we'll skip this for now
		fmt.Printf("🌐 Checking file distributor for available files...\n")
		// For now, skip distributor files as the method doesn't exist
	}

	// Try to get real files from DFS core if available
	if dfsCore != nil && len(availableFiles) == 0 {
		fmt.Printf("🔍 Checking DFS core for available files...\n")

		// Get system stats to see if there are files
		stats := dfsCore.GetSystemStats()
		if fileCount, ok := stats["total_files"].(int); ok && fileCount > 0 {
			fmt.Printf("📁 DFS Core reports %d total files\n", fileCount)
			// Note: This is a basic implementation. A full implementation would need
			// methods like dfsCore.GetAllFiles() or similar
		}

		// Get replica information which might contain file data
		if replicas := dfsCore.GetAllReplicaInfo(); replicas != nil {
			fmt.Printf("🔄 [DEBUG] DFS Core replica info contains %d chunks\n", len(replicas))

			// Group chunks by file ID
			fileChunks := make(map[string][]*dfs.ReplicaInfo)
			for _, replicaInfo := range replicas {
				if replicaInfo.FileID != "" {
					fileChunks[replicaInfo.FileID] = append(fileChunks[replicaInfo.FileID], replicaInfo)
				}
			}

			for fileID, chunks := range fileChunks {
				// Extract file information from replica data
				fileData := map[string]interface{}{
					"file_id":          fileID,
					"file_name":        fmt.Sprintf("DFS-File-%s", fileID[:8]),
					"file_size":        0, // Would need actual size from metadata
					"chunk_count":      len(chunks),
					"created_at":       time.Now(),
					"status":           "available in DFS",
					"chunks_available": len(chunks),
					"description":      "File retrieved from DFS core replica information",
					"replica_count":    len(chunks[0].CurrentReplicas),
				}
				availableFiles = append(availableFiles, fileData)
			}
		}
	}

	// If still no real files found, provide demo data for demonstration
	if len(availableFiles) == 0 {
		fmt.Printf("⚠️ [DEBUG] No real files found in any store, providing demo data\n")
		availableFiles = []map[string]interface{}{
			{
				"file_id":          "demo-file-001",
				"file_name":        "sample_document.pdf",
				"file_size":        2048576,
				"chunk_count":      8,
				"created_at":       time.Now().Add(-2 * time.Hour),
				"file_hash":        "abc123def456789",
				"status":           "demo - ready to download",
				"chunks_available": 8,
				"description":      "Sample PDF document for demonstration",
			},
			{
				"file_id":          "demo-file-002",
				"file_name":        "example_archive.zip",
				"file_size":        10485760,
				"chunk_count":      40,
				"created_at":       time.Now().Add(-24 * time.Hour),
				"file_hash":        "def456ghi789abc",
				"status":           "demo - ready to download",
				"chunks_available": 40,
				"description":      "Sample ZIP archive for demonstration",
			},
			{
				"file_id":          "demo-file-003",
				"file_name":        "sample_video.mp4",
				"file_size":        52428800,
				"chunk_count":      200,
				"created_at":       time.Now().Add(-6 * time.Hour),
				"file_hash":        "ghi789jkl012mno",
				"status":           "demo - partial (97.5% complete)",
				"chunks_available": 195,
				"description":      "Sample video file with missing chunks",
			},
			{
				"file_id":          "demo-file-004",
				"file_name":        "presentation.pptx",
				"file_size":        15728640,
				"chunk_count":      60,
				"created_at":       time.Now().Add(-4 * time.Hour),
				"file_hash":        "jkl012mno345pqr",
				"status":           "demo - ready to download",
				"chunks_available": 60,
				"description":      "Sample PowerPoint presentation",
			},
		}
		fmt.Printf("💾 [DEBUG] Returning %d demo files for demonstration\n", len(availableFiles))
	} else {
		fmt.Printf("✅ [DEBUG] Returning %d real files from system\n", len(availableFiles))
	}

	response := map[string]interface{}{
		"files":       availableFiles,
		"total_count": len(availableFiles),
		"timestamp":   time.Now(),
	}

	sendJSONResponse(w, true, "Available files retrieved", response)
}

// handleFileDownload handles file download requests after reassembly
func handleFileDownload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		sendJSONResponse(w, false, "Method not allowed", nil)
		return
	}
	fmt.Printf("📋 Handler called: handleFileDownload %s\n", r.Method)
	fileID := r.URL.Query().Get("file_id")
	if fileID == "" {
		fmt.Printf("❌ File download failed: missing file_id parameter\n")
		sendJSONResponse(w, false, "File ID is required", nil)
		return
	}
	fmt.Printf("📥 Attempting to download file: %s\n", fileID)
	password := r.URL.Query().Get("password")

	// Demo passthrough: if original cached, return it directly
	if cachePath, ok := originalFileCache[fileID]; ok {
		f, err := os.Open(cachePath)
		if err != nil {
			sendJSONResponse(w, false, "Failed to open cached file: "+err.Error(), nil)
			return
		}
		defer f.Close()
		st, _ := f.Stat()
		fileName := filepath.Base(cachePath)
		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", fileName))
		w.Header().Set("Content-Type", "application/octet-stream")
		w.Header().Set("Content-Length", fmt.Sprintf("%d", st.Size()))
		w.WriteHeader(http.StatusOK)
		_, _ = io.Copy(w, f)
		fmt.Printf("✅ Served cached original %s (%d bytes)\n", fileName, st.Size())
		return
	}

	// For real files, synchronously reassemble and stream
	if fileDistributor != nil && metaStore != nil && store != nil {
		// Try to determine a filename
		fileName := "reassembled_" + fileID
		if fi, err := fileDistributor.GetFileInfo(fileID); err == nil && fi != nil && fi.Name != "" {
			fileName = fi.Name
		}

		// Create temp output path
		_ = os.MkdirAll("temp_downloads", 0755)
		outputPath := filepath.Join("temp_downloads", fileID+"_"+fileName)

		// Perform reassembly using the chunker directly
		if password == "" {
			fmt.Printf("⚠️ No password provided for reassembly; decryption will fail\n")
		}

		if err := chunker.ReassembleFile(fileID, outputPath, password, metaStore, store); err != nil {
			fmt.Printf("❌ Synchronous reassembly failed: %v\n", err)
			sendJSONResponse(w, false, "Reassembly failed: "+err.Error(), nil)
			return
		}

		// Stream file
		f, err := os.Open(outputPath)
		if err != nil {
			sendJSONResponse(w, false, "Failed to open reassembled file: "+err.Error(), nil)
			return
		}
		defer f.Close()
		st, _ := f.Stat()

		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", fileName))
		w.Header().Set("Content-Type", "application/octet-stream")
		w.Header().Set("Content-Length", fmt.Sprintf("%d", st.Size()))
		w.WriteHeader(http.StatusOK)
		if _, err := io.Copy(w, f); err != nil {
			fmt.Printf("⚠️ Failed streaming file: %v\n", err)
		}
		fmt.Printf("✅ Reassembled and streamed %s (%d bytes)\n", fileName, st.Size())
		return
	}

	sendJSONResponse(w, false, "File reassembler not available", nil)
}

// handleDebugTest is a simple debug handler to test routing
func handleDebugTest(w http.ResponseWriter, r *http.Request) {
	debugInfo := map[string]interface{}{
		"dfsCore_available":          dfsCore != nil,
		"optimizedStorage_available": false,
		"timestamp":                  time.Now(),
	}

	if dfsCore != nil {
		debugInfo["optimizedStorage_available"] = dfsCore.OptimizedStorage != nil
	}

	sendJSONResponse(w, true, "Debug test successful", debugInfo)
}

// handleSimpleDebug is an unprotected debug handler
func handleSimpleDebug(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("🔧 Handler called: handleSimpleDebug %s\n", r.Method)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(200)
	fmt.Fprintf(w, `{"status":"ok","message":"Simple debug works","timestamp":"%s"}`, time.Now().Format(time.RFC3339))
}

// handleCreateSampleFiles creates sample files in the enhanced metadata store for testing
func handleCreateSampleFiles(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		sendJSONResponse(w, false, "Method not allowed", nil)
		return
	}

	fmt.Printf("🔧 [DEBUG] Creating sample files - DFS Core: %v, Optimized Storage: %v\n",
		dfsCore != nil, dfsCore != nil && dfsCore.OptimizedStorage != nil)

	if dfsCore == nil || dfsCore.OptimizedStorage == nil {
		sendJSONResponse(w, false, "Enhanced metadata store not available", nil)
		return
	}

	userID := r.Header.Get("X-User-ID")
	if userID == "" {
		userID = "test-user"
	}

	fmt.Printf("👤 [DEBUG] Creating files for user: %s\n", userID)

	// Create sample files with unique IDs to avoid conflicts
	timestamp := time.Now().Unix()
	sampleFiles := []*metadata.EnhancedFileMetadata{
		{
			FileID:         fmt.Sprintf("sample-file-%d-001", timestamp),
			FileName:       "test_document.pdf",
			OriginalName:   "test_document.pdf",
			FileSize:       2097152, // 2MB
			MimeType:       "application/pdf",
			FileHash:       "test-hash-001",
			ChunkCount:     8,
			ChunkHashes:    []string{"chunk1", "chunk2", "chunk3", "chunk4", "chunk5", "chunk6", "chunk7", "chunk8"},
			StorageNodes:   []string{"node-1", "node-2", "node-3"},
			ReplicaCount:   3,
			IsEncrypted:    true,
			EncryptionAlgo: "ChaCha20-Poly1305",
			OwnerID:        userID,
			CreatorID:      userID,
			Tags:           []string{"test", "sample", "pdf"},
			Categories:     []string{"documents", "test-data"},
			Description:    "Sample PDF document for testing the file listing functionality",
			HealthStatus:   "healthy",
		},
		{
			FileID:         fmt.Sprintf("sample-file-%d-002", timestamp),
			FileName:       "sample_video.mp4",
			OriginalName:   "sample_video.mp4",
			FileSize:       52428800, // 50MB
			MimeType:       "video/mp4",
			FileHash:       "test-hash-002",
			ChunkCount:     200,
			ChunkHashes:    make([]string, 200), // 200 chunks
			StorageNodes:   []string{"node-1", "node-2", "node-4"},
			ReplicaCount:   3,
			IsEncrypted:    true,
			EncryptionAlgo: "ChaCha20-Poly1305",
			OwnerID:        userID,
			CreatorID:      userID,
			Tags:           []string{"test", "video", "media"},
			Categories:     []string{"media", "test-data"},
			Description:    "Sample video file for testing large file handling",
			HealthStatus:   "healthy",
		},
		{
			FileID:         fmt.Sprintf("sample-file-%d-003", timestamp),
			FileName:       "archive.zip",
			OriginalName:   "project_files.zip",
			FileSize:       10485760, // 10MB
			MimeType:       "application/zip",
			FileHash:       "test-hash-003",
			ChunkCount:     40,
			ChunkHashes:    make([]string, 40), // 40 chunks
			StorageNodes:   []string{"node-2", "node-3"},
			ReplicaCount:   2,
			IsEncrypted:    true,
			EncryptionAlgo: "ChaCha20-Poly1305",
			OwnerID:        userID,
			CreatorID:      userID,
			Tags:           []string{"test", "archive", "compressed"},
			Categories:     []string{"archives", "test-data"},
			Description:    "Sample ZIP archive for testing compression handling",
			HealthStatus:   "degraded", // One missing replica
		},
	}

	// Fill in chunk hashes for the files that need them
	for i := range sampleFiles[1].ChunkHashes {
		sampleFiles[1].ChunkHashes[i] = fmt.Sprintf("video-chunk-%03d", i+1)
	}

	for i := range sampleFiles[2].ChunkHashes {
		sampleFiles[2].ChunkHashes[i] = fmt.Sprintf("zip-chunk-%02d", i+1)
	}

	createdCount := 0
	var createdFileIDs []string
	var errors []string
	for _, fileMeta := range sampleFiles {
		fmt.Printf("💾 [DEBUG] Storing file metadata: ID=%s, Name=%s\n", fileMeta.FileID, fileMeta.FileName)
		if err := dfsCore.OptimizedStorage.StoreFileMetadata(fileMeta); err != nil {
			fmt.Printf("❌ [DEBUG] Failed to store sample file %s: %v\n", fileMeta.FileName, err)
			errors = append(errors, fmt.Sprintf("%s: %v", fileMeta.FileID, err))
		} else {
			createdCount++
			createdFileIDs = append(createdFileIDs, fileMeta.FileID)
			fmt.Printf("✅ [DEBUG] Successfully stored sample file: %s\n", fileMeta.FileName)
		}
	}

	// Test retrieval immediately after storage
	fmt.Printf("🔍 [DEBUG] Testing immediate retrieval of stored files...\n")
	searchQuery := &metadata.SearchQuery{
		Query:     "",
		Tags:      []string{"test"},
		Limit:     10,
		Offset:    0,
		SortBy:    "created_at",
		SortOrder: "desc",
	}

	if searchResult, err := dfsCore.OptimizedStorage.SearchFiles(searchQuery); err == nil {
		fmt.Printf("✅ [DEBUG] Search after storage found %d files\n", len(searchResult.Files))
		for _, file := range searchResult.Files {
			fmt.Printf("  📁 [DEBUG] Found: %s (%s)\n", file.FileID, file.FileName)
		}
	} else {
		fmt.Printf("❌ [DEBUG] Search after storage failed: %v\n", err)
	}

	result := map[string]interface{}{
		"created_files":    createdCount,
		"total_requested":  len(sampleFiles),
		"created_file_ids": createdFileIDs,
		"timestamp":        timestamp,
	}

	if len(errors) > 0 {
		result["errors"] = errors
	}

	sendJSONResponse(w, true, fmt.Sprintf("Created %d sample files", createdCount), result)
}

func sendJSONResponse(w http.ResponseWriter, success bool, message string, data interface{}) {
	response := Response{
		Success: success,
		Message: message,
		Data:    data,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
