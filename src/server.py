import os
import sys
import uuid
import time
from typing import Dict, Any
try:
    from mcp.server.fastmcp import FastMCP
except ImportError:
    from fastmcp import FastMCP
from pydantic import BaseModel, Field

from src.mailru_client import MailRuClient
from src.webdav_client import WebDAVClient

# ==========================================
# 🛑 FAIL-FAST INITIALIZATION
# ==========================================
# If credentials are missing, we crash immediately. No silent failures.
try:
    mail_client = MailRuClient()
    dav_client = WebDAVClient()
except ValueError as e:
    print(f"CRITICAL STARTUP FAILURE: {e}", file=sys.stderr)
    sys.exit(1)

mcp = FastMCP("Mail.ru CRM Gateway")

# ==========================================
# 🛡️ HARDCORE HITL (Two-Phase Commit with TTL)
# ==========================================
PENDING_ACTIONS: Dict[str, Any] = {}
HITL_TTL_SECONDS = 3600  # 1 hour TTL
MAX_PENDING_ACTIONS = 100

def _cleanup_expired_actions() -> None:
    """Prune expired HITL actions from memory to prevent unbounded leaks."""
    now = time.time()
    expired = [t for t, data in PENDING_ACTIONS.items() if now - data.get("created_at", 0) > HITL_TTL_SECONDS]
    for t in expired:
        PENDING_ACTIONS.pop(t, None)

def request_hitl(action_type: str, details: dict) -> str:
    """Stage a destructive action and demand a confirmation token."""
    _cleanup_expired_actions()
    if len(PENDING_ACTIONS) >= MAX_PENDING_ACTIONS:
        oldest_token = min(PENDING_ACTIONS.keys(), key=lambda k: PENDING_ACTIONS[k].get("created_at", 0))
        PENDING_ACTIONS.pop(oldest_token, None)

    token = str(uuid.uuid4())[:8]
    PENDING_ACTIONS[token] = {
        "type": action_type,
        "details": details,
        "created_at": time.time()
    }
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
    _cleanup_expired_actions()
    if token not in PENDING_ACTIONS:
        return "❌ Error: Invalid, expired, or already executed HITL token."
    
    action = PENDING_ACTIONS.pop(token)
    
    try:
        if action["type"] == "mail_send":
            d = action["details"]
            mail_client.send_email(d["to"], d["subject"], d["body"], d.get("attachment"))
            return f"✅ Action Executed: Email sent to {d['to']}."

        elif action["type"] == "mail_move_message":
            d = action["details"]
            mail_client.move_message(d["uid"], d["to_folder"], d.get("from_folder", "INBOX"))
            return f"✅ Action Executed: Message {d['uid']} moved to {d['to_folder']}."
        
        elif action["type"] == "dav_delete":
            dav_client.delete(action["details"]["path"])
            return f"✅ Action Executed: {action['details']['path']} deleted."
            
        elif action["type"] == "dav_upload":
            dav_client.upload_file(action["details"]["local_path"], action["details"]["remote_path"])
            return f"✅ Action Executed: File uploaded to {action['details']['remote_path']}."
            
        elif action["type"] == "dav_create_folder":
            dav_client.create_directory(action["details"]["path"])
            return f"✅ Action Executed: Folder {action['details']['path']} created."
            
        else:
            return f"❌ Unknown action type: {action['type']}"
            
    except Exception as e:
        return f"❌ Execution failed: {e}"

# ==========================================
# 📧 Mail (IMAP / SMTP) Tools
# ==========================================

@mcp.tool()
def mail_read_inbox(limit: int = 10, folder: str = "INBOX", include_smart_folders: bool = True) -> str:
    """
    Fetch latest emails and threads.
    If folder is INBOX and include_smart_folders is True, aggregates across INBOX and
    Mail.ru smart subfolders (Newsletters, Social, News, Receipts) sorted newest first.
    """
    safe_limit = min(limit, 50)
    try:
        emails = mail_client.fetch_recent_emails(safe_limit, folder, include_smart_folders=include_smart_folders)
        if not emails:
            return "No emails found."
        
        result = []
        for e in emails:
            flags = ", ".join(e['flags'])
            fld_str = f" | Folder: {e['folder']}" if 'folder' in e else ""
            result.append(f"UID: {e['uid']}{fld_str} | From: {e['from']} | Subject: {e['subject']}\nDate: {e['date']} | Flags: {flags}\nSnippet: {e['text'][:200]}...")
        return "\n\n".join(result)
    except Exception as e:
        return f"Error reading inbox: {e}"

@mcp.tool()
def mail_get_body(uid: str, folder: str = "INBOX") -> str:
    """Extract full email text and html content by UID for AI summarization and analysis."""
    try:
        details = mail_client.get_email_body(uid, folder)
        if not details:
            return f"Email with UID {uid} not found in folder {folder}."
        
        content = f"UID: {details['uid']} | Folder: {details['folder']}\n"
        content += f"From: {details['from']} | To: {details['to']}\n"
        content += f"Subject: {details['subject']}\nDate: {details['date']}\n"
        content += "-" * 40 + "\n"
        body = details['text'] if details['text'] else details['html']
        content += body
        return content
    except Exception as e:
        return f"Error getting email body: {e}"

@mcp.tool()
def mail_send_draft(to_email: str, subject: str, body: str) -> str:
    """Save an email draft for ZavLab to review (No HITL required)."""
    try:
        success = mail_client.save_draft(to_email, subject, body)
        if success:
            return f"✅ Draft to '{to_email}' saved successfully in Drafts/Черновики."
        return "❌ Failed to save draft."
    except Exception as e:
        return f"Error saving draft: {e}"

@mcp.tool()
def mail_move_message(uid: str, to_folder: str, from_folder: str = "INBOX") -> str:
    """Move an email to another folder like 'Archive', 'SPAM', or 'REPLY_REQUIRED' (HITL protected)."""
    return request_hitl("mail_move_message", {"uid": uid, "to_folder": to_folder, "from_folder": from_folder})

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

ALLOWED_DOWNLOAD_ROOTS = [
    "/root/.agents",
    "/root/projects",
    "/tmp"
]

@mcp.tool()
def dav_download_file(remote_path: str, local_path: str) -> str:
    """Download documents to agent workspace or local storage. Safe read operation."""
    # Prevent Path Traversal outside designated workspace directories
    target_path = os.path.abspath(local_path)
    is_allowed = any(
        os.path.commonpath([root, target_path]) == root
        for root in ALLOWED_DOWNLOAD_ROOTS
    )
    if not is_allowed:
        return f"❌ Security Error: Path traversal detected. Downloads are restricted to: {', '.join(ALLOWED_DOWNLOAD_ROOTS)}"
    try:
        parent = os.path.dirname(target_path)
        if parent:
            os.makedirs(parent, exist_ok=True)
        dav_client.download_file(remote_path, target_path)
        return f"✅ File downloaded to {local_path} successfully."
    except Exception as e:
        return f"❌ Download failed: {e}"

@mcp.tool()
def dav_delete_file(path: str) -> str:
    """Delete documents or folders. (HITL protected)."""
    return request_hitl("dav_delete", {"path": path})

if __name__ == "__main__":
    mcp.run(transport="stdio")
