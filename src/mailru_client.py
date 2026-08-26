import os
import imaplib
import email
from email.header import decode_header
from datetime import datetime, timedelta
import time
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


    def get_imap_connection(self) -> imaplib.IMAP4_SSL:
        mail = imaplib.IMAP4_SSL(self.imap_host)
        mail.login(self.username, self.password)
        return mail

    def read_inbox_since(self, days: int = 1, folder: str = "INBOX") -> List[Dict[str, Any]]:
        """Fetch emails since a given number of days ago using IMAP SINCE."""
        mail = self.get_imap_connection()
        try:
            mail.select(f'"{folder}"')
            since_date = (datetime.now() - timedelta(days=days)).strftime("%d-%b-%Y")
            status, messages = mail.search(None, f'(SINCE "{since_date}")')
            
            emails = []
            if status == "OK" and messages[0]:
                email_ids = messages[0].split()
                # Get last 50 if there are many to avoid long processing
                for e_id in reversed(email_ids[-50:]):
                    res, msg_data = mail.fetch(e_id, '(RFC822)')
                    if res == "OK":
                        for response_part in msg_data:
                            if isinstance(response_part, tuple):
                                msg = email.message_from_bytes(response_part[1])
                                
                                # Decode subject
                                subject_header = msg["Subject"]
                                if subject_header:
                                    subject, encoding = decode_header(subject_header)[0]
                                    if isinstance(subject, bytes):
                                        subject = subject.decode(encoding if encoding else "utf-8", errors="replace")
                                else:
                                    subject = "No Subject"
                                    
                                emails.append({
                                    "uid": e_id.decode(),
                                    "subject": subject,
                                    "from": msg.get("From"),
                                    "date": msg.get("Date")
                                })
            return emails
        finally:
            try:
                mail.close()
            except Exception:
                pass
            mail.logout()

    def save_draft(self, to_email: str, subject: str, body: str, folder: str = "&BCcENQRBBD0ESwQ1-") -> bool:
        """Save an email to Drafts using IMAP APPEND."""
        mail = self.get_imap_connection()
        try:
            msg = email.message.EmailMessage()
            msg['Subject'] = subject
            msg['From'] = self.username
            msg['To'] = to_email
            msg.set_content(body)
            
            date_time = imaplib.Time2Internaldate(time.time())
            
            # Try appending to the primary Drafts folder, fallback to 'Drafts' if it fails
            try:
                status, _ = mail.append(f'"{folder}"', r'\Draft', date_time, msg.as_bytes())
                if status != 'OK':
                    raise Exception("Failed to append")
            except Exception:
                status, _ = mail.append('"Drafts"', r'\Draft', date_time, msg.as_bytes())
                if status != 'OK':
                    return False
            return True
        finally:
            mail.logout()

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
