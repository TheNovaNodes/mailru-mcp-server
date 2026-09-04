import unittest
from unittest.mock import patch, MagicMock, mock_open
import os
import sys
import time

sys.path.insert(0, os.path.abspath(os.path.join(os.path.dirname(__file__), "..")))

os.environ.setdefault("MAILRU_USERNAME", "test@mail.ru")
os.environ.setdefault("MAILRU_APP_PASS", "test_pass")

from src.mailru_client import MailRuClient
from src import server


class TestMailRuSMTPAndSearch(unittest.TestCase):
    def setUp(self):
        self.client = MailRuClient()

    @patch("smtplib.SMTP_SSL")
    def test_send_email_no_attachment(self, mock_smtp_cls):
        mock_server = MagicMock()
        mock_smtp_cls.return_value.__enter__.return_value = mock_server

        res = self.client.send_email("recipient@example.com", "Test Subject", "Test Body")
        self.assertTrue(res)
        mock_smtp_cls.assert_called_once_with("smtp.mail.ru", 465, timeout=15)
        mock_server.login.assert_called_once_with("test@mail.ru", "test_pass")
        mock_server.send_message.assert_called_once()

    @patch("smtplib.SMTP_SSL")
    @patch("os.path.exists", return_value=True)
    @patch("builtins.open", new_callable=mock_open, read_data=b"file content")
    def test_send_email_with_attachment(self, mock_file, mock_exists, mock_smtp_cls):
        mock_server = MagicMock()
        mock_smtp_cls.return_value.__enter__.return_value = mock_server

        res = self.client.send_email("recipient@example.com", "Invoice", "Please find attached", "/tmp/invoice.pdf")
        self.assertTrue(res)
        mock_smtp_cls.assert_called_once_with("smtp.mail.ru", 465, timeout=15)
        mock_server.send_message.assert_called_once()

    @patch("src.mailru_client.MailBox")
    def test_search_emails(self, mock_mailbox_cls):
        mock_box = MagicMock()
        mock_mailbox_instance = mock_mailbox_cls.return_value
        mock_mailbox_instance.login.return_value = mock_mailbox_instance
        mock_mailbox_instance.__enter__.return_value = mock_box

        msg1 = MagicMock()
        msg1.uid = "101"
        msg1.subject = "Subject 1"
        msg1.from_ = "a@b.com"
        msg1.date = None
        msg1.text = "Body 1"
        msg1.html = None

        mock_box.fetch.return_value = [msg1]

        res = self.client.search_emails("query_str")
        self.assertEqual(len(res), 1)
        self.assertEqual(res[0]["uid"], "101")
        self.assertIn("Body 1", res[0]["text_snippet"])
        mock_mailbox_cls.assert_called_once_with("imap.mail.ru", timeout=15)

    @patch.object(MailRuClient, "get_imap_connection")
    def test_read_inbox_since(self, mock_get_conn):
        mock_mail = MagicMock()
        mock_get_conn.return_value = mock_mail

        mock_mail.search.return_value = ("OK", [b"1 2"])
        # Mock fetch response
        raw_msg = (
            b"From: sender@example.com\r\n"
            b"Subject: =?utf-8?B?VGVzdA==?=\r\n"
            b"Date: Fri, 04 Sep 2026 12:00:00 +0000\r\n\r\n"
            b"Body"
        )
        mock_mail.fetch.return_value = ("OK", [(None, raw_msg)])

        emails = self.client.read_inbox_since(days=2)
        self.assertEqual(len(emails), 2)
        self.assertEqual(emails[0]["from"], "sender@example.com")
        self.assertEqual(emails[0]["subject"], "Test")


class TestServerMailTools(unittest.TestCase):
    def test_mail_search_thread_found_and_truncated(self):
        # 60 emails returned to test truncation (> safe_limit = 50)
        fake_emails = [{"uid": str(i), "from": f"u{i}@test.com", "subject": f"Sub {i}", "date": "2026-09-04", "text_snippet": "Hi"} for i in range(60)]
        with patch.object(server.mail_client, "search_emails", return_value=fake_emails):
            res = server.mail_search_thread("test")
            self.assertIn("Truncated. Found 60 emails", res)
            self.assertIn("UID: 0", res)

    def test_mail_search_thread_empty(self):
        with patch.object(server.mail_client, "search_emails", return_value=[]):
            res = server.mail_search_thread("nothing")
            self.assertEqual(res, "No emails found matching query.")

    def test_mail_search_thread_error(self):
        with patch.object(server.mail_client, "search_emails", side_effect=Exception("IMAP down")):
            res = server.mail_search_thread("err")
            self.assertIn("Error searching emails: IMAP down", res)

    def test_mail_read_inbox_empty_and_error(self):
        with patch.object(server.mail_client, "fetch_recent_emails", return_value=[]):
            res = server.mail_read_inbox()
            self.assertEqual(res, "No emails found.")

        with patch.object(server.mail_client, "fetch_recent_emails", side_effect=Exception("IMAP err")):
            res = server.mail_read_inbox()
            self.assertIn("Error reading inbox: IMAP err", res)

    def test_mail_read_inbox_found(self):
        fake_email = {
            "uid": "123",
            "folder": "INBOX",
            "from": "user@test.com",
            "subject": "Hello",
            "date": "2026-09-04T12:00:00Z",
            "flags": ["\\Seen"],
            "text": "Hello world snippet"
        }
        with patch.object(server.mail_client, "fetch_recent_emails", return_value=[fake_email]):
            res = server.mail_read_inbox()
            self.assertIn("UID: 123", res)
            self.assertIn("Folder: INBOX", res)
            self.assertIn("Subject: Hello", res)
            self.assertIn("Flags: \\Seen", res)

    def test_mail_get_body_not_found_and_error(self):
        with patch.object(server.mail_client, "get_email_body", return_value={}):
            res = server.mail_get_body("99999")
            self.assertIn("not found in folder", res)

        with patch.object(server.mail_client, "get_email_body", side_effect=Exception("Corrupt")):
            res = server.mail_get_body("111")
            self.assertIn("Error getting email body: Corrupt", res)

    def test_mail_get_body_found(self):
        fake_details = {
            "uid": "123",
            "folder": "INBOX",
            "from": "user@test.com",
            "to": "me@test.com",
            "subject": "Greetings",
            "date": "2026-09-04T12:00:00Z",
            "text": "Full body text content here",
            "html": ""
        }
        with patch.object(server.mail_client, "get_email_body", return_value=fake_details):
            res = server.mail_get_body("123")
            self.assertIn("UID: 123", res)
            self.assertIn("Full body text content here", res)

    def test_mail_send_draft_failure_and_error(self):
        with patch.object(server.mail_client, "save_draft", return_value=False):
            res = server.mail_send_draft("a@b.com", "S", "B")
            self.assertIn("Failed to save draft", res)

        with patch.object(server.mail_client, "save_draft", side_effect=Exception("Disk error")):
            res = server.mail_send_draft("a@b.com", "S", "B")
            self.assertIn("Error saving draft: Disk error", res)

    def test_mail_send_draft_success(self):
        with patch.object(server.mail_client, "save_draft", return_value=True):
            res = server.mail_send_draft("a@b.com", "Draft", "Content")
            self.assertIn("saved successfully in Drafts/Черновики", res)

    def test_mail_move_message_tool(self):
        token_msg = server.mail_move_message("777", "Archive", "INBOX")
        self.assertIn("ACTION BLOCKED (HITL REQUIRED)", token_msg)
        self.assertIn("mail_move_message", token_msg)

    def test_mail_send_reply_hitl(self):
        token_msg = server.mail_send_reply("client@test.com", "Re: Deal", "Agreed")
        self.assertIn("ACTION BLOCKED (HITL REQUIRED)", token_msg)
        token = token_msg.split("token: ")[1].strip()

        with patch.object(server.mail_client, "send_email", return_value=True) as mock_send:
            res = server.execute_pending_action(token)
            self.assertIn("Email sent to client@test.com", res)
            mock_send.assert_called_once_with("client@test.com", "Re: Deal", "Agreed", None)

    def test_mail_send_with_attachment_hitl(self):
        token_msg = server.mail_send_with_attachment("client@test.com", "Doc", "Body", "/tmp/doc.pdf")
        self.assertIn("ACTION BLOCKED (HITL REQUIRED)", token_msg)
        token = token_msg.split("token: ")[1].strip()

        with patch.object(server.mail_client, "send_email", return_value=True) as mock_send:
            res = server.execute_pending_action(token)
            self.assertIn("Email sent to client@test.com", res)
            mock_send.assert_called_once_with("client@test.com", "Doc", "Body", "/tmp/doc.pdf")

    def test_execute_pending_action_unknown_and_exception(self):
        # Unknown action
        server.PENDING_ACTIONS["test_unknown"] = {
            "type": "unsupported_action",
            "details": {},
            "created_at": time.time()
        }
        res = server.execute_pending_action("test_unknown")
        self.assertIn("Unknown action type", res)

        # Exception during execution
        server.PENDING_ACTIONS["test_fail"] = {
            "type": "mail_send",
            "details": {"to": "x", "subject": "y", "body": "z"},
            "created_at": time.time()
        }
        with patch.object(server.mail_client, "send_email", side_effect=Exception("SMTP down")):
            res = server.execute_pending_action("test_fail")
            self.assertIn("Execution failed: SMTP down", res)


if __name__ == "__main__":
    unittest.main()
