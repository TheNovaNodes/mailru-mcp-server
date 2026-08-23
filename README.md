<div align="center">
  <h1>📧 Mail.ru MCP Server (God Mode Edition)</h1>
  <p><b>A core component of TheNovaNodes Ecosystem</b></p>
  <p>
    <i>Stateless Model Context Protocol (MCP) gateway enabling Agentic AI control via Mail.ru IMAP, SMTP, and WebDAV.</i>
  </p>
</div>

---

## 🧪 Ecosystem Role
This repository serves as a secure, stateless bridge between autonomous AI agents (Kairos, Trickster) and the Mail.ru infrastructure. It strictly enforces Human-In-The-Loop (HITL) policies via a Two-Phase Commit system for data-mutating actions.

> **⚠️ CAPABILITIES NOTICE:**
> - **Mail (IMAP/SMTP):** 🟢 FULLY SUPPORTED (Production)
> - **Cloud Storage (WebDAV):** 🟢 FULLY SUPPORTED (Production)
> - **Contacts (CardDAV):** ❌ NOT SUPPORTED in `master` (Due to proprietary Mail.ru API quirks. See branch `agent/trickster-dav-expansion` for experimental RFC-compliant client).
> - **Calendar (CalDAV/EAS):** ❌ NOT SUPPORTED in `master`.

## 🏛️ Architecture & Storage
- **Stateless Proxy:** The MCP server does not store CRM state. Destructive actions generate UUID tokens stored in RAM for HITL verification.
- **Fail-Fast:** Server terminates immediately (`exit 1`) if credentials are missing.
- **Pagination:** Context blowouts are prevented by hard limits on directory listing and email fetching.

## 🔐 Security & Vault Protocol (CRITICAL)
**DO NOT USE `.env` FILES FOR SECRETS.** 
Execution Wrapper:
```bash
with-secret <VAULT_POINTER> --env MAILRU_APP_PASS -- python src/server.py
```

## 🛠️ The Absolute Tool Registry (Production Ready)

### 📧 Mail (IMAP / SMTP)
| Tool Name | Protocol | Description | HITL Required |
|-----------|----------|-------------|---------------|
| `mail_read_inbox` | IMAP | Fetch latest unread emails and threads (Paginated). | ❌ No |
| `mail_search_thread` | IMAP | Semantic and text search across mailboxes. | ❌ No |
| `mail_send_draft` | SMTP | Save a generated reply into the Drafts folder. | ❌ No |
| `mail_send_reply` | SMTP | Send a direct reply to a client. | 🚨 **YES** |
| `mail_send_with_attachment` | SMTP | Send email with files retrieved from WebDAV. | 🚨 **YES** |

### 🗂 Cloud Storage (WebDAV)
| Tool Name | Protocol | Description | HITL Required |
|-----------|----------|-------------|---------------|
| `dav_list_dir` | WebDAV | Read CRM folder structures and file metadata (Paginated). | ❌ No |
| `dav_create_folder` | WebDAV | Scaffold a new client directory. | 🚨 **YES** |
| `dav_upload_file` | WebDAV | Upload documents, invoices, or briefs. | 🚨 **YES** |
| `dav_download_file` | WebDAV | Download documents to agent memory. | ❌ No |
| `dav_delete_file` | WebDAV | Soft/Hard delete documents. | 🚨 **YES** |
| `execute_pending_action`| Internal | Confirms and executes any HITL blocked action. | ❌ No (Agent must have token) |
