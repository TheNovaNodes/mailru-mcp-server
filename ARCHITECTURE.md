# Architecture Decision Records (ADR) & System Design

## System Topology

```
┌──────────────────────────────────────────────────────────────┐
│                    MCP Client / Gateway                      │
└──────────────────────────────┬───────────────────────────────┘
                               │ JSON-RPC (MCP)
                               ▼
┌──────────────────────────────────────────────────────────────┐
│              MCP Router Gateway / Stdio Runner               │
│                 /etc/mcp-router/config.yaml                  │
└──────────────────────────────┬───────────────────────────────┘
                               │ stdio (JSON-RPC)
                               ▼
┌──────────────────────────────────────────────────────────────┐
│            Mail.ru MCP Server (Go Binary)                    │
│            /usr/local/bin/mailru-mcp-server                  │
├──────────────────────────────┬───────────────────────────────┤
│    internal/mail             │    internal/webdav            │
│  - IMAP TLS (:993)           │  - PROPFIND (XML)             │
│  - SMTP TLS (:465)           │  - GET/PUT/MKCOL/DELETE       │
│  - 15s Socket Deadlines      │  - Path Traversal Containment │
├──────────────────────────────┴───────────────────────────────┤
│    internal/hitl                                             │
│  - Two-Phase Commit Token Engine (3600s sliding TTL)         │
│  - Oldest-eviction bound (Capacity = 100)                    │
└──────────────────────────────────────────────────────────────┘
```

---

## Architecture Decision Records

### ADR-1: Standardized Transport & Router Integration
- **Context:** Individual MCP processes need multiplexing and isolation under `mcp-router` or standard MCP hosts.
- **Decision:** Operate as a standard `stdio` subprocess managed by process supervisors.
- **Status:** Accepted.

### ADR-2: Enforce Human-In-The-Loop (HITL) for Destructive Operations
- **Context:** AI agents must not autonomously send unauthorized emails or delete cloud storage files.
- **Decision:** Two-Phase Commit token workflow. Mutating operations stage action details in RAM and return a token. Execution requires explicit confirmation and calling `execute_pending_action(token)`.
- **Status:** Accepted.

### ADR-3: Amputation of Phantom CalDAV/CardDAV Protocols
- **Context:** Mail.ru does not expose public CalDAV endpoints and rejects CardDAV PUT contact operations.
- **Decision:** Excise phantom modules. Nextcloud (`nextcloud-mcp-control`) serves as the single source of truth for calendars and address books.
- **Status:** Accepted.

### ADR-4: Workspace Path Traversal Containment
- **Context:** Local file operations (`dav_download_file`, `dav_upload_file`, `mail_send_with_attachment`) could be exploited to overwrite or read sensitive host files (`/etc`, `~/.ssh`).
- **Decision:** Strict path validation restricting filesystem operations to authorized roots (user home, CWD, temporary directory, and customizable via `MAILRU_ALLOWED_ROOTS`).
- **Status:** Accepted.

### ADR-5: Fail-Safe Socket Timeout Doctrine
- **Context:** Unresponsive IMAP/SMTP/WebDAV sockets can block child processes and freeze the router.
- **Decision:** Explicit 15-second deadline on all network dials, reads, and writes.
- **Status:** Accepted.

### ADR-6: Full Migration to Go (Production Edition v2.0.0)
- **Context:** The legacy Python implementation suffered from Python interpreter overhead (~85MB RSS RAM per process), slow cold-start latency (~1.2s), virtualenv dependency fragility, and GIL contention.
- **Decision:** Rewrite the entire server in Go using `github.com/mark3labs/mcp-go`, `github.com/emersion/go-imap/v2`, and standard library `net/http`.
- **Consequences:**
  - **Memory:** Memory usage dropped by >80% (from ~85MB to ~15MB RSS).
  - **Cold Start:** Startup time reduced from ~1200ms to <10ms.
  - **Packaging:** Replaced heavy `.venv` with a single self-contained static binary (`13MB`).
  - **Type Safety:** Compile-time verification of all 13 tool inputs and protocol handlers.
  - **Ecosystem Alignment:** Native alignment with Go-based `mcp-router`.
- **Status:** Accepted (v2.0.0).
