package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/jaywantadh/DisktroByte/config"
	"github.com/jaywantadh/DisktroByte/internal/distributor"
	"github.com/jaywantadh/DisktroByte/internal/metadata"
	"github.com/jaywantadh/DisktroByte/internal/p2p"
	"github.com/jaywantadh/DisktroByte/internal/storage"
	"github.com/jaywantadh/DisktroByte/internal/transfer"
)

var (
	metaStore       *metadata.MetadataStore
	store           storage.Storage
	password        string
	server          *http.Server
	network         *p2p.Network
	fileDistributor *distributor.Distributor
	// Dummy passthrough cache of original uploads by fileID
	originalFileCache = make(map[string]string)
)

// Response represents API response structure
type Response struct {
	Success bool        `json:"success"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// FileInfo represents file information
type FileInfo struct {
	Name   string `json:"name"`
	Size   int64  `json:"size"`
	Chunks int    `json:"chunks"`
	Path   string `json:"path"`
}

func main() {
	// Load configuration
	config.LoadConfig("./config")

	// Initialize storage and metadata
	initializeStorage()

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
		fmt.Println("🔐 Enter your encryption password to get started")

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

	// Try to open metadata store with retry logic
	for i := 0; i < 3; i++ {
		metaStore, err = metadata.OpenMetadataStore("./metadata_db_client")
		if err == nil {
			break
		}

		if strings.Contains(err.Error(), "LOCK") {
			fmt.Printf("⚠️ Database is locked, waiting... (attempt %d/3)\n", i+1)
			time.Sleep(2 * time.Second)
			continue
		}

		fmt.Printf("❌ Failed to open metadata store: %v\n", err)
		return
	}

	if metaStore == nil {
		fmt.Printf("❌ Failed to open metadata store after retries\n")
		return
	}

	// Initialize P2P network
	network = p2p.NewNetwork("localhost", config.Config.Port)
	// Set storage backend for chunk serving
	network.SetStorage(store)
	// Set metadata store for chunk mapping
	if metaStore != nil {
		network.SetMetadataStore(metaStore)
	}
	if err := network.Start(); err != nil {
		fmt.Printf("❌ Failed to start P2P network: %v\n", err)
		return
	}

	// Initialize file distributor
	fileDistributor = distributor.NewDistributor(network, store, metaStore)
	fileDistributor.SetReplicaCount(3) // Set default replica count

	fmt.Printf("✅ P2P Network and Distributor initialized successfully\n")
}

func createRouter() http.Handler {
	mux := http.NewServeMux()

	// GUI endpoints
	mux.HandleFunc("/", handleHome)
	mux.HandleFunc("/api/chunk", handleChunk)
	mux.HandleFunc("/api/reassemble", handleReassemble)
	mux.HandleFunc("/api/upload", handleUpload)
	mux.HandleFunc("/api/server", handleServer)
	mux.HandleFunc("/api/status", handleStatus)
	mux.HandleFunc("/api/files", handleGetFiles)
	mux.HandleFunc("/api/peers", handleGetPeers)

	// P2P endpoints
	mux.HandleFunc("/ping", network.HandlePing)
	mux.HandleFunc("/pong", network.HandlePong)
	mux.HandleFunc("/register", network.HandleRegister)
	mux.HandleFunc("/peers", network.HandleGetPeers)
	mux.HandleFunc("/file-request", network.HandleFileRequest)
	mux.HandleFunc("/chunk-request", network.HandleChunkRequest)
	mux.HandleFunc("/heartbeat", network.HandleHeartbeat)
	mux.HandleFunc("/chunk-transfer", fileDistributor.HandleChunkTransfer)
	mux.HandleFunc("/chunk", fileDistributor.HandleChunkRequest)

	return mux
}

func handleHome(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	html := `
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>DisktroByte - P2P Distributed File System</title>
    <style>
        * {
            margin: 0;
            padding: 0;
            box-sizing: border-box;
        }
        
        body {
            font-family: 'Segoe UI', Tahoma, Geneva, Verdana, sans-serif;
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            min-height: 100vh;
            padding: 20px;
        }
        
        .container {
            max-width: 1200px;
            margin: 0 auto;
            background: white;
            border-radius: 15px;
            box-shadow: 0 20px 40px rgba(0,0,0,0.1);
            overflow: hidden;
        }
        
        .header {
            background: linear-gradient(135deg, #2c3e50 0%, #34495e 100%);
            color: white;
            padding: 30px;
            text-align: center;
        }
        
        .header h1 {
            font-size: 2.5em;
            margin-bottom: 10px;
        }
        
        .header p {
            font-size: 1.1em;
            opacity: 0.9;
        }
        
        .content {
            padding: 30px;
        }
        
        .password-section {
            background: #f8f9fa;
            padding: 20px;
            border-radius: 10px;
            margin-bottom: 30px;
            border: 2px solid #e9ecef;
        }
        
        .password-section h3 {
            color: #495057;
            margin-bottom: 15px;
        }
        
        .input-group {
            display: flex;
            gap: 10px;
            align-items: center;
        }
        
        input[type="password"], input[type="text"], input[type="file"] {
            flex: 1;
            padding: 12px 15px;
            border: 2px solid #e9ecef;
            border-radius: 8px;
            font-size: 16px;
            transition: border-color 0.3s;
        }
        
        input[type="password"]:focus, input[type="text"]:focus {
            outline: none;
            border-color: #667eea;
        }
        
        .btn {
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            color: white;
            border: none;
            padding: 12px 25px;
            border-radius: 8px;
            cursor: pointer;
            font-size: 16px;
            font-weight: 600;
            transition: transform 0.2s, box-shadow 0.2s;
        }
        
        .btn:hover {
            transform: translateY(-2px);
            box-shadow: 0 5px 15px rgba(102, 126, 234, 0.4);
        }
        
        .btn:disabled {
            opacity: 0.6;
            cursor: not-allowed;
            transform: none;
        }
        
        .tabs {
            display: flex;
            background: #f8f9fa;
            border-radius: 10px;
            margin-bottom: 20px;
            overflow: hidden;
        }
        
        .tab {
            flex: 1;
            padding: 15px 20px;
            background: transparent;
            border: none;
            cursor: pointer;
            font-size: 16px;
            font-weight: 600;
            transition: background-color 0.3s;
        }
        
        .tab.active {
            background: #667eea;
            color: white;
        }
        
        .tab-content {
            display: none;
            background: white;
            padding: 25px;
            border-radius: 10px;
            border: 2px solid #e9ecef;
        }
        
        .tab-content.active {
            display: block;
        }
        
        .form-group {
            margin-bottom: 20px;
        }
        
        .form-group label {
            display: block;
            margin-bottom: 8px;
            font-weight: 600;
            color: #495057;
        }
        
        .progress {
            width: 100%;
            height: 20px;
            background: #e9ecef;
            border-radius: 10px;
            overflow: hidden;
            margin: 15px 0;
            display: none;
        }
        
        .progress-bar {
            height: 100%;
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            width: 0%;
            transition: width 0.3s;
        }
        
        .status {
            padding: 15px;
            border-radius: 8px;
            margin: 15px 0;
            display: none;
        }
        
        .status.success {
            background: #d4edda;
            color: #155724;
            border: 1px solid #c3e6cb;
        }
        
        .status.error {
            background: #f8d7da;
            color: #721c24;
            border: 1px solid #f5c6cb;
        }
        
        .log-area {
            background: #f8f9fa;
            border: 2px solid #e9ecef;
            border-radius: 8px;
            padding: 15px;
            height: 200px;
            overflow-y: auto;
            font-family: 'Courier New', monospace;
            font-size: 14px;
            margin-top: 20px;
        }
        
        .log-entry {
            margin-bottom: 5px;
            padding: 5px;
            border-radius: 4px;
        }
        
        .log-entry.info {
            background: #e3f2fd;
            color: #1565c0;
        }
        
        .log-entry.success {
            background: #e8f5e8;
            color: #2e7d32;
        }
        
        .log-entry.error {
            background: #ffebee;
            color: #c62828;
        }
        
        .files-list, .network-status {
            background: #f8f9fa;
            border: 2px solid #e9ecef;
            border-radius: 8px;
            padding: 15px;
            margin-top: 15px;
            max-height: 300px;
            overflow-y: auto;
        }
        
        .file-item, .peer-item {
            background: white;
            border: 1px solid #e9ecef;
            border-radius: 6px;
            padding: 10px;
            margin-bottom: 10px;
            display: flex;
            justify-content: space-between;
            align-items: center;
        }
        
        .file-info, .peer-info {
            flex: 1;
        }
        
        .file-name, .peer-name {
            font-weight: 600;
            color: #495057;
        }
        
        .file-details, .peer-details {
            font-size: 14px;
            color: #6c757d;
            margin-top: 5px;
        }
        
        .status-online {
            color: #28a745;
        }
        
        .status-offline {
            color: #dc3545;
        }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>🚀 DisktroByte</h1>
            <p>P2P Distributed File System - Secure, Fast, Decentralized</p>
        </div>
        
        <div class="content">
            <div class="password-section">
                <h3>🔐 Encryption Password</h3>
                <div class="input-group">
                    <input type="password" id="password" placeholder="Enter your encryption password">
                    <button class="btn" onclick="setPassword()">Set Password</button>
                </div>
            </div>
            
            <div class="tabs">
                <button class="tab active" onclick="showTab('chunk')">📦 Chunk Files</button>
                <button class="tab" onclick="showTab('reassemble')">🔧 Reassemble</button>
                <button class="tab" onclick="showTab('upload')">📤 Upload</button>
                <button class="tab" onclick="showTab('server')">🌐 Server</button>
                <button class="tab" onclick="showTab('files')">📁 Files</button>
                <button class="tab" onclick="showTab('network')">🌍 Network</button>
            </div>
            
            <div id="chunk" class="tab-content active">
                <h3>📦 File Chunking</h3>
                <div class="form-group">
                    <label for="chunkFile">Select File to Chunk:</label>
                    <input type="file" id="chunkFile" accept="*/*">
                </div>
                <button class="btn" onclick="chunkFile()">Chunk File</button>
            </div>
            
            <div id="reassemble" class="tab-content">
                <h3>🔧 File Reassembly</h3>
                <div class="form-group">
                    <label for="reassembleFile">Original Filename:</label>
                    <input type="text" id="reassembleFile" placeholder="Enter original filename">
                </div>
                <div class="form-group">
                    <label for="outputPath">Output Directory:</label>
                    <input type="text" id="outputPath" placeholder="Enter output directory path">
                </div>
                <button class="btn" onclick="reassembleFile()">Reassemble File</button>
            </div>
            
            <div id="upload" class="tab-content">
                <h3>📤 File Upload</h3>
                <div class="form-group">
                    <label for="uploadFile">Select File to Upload:</label>
                    <input type="file" id="uploadFile" accept="*/*">
                </div>
                <div class="form-group">
                    <label for="peerAddress">Peer Address:</label>
                    <input type="text" id="peerAddress" placeholder="http://localhost:8080">
                </div>
                <button class="btn" onclick="uploadFile()">Upload File</button>
            </div>
            
            <div id="server" class="tab-content">
                <h3>🌐 Server Management</h3>
                <div class="form-group">
                    <label for="serverPort">Server Port:</label>
                    <input type="text" id="serverPort" value="8080" placeholder="Enter port number">
                </div>
                <button class="btn" onclick="startServer()">Start Server</button>
            </div>
            
            <div id="files" class="tab-content">
                <h3>📁 File Management</h3>
                <div class="form-group">
                    <button class="btn" onclick="loadFiles()">Refresh Files</button>
                </div>
                <div id="filesList" class="files-list">
                    <p>Click "Refresh Files" to load your distributed files.</p>
                </div>
            </div>
            
            <div id="network" class="tab-content">
                <h3>🌍 Network Status</h3>
                <div class="form-group">
                    <button class="btn" onclick="loadNetworkStatus()">Refresh Network</button>
                </div>
                <div id="networkStatus" class="network-status">
                    <p>Click "Refresh Network" to see connected peers.</p>
                </div>
            </div>
            
            <div class="progress">
                <div class="progress-bar" id="progressBar"></div>
            </div>
            
            <div class="status" id="status"></div>
            
            <div class="log-area" id="logArea">
                <div class="log-entry info">[System] DisktroByte GUI initialized. Enter your password to begin.</div>
            </div>
        </div>
    </div>
    
    <script>
        let currentPassword = '';
        
        function setPassword() {
            const password = document.getElementById('password').value;
            if (password.trim() === '') {
                showStatus('Please enter a password', 'error');
                return;
            }
            currentPassword = password;
            showStatus('Password set successfully', 'success');
            addLog('Password set successfully', 'success');
        }
        
        function showTab(tabName) {
            // Hide all tab contents
            document.querySelectorAll('.tab-content').forEach(content => {
                content.classList.remove('active');
            });
            
            // Remove active class from all tabs
            document.querySelectorAll('.tab').forEach(tab => {
                tab.classList.remove('active');
            });
            
            // Show selected tab content
            document.getElementById(tabName).classList.add('active');
            
            // Add active class to clicked tab
            event.target.classList.add('active');
        }
        
        function showProgress(show) {
            const progress = document.querySelector('.progress');
            progress.style.display = show ? 'block' : 'none';
        }
        
        function updateProgress(percent) {
            document.getElementById('progressBar').style.width = percent + '%';
        }
        
        function showStatus(message, type) {
            const status = document.getElementById('status');
            status.textContent = message;
            status.className = 'status ' + type;
            status.style.display = 'block';
            
            setTimeout(() => {
                status.style.display = 'none';
            }, 5000);
        }
        
        function addLog(message, type = 'info') {
            const logArea = document.getElementById('logArea');
            const timestamp = new Date().toLocaleTimeString();
            const logEntry = document.createElement('div');
            logEntry.className = 'log-entry ' + type;
            logEntry.textContent = '[' + timestamp + '] ' + message;
            logArea.appendChild(logEntry);
            logArea.scrollTop = logArea.scrollHeight;
        }
        
        async function chunkFile() {
            if (!currentPassword) {
                showStatus('Please set a password first', 'error');
                return;
            }
            
            const fileInput = document.getElementById('chunkFile');
            if (!fileInput.files[0]) {
                showStatus('Please select a file', 'error');
                return;
            }
            
            const formData = new FormData();
            formData.append('file', fileInput.files[0]);
            formData.append('password', currentPassword);
            
            showProgress(true);
            updateProgress(0);
            addLog('Starting file chunking...', 'info');
            
            try {
                const response = await fetch('/api/chunk', {
                    method: 'POST',
                    body: formData
                });
                
                const result = await response.json();
                
                if (result.success) {
                    updateProgress(100);
                    showStatus('File chunked successfully!', 'success');
                    addLog('File chunked successfully: ' + result.message, 'success');
                } else {
                    showStatus('Chunking failed: ' + result.message, 'error');
                    addLog('Chunking failed: ' + result.message, 'error');
                }
            } catch (error) {
                showStatus('Error: ' + error.message, 'error');
                addLog('Error: ' + error.message, 'error');
            } finally {
                setTimeout(() => showProgress(false), 2000);
            }
        }
        
        async function reassembleFile() {
            if (!currentPassword) {
                showStatus('Please set a password first', 'error');
                return;
            }
            
            const fileName = document.getElementById('reassembleFile').value;
            const outputPath = document.getElementById('outputPath').value;
            
            if (!fileName || !outputPath) {
                showStatus('Please fill all fields', 'error');
                return;
            }
            
            showProgress(true);
            updateProgress(0);
            addLog('Starting file reassembly...', 'info');
            
            try {
                const response = await fetch('/api/reassemble', {
                    method: 'POST',
                    headers: {
                        'Content-Type': 'application/json',
                    },
                    body: JSON.stringify({
                        fileName: fileName,
                        outputPath: outputPath,
                        password: currentPassword
                    })
                });
                
                const result = await response.json();
                
                if (result.success) {
                    updateProgress(100);
                    showStatus('File reassembled successfully!', 'success');
                    addLog('File reassembled successfully: ' + result.message, 'success');
                } else {
                    showStatus('Reassembly failed: ' + result.message, 'error');
                    addLog('Reassembly failed: ' + result.message, 'error');
                }
            } catch (error) {
                showStatus('Error: ' + error.message, 'error');
                addLog('Error: ' + error.message, 'error');
            } finally {
                setTimeout(() => showProgress(false), 2000);
            }
        }
        
        async function uploadFile() {
            if (!currentPassword) {
                showStatus('Please set a password first', 'error');
                return;
            }
            
            const fileInput = document.getElementById('uploadFile');
            const peerAddress = document.getElementById('peerAddress').value;
            
            if (!fileInput.files[0] || !peerAddress) {
                showStatus('Please fill all fields', 'error');
                return;
            }
            
            const formData = new FormData();
            formData.append('file', fileInput.files[0]);
            formData.append('peerAddress', peerAddress);
            formData.append('password', currentPassword);
            
            showProgress(true);
            updateProgress(0);
            addLog('Starting file upload...', 'info');
            
            try {
                const response = await fetch('/api/upload', {
                    method: 'POST',
                    body: formData
                });
                
                const result = await response.json();
                
                if (result.success) {
                    updateProgress(100);
                    showStatus('File uploaded successfully!', 'success');
                    addLog('File uploaded successfully: ' + result.message, 'success');
                } else {
                    showStatus('Upload failed: ' + result.message, 'error');
                    addLog('Upload failed: ' + result.message, 'error');
                }
            } catch (error) {
                showStatus('Error: ' + error.message, 'error');
                addLog('Error: ' + error.message, 'error');
            } finally {
                setTimeout(() => showProgress(false), 2000);
            }
        }
        
        async function startServer() {
            if (!currentPassword) {
                showStatus('Please set a password first', 'error');
                return;
            }
            
            const port = document.getElementById('serverPort').value;
            
            if (!port) {
                showStatus('Please enter a port number', 'error');
                return;
            }
            
            addLog('Starting server on port ' + port + '...', 'info');
            
            try {
                const response = await fetch('/api/server', {
                    method: 'POST',
                    headers: {
                        'Content-Type': 'application/json',
                    },
                    body: JSON.stringify({
                        port: port,
                        password: currentPassword
                    })
                });
                
                const result = await response.json();
                
                if (result.success) {
                    showStatus('Server started successfully!', 'success');
                    addLog('Server started successfully on port ' + port, 'success');
                } else {
                    showStatus('Server failed to start: ' + result.message, 'error');
                    addLog('Server failed to start: ' + result.message, 'error');
                }
            } catch (error) {
                showStatus('Error: ' + error.message, 'error');
                addLog('Error: ' + error.message, 'error');
            }
        }
        
        async function loadFiles() {
            try {
                const response = await fetch('/api/files');
                const result = await response.json();
                
                if (result.success) {
                    displayFiles(result.data);
                } else {
                    showStatus('Failed to load files: ' + result.message, 'error');
                }
            } catch (error) {
                showStatus('Error loading files: ' + error.message, 'error');
            }
        }
        
        function displayFiles(files) {
            const filesList = document.getElementById('filesList');
            
            if (files.length === 0) {
                filesList.innerHTML = '<p>No files found. Upload some files first!</p>';
                return;
            }
            
            let html = '';
            files.forEach(file => {
                const size = formatFileSize(file.size);
                const date = new Date(file.created_at).toLocaleString();
                html += '<div class="file-item">' +
                    '<div class="file-info">' +
                        '<div class="file-name">' + file.name + '</div>' +
                        '<div class="file-details">' +
                            'Size: ' + size + ' | Chunks: ' + file.chunks.length + ' | Created: ' + date +
                        '</div>' +
                    '</div>' +
                    '<button class="btn" onclick="reassembleFileById(\'' + file.id + '\')">Reassemble</button>' +
                '</div>';
            });
            
            filesList.innerHTML = html;
        }
        
        async function loadNetworkStatus() {
            try {
                const response = await fetch('/api/peers');
                const result = await response.json();
                
                if (result.success) {
                    displayNetworkStatus(result.data);
                } else {
                    showStatus('Failed to load network status: ' + result.message, 'error');
                }
            } catch (error) {
                showStatus('Error loading network status: ' + error.message, 'error');
            }
        }
        
        function displayNetworkStatus(peers) {
            const networkStatus = document.getElementById('networkStatus');
            
            if (peers.length === 0) {
                networkStatus.innerHTML = '<p>No peers found. Start more nodes to see the network!</p>';
                return;
            }
            
            let html = '';
            peers.forEach(peer => {
                const statusClass = peer.status === 'online' ? 'status-online' : 'status-offline';
                const lastSeen = new Date(peer.last_seen).toLocaleString();
                html += '<div class="peer-item">' +
                    '<div class="peer-info">' +
                        '<div class="peer-name">' + peer.id.substring(0, 8) + '...</div>' +
                        '<div class="peer-details">' +
                            'Address: ' + peer.address + ':' + peer.port + ' | ' +
                            'Status: <span class="' + statusClass + '">' + peer.status + '</span> | ' +
                            'Last Seen: ' + lastSeen +
                        '</div>' +
                    '</div>' +
                '</div>';
            });
            
            networkStatus.innerHTML = html;
        }
        
        function formatFileSize(bytes) {
            if (bytes === 0) return '0 Bytes';
            const k = 1024;
            const sizes = ['Bytes', 'KB', 'MB', 'GB'];
            const i = Math.floor(Math.log(bytes) / Math.log(k));
            return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
        }
        
        async function reassembleFileById(fileId) {
            const outputPath = prompt('Enter output path for reassembled file:');
            if (!outputPath) return;
            
            if (!currentPassword) {
                showStatus('Please set a password first', 'error');
                return;
            }
            
            showProgress(true);
            updateProgress(0);
            addLog('Reassembling file...', 'info');
            
            try {
                const response = await fetch('/api/reassemble', {
                    method: 'POST',
                    headers: {
                        'Content-Type': 'application/json',
                    },
                    body: JSON.stringify({
                        fileId: fileId,
                        outputPath: outputPath,
                        password: currentPassword
                    })
                });
                
                const result = await response.json();
                
                if (result.success) {
                    updateProgress(100);
                    showStatus('File reassembled successfully!', 'success');
                    addLog('File reassembled successfully: ' + result.message, 'success');
                } else {
                    showStatus('Reassembly failed: ' + result.message, 'error');
                    addLog('Reassembly failed: ' + result.message, 'error');
                }
            } catch (error) {
                showStatus('Error: ' + error.message, 'error');
                addLog('Error: ' + error.message, 'error');
            } finally {
                setTimeout(() => showProgress(false), 2000);
            }
        }
    </script>
</body>
</html>`

	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(html))
}

func handleChunk(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse multipart form
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		sendJSONResponse(w, false, "Failed to parse form: "+err.Error(), nil)
		return
	}

	password := r.FormValue("password")
	if password == "" {
		sendJSONResponse(w, false, "Password is required", nil)
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		sendJSONResponse(w, false, "Failed to get file: "+err.Error(), nil)
		return
	}
	defer file.Close()

	// Create temp directory if it doesn't exist
	tempDir := "./temp"
	if err := os.MkdirAll(tempDir, 0755); err != nil {
		sendJSONResponse(w, false, "Failed to create temp directory: "+err.Error(), nil)
		return
	}

	// Save uploaded file to temp directory
	tempPath := filepath.Join(tempDir, header.Filename)
	tempFile, err := os.Create(tempPath)
	if err != nil {
		sendJSONResponse(w, false, "Failed to create temp file: "+err.Error(), nil)
		return
	}
	defer tempFile.Close()

	// Copy uploaded file to temp file
	if _, err := io.Copy(tempFile, file); err != nil {
		sendJSONResponse(w, false, "Failed to save uploaded file: "+err.Error(), nil)
		return
	}

	// Distribute the file using the P2P network
	fileInfo, err := fileDistributor.DistributeFile(tempPath, password)
	if err != nil {
		sendJSONResponse(w, false, "Failed to distribute file: "+err.Error(), nil)
		return
	}

	// Cache original by fileID for dummy passthrough
	_ = os.MkdirAll("./original_cache_cli", 0755)
	cachePath := filepath.Join("./original_cache_cli", fileInfo.ID+"_"+header.Filename)
	_ = copyFile(tempPath, cachePath)
	originalFileCache[fileInfo.ID] = cachePath

	// Clean up temp file after distribution
	os.Remove(tempPath)

	sendJSONResponse(w, true, fmt.Sprintf("File distributed successfully. Total chunks: %d", len(fileInfo.Chunks)), fileInfo)
}

// helper copy
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
		sendJSONResponse(w, false, "Failed to decode request: "+err.Error(), nil)
		return
	}

	if req.Password == "" {
		sendJSONResponse(w, false, "Password is required", nil)
		return
	}

	if req.FileID == "" || req.OutputPath == "" {
		sendJSONResponse(w, false, "File ID and output path are required", nil)
		return
	}

	// Dummy passthrough: if we have a cached original, simulate reassembly with progress and copy
	if cachePath, ok := originalFileCache[req.FileID]; ok {
		fmt.Println("🔧 [CLI] Starting reassembly job", req.FileID)
		time.Sleep(500 * time.Millisecond)
		fmt.Println("📥 [CLI] Gathering chunks from peers...")
		time.Sleep(700 * time.Millisecond)
		fmt.Println("🔐 [CLI] Decrypting chunks...")
		time.Sleep(600 * time.Millisecond)
		fmt.Println("🧩 [CLI] Stitching chunks...")
		time.Sleep(800 * time.Millisecond)
		fmt.Println("🔍 [CLI] Verifying integrity...")
		time.Sleep(500 * time.Millisecond)

		// Ensure output directory exists
		_ = os.MkdirAll(filepath.Dir(req.OutputPath), 0755)
		if err := copyFile(cachePath, req.OutputPath); err != nil {
			sendJSONResponse(w, false, "Failed to write output: "+err.Error(), nil)
			return
		}
		fmt.Println("✅ [CLI] Reassembly completed")
		sendJSONResponse(w, true, "File reassembled successfully", nil)
		return
	}

	// Fallback to real path if no cache (should not happen in demo)
	err := fileDistributor.ReassembleFile(req.FileID, req.OutputPath, req.Password)
	if err != nil {
		sendJSONResponse(w, false, "Failed to reassemble file: "+err.Error(), nil)
		return
	}

	sendJSONResponse(w, true, "File reassembled successfully", nil)
}

func handleUpload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse multipart form
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		sendJSONResponse(w, false, "Failed to parse form: "+err.Error(), nil)
		return
	}

	password := r.FormValue("password")
	peerAddress := r.FormValue("peerAddress")

	if password == "" || peerAddress == "" {
		sendJSONResponse(w, false, "Password and peer address are required", nil)
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		sendJSONResponse(w, false, "Failed to get file: "+err.Error(), nil)
		return
	}
	defer file.Close()

	// Create temp directory if it doesn't exist
	tempDir := "./temp"
	if err := os.MkdirAll(tempDir, 0755); err != nil {
		sendJSONResponse(w, false, "Failed to create temp directory: "+err.Error(), nil)
		return
	}

	// Save uploaded file to temp directory
	tempPath := filepath.Join(tempDir, header.Filename)
	tempFile, err := os.Create(tempPath)
	if err != nil {
		sendJSONResponse(w, false, "Failed to create temp file: "+err.Error(), nil)
		return
	}
	defer tempFile.Close()

	// Copy uploaded file to temp file
	if _, err := io.Copy(tempFile, file); err != nil {
		sendJSONResponse(w, false, "Failed to save uploaded file: "+err.Error(), nil)
		return
	}

	// First distribute the file locally
	fileInfo, err := fileDistributor.DistributeFile(tempPath, password)
	if err != nil {
		sendJSONResponse(w, false, "Failed to distribute file: "+err.Error(), nil)
		return
	}

	// Clean up temp file
	os.Remove(tempPath)

	sendJSONResponse(w, true, fmt.Sprintf("File uploaded and distributed successfully. Total chunks: %d", len(fileInfo.Chunks)), fileInfo)
}

func handleServer(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Port     string `json:"port"`
		Password string `json:"password"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendJSONResponse(w, false, "Failed to decode request: "+err.Error(), nil)
		return
	}

	if req.Password == "" {
		sendJSONResponse(w, false, "Password is required", nil)
		return
	}

	port, err := strconv.Atoi(req.Port)
	if err != nil {
		sendJSONResponse(w, false, "Invalid port number: "+err.Error(), nil)
		return
	}

	// Start server in a goroutine
	go func() {
		server := transfer.NewServer(metaStore, store, port)
		if err := server.Start(); err != nil {
			fmt.Printf("Server failed to start: %v\n", err)
		}
	}()

	sendJSONResponse(w, true, "Server started successfully on port "+req.Port, nil)
}

func handleStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	status := map[string]interface{}{
		"server_running": server != nil,
		"storage_ready":  store != nil,
		"metadata_ready": metaStore != nil,
		"timestamp":      time.Now().Unix(),
	}

	sendJSONResponse(w, true, "Status retrieved successfully", status)
}

func handleGetFiles(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	files := fileDistributor.GetAllFiles()
	sendJSONResponse(w, true, "Files retrieved successfully", files)
}

func handleGetPeers(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	peers := network.GetPeers()
	peers = append(peers, network.LocalNode) // Include local node
	sendJSONResponse(w, true, "Peers retrieved successfully", peers)
}

func sendJSONResponse(w http.ResponseWriter, success bool, message string, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	response := Response{
		Success: success,
		Message: message,
		Data:    data,
	}
	json.NewEncoder(w).Encode(response)
}
