# Security Policy

TheNovaNodes Collective takes security and operational integrity with utmost seriousness.

---

## 🔒 Supported Versions

| Version | Supported          | Runtime    | Status |
| ------- | ------------------ | ---------- | ------ |
| 2.x     | :white_check_mark: | Go 1.22+   | Production Current |
| 1.x     | :x:                | Python 3.12| Deprecated |

---

## 🛡️ Security Architecture & Threat Model

### 1. Zero Direct Network Exposure for Autonomous Agents
Autonomous agents never connect directly to IMAP, SMTP, or WebDAV sockets and never handle raw credentials. All communication is mediated via the Model Context Protocol (MCP) through the centralized `mcp-router` gateway.

### 2. Secrets Handling & Credential Isolation
- Credentials (`MAILRU_USERNAME`, `MAILRU_APP_PASS`) are injected strictly via environment variables at process spawn time from `/root/projects/TheNovaNodes/mcp-router/.env` (`chmod 600`, included in `.gitignore`).
- No plaintext credentials, session tokens, or private keys are ever stored in Git repositories, log files, or database caches.
- A fail-fast check terminates the server immediately (`exit 1`) on startup if essential credentials are absent.

### 3. Human-In-The-Loop (HITL) Two-Phase Commit
- Mutating and outbound operations (`mail_send_reply`, `mail_send_with_attachment`, `mail_move_message`, `dav_create_folder`, `dav_upload_file`, `dav_delete_file`) are classified as High Risk (R3/R4).
- Mutating tools **never** execute in a single turn. They generate a single-use token with a sliding 3600-second TTL stored in thread-safe memory (`sync.Mutex`).
- Execution requires explicit human confirmation from ЗавЛаб followed by a call to `execute_pending_action(token)`. Replay attacks are prevented by atomically popping and invalidating tokens upon consumption.

### 4. Path Traversal & Workspace Containment Defense
- The `dav_download_file` tool validates target destinations using `webdav.ValidateDownloadPath()`.
- Downloads are strictly confined to authorized ecosystem roots (`/root/.agents`, `/root/projects`, `/tmp`, and CWD).
- Direct or indirect attempts to escape into root directories (`/etc`, `/root/.ssh`, `/var`, `/proc`) are halted with a security violation.

### 5. Memory Safety & Deadlock Defense
- The Go runtime enforces memory safety, bounds checking, and strong static typing across all tool inputs and network buffers.
- All IMAP, SMTP, and WebDAV socket calls enforce an explicit 15-second deadline (`context.WithTimeout` / `net.Dialer.Timeout`) to prevent hanging TCP states or socket exhaustion.

---

## 🚨 Reporting a Vulnerability

If you discover a security vulnerability within `mailru-mcp-server`:

1. **Do not** open a public issue on GitHub.
2. Report the vulnerability directly to **ЗавЛаб** or via private Telegram channel to the NovaNodes Security Sentinel.
3. Provide:
   - Description of the vulnerability.
   - Steps to reproduce or proof-of-concept payload.
   - Affected tools or endpoints.
4. The security team will triage, patch on a private security branch, verify via unit tests, and deploy following collective review.
