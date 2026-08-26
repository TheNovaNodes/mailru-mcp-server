import sys
import uuid
from typing import Dict, Any
from mcp.server.fastmcp import FastMCP
from pydantic import BaseModel, Field

from src.mailru_client import MailRuClient
from src.webdav_client import WebDAVClient
from src.caldav_client import CalDAVClient
from src.carddav_client import CardDAVClient

# ==========================================
# 🛑 FAIL-FAST INITIALIZATION
# ==========================================
# If credentials are missing, we crash immediately. No silent failures.
try:
    mail_client = MailRuClient()
    dav_client = WebDAVClient()
    caldav_client = CalDAVClient()
    carddav_client = CardDAVClient()
except ValueError as e:
    print(f"CRITICAL STARTUP FAILURE: {e}", file=sys.stderr)
    sys.exit(1)

mcp = FastMCP("Mail.ru CRM Gateway", dependencies=["imap-tools", "webdavclient3", "requests"])

# ==========================================
# 🛡️ HARDCORE HITL (Two-Phase Commit)
# ==========================================
PENDING_ACTIONS: Dict[str, Any] = {}

def request_hitl(action_type: str, details: dict) -> str:
    """Stage a destructive action and demand a confirmation token."""
    token = str(uuid.uuid4())[:8]
    PENDING_ACTIONS[token] = {"type": action_type, "details": details}
    return (
        f"🚨 ACTION BLOCKED (HITL REQUIRED) 🚨\n"
        f"Type: {action_type}\n"
        f"Details: {details}\n"
        f"To execute, you MUST call `execute_pending_action` with token: {token}"
    )

@mcp.tool()
def execute_pending_action(token: str) -> str:
    """
    Execute a destructive action that was previously blocked by HITL.
    The agent must retrieve the token from the blocked action response.
    """
    if token not in PENDING_ACTIONS:
        return "❌ Error: Invalid, expired, or already executed HITL token."
    
    action = PENDING_ACTIONS.pop(token)
    
    try:
        if action["type"] == "mail_send":
            d = action["details"]
            mail_client.send_email(d["to"], d["subject"], d["body"], d.get("attachment"))
            return f"✅ Action Executed: Email sent to {d['to']}."
        
        elif action["type"] == "dav_delete":
            dav_client.delete(action["details"]["path"])
            return f"✅ Action Executed: {action['details']['path']} deleted."
            
        elif action["type"] == "dav_upload":
            dav_client.upload_file(action["details"]["local_path"], action["details"]["remote_path"])
            return f"✅ Action Executed: File uploaded to {action['details']['remote_path']}."
            
        elif action["type"] == "dav_create_folder":
            dav_client.create_directory(action["details"]["path"])
            return f"✅ Action Executed: Folder {action['details']['path']} created."
            
        elif action["type"] == "calendar_create_event":
            d = action["details"]
            caldav_client.create_event(d["title"], d["start_iso"], d["end_iso"])
            return f"✅ Action Executed: Event '{d['title']}' scheduled."

        elif action["type"] == "contact_create":
            d = action["details"]
            carddav_client.create_contact(d["name"], d["email"], d.get("phone"))
            return f"✅ Action Executed: Contact '{d['name']}' saved to Address Book."
            
        else:
            return f"❌ Unknown action type: {action['type']}"
            
    except Exception as e:
        return f"❌ Execution failed: {e}"

# ==========================================
# 📧 Mail (IMAP / SMTP) Tools
# ==========================================

@mcp.tool()
def mail_read_inbox(limit: int = 10, folder: str = "INBOX") -> str:
    """Fetch latest unread emails and threads."""
    safe_limit = min(limit, 50)
    try:
        emails = mail_client.fetch_recent_emails(safe_limit, folder)
        if not emails:
            return "No emails found."
        
        result = []
        for e in emails:
            flags = ", ".join(e['flags'])
            result.append(f"UID: {e['uid']} | From: {e['from']} | Subject: {e['subject']}\nDate: {e['date']} | Flags: {flags}\nSnippet: {e['text'][:200]}...")
        return "\n\n".join(result)
    except Exception as e:
        return f"Error reading inbox: {e}"

@mcp.tool()
def mail_search_thread(query: str, folder: str = "INBOX", limit: int = 20) -> str:
    """Semantic and text search across mailboxes."""
    safe_limit = min(limit, 50)
    try:
        emails = mail_client.search_emails(query, folder)
        if not emails:
            return "No emails found matching query."
        
        result = []
        for e in emails[:safe_limit]:
            result.append(f"UID: {e['uid']} | From: {e['from']} | Subject: {e['subject']}\nDate: {e['date']}\nSnippet: {e['text_snippet']}")
        
        out = "\n\n".join(result)
        if len(emails) > safe_limit:
            out += f"\n\n... (Truncated. Found {len(emails)} emails, showing first {safe_limit}. Be more specific!)"
        return out
    except Exception as e:
        return f"Error searching emails: {e}"

@mcp.tool()
def mail_send_reply(to_email: str, subject: str, body: str) -> str:
    """Send a direct reply to a client (HITL protected)."""
    return request_hitl("mail_send", {"to": to_email, "subject": subject, "body": body})

@mcp.tool()
def mail_send_with_attachment(to_email: str, subject: str, body: str, attachment_path: str) -> str:
    """Send email with files retrieved from WebDAV/local storage (HITL protected)."""
    return request_hitl("mail_send", {"to": to_email, "subject": subject, "body": body, "attachment": attachment_path})

# ==========================================
# 🗂 Cloud Storage (WebDAV) Tools
# ==========================================

@mcp.tool()
def dav_list_dir(path: str = "/", offset: int = 0, limit: int = 50) -> str:
    """Read CRM folder structures and file metadata. Uses pagination."""
    try:
        contents = dav_client.list_directory(path)
        total_files = len(contents)
        
        safe_limit = min(limit, 100)
        paginated_contents = contents[offset:offset+safe_limit]
        
        header = f"📁 Directory: {path} (Showing {offset} to {offset+len(paginated_contents)} of {total_files} total items)\n"
        header += "-"*40 + "\n"
        
        if total_files == 0:
            return header + "Directory is empty."
            
        return header + "\n".join(paginated_contents)
    except Exception as e:
        return f"WebDAV list failed: {e}"

@mcp.tool()
def dav_create_folder(path: str) -> str:
    """Scaffold a new client directory (HITL protected)."""
    return request_hitl("dav_create_folder", {"path": path})

@mcp.tool()
def dav_upload_file(local_path: str, remote_path: str) -> str:
    """Upload documents to WebDAV (HITL protected)."""
    return request_hitl("dav_upload", {"local_path": local_path, "remote_path": remote_path})

@mcp.tool()
def dav_download_file(remote_path: str, local_path: str) -> str:
    """Download documents to agent memory (local storage). Safe read operation."""
    try:
        dav_client.download_file(remote_path, local_path)
        return f"✅ File downloaded to {local_path} successfully."
    except Exception as e:
        return f"❌ Download failed: {e}"

@mcp.tool()
def dav_delete_file(path: str) -> str:
    """Delete documents or folders. (HITL protected)."""
    return request_hitl("dav_delete", {"path": path})

# ==========================================
# 📅 Calendar & Contacts (CalDAV / CardDAV)
# ==========================================

@mcp.tool()
def calendar_list_events(days_ahead: int = 7) -> str:
    """Read ZavLab's schedule to avoid double-booking."""
    try:
        events = caldav_client.list_events(days_ahead)
        return "\n\n".join(events)
    except Exception as e:
        return f"Failed to fetch calendar events: {e}"

@mcp.tool()
def calendar_create_event(title: str, start_iso: str, end_iso: str) -> str:
    """Propose and schedule a meeting with a client (HITL protected)."""
    return request_hitl("calendar_create_event", {"title": title, "start_iso": start_iso, "end_iso": end_iso})

@mcp.tool()
def contact_search(query: str) -> str:
    """Search for client vCard by email or name."""
    try:
        contacts = carddav_client.search_contacts(query)
        res = []
        for c in contacts:
            if "error" in c: res.append(f"Error: {c['error']}")
            elif "info" in c: res.append(c["info"])
            else: res.append(c["vcard"])
        return "\n\n".join(res)
    except Exception as e:
        return f"Failed to search contacts: {e}"

@mcp.tool()
def contact_create(name: str, email: str, phone: str = "") -> str:
    """Create a new lead in the address book (HITL protected)."""
    return request_hitl("contact_create", {"name": name, "email": email, "phone": phone})

if __name__ == "__main__":
    mcp.run(transport="stdio")
