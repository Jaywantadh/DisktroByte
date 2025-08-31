# 🚀 DisktroByte - P2P Distributed File System

A revolutionary peer-to-peer distributed file system built in Go, featuring automatic file chunking, compression, encryption, and distributed storage across a network of nodes.

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
- **📱 Web-based GUI** for easy management
- **⚡ High performance** with parallel processing

### Use Cases

- **Distributed Backup Systems**: Redundant file storage across multiple locations
- **Content Distribution Networks**: Efficient file sharing in P2P networks
- **Edge Computing**: Distributed storage for edge devices
- **Blockchain Storage**: Decentralized file storage for blockchain applications
- **Research Data Sharing**: Secure, distributed research data management

## 🏗️ System Architecture

![System-Architecture](https://github.com/Jaywantadh/Images/blob/main/System-Architecture.png)

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

### 🎨 User Interface
- **Modern Web GUI**: Beautiful, responsive interface
- **Real-time Updates**: Live status updates and progress tracking
- **Cross-platform**: Works on Windows, macOS, and Linux
- **Intuitive Design**: Easy-to-use interface for all skill levels

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
   go build ./cmd/cli
   ```

4. **Run the Application**
   ```bash
   # Option 1: Direct execution
   go run ./cmd/cli/main.go
   
   # Option 2: Using the provided script (Windows)
   start-gui.bat
   
   # Option 3: Using Make (if available)
   make gui
   ```

## 🚀 Quick Start

### 1. Start the Application

```bash
go run ./cmd/cli/main.go
```

### 2. Access the Web Interface

Open your browser and navigate to:
```
http://localhost:8080
```

### 3. Set Your Password

1. Enter your encryption password in the password field
2. Click "Set Password"
3. This password will be used for all file operations

### 4. Upload and Distribute Files

1. Go to the "Chunk Files" tab
2. Select a file to upload
3. Click "Chunk File"
4. Watch as your file is automatically:
   - Chunked into smaller pieces
   - Compressed (if beneficial)
   - Encrypted with your password
   - Distributed across the network

### 5. View Network Status

1. Go to the "Network" tab
2. Click "Refresh Network"
3. Monitor connected peers and their health status

## 📖 Usage Guide

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

#### Reassembling Files
**Purpose**: Reconstruct original files from distributed chunks

**Steps**:
1. Go to "Files" tab
2. Click "Refresh Files" to load your file list
3. Click "Reassemble" on any file
4. Enter the output path for the reassembled file
5. Wait for completion

**What Happens**:
- System locates all chunks for the file
- Downloads missing chunks from peers (if needed)
- Decrypts all chunks using your password
- Decompresses chunks (if they were compressed)
- Reassembles chunks in correct order
- Saves the complete file to your specified location

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
│   └── cli/
│       └── main.go              # Main application entry point
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
├── samples/                    # Sample files for testing
├── tests/                      # Test files
├── web/                        # Web assets (if any)
├── go.mod                      # Go module file
├── go.sum                      # Go module checksums
├── Makefile                    # Build automation
├── README.md                   # This file
└── start-gui.bat              # Windows startup script
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

## 📄 License

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
