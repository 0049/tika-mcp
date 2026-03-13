# tika-mcp

A **Model Context Protocol (MCP) server** written in Go that exposes [Apache Tika](https://tika.apache.org/) document processing capabilities to LLM agents.

Enables AI assistants to extract text, metadata, and content type information from 1,000+ document formats including PDF, DOCX, XLSX, PPTX, HTML, images (with OCR), emails, and more.

---

## Features

| Tool | Description |
|---|---|
| `tika_extract_text` | Extract plain text from any document |
| `tika_extract_html` | Extract content as annotated HTML |
| `tika_extract_xml` | Extract content as XHTML/XML |
| `tika_get_metadata` | Get all document metadata (author, dates, page count…) |
| `tika_get_metadata_field` | Get a single metadata field |
| `tika_detect_content_type` | Detect MIME type of a document |
| `tika_detect_language` | Detect natural language of document text |
| `tika_extract_from_url` | Fetch + extract a remote document by URL |
| `tika_server_version` | Get Apache Tika server version |
| `tika_list_mime_types` | List all supported MIME types |
| `tika_list_parsers` | List all available parsers |
| `tika_health_check` | Check if Tika server is reachable |

---

## Prerequisites

- **Apache Tika Server** running and accessible
- **Go 1.22+** (to build) or **Docker**

### Start Apache Tika Server

**Option A — Docker (recommended):**
```bash
docker run -d -p 9998:9998 apache/tika:2.9.1-full
```
The `-full` image includes OCR support (Tesseract).

**Option B — Java JAR:**
```bash
# Download from https://tika.apache.org/download.html
java -jar tika-server-standard-2.9.1.jar --port 9998
```

---

## Installation

### Build from source
```bash
git clone https://github.com/your-org/tika-mcp
cd tika-mcp
go build -o tika-mcp .
```

### Docker
```bash
docker build -t tika-mcp .
```

### systemd (Linux production install)

Two unit files are provided: `tika.service` (Apache Tika JVM process) and `tika-mcp.service` (this MCP server). `tika-mcp.service` declares `After=tika.service` so systemd always starts them in the right order.

```bash
# 1. Install the tika-mcp binary
sudo install -m 755 tika-mcp /usr/local/bin/tika-mcp

# 2. Install Apache Tika JAR (adjust version as needed)
sudo mkdir -p /opt/tika
sudo install -m 644 tika-server-standard-2.9.1.jar /opt/tika/tika-server.jar

# 3. Create dedicated service accounts (no login shell, no home dir)
sudo useradd --system --no-create-home --shell /usr/sbin/nologin tika
sudo useradd --system --no-create-home --shell /usr/sbin/nologin tika-mcp

# 4. Create log and cache directories
sudo mkdir -p /var/log/tika-mcp /var/cache/tika
sudo chown tika-mcp:tika-mcp /var/log/tika-mcp
sudo chown tika:tika /var/cache/tika

# 5. Install environment file
sudo mkdir -p /etc/tika-mcp /etc/tika
sudo install -m 640 tika-mcp.env /etc/tika-mcp/tika-mcp.env
sudo chown root:tika-mcp /etc/tika-mcp/tika-mcp.env

# 6. Install and enable unit files
sudo install -m 644 tika.service     /etc/systemd/system/tika.service
sudo install -m 644 tika-mcp.service /etc/systemd/system/tika-mcp.service
sudo systemctl daemon-reload

# 7. Enable both services at boot and start them now
sudo systemctl enable --now tika.service
sudo systemctl enable --now tika-mcp.service
```

**Verify they are running:**
```bash
sudo systemctl status tika.service
sudo systemctl status tika-mcp.service

# Follow live logs
journalctl -u tika-mcp -f
journalctl -u tika -f
```

**Common management commands:**
```bash
# Restart after config change
sudo systemctl restart tika-mcp

# Change Tika URL or listen port
sudo nano /etc/tika-mcp/tika-mcp.env
sudo systemctl restart tika-mcp

# Stop both services
sudo systemctl stop tika-mcp tika

# Disable from autostart
sudo systemctl disable tika-mcp tika
```

---

## Usage

### Transport modes

tika-mcp supports two transport modes:

#### 1. stdio (default) — for local MCP clients (Claude Desktop, etc.)

```bash
# With Tika running on localhost:9998
./tika-mcp

# Custom Tika URL
./tika-mcp --tika-url http://my-tika-host:9998

# Via environment variable
TIKA_URL=http://my-tika-host:9998 ./tika-mcp
```

#### 2. HTTP — for remote/networked MCP clients

```bash
./tika-mcp --transport http --addr :8080
```

The HTTP endpoint accepts `POST /mcp` with a JSON-RPC 2.0 request body.

---

## MCP Client Configuration

### Claude Desktop (`claude_desktop_config.json`)

```json
{
  "mcpServers": {
    "tika": {
      "command": "/path/to/tika-mcp",
      "args": ["--tika-url", "http://localhost:9998"],
      "env": {}
    }
  }
}
```

### Claude Desktop with Docker
```json
{
  "mcpServers": {
    "tika": {
      "command": "docker",
      "args": [
        "run", "--rm", "-i",
        "--network", "host",
        "-e", "TIKA_URL=http://localhost:9998",
        "tika-mcp"
      ]
    }
  }
}
```

### HTTP transport (remote clients)
Point your MCP client to: `http://your-server:8080/mcp`

---

## Docker Compose (full stack)

```bash
# Starts both Tika and tika-mcp (HTTP mode on :8080)
docker-compose up -d
```

---

## Tool Reference

All document tools accept **base64-encoded** file content.

### tika_extract_text
```json
{
  "content_base64": "<base64-encoded bytes>",
  "filename": "report.pdf",      // optional — helps parser selection
  "content_type": "application/pdf"  // optional — MIME type hint
}
```

### tika_get_metadata_field
```json
{
  "content_base64": "<base64-encoded bytes>",
  "field": "Author"
}
```
Common fields: `Author`, `title`, `Creation-Date`, `Last-Modified`, 
`Content-Type`, `xmpTPg:NPages` (page count), `dc:creator`, `dc:description`.

### tika_extract_from_url
```json
{
  "url": "https://example.com/document.pdf"
}
```

---

## CLI Flags

| Flag | Env Var | Default | Description |
|---|---|---|---|
| `--tika-url` | `TIKA_URL` | `http://localhost:9998` | Apache Tika server URL |
| `--transport` | `MCP_TRANSPORT` | `stdio` | Transport: `stdio` or `http` |
| `--addr` | `MCP_ADDR` | `:8080` | HTTP listen address |
| `--version` | — | — | Print version and exit |

---

## Development

```bash
# Run tests
go test ./...

# Run with verbose output
go test -v ./...

# Build
go build -o tika-mcp .

# Test health check manually (requires Tika running)
echo '{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"tika_health_check","arguments":{}}}' \
  | ./tika-mcp
```

---

## Architecture

```
┌─────────────────────────────────┐
│           MCP Client            │
│  (Claude Desktop / AI Agent)    │
└──────────────┬──────────────────┘
               │ MCP Protocol (JSON-RPC 2.0)
               │ stdio OR HTTP POST /mcp
┌──────────────▼──────────────────┐
│           tika-mcp              │
│  ┌─────────────────────────┐   │
│  │   MCPServer (server.go) │   │
│  │  - initialize           │   │
│  │  - tools/list           │   │
│  │  - tools/call           │   │
│  └──────────┬──────────────┘   │
│             │                   │
│  ┌──────────▼──────────────┐   │
│  │   TikaClient (tika.go)  │   │
│  │  - ExtractText/HTML/XML │   │
│  │  - GetMetadata          │   │
│  │  - DetectContentType    │   │
│  │  - DetectLanguage       │   │
│  └──────────┬──────────────┘   │
└─────────────┼───────────────────┘
              │ HTTP REST
┌─────────────▼───────────────────┐
│      Apache Tika Server         │
│         (port 9998)             │
└─────────────────────────────────┘
```

---

## Supported Formats (via Apache Tika)

PDF, Microsoft Office (DOC/DOCX/XLS/XLSX/PPT/PPTX), OpenDocument, RTF, HTML, XML, JSON, CSV, plain text, images (JPEG/PNG/TIFF with OCR), email (EML/MSG), ZIP/TAR archives, audio/video metadata, iCalendar, vCard, and [1000+ more](https://tika.apache.org/2.9.1/formats.html).
