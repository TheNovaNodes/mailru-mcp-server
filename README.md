# 📧 Mail.ru MCP Server

Stateless Model Context Protocol (MCP) server providing agentic AI access to the Mail.ru ecosystem (IMAP, SMTP, WebDAV, CardDAV).

## 🎯 Purpose
This server acts as a secure, stateless bridge between AI agents (like Kairos/Trickster) and the Mail.ru backend to enable Agentic CRM workflows.

## 🛠️ Planned MCP Tools
- **Mail (IMAP/SMTP):** `search_emails`, `read_email`, `send_email`
- **Cloud (WebDAV):** `create_folder`, `upload_document`, `download_document`
- **Contacts (CardDAV):** `create_contact`, `update_notes`

## 🚀 Quick Start
This project relies on strict security policies. Do NOT use `.env` files for production credentials. See `SECRETS.md` for details.
