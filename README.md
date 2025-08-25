# DisktroByte

DisktroByte is a lightweight, high-performance, enterprise-ready P2P distributed file system. It is designed for organizational use, enabling secure, fast, and decentralized file sharing without the complexity of blockchain technology.

## Features

-   **Peer Discovery and Registration:** Uses HTTP-based endpoints (`/ping`, `/peers`) for peer discovery and registration.
-   **Liveness Check:** Actively monitors peers to ensure they are online.
-   **Compression:** Includes a compression module with media-type detection to skip non-beneficial file types (e.g., `.mp4`, `.jpg`, `.zip`).
-   **File Chunking:** Splits files into smaller, transportable chunks.
-   **Encryption:** Secures file chunks using ChaCha20-Poly1305.
-   **Pluggable Storage:** Supports different storage backends, with a local filesystem implementation provided.
-   **File Transfer:** A client-server model for transferring files between peers.
-   **CLI Interface:** A command-line interface for interacting with the application.

## Project Structure

```
disktrobyte/
├── cmd/
│   ├── cli/
│   │   └── main.go             # CLI entry point
│   └── gui/                    # GUI entry (to be developed later)
├── internal/
│   ├── peer/                   # Node peer logic
│   ├── chunker/                # File chunking logic
│   ├── compressor/             # Compression algorithms and strategy
│   ├── encryptor/              # Encryption logic and algorithm choice
│   ├── storage/                # Pluggable storage backends
│   ├── metadata/               # Metadata manager (e.g., BadgerDB abstraction)
│   ├── distributor/            # Distribution + Replication manager
│   ├── discovery/              # P2P node discovery and connection
│   ├── transfer/               # Upload/download logic
│   └── utils/                  # Helper functions and utilities
├── pkg/                        # Public-facing packages
├── config/                     # Config files (YAML, JSON)
│   ├── config.go               # Config loader using Viper
│   └── config.yaml             # Default config template
├── scripts/                    # Helper scripts (e.g., benchmarks)
├── web/                        # Frontend (to be developed later)
│   └── tauri/                  # Tauri-based GUI wrapper
├── tests/                      # Integration and unit tests
├── api/                        # gRPC or REST APIs
├── .env                        # Environment variables template
├── .gitignore
├── go.mod                      # Go module dependencies
├── go.sum
├── Makefile                    # Common CLI tasks (build, run, test)
├── README.md                   # Project overview and usage
└── LICENSE
```

### File Explanations

-   `cmd/cli/main.go`: The main entry point for the command-line interface. It handles command parsing and orchestrates the application's workflow.
-   `internal/chunker/`: Contains the logic for splitting files into chunks and reassembling them.
-   `internal/compressor/`: Implements the compression logic, including the decision of whether to compress a file based on its type.
-   `internal/encryptor/`: Handles the encryption and decryption of file chunks.
-   `internal/storage/`: Defines the `Storage` interface and provides a `LocalStorage` implementation for storing chunks on the local filesystem.
-   `internal/metadata/`: Manages the metadata for files and chunks using a BadgerDB database.
-   `internal/transfer/`: Implements the client-server model for transferring files between peers.
-   `config/`: Contains the application's configuration files and the logic for loading them.

## Workflow

1.  **Chunking:** When a file is added to `DisktroByte`, it is first split into smaller chunks.
2.  **Compression:** Each chunk is then compressed, unless it's a file type that won't benefit from compression.
3.  **Encryption:** The compressed chunks are encrypted using ChaCha20-Poly1305.
4.  **Storage:** The encrypted chunks are stored using the configured storage backend (e.g., the local filesystem).
5.  **Metadata:** Metadata for the file and its chunks is stored in a BadgerDB database.
6.  **Transfer:** When a peer requests a file, the client initiates a transfer with the server, and the chunks are sent over the network.
7.  **Reassembly:** The receiving peer reassembles the file from the chunks, decrypts them, and decompresses them.

## Commands

### Build the Application

```bash
go build ./cmd/cli
```

### Run the Application

#### Start the Server

To start the `DisktroByte` server, run the following command:

```bash
go run cmd/cli/main.go server
```

The server will start on the port specified in the `config/config.yaml` file (default is `8080`).

#### Chunk a File

To split a file into chunks, use the `chunk` command:

```bash
go run cmd/cli/main.go chunk <file_path>
```

This will create a series of chunk files in the `output_chunks` directory.

#### Upload a File

To upload a file to a peer, use the `upload` command:

```bash
go run cmd/cli/main.go upload <file_path> <peer_address>
```

-   `<file_path>`: The path to the file you want to upload.
-   `<peer_address>`: The address of the peer you want to upload the file to (e.g., `http://localhost:8080`).

#### Reassemble a File

To reassemble a file from its chunks, use the `reassemble` command:

```bash
go run cmd/cli/main.go reassemble <file_name> <output_path>
```

-   `<file_name>`: The name of the file as it was originally chunked.
-   `<output_path>`: The path where the reassembled file will be saved.
