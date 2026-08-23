import os
import smtplib
from email.message import EmailMessage
from typing import List, Dict, Any
from imap_tools import MailBox, AND

class MailRuClient:
    def __init__(self):
        # Credentials loaded purely from ENV (injected via with-secret)
        self.username = os.environ.get("MAILRU_USERNAME")
        self.password = os.environ.get("MAILRU_APP_PASS")
        self.imap_host = os.environ.get("MAILRU_IMAP_HOST", "imap.mail.ru")
        self.smtp_host = os.environ.get("MAILRU_SMTP_HOST", "smtp.mail.ru")
        
        if not self.username or not self.password:
            raise ValueError("MAILRU_USERNAME and MAILRU_APP_PASS must be set in environment.")

    def fetch_recent_emails(self, limit: int = 10, folder: str = "INBOX") -> List[Dict[str, Any]]:
        """Fetch recent emails from a specific folder."""
        emails = []
        with MailBox(self.imap_host).login(self.username, self.password, initial_folder=folder) as mailbox:
            # Fetch last 'limit' emails in reverse order (newest first)
            for msg in mailbox.fetch(limit=limit, reverse=True):
                emails.append({
                    "uid": msg.uid,
                    "subject": msg.subject,
                    "from": msg.from_,
                    "to": msg.to,
                    "date": msg.date.isoformat(),
                    "text": msg.text or msg.html,
                    "flags": msg.flags
                })
        return emails

    def search_emails(self, query: str, folder: str = "INBOX") -> List[Dict[str, Any]]:
        """Search emails by text/subject."""
        emails = []
        with MailBox(self.imap_host).login(self.username, self.password, initial_folder=folder) as mailbox:
            # Simple text search across all fields
            for msg in mailbox.fetch(AND(text=query)):
                emails.append({
                    "uid": msg.uid,
                    "subject": msg.subject,
                    "from": msg.from_,
                    "date": msg.date.isoformat(),
                    "text_snippet": (msg.text or msg.html)[:500] + "..."
                })
        return emails

    def send_email(self, to_email: str, subject: str, body: str, attachment_path: str = None) -> bool:
        """Send an email via SMTP."""
        msg = EmailMessage()
        msg['Subject'] = subject
        msg['From'] = self.username
        msg['To'] = to_email
        msg.set_content(body)
        
        if attachment_path and os.path.exists(attachment_path):
            with open(attachment_path, 'rb') as f:
                file_data = f.read()
                file_name = os.path.basename(attachment_path)
            msg.add_attachment(file_data, maintype='application', subtype='octet-stream', filename=file_name)
            
        with smtplib.SMTP_SSL(self.smtp_host, 465) as server:
            server.login(self.username, self.password)
            server.send_message(msg)
            
        return True
