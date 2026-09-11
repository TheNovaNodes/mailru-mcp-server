<div align="center">
  <h1>📧 Mail.ru MCP Server (Go Production Edition)</h1>
  <p><b>A core component of TheNovaNodes Ecosystem</b></p>
  <p>
    <i>High-performance, stateless Model Context Protocol (MCP) gateway written in Go, enabling Agentic AI control via Mail.ru IMAP, SMTP, and WebDAV.</i>
  </p>
  <p>
    [![CI](https://github.com/TheNovaNodes/mailru-mcp-server/actions/workflows/ci.yml/badge.svg)](https://github.com/TheNovaNodes/mailru-mcp-server/actions/workflows/ci.yml)
    <img src="https://img.shields.io/badge/go-1.22+-00ADD8.svg?logo=go" alt="Go Version" />
    <img src="https://img.shields.io/badge/tests-passing-brightgreen.svg" alt="Tests" />
    <img src="https://img.shields.io/badge/protocol-MCP%20JSON--RPC-green.svg" alt="MCP" />
    <img src="https://img.shields.io/badge/memory-15MB%20RSS-blue.svg" alt="Memory" />
    <img src="https://img.shields.io/badge/license-MIT-blue.svg" alt="License" />
  </p>
</div>

---

## 🧪 Ecosystem Role
This repository serves as a hardened, stateless bridge between autonomous AI agents (e.g. Kairos, Trickster, Prometheus) and Mail.ru services. It strictly enforces Human-In-The-Loop (HITL) policies via an atomic Two-Phase Commit token system with sliding TTL for all mutating or outbound operations.

> **✅ CAPABILITIES MATRIX:**
> - **Mail (IMAP / SMTP):** 🟢 FULLY SUPPORTED (TLS on 993/465, strict 15s socket deadlines, aggregates INBOX & smart subfolders, draft generation, full body extraction, semantic thread search).
> - **Cloud Storage (WebDAV):** 🟢 FULLY SUPPORTED (PROPFIND XML parsing, 15s HTTP timeout, paginated listing, allowed-roots path traversal defense, upload, create folder, delete).
> - **Calendar & Contacts:** 🏛️ **NATIVELY DELEGATED TO NEXTCLOUD** (`nextcloud-mcp-control` / `nextcloud-gateway`). Mail.ru does not provide a public CalDAV server and rejects CardDAV PUT requests. Schedule and address book management are maintained via Nextcloud.

---

## 🏛️ Architecture & Highlights

- **Native Go Implementation:** Built with `github.com/mark3labs/mcp-go`, `github.com/emersion/go-imap/v2`, and standard library `net/http`.
- **Zero Interpreter Overhead:** Single self-contained static binary (`~13MB`) replacing Python virtual environment. Memory drops from ~85MB to ~15MB RSS. Cold-start latency drops from ~1.2s to <10ms.
- **Fail-Safe Socket Timeouts:** Every network call (IMAP TLS, SMTP TLS, WebDAV HTTP) enforces an explicit 15-second deadline (`MAILRU_TIMEOUT=15`), eliminating deadlock hazards.
- **Stateless Two-Phase Commit (HITL):** Destructive actions generate cryptographically secure single-use tokens stored in memory with a 3600-second TTL and automatic eviction when capacity (`MAX_PENDING_ACTIONS = 100`) is reached.
- **Workspace Path Traversal Defense:** All file downloads, uploads, and email attachments strictly validate that paths resolve inside designated operational roots (user home, CWD, system temporary directory, or customizable via `MAILRU_ALLOWED_ROOTS`).
- **Fail-Fast Initialization:** Server terminates immediately (`exit 1`) during startup if essential credentials (`MAILRU_USERNAME`, `MAILRU_APP_PASS`) are missing.

---

## 🛠️ MCP Tool Registry (13 Tools)

| Tool Name | Risk / HITL | Parameters | Description |
| :--- | :---: | :--- | :--- |
| `execute_pending_action` | 🟢 Safe | `token` (str) | Executes a staged action blocked by HITL. |
| `mail_read_inbox` | 🟢 Safe | `limit` (int), `folder` (str), `include_smart_folders` (bool) | Fetches latest emails across INBOX and Mail.ru smart folders. |
| `mail_get_body` | 🟢 Safe | `uid` (str), `folder` (str) | Extracts full email text and HTML content by message UID. |
| `mail_send_draft` | 🟢 Safe | `to_email` (str), `subject` (str), `body` (str) | Saves an email draft for human review. |
| `mail_search_thread` | 🟢 Safe | `query` (str), `folder` (str), `limit` (int) | Searches email subject and body across specified folder. |
| `mail_send_reply` | 🔴 **HITL** | `to_email` (str), `subject` (str), `body` (str) | Sends an email reply via SMTP TLS (Port 465). |
| `mail_send_with_attachment`| 🔴 **HITL** | `to_email`, `subject`, `body`, `attachment_path` (str) | Sends email with local attachment via MIME multipart. |
| `mail_move_message` | 🔴 **HITL** | `uid` (str), `to_folder` (str), `from_folder` (str) | Moves message between IMAP folders. |
| `dav_list_dir` | 🟢 Safe | `path` (str), `offset` (int), `limit` (int) | Lists WebDAV directory contents with pagination. |
| `dav_download_file` | 🟢 Safe | `remote_path` (str), `local_path` (str) | Downloads file to local workspace with path containment check. |
| `dav_upload_file` | 🔴 **HITL** | `local_path` (str), `remote_path` (str) | Uploads local file to WebDAV cloud storage. |
| `dav_create_folder` | 🔴 **HITL** | `path` (str) | Creates remote directory in WebDAV cloud storage. |
| `dav_delete_file` | 🔴 **HITL** | `path` (str) | Deletes remote file or directory in WebDAV. |

---

## 🚀 Building & Deployment

### Compilation
```bash
# Build binary
make build
# Or directly with go
go build -v -o mailru-mcp-server .
```

### Configuration (Environment Variables)
```bash
export MAILRU_USERNAME="user@mail.ru"
export MAILRU_APP_PASS="app-specific-password"
export MAILRU_TIMEOUT="15"                # Optional (default: 15 seconds)
export MAILRU_IMAP_HOST="imap.mail.ru"    # Optional (default: imap.mail.ru)
export MAILRU_SMTP_HOST="smtp.mail.ru"    # Optional (default: smtp.mail.ru)
export MAILRU_WEBDAV_HOST="https://webdav.cloud.mail.ru" # Optional
export MAILRU_ALLOWED_ROOTS=""            # Optional (comma-separated allowed roots)
export MAILRU_OPERATOR_NAME="ZavLab"      # Optional (operator confirmation name)
```

### Deployment via `mcp-router` or MCP Client
Example configuration in `config.yaml` or client settings:
```yaml
  mailru:
    transport: stdio
    command: /usr/local/bin/mailru-mcp-server
    env:
      MAILRU_USERNAME: ${MAILRU_USERNAME}
      MAILRU_APP_PASS: ${MAILRU_APP_PASS}
    prefix: mailru__
```

---

## 🧪 Testing & Verification

```bash
# Run unit tests with race detection and coverage
make test

# Run linter
make lint

# Generate coverage breakdown
make coverage
```

---

## 📜 Governance & Standards
- **License:** [MIT License](LICENSE)
- **Contribution Guide:** [CONTRIBUTING.md](CONTRIBUTING.md)
- **Security Policy:** [SECURITY.md](SECURITY.md)
- **Changelog:** [CHANGELOG.md](CHANGELOG.md)
- **Architecture Decisions:** [ARCHITECTURE.md](ARCHITECTURE.md)
