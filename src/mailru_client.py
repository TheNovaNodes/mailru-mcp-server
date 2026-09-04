import os
import imaplib
import email
from email.header import decode_header
from datetime import datetime, timedelta, timezone
import time
import smtplib
from email.message import EmailMessage
from typing import List, Dict, Any
from imap_tools import MailBox, AND

DEFAULT_TIMEOUT = 15

def _normalize_date(dt: Any) -> datetime:
    if not dt:
        return datetime.min.replace(tzinfo=timezone.utc)
    if dt.tzinfo is None or dt.tzinfo.utcoffset(dt) is None:
        return dt.replace(tzinfo=timezone.utc)
    return dt.astimezone(timezone.utc)

class MailRuClient:
    def __init__(self):
        # Credentials loaded purely from ENV (injected via with-secret)
        self.username = os.environ.get("MAILRU_USERNAME")
        self.password = os.environ.get("MAILRU_APP_PASS")
        self.imap_host = os.environ.get("MAILRU_IMAP_HOST", "imap.mail.ru")
        self.smtp_host = os.environ.get("MAILRU_SMTP_HOST", "smtp.mail.ru")
        self.timeout = int(os.environ.get("MAILRU_TIMEOUT", str(DEFAULT_TIMEOUT)))
        
        if not self.username or not self.password:
            raise ValueError("MAILRU_USERNAME and MAILRU_APP_PASS must be set in environment.")

    def fetch_recent_emails(self, limit: int = 10, folder: str = "INBOX", include_smart_folders: bool = True) -> List[Dict[str, Any]]:
        """
        Fetch recent emails.
        If folder == "INBOX" and include_smart_folders is True,
        aggregates across INBOX and Mail.ru smart subfolders (Newsletters, Social, News, Receipts).
        Returns emails sorted newest first.
        """
        folders_to_scan = [folder]
        if folder == "INBOX" and include_smart_folders:
            folders_to_scan = ["INBOX", "INBOX/Newsletters", "INBOX/Social", "INBOX/News", "INBOX/Receipts"]

        emails = []
        with MailBox(self.imap_host, timeout=self.timeout).login(self.username, self.password) as mailbox:
            for fld in folders_to_scan:
                try:
                    mailbox.folder.set(fld)
                    for msg in mailbox.fetch(limit=limit, reverse=True, mark_seen=False):
                        emails.append({
                            "uid": msg.uid,
                            "folder": fld,
                            "subject": msg.subject or "(No Subject)",
                            "from": msg.from_,
                            "to": msg.to,
                            "date": msg.date.isoformat() if msg.date else "",
                            "date_raw": msg.date,
                            "text": msg.text or msg.html or "",
                            "flags": msg.flags
                        })
                except Exception:
                    continue

        # Sort descending by date so newest emails from all folders are on top (tz-safe)
        emails.sort(key=lambda x: _normalize_date(x.get("date_raw")), reverse=True)
        return emails[:limit]

    def search_emails(self, query: str, folder: str = "INBOX") -> List[Dict[str, Any]]:
        """Search emails by text/subject."""
        emails = []
        with MailBox(self.imap_host, timeout=self.timeout).login(self.username, self.password, initial_folder=folder) as mailbox:
            for msg in mailbox.fetch(AND(text=query), reverse=True, mark_seen=False):
                emails.append({
                    "uid": msg.uid,
                    "folder": folder,
                    "subject": msg.subject or "(No Subject)",
                    "from": msg.from_,
                    "date": msg.date.isoformat() if msg.date else "",
                    "text_snippet": (msg.text or msg.html or "")[:500] + "..."
                })
        return emails

    def get_email_body(self, uid: str, folder: str = "INBOX") -> Dict[str, Any]:
        """Fetch full email content by UID from a folder."""
        with MailBox(self.imap_host, timeout=self.timeout).login(self.username, self.password, initial_folder=folder) as mailbox:
            for msg in mailbox.fetch(AND(uid=uid), limit=1, mark_seen=False):
                return {
                    "uid": msg.uid,
                    "folder": folder,
                    "subject": msg.subject or "(No Subject)",
                    "from": msg.from_,
                    "to": msg.to,
                    "date": msg.date.isoformat() if msg.date else "",
                    "text": msg.text or "",
                    "html": msg.html or ""
                }
        return {}

    def move_message(self, uid: str, to_folder: str, from_folder: str = "INBOX") -> bool:
        """Move email by UID to another folder."""
        with MailBox(self.imap_host, timeout=self.timeout).login(self.username, self.password, initial_folder=from_folder) as mailbox:
            mailbox.move(uid, to_folder)
        return True

    def get_imap_connection(self) -> imaplib.IMAP4_SSL:
        mail = imaplib.IMAP4_SSL(self.imap_host, timeout=self.timeout)
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
                for e_id in reversed(email_ids[-50:]):
                    res, msg_data = mail.fetch(e_id, '(RFC822)')
                    if res == "OK":
                        for response_part in msg_data:
                            if isinstance(response_part, tuple):
                                msg = email.message_from_bytes(response_part[1])
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

    def save_draft(self, to_email: str, subject: str, body: str, folder: str = "Черновики") -> bool:
        """Save an email to Drafts using imap_tools append."""
        msg = EmailMessage()
        msg['Subject'] = subject
        msg['From'] = self.username
        msg['To'] = to_email
        msg.set_content(body)

        with MailBox(self.imap_host, timeout=self.timeout).login(self.username, self.password) as mailbox:
            # Try specified folder (default 'Черновики' for Mail.ru), fallback to 'Drafts'
            try:
                mailbox.append(msg.as_bytes(), folder, flag_set=['\\Draft'])
                return True
            except Exception:
                mailbox.append(msg.as_bytes(), "Drafts", flag_set=['\\Draft'])
                return True

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
            
        with smtplib.SMTP_SSL(self.smtp_host, 465, timeout=self.timeout) as server:
            server.login(self.username, self.password)
            server.send_message(msg)
            
        return True
