# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Scan Drop is a simple FTP server implementation written in Go designed specifically for document scanners. The server accepts any username/password combination and provides basic FTP functionality for receiving scanned documents and uploading them directly to Paperless-ngx via REST API.

## Architecture

- **Single file implementation**: The entire FTP server is contained in `cmd/scan-drop/main.go`
- **Config struct**: Environment-based configuration management with defaults
- **FTPServer struct**: Core server type that manages connections and Paperless-ngx integration
- **Protocol support**: Implements essential FTP commands (USER, PASS, STOR, LIST, PORT, EPRT, etc.)
- **Active data connections**: Supports PORT/EPRT commands for data transfer (PASV not supported)
- **In-memory processing**: Documents are processed entirely in memory for efficiency
- **Direct API integration**: Documents are uploaded directly to Paperless-ngx REST API

## Build and Run Commands

```bash
# Build the project
go build -o scan-drop cmd/scan-drop/main.go

# Run with default settings (requires PAPERLESS_TOKEN)
PAPERLESS_TOKEN=your_api_token ./scan-drop

# Run with custom configuration
FTP_PORT=2121 PAPERLESS_URL=http://paperless:8000 PAPERLESS_TOKEN=your_token LOG_LEVEL=DEBUG ./scan-drop

# All environment variables
FTP_PORT=2121 \
PAPERLESS_URL=http://localhost:8000 \
PAPERLESS_TOKEN=your_api_token \
HTTP_TIMEOUT=30s \
LOG_LEVEL=INFO \
./scan-drop
```

## Key Implementation Details

- **Default port**: 2121 (configurable via `FTP_PORT` environment variable)
- **Required configuration**: `PAPERLESS_TOKEN` environment variable must be set
- **Paperless-ngx integration**:
  - **API URL**: Configurable via `PAPERLESS_URL` (default: `http://localhost:8000`)
  - **Authentication**: Token-based via `PAPERLESS_TOKEN` environment variable (required)
  - **HTTP timeout**: Configurable via `HTTP_TIMEOUT` (default: 30s)
- **Logging**: Configurable via `LOG_LEVEL` environment variable
  - INFO: Server events, document uploads, connection errors, API responses
  - DEBUG: All INFO logs plus FTP command/response traffic
- **Security**: No FTP authentication - accepts any username/password
- **File handling**: Documents are processed entirely in memory and uploaded directly to API (no temporary files)
- **Memory efficiency**: Files are read once from FTP data connection and immediately uploaded to Paperless-ngx
- **Error handling**: Comprehensive error handling with structured logging for debugging

## Docker Usage

The application is available as a Docker image published to GitHub Container Registry:

```bash
# Run with Paperless-ngx API integration
docker run -d \
  --name scan-drop \
  -p 2121:2121 \
  -e PAPERLESS_URL=http://paperless:8000 \
  -e PAPERLESS_TOKEN=your_api_token \
  -e LOG_LEVEL=INFO \
  ghcr.io/yourusername/scan-drop:latest
```

### Docker Compose Integration

For integration with Paperless-ngx, add this service to your docker-compose.yml:

```yaml
services:
  scan-drop:
    image: ghcr.io/yourusername/scan-drop:latest
    container_name: scan-drop
    ports:
      - "2121:2121"
    environment:
      - PAPERLESS_URL=http://paperless-ngx:8000
      - PAPERLESS_TOKEN=${PAPERLESS_API_TOKEN}
      - LOG_LEVEL=INFO
    restart: unless-stopped
    depends_on:
      - paperless-ngx

```

**Environment Variables**:
- `PAPERLESS_URL`: URL of your Paperless-ngx instance (default: `http://localhost:8000`)
- `PAPERLESS_TOKEN`: API token for authentication (**required**)
- `FTP_PORT`: FTP server port (default: `2121`)
- `LOG_LEVEL`: Logging level (`DEBUG`, `INFO`, `WARN`, `ERROR`)
- `HTTP_TIMEOUT`: API request timeout (default: `30s`)

## Testing

The server can be tested with any FTP client:
```bash
ftp localhost 2121
# Use any username/password combination
```