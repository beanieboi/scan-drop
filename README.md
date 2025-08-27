# Scan Drop - FTP Server for Document Scanners

A lightweight, minimal FTP server designed specifically for receiving scanned documents from network scanners and uploading them directly to [Paperless-ngx](https://github.com/paperless-ngx/paperless-ngx) via REST API.

## 🎯 Purpose

Many document scanners support FTP as a delivery method for scanned files, but setting up a full FTP server can be overkill for this simple use case. Scan Drop provides just enough functionality to:

- Accept scanned documents from network scanners via FTP
- Upload files directly to Paperless-ngx via REST API
- Process documents entirely in memory for efficiency

## ✨ Features

- **Zero configuration authentication** - accepts any username/password
- **Direct API integration** - uploads documents directly to Paperless-ngx REST API
- **In-memory processing** - documents are processed entirely in memory (no temporary files)
- **Scanner compatibility** - supports active (PORT/EPRT) data transfer modes
- **Lightweight** - single Go binary with no external dependencies
- **Token-based authentication** - secure API authentication with Paperless-ngx
- **Comprehensive logging** - logs all FTP commands for debugging scanner issues

## 🚀 Quick Start

### Build from Source

```bash
go build -o scan-drop ./cmd/scan-drop
```

### Run with Default Settings

```bash
# Requires PAPERLESS_TOKEN environment variable
PAPERLESS_TOKEN=your_api_token ./scan-drop
```

### Run with Custom Settings

```bash
# Custom port and Paperless-ngx URL
FTP_PORT=3021 PAPERLESS_URL=http://paperless:8000 PAPERLESS_TOKEN=your_token ./scan-drop

# Enable debug logging (shows FTP command/response traffic)
LOG_LEVEL=DEBUG PAPERLESS_TOKEN=your_token ./scan-drop

# Combined configuration
FTP_PORT=2121 PAPERLESS_URL=http://localhost:8000 PAPERLESS_TOKEN=your_token LOG_LEVEL=DEBUG ./scan-drop
```

## 🔧 Configuration

All configuration is done through environment variables:

### Environment Variables

- **`FTP_PORT`**: FTP server port (default: `2121`)
- **`PAPERLESS_URL`**: Paperless-ngx instance URL (default: `http://localhost:8000`)
- **`PAPERLESS_TOKEN`**: Paperless-ngx API token (**required**)
- **`HTTP_TIMEOUT`**: API request timeout (default: `30s`)
- **`LOG_LEVEL`**: Controls logging verbosity (default: `INFO`)
  - `INFO`: Shows server startup, connections, document uploads, API responses, and errors
  - `DEBUG`: Shows all INFO logs plus FTP command/response traffic for debugging scanner issues
  - `WARN`: Shows only warnings and errors
  - `ERROR`: Shows only errors

### Examples

```bash
# Default settings (requires PAPERLESS_TOKEN)
PAPERLESS_TOKEN=your_api_token ./scan-drop

# Custom port
FTP_PORT=3021 PAPERLESS_TOKEN=your_api_token ./scan-drop

# Custom Paperless-ngx URL
PAPERLESS_URL=http://paperless:8000 PAPERLESS_TOKEN=your_api_token ./scan-drop

# Debug mode for troubleshooting
LOG_LEVEL=DEBUG PAPERLESS_TOKEN=your_api_token ./scan-drop

# Full configuration
FTP_PORT=2121 PAPERLESS_URL=http://localhost:8000 PAPERLESS_TOKEN=your_api_token HTTP_TIMEOUT=30s LOG_LEVEL=INFO ./scan-drop
```

## 📱 Scanner Setup

Configure your document scanner with these FTP settings:

- **Server**: Your Paperless-ngx host IP
- **Port**: 2121 (or your custom port)
- **Username**: Any value (e.g., `scanner`) - *authentication is intentionally disabled*
- **Password**: Any value (e.g., `password`) - *authentication is intentionally disabled*
- **Directory**: `/` or leave blank
- **Transfer Mode**: Active (PORT/EPRT)

## 🐳 Docker Integration

If running Paperless-ngx in Docker, you can run scan-drop alongside:

```bash
# Run scan-drop in Docker network
docker run -d \
  --name scan-drop \
  --network paperless_default \
  -p 2121:2121 \
  -e PAPERLESS_URL=http://paperless-ngx:8000 \
  -e PAPERLESS_TOKEN=your_api_token \
  -e LOG_LEVEL=INFO \
  ghcr.io/beanieboi/scan-drop:latest
```

### Docker Compose Integration

For integration with Paperless-ngx, add this service to your docker-compose.yml:

```yaml
services:
  scan-drop:
    image: ghcr.io/beanieboi/scan-drop:latest
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

## 🔍 Supported FTP Commands

The server implements these FTP commands for scanner compatibility:

- `USER`, `PASS` - Authentication (accepts anything)
- `STOR` - Store/upload files (uploads directly to Paperless-ngx API)
- `RETR` - Retrieve/download files (not supported - returns error)
- `LIST` - Directory listing (returns empty)
- `DELE` - Delete files (not supported - returns error)
- `PORT`, `EPRT` - Active mode data connection
- `PWD`, `CWD` - Directory navigation
- `TYPE`, `SYST`, `FEAT` - System information
- `QUIT` - Disconnect

## 📁 File Handling

- Files are processed entirely in memory (no temporary files created)
- Documents are uploaded directly to Paperless-ngx via REST API
- Filenames are preserved and passed to Paperless-ngx
- Paperless-ngx processes documents immediately upon API upload

## 🔒 Security Notes

This FTP server is designed for trusted local networks only and **intentionally disables FTP authentication** to simplify scanner setup:

- **No FTP authentication** - any FTP credentials are accepted (by design for ease of setup)
- **Secure API authentication** - uses token-based authentication with Paperless-ngx API
- **No encryption** - FTP transfers are in plain text (use trusted networks only)
- **In-memory processing** - no files stored locally, direct API upload
- **API security** - all uploads authenticated via Paperless-ngx API token

⚠️ **Important**: The lack of FTP authentication is a deliberate design choice to eliminate scanner configuration issues. Only run this server on trusted networks where unauthorized FTP access is not a concern.

## 🐛 Troubleshooting

### Scanner Can't Connect

1. Check if the port is accessible: `telnet your-server 2121`
2. Verify firewall settings allow the FTP port
3. Check server logs for connection attempts

### Files Not Appearing in Paperless-ngx

1. Verify `PAPERLESS_TOKEN` environment variable is set correctly
2. Check `PAPERLESS_URL` points to the correct Paperless-ngx instance
3. Verify the API token has permissions to upload documents
4. Monitor scan-drop logs for API errors
5. Check Paperless-ngx logs for processing errors

### Scanner Reports Login Failed

Some scanners are strict about FTP responses. Try:
1. Using simple usernames like `user` or `scanner`
2. Using simple passwords like `password` or `scan`
3. Checking scanner documentation for specific FTP requirements
4. Enable debug logging with `LOG_LEVEL=DEBUG` to see the exact FTP commands and responses
