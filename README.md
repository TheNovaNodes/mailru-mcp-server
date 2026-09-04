<div align="center">
  <h1>📧 Mail.ru MCP Server (Production Edition)</h1>
  <p><b>A core component of TheNovaNodes Ecosystem</b></p>
  <p>
    <i>Stateless Model Context Protocol (MCP) gateway enabling Agentic AI control via Mail.ru IMAP, SMTP, WebDAV, CalDAV, and CardDAV.</i>
  </p>
  <p>
    <img src="https://img.shields.io/badge/tests-57%20passed-brightgreen.svg" alt="Tests" />
    <img src="https://img.shields.io/badge/coverage-96%25-brightgreen.svg" alt="Coverage" />
    <img src="https://img.shields.io/badge/python-3.12-blue.svg" alt="Python" />
    <img src="https://img.shields.io/badge/protocol-MCP%20JSON--RPC-green.svg" alt="MCP" />
  </p>
</div>

---

## 🧪 Ecosystem Role
This repository serves as a secure, stateless bridge between autonomous AI agents (e.g. Kairos, Trickster, Prometheus) and the Mail.ru infrastructure. It strictly enforces Human-In-The-Loop (HITL) policies via a Two-Phase Commit token system for any data-mutating or outbound actions.

> **✅ CAPABILITIES MATRIX:**
> - **Mail (IMAP / SMTP):** 🟢 FULLY SUPPORTED (Aggregates INBOX & smart folders, draft generation, body extraction).
> - **Cloud Storage (WebDAV):** 🟢 FULLY SUPPORTED (Paginated listing, download, upload, create folder, delete).
> - **Calendar (CalDAV):** 🟢 SUPPORTED (RFC 4791 / Mail.ru CalDAV `calendar_list_events`, `calendar_create_event`).
> - **Contacts (CardDAV):** 🟢 SUPPORTED (RFC 6352 / Mail.ru CardDAV `contact_search`, `contact_create`).

---

## 🏛️ Architecture & Deployment

- **Production Topology:** Operates as a managed `stdio` child backend inside `mcp-router.service` on port `:8090`. Multiplexed and throttled with per-agent ACL.
- **Stateless Proxy:** The MCP server does not persist state in SQLite. Destructive actions generate cryptographically random 8-character tokens stored in volatile RAM for HITL confirmation.
- **Fail-Fast:** Server terminates immediately (`exit 1`) during initialization if essential credentials (`MAILRU_USERNAME`, `MAILRU_APP_PASS`) are missing.
- **Pagination & Safe Reads:** Prevents context window explosion with pagination (`limit` / `offset`), and suppresses `mark_seen` during triage to prevent accidental state changes.

### 🚀 Running the Server

#### Production (via `mcp-router`):
Configured in `/root/projects/TheNovaNodes/mcp-router/config.yaml`:
```yaml
  mailru:
    transport: stdio
    command: /root/projects/TheNovaNodes/mailru-mcp-server/.venv/bin/python
    args:
    - -m
    - src.server
    env:
      PYTHONPATH: /root/projects/TheNovaNodes/mailru-mcp-server
      MAILRU_USERNAME: ${MAILRU_USERNAME}
      MAILRU_APP_PASS: ${MAILRU_APP_PASS}
    prefix: mailru__
```

#### Standalone Execution (for testing or debugging):
```bash
MAILRU_USERNAME="user@mail.ru" MAILRU_APP_PASS="app-password" python -m src.server
```

---

## 🛠️ The Absolute Tool Registry (17 Tools)

### 📧 1. Mail (IMAP / SMTP) Triage & Drafting
| Tool Name | Protocol | Arguments & Defaults | Description | HITL Required |
|-----------|----------|----------------------|-------------|---------------|
| `mail_read_inbox` | IMAP | `limit: int = 10`<br>`folder: str = "INBOX"`<br>`include_smart_folders: bool = True` | Fetches recent incoming emails. Aggregates across INBOX and Mail.ru smart subfolders (`Newsletters`, `Social`, `News`, `Receipts`). Sorts newest first. | ❌ No |
| `mail_get_body` | IMAP | `uid: str`<br>`folder: str = "INBOX"` | Extracts complete text and HTML body by message UID for AI summarization. | ❌ No |
| `mail_search_thread` | IMAP | `query: str`<br>`folder: str = "INBOX"`<br>`limit: int = 20` | Full-text and header search across mailboxes with safe truncation. | ❌ No |
| `mail_send_draft` | IMAP | `to_email: str`<br>`subject: str`<br>`body: str` | Appends a draft directly into `Черновики` / `Drafts` via IMAP for ZavLab's manual review. | ❌ No |
| `mail_move_message` | IMAP | `uid: str`<br>`to_folder: str`<br>`from_folder: str = "INBOX"` | Moves an email by UID to another folder (`Archive`, `SPAM`, `REPLY_REQUIRED`). | 🚨 **YES (HITL)** |
| `mail_send_reply` | SMTP | `to_email: str`<br>`subject: str`<br>`body: str` | Sends direct email reply via SMTP. | 🚨 **YES (HITL)** |
| `mail_send_with_attachment` | SMTP | `to_email: str`<br>`subject: str`<br>`body: str`<br>`attachment_path: str` | Sends email with attachment retrieved from WebDAV/local filesystem. | 🚨 **YES (HITL)** |

### 🗂 2. Cloud Storage (WebDAV)
| Tool Name | Protocol | Arguments & Defaults | Description | HITL Required |
|-----------|----------|----------------------|-------------|---------------|
| `dav_list_dir` | WebDAV | `path: str = "/"`<br>`offset: int = 0`<br>`limit: int = 50` | Paginated directory and file metadata listing. | ❌ No |
| `dav_download_file` | WebDAV | `remote_path: str`<br>`local_path: str` | Downloads remote file to agent local workspace. Strictly validates against Path Traversal. | ❌ No |
| `dav_create_folder` | WebDAV | `path: str` | Scaffolds a new directory on Cloud Mail.ru. | 🚨 **YES (HITL)** |
| `dav_upload_file` | WebDAV | `local_path: str`<br>`remote_path: str` | Uploads local documents or invoices to WebDAV. | 🚨 **YES (HITL)** |
| `dav_delete_file` | WebDAV | `path: str` | Deletes remote documents or directories. | 🚨 **YES (HITL)** |

### 📅 3. Calendar & Scheduling (CalDAV)
| Tool Name | Protocol | Arguments & Defaults | Description | HITL Required |
|-----------|----------|----------------------|-------------|---------------|
| `calendar_list_events` | CalDAV | `days_ahead: int = 7` | Reads upcoming events via CalDAV `REPORT` to check availability and avoid collisions. | ❌ No |
| `calendar_create_event` | CalDAV | `title: str`<br>`start_iso: str`<br>`end_iso: str` | Proposes and schedules a meeting on Mail.ru Calendar. | 🚨 **YES (HITL)** |

### 👥 4. Address Book & Contacts (CardDAV)
| Tool Name | Protocol | Arguments & Defaults | Description | HITL Required |
|-----------|----------|----------------------|-------------|---------------|
| `contact_search` | CardDAV | `query: str` | Searches address book for client vCards by name or email via CardDAV `REPORT`. | ❌ No |
| `contact_create` | CardDAV | `name: str`<br>`email: str`<br>`phone: str = ""` | Creates a new contact entry in the Mail.ru Address Book via CardDAV `PUT`. | 🚨 **YES (HITL)** |

### 🛡️ 5. Two-Phase Commit Execution
| Tool Name | Protocol | Arguments & Defaults | Description | HITL Required |
|-----------|----------|----------------------|-------------|---------------|
| `execute_pending_action` | Internal | `token: str` | Validates one-time token and executes previously blocked mutating action. | ❌ No (Requires Token) |

---

## 🛡️ Human-In-The-Loop (HITL) Workflow

When an autonomous agent invokes a mutating tool (e.g. `mail_send_reply`), the server stages the request in memory and blocks execution:

```text
🚨 ACTION BLOCKED (HITL REQUIRED) 🚨
Type: mail_send
Details: {'to': 'client@example.com', 'subject': 'Agreement', 'body': '...'}
To execute, you MUST call `execute_pending_action` with token: a1b2c3d4
```

1. The agent presents the details to ZaVLab for review.
2. Upon explicit human approval, the agent executes:
   ```python
   execute_pending_action(token="a1b2c3d4")
   ```
3. The server executes the action, invalidates the token, and returns the result. Tokens are single-use and cannot be replayed.

---

## 🧪 Testing & Quality Assurance

The test suite covers unit tests, mock integration, timezone normalization, path traversal protection, and error paths:

```bash
# Run all tests
python -m unittest discover tests -v

# Run coverage analysis
coverage run -m unittest discover tests
coverage report -m
```

Target standard: **$\ge 90\%$ line coverage across all modules**.

---

## 🤝 Contributing (Agent Doctrine)
- **Workflow:** GitHub Flow. Direct pushes to `master` strictly forbidden.
- **Pre-commit:** Verify changes with `python -m py_compile src/*.py` and `python -m unittest discover tests`.
- **Pre-push:** Create dedicated branches (`feat/*`, `fix/*`), submit PR, and verify cloud CI status (`gh pr checks`).

