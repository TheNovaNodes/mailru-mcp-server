<div align="center">
  <h1>📧 Mail.ru MCP Server (God Mode Edition)</h1>
  <p><b>A core component of TheNovaNodes Ecosystem</b></p>
  <p>
    <i>Stateless Model Context Protocol (MCP) gateway enabling absolute Agentic AI control via Mail.ru IMAP, SMTP, WebDAV, CardDAV, and ActiveSync.</i>
  </p>
</div>

---

## 🧪 Ecosystem Role
This repository serves as a secure, stateless bridge between autonomous AI agents (Kairos, Trickster) and the entire Mail.ru infrastructure. It strictly enforces Human-In-The-Loop (HITL) policies for data-mutating actions, transforming a standard mailbox into a fully automated, agentic CRM and Calendar orchestrator.

## 🏛️ Architecture & Storage
- **Stateless Proxy:** The MCP server does not store CRM state.
- **Collision-Free Filesystem:** WebDAV folders strictly follow the `HumanName__c_UUID` pattern.
- **Asynchronous Polling:** Email ingestion is handled via background pollers.

## 🔐 Security & Vault Protocol (CRITICAL)
**DO NOT USE `.env` FILES FOR SECRETS.** 
Execution Wrapper:
```bash
with-secret <VAULT_POINTER> --env MAILRU_APP_PASS -- python src/server.py
```

## 🛠️ The Absolute Tool Registry
The following represents the exhaustive mapping of MCP tools to Mail.ru protocols, designed for maximum AI autonomy within safe boundaries.

### 📧 Mail (IMAP / SMTP)
| Tool Name | Protocol | Description | HITL Required |
|-----------|----------|-------------|---------------|
| `mail_read_inbox` | IMAP | Fetch latest unread emails and threads. | ❌ No |
| `mail_search_thread` | IMAP | Semantic and text search across mailboxes. | ❌ No |
| `mail_get_body` | IMAP | Extract and decode raw MIME content / attachments. | ❌ No |
| `mail_mark_read` | IMAP | Mark emails as read/important. | ❌ No |
| `mail_move_folder` | IMAP | Move emails to archive or client-specific folders. | ⚠️ Yes (If bulk) |
| `mail_send_draft` | SMTP | Save a generated reply into the Drafts folder. | ❌ No |
| `mail_send_reply` | SMTP | Send a direct reply to a client. | 🚨 **YES (R3)** |
| `mail_send_with_attachment` | SMTP | Send email with files retrieved from WebDAV. | 🚨 **YES (R3)** |

### 🗂 Cloud Storage (WebDAV)
| Tool Name | Protocol | Description | HITL Required |
|-----------|----------|-------------|---------------|
| `dav_list_dir` | WebDAV | Read CRM folder structures and file metadata. | ❌ No |
| `dav_create_folder` | WebDAV | Scaffold a new client directory (`Name__c_UUID`). | ⚠️ Yes (R2) |
| `dav_upload_file` | WebDAV | Upload documents, invoices, or briefs. | ⚠️ Yes (R2) |
| `dav_download_file` | WebDAV | Download documents to agent memory. | ❌ No |
| `dav_move_file` | WebDAV | Move/Rename files across the CRM filesystem. | ⚠️ Yes (R2) |
| `dav_delete_file` | WebDAV | Soft/Hard delete documents. | 🚨 **YES (R4)** |

### 👥 CRM Contacts (CardDAV)
| Tool Name | Protocol | Description | HITL Required |
|-----------|----------|-------------|---------------|
| `contact_search` | CardDAV | Search for client vCard by email or name. | ❌ No |
| `contact_create` | CardDAV | Create a new lead in the address book. | ⚠️ Yes (R2) |
| `contact_update` | CardDAV | Append CRM stage or UUID to contact notes. | ⚠️ Yes (R2) |

### 📅 Scheduling & Time (ActiveSync)
| Tool Name | Protocol | Description | HITL Required |
|-----------|----------|-------------|---------------|
| `calendar_list_events` | EAS | Read ZavLab's schedule to avoid double-booking. | ❌ No |
| `calendar_create_event`| EAS | Propose and schedule a meeting with a client. | 🚨 **YES (R3)** |
| `calendar_cancel_event`| EAS | Cancel or reschedule an existing event. | 🚨 **YES (R4)** |

## 🤝 Contributing (Agent Doctrine)
- **Workflow:** GitHub Flow. Direct pushes to `master` banned.
- **AI Delegation:** Work executed on branches prefixed with `jules/` or `agent/`.
