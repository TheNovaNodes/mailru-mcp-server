# 🏛️ Architecture Decision Record (ADR)

**Status:** Approved
**Designers:** Kairos & Manus AI

## 1. Core State (Stateless MCP)
The MCP server is strictly stateless. It does not contain an internal SQLite database for CRM logic. All state (deals, tasks, aliases) is maintained by the upstream agent ecosystem (TheNovaNodes).

## 2. WebDAV Directory Structure
To prevent naming collisions, all client directories follow the `Human Name + UUIDv7` pattern.
Example: `/CRM/ООО_Вектор__c_0192f4c1/`

## 3. Asynchronous IMAP Polling
The MCP server itself does not run background listeners. A separate cron-worker in the agent ecosystem will poll the MCP server's read tools and cache emails to a local database to trigger AI analysis.

## 4. Human-in-the-Loop (HITL) Security
Risk-based isolation:
- **R0-R1:** Safe read operations are allowed fully autonomously.
- **R2-R3:** Mutating operations (`send_email`, `dav_put`, `dav_delete`) require explicit Human-in-the-Loop confirmation in Telegram before execution.
