import sys
from mcp.server.fastmcp import FastMCP
from pydantic import BaseModel, Field

from src.mailru_client import MailRuClient
from src.webdav_client import WebDAVClient

# Create the MCP Server
mcp = FastMCP("Mail.ru CRM Gateway", dependencies=["imap-tools", "webdavclient3"])

try:
    mail_client = MailRuClient()
    dav_client = WebDAVClient()
except ValueError as e:
    # We allow the server to start, but it might fail on execution if env vars are missing.
    # In a real vault environment, they are injected before runtime.
    pass

# ==========================================
# 📧 Mail (IMAP / SMTP) Tools
# ==========================================

@mcp.tool()
def mail_read_inbox(limit: int = 10, folder: str = "INBOX") -> str:
    """
    Fetch latest unread emails and threads.
    Safe read operation (R0-R1).
    """
    try:
        emails = mail_client.fetch_recent_emails(limit, folder)
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
def mail_search_thread(query: str, folder: str = "INBOX") -> str:
    """
    Semantic and text search across mailboxes.
    Safe read operation (R0-R1).
    """
    try:
        emails = mail_client.search_emails(query, folder)
        if not emails:
            return "No emails found matching query."
        
        result = []
        for e in emails:
            result.append(f"UID: {e['uid']} | From: {e['from']} | Subject: {e['subject']}\nDate: {e['date']}\nSnippet: {e['text_snippet']}")
        return "\n\n".join(result)
    except Exception as e:
        return f"Error searching emails: {e}"

@mcp.tool()
def mail_send_reply(to_email: str, subject: str, body: str) -> str:
    """
    Send a direct reply to a client.
    🚨 HITL REQUIRED (R3): You MUST confirm this action with the user before executing.
    """
    try:
        mail_client.send_email(to_email, subject, body)
        return f"Email sent successfully to {to_email} with subject '{subject}'."
    except Exception as e:
        return f"Failed to send email: {e}"

@mcp.tool()
def mail_send_with_attachment(to_email: str, subject: str, body: str, attachment_path: str) -> str:
    """
    Send email with files retrieved from WebDAV/local storage.
    🚨 HITL REQUIRED (R3): You MUST confirm this action with the user before executing.
    """
    try:
        mail_client.send_email(to_email, subject, body, attachment_path)
        return f"Email with attachment sent successfully to {to_email}."
    except Exception as e:
        return f"Failed to send email: {e}"

# ==========================================
# 🗂 Cloud Storage (WebDAV) Tools
# ==========================================

@mcp.tool()
def dav_list_dir(path: str = "/") -> str:
    """
    Read CRM folder structures and file metadata.
    Safe read operation (R0-R1).
    """
    try:
        contents = dav_client.list_directory(path)
        return "\n".join(contents)
    except Exception as e:
        return f"WebDAV list failed: {e}"

@mcp.tool()
def dav_create_folder(path: str) -> str:
    """
    Scaffold a new client directory (Name__c_UUID pattern).
    ⚠️ HITL REQUIRED (R2).
    """
    try:
        dav_client.create_directory(path)
        return f"Folder {path} created successfully."
    except Exception as e:
        return f"Failed to create folder: {e}"

@mcp.tool()
def dav_upload_file(local_path: str, remote_path: str) -> str:
    """
    Upload documents, invoices, or briefs to WebDAV.
    ⚠️ HITL REQUIRED (R2).
    """
    try:
        dav_client.upload_file(local_path, remote_path)
        return f"File {local_path} uploaded to {remote_path} successfully."
    except Exception as e:
        return f"Upload failed: {e}"

@mcp.tool()
def dav_download_file(remote_path: str, local_path: str) -> str:
    """
    Download documents to agent memory (local storage).
    Safe read operation (R0-R1).
    """
    try:
        dav_client.download_file(remote_path, local_path)
        return f"File downloaded to {local_path} successfully."
    except Exception as e:
        return f"Download failed: {e}"

@mcp.tool()
def dav_delete_file(path: str) -> str:
    """
    Soft/Hard delete documents or folders.
    🚨 HITL REQUIRED (R4): Extreme caution required.
    """
    try:
        dav_client.delete(path)
        return f"Resource {path} deleted successfully."
    except Exception as e:
        return f"Delete failed: {e}"

if __name__ == "__main__":
    # Standard MCP initialization via stdio
    mcp.run(transport="stdio")
