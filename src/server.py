from mcp.server.fastmcp import FastMCP
from typing import List
import os
import sys

sys.path.append(os.path.dirname(os.path.abspath(__file__)))
from mailru_client import MailRuClient
from webdav_client import WebDAVClient

mcp = FastMCP("Mail.ru CRM Gateway - God Mode", dependencies=["imap-tools", "webdavclient3", "pydantic"])

def get_mail() -> MailRuClient: return MailRuClient()
def get_dav() -> WebDAVClient: return WebDAVClient()

# ==========================================
# 📧 1. IMAP / SMTP Tools (Mail)
# ==========================================
@mcp.tool()
def mail_read_inbox(limit: int = 10) -> str:
    """Fetch latest unread emails and threads."""
    return get_mail().fetch_recent_emails(limit)

@mcp.tool()
def mail_search_thread(query: str) -> str:
    """Semantic and text search across mailboxes."""
    return get_mail().search_emails(query)

@mcp.tool()
def mail_get_body(uid: str) -> str:
    """Extract and decode raw MIME content / attachments."""
    return f"NOT IMPLEMENTED YET: Fetching body for UID {uid}"

@mcp.tool()
def mail_mark_read(uid: str) -> str:
    """Mark emails as read/important."""
    return f"NOT IMPLEMENTED YET: Marked {uid} as read"

@mcp.tool()
def mail_move_folder(uid: str, folder: str) -> str:
    """Move emails to archive or client-specific folders."""
    return f"NOT IMPLEMENTED YET: Moved {uid} to {folder}"

@mcp.tool()
def mail_send_reply(to: str, subject: str, body: str) -> str:
    """Send a direct reply to a client. (Requires HITL)"""
    return f"HITL REQUIRED: Ready to send email to {to}."

@mcp.tool()
def mail_send_with_attachment(to: str, subject: str, body: str, dav_file_path: str) -> str:
    """Send email with files retrieved from WebDAV. (Requires HITL)"""
    return f"HITL REQUIRED: Ready to send email to {to} with attachment {dav_file_path}."

# ==========================================
# 🗂 2. WebDAV Tools (Cloud CRM)
# ==========================================
@mcp.tool()
def dav_list_dir(path: str = "/") -> str:
    """Read CRM folder structures and file metadata."""
    try:
        items = get_dav().list_directory(path)
        return f"Contents of {path}:\n" + "\n".join([f"- {i}" for i in items])
    except Exception as e:
        return f"Error: {e}"

@mcp.tool()
def dav_create_folder(path: str) -> str:
    """Scaffold a new client directory. (Requires HITL)"""
    try:
        get_dav().create_directory(path)
        return f"SUCCESS: Created folder {path}"
    except Exception as e:
        return f"Error: {e}"

@mcp.tool()
def dav_upload_file(local_path: str, remote_path: str) -> str:
    """Upload documents, invoices, or briefs. (Requires HITL)"""
    return f"NOT IMPLEMENTED YET: Upload {local_path} to {remote_path}"

@mcp.tool()
def dav_download_file(remote_path: str, local_path: str) -> str:
    """Download documents to agent memory."""
    return f"NOT IMPLEMENTED YET: Download {remote_path} to {local_path}"

@mcp.tool()
def dav_move_file(source: str, dest: str) -> str:
    """Move/Rename files across the CRM filesystem. (Requires HITL)"""
    return f"NOT IMPLEMENTED YET: Moved {source} to {dest}"

@mcp.tool()
def dav_delete_file(path: str) -> str:
    """Soft/Hard delete documents. (Requires HITL)"""
    return f"🚨 DANGER HITL REQUIRED: Ready to delete {path}"

# ==========================================
# 👥 3. CardDAV Tools (Contacts)
# ==========================================
@mcp.tool()
def contact_search(query: str) -> str:
    """Search for client vCard by email or name."""
    return f"NOT IMPLEMENTED YET: Search contact {query}"

@mcp.tool()
def contact_create(name: str, email: str, phone: str) -> str:
    """Create a new lead in the address book. (Requires HITL)"""
    return f"NOT IMPLEMENTED YET: Created contact {name}"

# ==========================================
# 📅 4. ActiveSync Tools (Calendar)
# ==========================================
@mcp.tool()
def calendar_list_events(days_ahead: int = 7) -> str:
    """Read ZavLab's schedule to avoid double-booking."""
    return f"NOT IMPLEMENTED YET: Fetched schedule for {days_ahead} days"

@mcp.tool()
def calendar_create_event(title: str, start_iso: str, end_iso: str) -> str:
    """Propose and schedule a meeting with a client. (Requires HITL)"""
    return f"HITL REQUIRED: Ready to schedule {title} at {start_iso}"

if __name__ == "__main__":
    mcp.run()
