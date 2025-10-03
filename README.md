# 🚀 DisktroByte - P2P Distributed File System

A revolutionary peer-to-peer distributed file system built in Go, featuring automatic file chunking, compression, encryption, distributed storage, and intelligent file reassembly across a network of nodes.

## Documentation
### Tap here ⬇️

[![Documentation](https://github.com/Jaywantadh/Images/blob/main/DisktroByteLogo.png)](https://long-phase-95351982.figma.site/#overview)

## 📋 Table of Contents

- [Overview](#Overview)
- [System Architecture](#System-Architecture)
- [Key Features](#key-features)
- [Installation](#installation)
- [Quick Start](#quick-start)
- [Usage Guide](#usage-guide)
- [System Design](#system-design)
- [API Reference](#api-reference)
- [Developer Guide](#developer-guide)
- [Troubleshooting](#troubleshooting)
- [Contributing](#contributing)
- [License](#license)

## 🌟 Overview

![Overveiw](https://github.com/Jaywantadh/Images/blob/main/Overveiw.png)

DisktroByte is a cutting-edge distributed file system that transforms traditional file storage into a decentralized, fault-tolerant network. Built with modern Go technologies, it provides:


- **🔐 End-to-end encryption** using ChaCha20-Poly1305
- **📦 Intelligent compression** with LZ4 algorithm
- **🌐 P2P networking** for decentralized storage
- **🔄 Automatic replication** for fault tolerance
- **🔧 File reassembly** with intelligent chunk reconstruction
- **📱 Modern Web GUI** with real-time updates
- **⚡ High performance** with parallel processing
- **🔍 Smart file management** with metadata tracking

### Use Cases

- **Distributed Backup Systems**: Redundant file storage across multiple locations
- **Content Distribution Networks**: Efficient file sharing in P2P networks
- **Edge Computing**: Distributed storage for edge devices
- **Blockchain Storage**: Decentralized file storage for blockchain applications
- **Research Data Sharing**: Secure, distributed research data management

## 📸 Screenshots & Visual Overview

### Main Dashboard
![Main Dashboard](assets/images/dashboard.png)
*Modern web interface with role-based navigation and real-time status updates*

### File Reassembly Interface
![File Reassembly](assets/images/file-reassembly.png)
*Intelligent file reconstruction with progress tracking and direct downloads*

### Network Status Monitor
![Network Status](assets/images/network-status.png)
*Real-time peer monitoring with health indicators and connectivity status*

### Authentication System
![Login Interface](assets/images/login-interface.png)
*Secure user authentication with registration and session management*

### File Upload & Processing
![File Upload](assets/images/file-upload.png)
*Drag-and-drop file upload with chunking progress and distribution status*

---

## 🏗️ System Architecture

![System-Architecture](https://github.com/Jaywantadh/Images/blob/main/System-Architecture.png)

### Distributed Network Topology
```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   Node A        │    │   Node B        │    │   Node C        │
│                 │    │                 │    │                 │
│  ┌───────────┐  │    │  ┌───────────┐  │    │  ┌───────────┐  │
│  │   GUI     │  │    │  │   GUI     │  │    │  │   GUI     │  │
│  │  (Web)    │  │    │  │  (Web)    │  │    │  │  (Web)    │  │
│  └───────────┘  │    │  └───────────┘  │    │  └───────────┘  │
│        │        │    │        │        │    │        │        │
│  ┌───────────┐  │    │  ┌───────────┐  │    │  ┌───────────┐  │
│  │ P2P       │◄─┴────┴─►│ P2P       │◄─┴────┴─►│ P2P       │  │
│  │ Network   │  │    │  │ Network   │  │    │  │ Network   │  │
│  └───────────┘  │    │  └───────────┘  │    │  └───────────┘  │
│        │        │    │        │        │    │        │        │
│  ┌───────────┐  │    │  ┌───────────┐  │    │  ┌───────────┐  │
│  │ File      │  │    │  │ File      │  │    │  │ File      │  │
│  │ Distributor│  │    │  │ Distributor│  │    │  │ Distributor│  │
│  └───────────┘  │    │  └───────────┘  │    │  └───────────┘  │
│        │        │    │        │        │    │        │        │
│  ┌───────────┐  │    │  ┌───────────┐  │    │  ┌───────────┐  │
│  │ Storage   │  │    │  │ Storage   │  │    │  │ Storage   │  │
│  │ Layer     │  │    │  │ Layer     │  │    │  │ Layer     │  │
│  └───────────┘  │    │  └───────────┘  │    │  └───────────┘  │
└─────────────────┘    └─────────────────┘    └─────────────────┘
```

### DisktroByte vs Traditional Storage
![Comparison Chart](assets/images/comparison-chart.png)

| Feature | Traditional Storage | DisktroByte P2P |
|---------|--------------------|-----------------|
| **Single Point of Failure** | ❌ High Risk | ✅ Eliminated |
| **Scalability** | ⚠️ Limited | ✅ Infinite |
| **Data Security** | ⚠️ Server Dependent | ✅ End-to-End Encrypted |
| **File Accessibility** | 🖥️ Server Required | 🌐 Distributed Access |
| **Storage Costs** | 💰 High & Recurring | 🔄 Shared & Reduced |
| **Network Resilience** | ❌ Central Dependency | 🔍 Self-Healing Network |

### Core Components

1. **Web GUI**: Modern, responsive interface for file management
2. **P2P Network**: Decentralized communication between nodes
3. **File Distributor**: Manages file chunking, distribution, and replication
4. **Storage Layer**: Local file system and metadata management
5. **Encryption Engine**: ChaCha20-Poly1305 encryption/decryption
6. **Compression Engine**: LZ4 compression for optimal storage

## ✨ Key Features

![Key-Features](https://github.com/Jaywantadh/Images/blob/main/Key-features.png)

### 🔐 Security Features
- **ChaCha20-Poly1305 Encryption**: Military-grade encryption for all files
- **Password-based Key Derivation**: Secure key generation from user passwords
- **End-to-end Encryption**: Files encrypted before leaving the source node
- **Secure P2P Communication**: Encrypted communication between nodes

### 📦 Storage Features
- **Intelligent Chunking**: Automatic file splitting into optimal chunk sizes
- **LZ4 Compression**: High-speed compression for compressible files
- **Smart Compression Detection**: Skips compression for already compressed files
- **Metadata Management**: BadgerDB for efficient metadata storage

### 🌐 Network Features
- **P2P Architecture**: Decentralized network without central servers
- **Automatic Node Discovery**: Dynamic peer discovery and registration
- **Heartbeat Monitoring**: Real-time health monitoring of network nodes
- **Fault Tolerance**: Automatic failover and recovery mechanisms

### 🎨 User Interface Features
- **Modern Web GUI**: Beautiful, responsive interface with role-based access
- **File Reassembly Dashboard**: Intuitive file management and reassembly interface
- **Real-time Updates**: Live status updates and progress tracking
- **Cross-platform**: Works on Windows, macOS, and Linux
- **Authentication System**: Secure user management with session handling
- **Drag & Drop Support**: Easy file upload with visual feedback
- **Progress Visualization**: Real-time progress bars and status indicators
- **Responsive Design**: Optimized for desktop, tablet, and mobile devices
- **Dark/Light Theme**: Automatic theme detection and manual toggle
- **Intuitive Navigation**: Clean, organized interface for all skill levels

### 🆕 New Features (Latest Updates)

#### 🔧 Advanced File Reassembly System
![File Reassembly Demo](assets/images/reassembly-demo.gif)
- **Smart File Detection**: Automatic file type recognition with appropriate icons
- **Visual Progress Tracking**: Real-time progress bars during file reconstruction
- **Chunk Status Monitoring**: Visual indicators for available vs missing chunks
- **Direct Browser Downloads**: One-click download of reassembled files
- **Demo File Testing**: Interactive demo files for testing without real data
- **History Tracking**: Complete log of reassembly operations and downloads

#### 🔐 Enhanced Authentication & Security
![Authentication Flow](assets/images/auth-flow.png)
- **User Registration System**: Complete user onboarding with profile management
- **Session Management**: Secure session handling with automatic timeout
- **Role-based Access Control**: Different permission levels (admin, user, viewer)
- **Password Security**: Secure password hashing and validation
- **API Authentication**: Protected endpoints with middleware validation

#### 📊 Real-time Dashboard & Monitoring
![Dashboard Overview](assets/images/dashboard-overview.png)
- **Live Network Status**: Real-time peer connectivity and health monitoring
- **File Statistics**: Visual charts showing storage usage and distribution
- **Performance Metrics**: System performance indicators and resource usage
- **Activity Logs**: Comprehensive logging of all system operations
- **Alert System**: Real-time notifications for important events

#### 🌐 Enhanced P2P Networking
![Network Topology](assets/images/network-topology.png)
- **Automatic Peer Discovery**: Dynamic discovery and connection to network peers
- **Load Balancing**: Intelligent distribution of chunks across available nodes
- **Fault Tolerance**: Automatic failover and recovery mechanisms
- **Bandwidth Optimization**: Smart bandwidth usage and throttling
- **Network Diagnostics**: Built-in tools for network troubleshooting

## 🛠️ Installation

### Prerequisites

- **Go 1.19+**: [Download Go](https://golang.org/dl/)
- **Git**: [Download Git](https://git-scm.com/downloads)
- **Web Browser**: Chrome, Firefox, Safari, or Edge

### Installation Steps

1. **Clone the Repository**
   ```bash
   git clone https://github.com/jaywantadh/DisktroByte.git
   cd DisktroByte
   ```

2. **Install Dependencies**
   ```bash
   go mod tidy
   ```

3. **Build the Application**
   ```bash
   # Build the GUI version (recommended)
   go build -o disktroByte.exe ./cmd/gui
   
   # Or build the CLI version
   go build -o disktroByte-cli.exe ./cmd/cli
   ```

4. **Run the Application**
   ```bash
   # Option 1: Run the GUI version
   ./disktroByte.exe
   
   # Option 2: Direct execution
   go run ./cmd/gui/main.go
   
   # Option 3: CLI version
   go run ./cmd/cli/main.go
   ```

## Data Flow

![Data-Flow](https://github.com/Jaywantadh/Images/blob/main/DataFlow.png)

## 🚀 Quick Start

### 1. Start the Application

```bash
# Run the GUI version
go run ./cmd/gui/main.go

# Or run the built executable
./disktroByte.exe
```

### 2. Access the Web Interface

Open your browser and navigate to:
```
http://localhost:8080
```
*Note: If port 8080 is busy, the application will automatically use the next available port*

### 3. Create Account & Login

1. Click "Register" to create a new account
2. Fill in username, password, and optional details
3. Login with your credentials
4. The system will create a secure session for your operations

### 4. Upload and Distribute Files

1. Go to the "Send Files" tab
2. Select a file to upload
3. Click "Send File"
4. Watch as your file is automatically:
   - Chunked into smaller pieces
   - Compressed (if beneficial)
   - Encrypted securely
   - Distributed across the network

### 5. File Reassembly

1. Go to the "File Reassembly" (🔧) tab
2. Click "Refresh Available Files" to load files
3. Browse available files with completion status
4. Click "Reassemble" or "Prepare" for demo files
5. Download completed files directly from the interface

### 6. View Network Status

1. Go to the "Network" tab
2. Click "Refresh Network"
3. Monitor connected peers and their health status

## 🎦 Interactive Demo & Screenshots

### Try the File Reassembly Feature
![File Reassembly Interface](assets/images/file-reassembly-interface.png)
*Experience our new file reassembly system with these demo files:*

| Demo File | Type | Size | Status | Preview |
|-----------|------|------|--------|---------|
| 📑 sample_document.pdf | PDF | 2MB | ✅ Ready | ![PDF Preview](assets/images/pdf-preview.png) |
| 🗜️ example_archive.zip | ZIP | 10MB | ✅ Ready | ![ZIP Preview](assets/images/zip-preview.png) |
| 🎥 sample_video.mp4 | MP4 | 50MB | ⚠️ 97.5% | ![MP4 Preview](assets/images/mp4-preview.png) |
| 📊 presentation.pptx | PPTX | 15MB | ✅ Ready | ![PPTX Preview](assets/images/pptx-preview.png) |

### Real-time Progress Tracking
![Progress Tracking](assets/images/progress-tracking.gif)
*Watch files being reassembled with visual progress indicators and status updates*

### Network Visualization
![Network Graph](assets/images/network-graph.png)
*Monitor your P2P network with our interactive network graph showing peer connections and health status*

---

## 📜 Usage Guide

### File Operations

#### Chunking Files
**Purpose**: Split large files into manageable chunks for distributed storage

**Steps**:
1. Navigate to "Chunk Files" tab
2. Click "Choose File" and select your file
3. Ensure your password is set
4. Click "Chunk File"
5. Monitor progress in the log area

**What Happens**:
- File is read and analyzed
- Optimal chunk size is determined
- File is split into chunks
- Each chunk is compressed (if beneficial)
- Chunks are encrypted with your password
- Chunks are stored locally and distributed to peers

#### File Reassembly
**Purpose**: Reconstruct original files from distributed chunks with intelligent management

**Steps**:
1. Go to "File Reassembly" (🔧) tab
2. Click "Refresh Available Files" to load your file list
3. View file completion status and chunk availability
4. Click "Reassemble" (for real files) or "Prepare" (for demo files)
5. Monitor real-time progress and status updates
6. Download completed files directly from the interface

**Features**:
- **Visual Progress Tracking**: Real-time progress bars and status updates
- **File Type Detection**: Smart icons and handling for different file types (PDF, ZIP, MP4, PPTX)
- **Demo File Support**: Test the interface with realistic demo files
- **Chunk Status Monitoring**: See exactly which chunks are available vs missing
- **Direct Download**: Download reassembled files directly from the web interface
- **History Tracking**: Keep track of reassembly operations and downloads

**What Happens**:
- System locates all chunks for the file across the network
- Downloads missing chunks from peers (if needed)
- Decrypts all chunks using secure encryption
- Decompresses chunks (if they were compressed)
- Reassembles chunks in correct order with integrity verification
- Makes the complete file available for download

#### Uploading to Specific Peers
**Purpose**: Upload files to specific network nodes

**Steps**:
1. Go to "Upload" tab
2. Select a file to upload
3. Enter the peer address (e.g., `http://localhost:8081`)
4. Click "Upload File"

### Network Management

#### Starting a Server
**Purpose**: Make your node available to other peers

**Steps**:
1. Go to "Server" tab
2. Enter the port number (default: 8080)
3. Click "Start Server"

#### Monitoring Network Health
**Purpose**: Monitor the health and status of connected peers

**Steps**:
1. Go to "Network" tab
2. Click "Refresh Network"
3. View peer status, last seen times, and health indicators

### Advanced Features

#### File Management
- **View Distributed Files**: See all files you've distributed
- **File Information**: View file size, chunk count, creation date
- **Replication Status**: Monitor chunk replication across nodes

#### Network Diagnostics
- **Peer Discovery**: Automatic discovery of new nodes
- **Health Monitoring**: Real-time health checks every 30 seconds
- **Status Tracking**: Online/offline status for all peers

## 🏗️ System Design

### Architecture Overview

DisktroByte follows a modular, layered architecture:

```
┌─────────────────────────────────────────────────────────────┐
│                    Web GUI Layer                            │
├─────────────────────────────────────────────────────────────┤
│                    HTTP API Layer                           │
├─────────────────────────────────────────────────────────────┤
│                    P2P Network Layer                        │
├─────────────────────────────────────────────────────────────┤
│                    File Distribution Layer                  │
├─────────────────────────────────────────────────────────────┤
│                    Storage Layer                            │
├─────────────────────────────────────────────────────────────┤
│                    Encryption/Compression Layer             │
└─────────────────────────────────────────────────────────────┘
```

### Component Details

#### 1. Web GUI Layer (`cmd/cli/main.go`)
- **Purpose**: User interface and interaction
- **Technology**: HTML5, CSS3, JavaScript
- **Features**: Real-time updates, progress tracking, responsive design

#### 2. HTTP API Layer
- **Endpoints**: RESTful API for all operations
- **Authentication**: Password-based encryption
- **Response Format**: JSON with success/error indicators

#### 3. P2P Network Layer (`internal/p2p/network.go`)
- **Protocol**: HTTP-based P2P communication
- **Discovery**: Automatic peer discovery and registration
- **Health Monitoring**: Heartbeat system for peer health

#### 4. File Distribution Layer (`internal/distributor/distributor.go`)
- **Chunking**: Intelligent file splitting
- **Replication**: Automatic chunk replication across nodes
- **Recovery**: Fault-tolerant chunk retrieval

#### 5. Storage Layer (`internal/storage/`)
- **Local Storage**: File system-based chunk storage
- **Metadata**: BadgerDB for efficient metadata management
- **Indexing**: Fast chunk lookup and retrieval

#### 6. Encryption/Compression Layer
- **Encryption**: ChaCha20-Poly1305 for file security
- **Compression**: LZ4 for storage optimization
- **Detection**: Smart compression detection

### Data Flow

#### File Upload Process
```
1. User selects file → GUI
2. File uploaded → HTTP API
3. File saved → Temp storage
4. File chunked → Chunker
5. Chunks compressed → Compressor
6. Chunks encrypted → Encryptor
7. Chunks stored → Local storage
8. Chunks distributed → P2P network
9. Metadata updated → BadgerDB
10. Success response → GUI
```

#### File Download Process
```
1. User requests file → GUI
2. File metadata retrieved → BadgerDB
3. Chunk locations determined → Distributor
4. Missing chunks downloaded → P2P network
5. Chunks decrypted → Encryptor
6. Chunks decompressed → Compressor
7. File reassembled → Chunker
8. File saved → User location
9. Success response → GUI
```

### Security Model

#### Encryption Flow
```
1. User Password → Key Derivation Function
2. Derived Key → ChaCha20-Poly1305
3. File Data → Encrypted Chunks
4. Encrypted Chunks → Distributed Storage
```

#### Authentication Model
- **Password-based**: Single password for all operations
- **Session-less**: No persistent sessions
- **Stateless**: Each request authenticated independently

## 📚 API Reference

### HTTP Endpoints

#### GUI Endpoints

##### `GET /`
- **Purpose**: Serve the main web interface
- **Response**: HTML page with GUI

##### `POST /api/chunk`
- **Purpose**: Chunk and distribute a file
- **Content-Type**: `multipart/form-data`
- **Parameters**:
  - `file`: File to chunk
  - `password`: Encryption password
- **Response**: JSON with success status and file info

##### `POST /api/reassemble`
- **Purpose**: Reassemble a file from chunks
- **Content-Type**: `application/json`
- **Body**:
  ```json
  {
    "fileId": "string",
    "outputPath": "string",
    "password": "string"
  }
  ```
- **Response**: JSON with success status

##### `POST /api/upload`
- **Purpose**: Upload file to specific peer
- **Content-Type**: `multipart/form-data`
- **Parameters**:
  - `file`: File to upload
  - `peerAddress`: Target peer address
  - `password`: Encryption password
- **Response**: JSON with success status

##### `GET /api/files`
- **Purpose**: Get list of distributed files
- **Response**: JSON array of file information

##### `GET /api/peers`
- **Purpose**: Get list of connected peers
- **Response**: JSON array of peer information

##### `GET /api/files/available`
- **Purpose**: Get list of files available for reassembly
- **Authentication**: Required
- **Response**: JSON with files array and metadata
- **Example Response**:
  ```json
  {
    "success": true,
    "message": "Available files retrieved",
    "data": {
      "files": [
        {
          "file_id": "demo-file-001",
          "file_name": "sample_document.pdf",
          "file_size": 2048576,
          "chunk_count": 8,
          "chunks_available": 8,
          "status": "demo - ready to download",
          "description": "Sample PDF document for demonstration"
        }
      ],
      "total_count": 4,
      "timestamp": "2025-09-08T18:00:00Z"
    }
  }
  ```

##### `GET /api/files/download?file_id=<file_id>`
- **Purpose**: Download a reassembled file
- **Authentication**: Required
- **Parameters**: `file_id`: File identifier
- **Response**: Binary file data with appropriate headers
- **Headers**: 
  - `Content-Disposition: attachment; filename="filename"`
  - `Content-Type: application/octet-stream` (or specific MIME type)
  - `Content-Length: <file_size>`

#### P2P Communication Endpoints

##### `GET /ping`
- **Purpose**: Health check endpoint
- **Response**: "pong" text

##### `POST /register`
- **Purpose**: Register a new peer
- **Content-Type**: `application/json`
- **Body**: Node information
- **Response**: Local node information

##### `GET /peers`
- **Purpose**: Get all known peers
- **Response**: JSON array of peer information

##### `POST /chunk-transfer`
- **Purpose**: Receive chunk from another node
- **Content-Type**: `application/json`
- **Body**: Chunk transfer request
- **Response**: Success status

##### `GET /chunk?id=<chunk_id>`
- **Purpose**: Download chunk data
- **Parameters**: `id`: Chunk identifier
- **Response**: Binary chunk data

### Response Formats

#### Success Response
```json
{
  "success": true,
  "message": "Operation completed successfully",
  "data": {
    // Operation-specific data
  }
}
```

#### Error Response
```json
{
  "success": false,
  "message": "Error description",
  "data": null
}
```

## 👨‍💻 Developer Guide

### Project Structure

```
DisktroByte/
├── cmd/
│   ├── cli/
│   │   └── main.go             # CLI application entry point
│   └── gui/
│       ├── main.go             # GUI application entry point
│       ├── templates.go        # HTML templates for web interface
│       └── javascript.go       # JavaScript for web interface
├── config/
│   ├── config.go               # Configuration management
│   └── config.yaml             # Configuration file
├── internal/
│   ├── chunker/
│   │   ├── chunker.go          # File chunking logic
│   │   └── reassemble.go       # File reassembly logic
│   ├── compressor/
│   │   └── compressor.go       # Compression utilities
│   ├── distributor/
│   │   └── distributor.go      # File distribution logic
│   ├── encryptor/
│   │   └── encryptor.go        # Encryption utilities
│   ├── metadata/
│   │   └── metadata.go         # Metadata management
│   ├── p2p/
│   │   └── network.go          # P2P networking
│   ├── storage/
│   │   ├── storage.go          # Storage interface
│   │   └── local.go            # Local storage implementation
│   └── utils/
│       └── types.go            # Common types and utilities
├── pkg/
│   ├── env/
│   │   └── env.go              # Environment utilities
│   ├── httpserver/
│   │   └── server.go           # HTTP server utilities
│   └── logging/
│       └── logger.go           # Logging utilities
├── assets/                     # Visual assets and documentation images
│   ├── images/                # Images for README and documentation
│   │   ├── screenshots/       # Application screenshots
│   │   ├── demos/             # Demo GIFs and animations
│   │   ├── diagrams/          # Architecture and flow diagrams
│   │   └── logos/             # Project logos and icons
│   └── README.md               # Assets documentation
├── samples/                    # Sample files for testing
├── tests/                      # Test files
├── temp_downloads/             # Temporary download directory
├── output_chunks/              # Local chunk storage
├── metadata_db_client/         # BadgerDB metadata (client)
├── metadata_db_server/         # BadgerDB metadata (server)
├── go.mod                      # Go module file
├── go.sum                      # Go module checksums
├── .gitignore                  # Git ignore rules
├── README.md                   # This file
├── disktroByte.exe            # Built GUI executable
└── disktroByte-cli.exe        # Built CLI executable
```

### Key Components

#### 1. Main Application (`cmd/cli/main.go`)
```go
// Global variables for application state
var (
    metaStore      *metadata.MetadataStore
    store          storage.Storage
    password       string
    server         *http.Server
    network        *p2p.Network
    fileDistributor *distributor.Distributor
)
```

#### 2. P2P Network (`internal/p2p/network.go`)
```go
type Network struct {
    LocalNode       *Node
    Peers           map[string]*Node
    mu              sync.RWMutex
    heartbeatTicker *time.Ticker
    stopChan        chan bool
}
```

#### 3. File Distributor (`internal/distributor/distributor.go`)
```go
type Distributor struct {
    network      *p2p.Network
    store        storage.Storage
    metaStore    *metadata.MetadataStore
    files        map[string]*FileInfo
    chunks       map[string]*ChunkInfo
    mu           sync.RWMutex
    replicaCount int
}
```

### Development Setup

#### 1. Environment Setup
```bash
# Clone repository
git clone https://github.com/jaywantadh/DisktroByte.git
cd DisktroByte

# Install dependencies
go mod tidy

# Run tests
go test ./...

# Build application
go build ./cmd/cli
```

#### 2. Configuration
Edit `config/config.yaml`:
```yaml
port: 8080
chunk_size: 1048576  # 1MB chunks
replica_count: 3
heartbeat_interval: 30
```

#### 3. Development Commands
```bash
# Run in development mode
go run ./cmd/cli/main.go

# Run with specific port
PORT=8081 go run ./cmd/cli/main.go

# Run tests with coverage
go test -cover ./...

# Build for different platforms
GOOS=linux GOARCH=amd64 go build ./cmd/cli
GOOS=windows GOARCH=amd64 go build ./cmd/cli
GOOS=darwin GOARCH=amd64 go build ./cmd/cli
```

### Adding New Features

#### 1. Adding New API Endpoints
```go
// In main.go, add to createRouter()
mux.HandleFunc("/api/new-endpoint", handleNewEndpoint)

// Implement the handler
func handleNewEndpoint(w http.ResponseWriter, r *http.Request) {
    // Your logic here
    sendJSONResponse(w, true, "Success", data)
}
```

#### 2. Adding New P2P Messages
```go
// In network.go, add new message type
const (
    MessageTypePing = "ping"
    MessageTypePong = "pong"
    MessageTypeNew  = "new_message_type"
)

// Add handler
func (n *Network) HandleNewMessage(w http.ResponseWriter, r *http.Request) {
    // Implementation
}
```

#### 3. Adding New Storage Backends
```go
// Implement the Storage interface
type NewStorage struct {
    // Your storage implementation
}

func (s *NewStorage) Put(chunkData io.Reader) (string, error) {
    // Implementation
}

func (s *NewStorage) Get(id string) (io.ReadCloser, error) {
    // Implementation
}
```

### Testing

#### Unit Tests
```bash
# Run all tests
go test ./...

# Run specific package tests
go test ./internal/chunker

# Run tests with verbose output
go test -v ./...

# Run tests with coverage
go test -cover ./...
```

#### Integration Tests
```bash
# Start multiple nodes for testing
go run ./cmd/cli/main.go &
sleep 2
go run ./cmd/cli/main.go -port 8081 &
sleep 2
go run ./cmd/cli/main.go -port 8082 &
```

#### Performance Testing
```bash
# Test with large files
dd if=/dev/zero of=testfile.dat bs=1M count=100
# Upload testfile.dat through GUI
```

### Debugging

#### Logging
```go
// Add debug logging
fmt.Printf("DEBUG: Processing chunk %s\n", chunkID)
```

#### Network Debugging
```bash
# Check network connectivity
curl http://localhost:8080/ping
curl http://localhost:8081/ping

# Check peer registration
curl -X POST http://localhost:8080/register \
  -H "Content-Type: application/json" \
  -d '{"id":"test","address":"localhost","port":8081}'
```

#### Database Debugging
```bash
# Check BadgerDB files
ls -la metadata_db_client/
ls -la metadata_db_server/
```

## 🔧 Troubleshooting

### Common Issues

#### 1. Port Already in Use
**Problem**: `bind: Only one usage of each socket address`
**Solution**: 
- The application automatically tries the next available port
- Check the console output for the actual port being used
- Kill existing processes: `taskkill /F /IM DisktroByte.exe`

#### 2. Database Lock Error
**Problem**: `Cannot create lock file... Another process is using this Badger database`
**Solution**:
- The application automatically retries with exponential backoff
- Manually remove lock file: `Remove-Item metadata_db_client\LOCK -Force`
- Restart the application

#### 3. File Upload Fails
**Problem**: `Failed to chunk file: failed to open file`
**Solution**:
- Ensure the temp directory exists
- Check file permissions
- Verify the file path is correct

#### 4. Network Connection Issues
**Problem**: Peers not connecting or showing offline
**Solution**:
- Check firewall settings
- Verify port accessibility
- Ensure peers are running on correct ports
- Check network connectivity between nodes

#### 5. Memory Issues
**Problem**: High memory usage with large files
**Solution**:
- Reduce chunk size in configuration
- Process files in smaller batches
- Monitor system resources

### Performance Optimization

#### 1. Chunk Size Optimization
```yaml
# config/config.yaml
chunk_size: 1048576  # 1MB - adjust based on your needs
```

#### 2. Replica Count Adjustment
```yaml
# config/config.yaml
replica_count: 3  # Balance between redundancy and storage
```

#### 3. Network Optimization
```yaml
# config/config.yaml
heartbeat_interval: 30  # Adjust based on network stability
```

### Monitoring and Logs

#### Application Logs
- Check console output for real-time logs
- Log levels: INFO, WARNING, ERROR, DEBUG
- Timestamp format: `[HH:MM:SS]`

#### Network Monitoring
- Use the "Network" tab in GUI
- Monitor peer health status
- Check last seen timestamps

#### Storage Monitoring
- Monitor `output_chunks/` directory size
- Check `metadata_db_client/` for metadata growth
- Monitor disk space usage

## 🤝 Contributing

### Development Workflow

1. **Fork the Repository**
   ```bash
   git clone https://github.com/your-username/DisktroByte.git
   cd DisktroByte
   ```

2. **Create Feature Branch**
   ```bash
   git checkout -b feature/your-feature-name
   ```

3. **Make Changes**
   - Follow Go coding standards
   - Add tests for new features
   - Update documentation

4. **Test Your Changes**
   ```bash
   go test ./...
   go build ./cmd/cli
   go run ./cmd/cli/main.go
   ```

5. **Commit and Push**
   ```bash
   git add .
   git commit -m "Add feature: description"
   git push origin feature/your-feature-name
   ```

6. **Create Pull Request**
   - Provide detailed description
   - Include test results
   - Reference related issues

### Code Standards

#### Go Code Style
- Follow [Effective Go](https://golang.org/doc/effective_go.html)
- Use `gofmt` for formatting
- Run `golint` for code quality
- Add comments for exported functions

#### Testing Standards
- Unit tests for all new functions
- Integration tests for API endpoints
- Performance tests for critical paths
- Minimum 80% code coverage

#### Documentation Standards
- Update README.md for new features
- Add inline comments for complex logic
- Document API changes
- Include usage examples

### Areas for Contribution

#### High Priority
- [ ] Performance optimization
- [ ] Additional storage backends
- [ ] Enhanced security features
- [ ] Mobile app development

#### Medium Priority
- [ ] WebRTC integration
- [ ] Blockchain integration
- [ ] Advanced compression algorithms
- [ ] Machine learning optimization

#### Low Priority
- [ ] Additional UI themes
- [ ] Plugin system
- [ ] Advanced analytics
- [ ] Multi-language support

## 🎆 Recent Achievements

### 🔧 File Reassembly Feature (Latest)
![File Reassembly Banner](assets/images/file-reassembly-banner.png)

We've recently implemented a comprehensive **File Reassembly System** that brings intelligent file reconstruction to DisktroByte:

#### ✨ **What's New**

- **🎨 Modern Web Interface**: Beautiful file reassembly dashboard with real-time progress tracking
- **📊 Visual Progress Bars**: Live status updates during file reconstruction process
- **📁 Smart File Type Detection**: Automatic recognition and proper handling of different file formats
- **🎯 Chunk Status Monitoring**: Visual indicators showing exactly which chunks are available vs missing
- **📥 Direct Downloads**: One-click download of reassembled files directly from the web interface
- **🎦 Demo File Support**: Interactive demo files to test the interface without real data

#### 🚀 **Key Features**

| Feature | Description | Status |
|---------|-------------|--------|
| **File Browsing** | Browse available files with metadata and completion status | ✅ Complete |
| **Progress Tracking** | Real-time progress bars and status updates | ✅ Complete |
| **File Type Icons** | Smart icons for PDF, ZIP, MP4, PPTX files | ✅ Complete |
| **Demo Files** | 4 realistic demo files for testing (PDF, ZIP, MP4, PPTX) | ✅ Complete |
| **Authentication** | Secure user authentication and session management | ✅ Complete |
| **Direct Download** | Download reassembled files directly from browser | ✅ Complete |
| **History Tracking** | Track reassembly operations and downloads | ✅ Complete |
| **Error Handling** | Comprehensive error handling and user feedback | ✅ Complete |

#### 📊 **Demo Files Available**

1. **📑 sample_document.pdf** (2MB) - Ready to download
2. **🗜️ example_archive.zip** (10MB) - Ready to download  
3. **🎥 sample_video.mp4** (50MB) - Partial (97.5% complete)
4. **📊 presentation.pptx** (15MB) - Ready to download

#### 🛠️ **Technical Implementation**

- **Backend**: New API endpoints (`/api/files/available`, `/api/files/download`)
- **Frontend**: Enhanced JavaScript with progress tracking and file management
- **Database**: Improved metadata storage and retrieval
- **Security**: Authentication-protected endpoints with proper session handling
- **UX**: Intuitive interface with visual feedback and error handling

#### 🎯 **Usage**

1. Navigate to the **File Reassembly** (🔧) tab
2. Click **"Refresh Available Files"** to load the file list
3. View files with completion percentages and status indicators
4. Click **"Reassemble"** (real files) or **"Prepare"** (demo files)
5. Monitor real-time progress with visual feedback
6. Download completed files directly from the interface

This feature represents a significant milestone in making DisktroByte more user-friendly and production-ready for distributed file management scenarios.

---

## 📝 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 🙏 Acknowledgments

- **Go Team**: For the excellent programming language
- **BadgerDB**: For the high-performance key-value store
- **LZ4**: For the fast compression algorithm
- **ChaCha20-Poly1305**: For the secure encryption
- **Open Source Community**: For inspiration and support

## 📞 Support

### Getting Help

1. **Documentation**: Check this README and inline code comments
2. **Issues**: Create an issue on GitHub for bugs or feature requests
3. **Discussions**: Use GitHub Discussions for questions and ideas
4. **Email**: Contact the maintainers for private support

### Reporting Bugs

When reporting bugs, please include:

1. **Environment Details**:
   - Operating System and version
   - Go version
   - DisktroByte version

2. **Steps to Reproduce**:
   - Detailed step-by-step instructions
   - Sample files (if applicable)
   - Expected vs actual behavior

3. **Logs and Error Messages**:
   - Console output
   - Error messages
   - Network logs

4. **Additional Context**:
   - File sizes and types
   - Network configuration
   - System resources

---

**Made with ❤️ by the DisktroByte Team**

*Empowering decentralized file storage for the future.*
