# SoundCloud MP3 Downloader 🎵

[![Go Version](https://img.shields.io/badge/Go-1.21+-blue.svg)](https://golang.org)
[![License](https://img.shields.io/badge/License-MIT-green.svg)](LICENSE)
[![Platform](https://img.shields.io/badge/Platform-Windows%20%7C%20Linux%20%7C%20macOS-lightgrey.svg)]()
[![gRPC](https://img.shields.io/badge/gRPC-Supported-brightgreen.svg)](https://grpc.io/)

A fast command-line tool to download MP3 files from SoundCloud tracks with both standalone and gRPC server-client architectures.

## ✨ Features

- 🚀 Download MP3 files from any public SoundCloud track
- 📁 Automatic output directory creation
- 🎯 Simple CLI interface
- 🔄 Progress feedback during download
- 🌐 Cross-platform compatibility
- 🔌 **NEW: gRPC server-client architecture**
- 📊 **NEW: Download status monitoring**
- 📋 **NEW: Download history tracking**

## 🏗️ Architecture Options

### 1. Standalone Mode (Original)
Single executable that downloads directly.

### 2. gRPC Mode (New)
Server-client architecture with:
- **Server**: Handles downloads with status tracking
- **Client**: Communicates with server for downloads
- **Benefits**: Concurrent downloads, status monitoring, download history

## 🚀 Quick Start

### Prerequisites
- Go 1.21+ or download the pre-built executable
- For gRPC mode: Protocol Buffers compiler (`protoc`)

### Installation

```bash
# Clone the repository
git clone <your-repo-url>
cd SoundCloudDownloader

# Build the application
go mod tidy
go build -o soundcloud-downloader
```

### Usage

#### Standalone Mode
```bash
# Basic download
./soundcloud-downloader "https://soundcloud.com/artist/track-name"

# Custom output directory
./soundcloud-downloader -o ~/Music "https://soundcloud.com/artist/track-name"

# Windows batch file
download.bat "https://soundcloud.com/artist/track-name"

# Linux/Mac shell script
./download.sh "https://soundcloud.com/artist/track-name"
```

#### gRPC Mode
```bash
# Windows - Quick setup
run-grpc.bat

# Manual setup
make deps
make all

# Start server (in one terminal)
./bin/server

# Use client (in another terminal)
./bin/client download "https://soundcloud.com/artist/track-name"
./bin/client list 10
```

## 📖 Command Options

### Standalone Mode
| Flag | Description | Default |
|------|-------------|---------|
| `-o, --output` | Output directory | `downloads/` |
| `-h, --help` | Show help | - |

### gRPC Client Commands
| Command | Description | Example |
|---------|-------------|---------|
| `download <url> [dir] [filename]` | Download a track | `client download "URL" "Music" "track.mp3"` |
| `list [limit]` | List recent downloads | `client list 10` |

## 🔧 How It Works

### Standalone Mode
1. **Fetch** SoundCloud track page
2. **Extract** track ID and client ID
3. **Get** direct stream URL via API
4. **Download** MP3 file

### gRPC Mode
1. **Client** sends download request to server
2. **Server** starts background download process
3. **Server** updates download status in real-time
4. **Client** monitors progress via status requests
5. **Server** saves file and updates completion status

## 📁 Output

- Files saved as: `soundcloud_[track_id].mp3` (or custom filename)
- Default location: `downloads/` folder
- gRPC mode: Download history and status tracking

## ⚠️ Limitations

- Public tracks only
- 128kbps MP3 quality
- Requires streaming availability
- May not work with download-restricted tracks

## 🛠️ Troubleshooting

| Error | Solution |
|-------|----------|
| `could not find client_id` | SoundCloud page structure changed |
| `could not find stream URL` | Track may be private/restricted |
| `failed to download file` | Check network connection |
| `invalid SoundCloud URL` | Verify URL format |
| `gRPC connection failed` | Ensure server is running on :50051 |

## 📄 License

MIT License - see [LICENSE](LICENSE) file for details.

## ⚖️ Legal Notice

**Educational use only.** Respect copyright laws and only download content you have permission to access.