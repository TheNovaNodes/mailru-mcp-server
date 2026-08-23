<div align="center">
  <h1>📧 Mail.ru MCP Server</h1>
  <p><b>A core component of TheNovaNodes Ecosystem</b></p>
  <p>
    <i>Stateless Model Context Protocol (MCP) gateway enabling Agentic AI workflows via Mail.ru IMAP, WebDAV, and CardDAV.</i>
  </p>
</div>

---

## 🧪 Ecosystem Role
This repository is part of **TheNovaNodes** ecosystem. It serves as a secure, stateless bridge between autonomous AI agents (such as Kairos, Trickster, and Nova) and the Mail.ru backend. It strictly enforces Human-In-The-Loop (HITL) policies for all data-mutating actions.

## 🏛️ Architecture & Storage
Based on the architectural audit by Manus AI, this server adheres to a **Stateless Design Pattern**:
- **Stateless Proxy:** The MCP server does not store CRM state (Deals, Clients, Funnels). All state is orchestrated by TheNovaNodes core database.
- **Collision-Free Filesystem:** WebDAV folders strictly follow the `HumanName__c_UUID` pattern (e.g., `/CRM/Vector_LLC__c_0192f4c1/`).
- **Asynchronous Polling:** Email ingestion is handled via background pollers, triggering AI reasoning cycles only upon new artifact delivery.

## 🔐 Security & Vault Protocol (CRITICAL)
**DO NOT USE `.env` FILES FOR SECRETS.** 
Following the ecosystem's Secret Management Law, this server requires memory-only injection of the Mail.ru App Password via the `agent-vault` architecture.

### Execution Wrapper:
```bash
with-secret <VAULT_POINTER> --env MAILRU_APP_PASS -- python src/server.py
```

## 🛠️ MCP Tools Exposed
| Module | Tool | Description | HITL Required |
|--------|------|-------------|---------------|
| **IMAP** | `search_emails` | Find emails by semantic/text query | ❌ No |
| **IMAP** | `fetch_recent_emails`| Read latest inbox artifacts | ❌ No |
| **WebDAV**| `list_cloud_directory`| Read CRM folder structures | ❌ No |
| **WebDAV**| `create_crm_client_folder`| Scaffold a new client directory | ⚠️ Yes |

## 🤝 Contributing (Agent Doctrine)
- **Workflow:** GitHub Flow is mandatory. Direct pushes to the `master` / `main` branch are strictly banned.
- **Commits:** Conventional Commits only (`feat:`, `fix:`, `docs:`, `refactor:`).
- **AI Delegation:** All work delegated to AI agents must be executed on branches prefixed with `jules/` or `agent/`.
- **Merge Policy:** Squash and merge is mandatory.
