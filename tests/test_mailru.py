import unittest
from unittest.mock import patch, MagicMock
from datetime import datetime, timezone, timedelta
import time
import os
import sys

# Ensure project root is in sys.path
sys.path.insert(0, os.path.abspath(os.path.join(os.path.dirname(__file__), "..")))

# Set mock env vars before loading server module to prevent Fail-Fast crash on import
os.environ.setdefault("MAILRU_USERNAME", "test@mail.ru")
os.environ.setdefault("MAILRU_APP_PASS", "test_pass")
os.environ.setdefault("WEBDAV_LOGIN", "test_dav")
os.environ.setdefault("WEBDAV_PASSWORD", "test_pass")
os.environ.setdefault("CALDAV_URL", "https://caldav.example.com")
os.environ.setdefault("CALDAV_USER", "test_user")
os.environ.setdefault("CALDAV_PASS", "test_pass")
os.environ.setdefault("CARDDAV_URL", "https://carddav.example.com")
os.environ.setdefault("CARDDAV_USER", "test_user")
os.environ.setdefault("CARDDAV_PASS", "test_pass")

from src.mailru_client import _normalize_date, MailRuClient
from src import server


class TestDateNormalization(unittest.TestCase):
    def test_normalize_date_various_inputs(self):
        # 1. UTC aware
        dt_utc = datetime(2026, 9, 4, 12, 0, tzinfo=timezone.utc)
        self.assertEqual(_normalize_date(dt_utc), dt_utc)

        # 2. Naive
        dt_naive = datetime(2026, 9, 4, 12, 0)
        normalized_naive = _normalize_date(dt_naive)
        self.assertEqual(normalized_naive.tzinfo, timezone.utc)
        self.assertEqual(normalized_naive.year, 2026)

        # 3. Offset aware (+3 hours)
        tz_msk = timezone(timedelta(hours=3))
        dt_msk = datetime(2026, 9, 4, 15, 0, tzinfo=tz_msk)
        self.assertEqual(_normalize_date(dt_msk), dt_utc)

        # 4. None
        normalized_none = _normalize_date(None)
        self.assertEqual(normalized_none.tzinfo, timezone.utc)
        self.assertEqual(normalized_none, datetime.min.replace(tzinfo=timezone.utc))

    def test_sorting_mixed_dates(self):
        d_none = None
        d_naive = datetime(2025, 1, 1, 10, 0)
        d_utc_old = datetime(2024, 1, 1, 10, 0, tzinfo=timezone.utc)
        d_msk_new = datetime(2026, 9, 4, 15, 0, tzinfo=timezone(timedelta(hours=3)))

        items = [
            {"id": 1, "date_raw": d_none},
            {"id": 2, "date_raw": d_naive},
            {"id": 3, "date_raw": d_utc_old},
            {"id": 4, "date_raw": d_msk_new},
        ]
        items.sort(key=lambda x: _normalize_date(x["date_raw"]), reverse=True)
        sorted_ids = [i["id"] for i in items]
        self.assertEqual(sorted_ids, [4, 2, 3, 1])


class TestMailRuClient(unittest.TestCase):
    @patch.dict(os.environ, {"MAILRU_USERNAME": "test@mail.ru", "MAILRU_APP_PASS": "secret123"})
    def setUp(self):
        self.client = MailRuClient()

    def test_init_raises_without_creds(self):
        with patch.dict(os.environ, {}, clear=True):
            with self.assertRaises(ValueError):
                MailRuClient()

    @patch("src.mailru_client.MailBox")
    def test_fetch_recent_emails_aggregates_smart_folders(self, mock_mailbox_cls):
        mock_box = MagicMock()
        mock_mailbox_instance = mock_mailbox_cls.return_value
        mock_mailbox_instance.login.return_value = mock_mailbox_instance
        mock_mailbox_instance.__enter__.return_value = mock_box

        msg_inbox = MagicMock()
        msg_inbox.uid = "101"
        msg_inbox.subject = "Urgent Request"
        msg_inbox.from_ = "client@domain.com"
        msg_inbox.to = ("test@mail.ru",)
        msg_inbox.date = datetime(2026, 9, 4, 10, 0, tzinfo=timezone.utc)
        msg_inbox.text = "Please check doc"
        msg_inbox.html = ""
        msg_inbox.flags = ()

        msg_news = MagicMock()
        msg_news.uid = "102"
        msg_news.subject = "Weekly Digest"
        msg_news.from_ = "newsletter@news.com"
        msg_news.to = ("test@mail.ru",)
        msg_news.date = datetime(2026, 9, 4, 12, 0, tzinfo=timezone.utc)
        msg_news.text = "Here is the news"
        msg_news.html = ""
        msg_news.flags = ()

        def fetch_side_effect(limit, reverse, mark_seen):
            current_folder = mock_box.folder.set.call_args[0][0] if mock_box.folder.set.call_args else "INBOX"
            if current_folder == "INBOX":
                return [msg_inbox]
            elif current_folder == "INBOX/Newsletters":
                return [msg_news]
            return []

        mock_box.fetch.side_effect = fetch_side_effect

        results = self.client.fetch_recent_emails(limit=10, folder="INBOX", include_smart_folders=True)
        self.assertEqual(len(results), 2)
        # Verify sorted newest first: msg_news (12:00) before msg_inbox (10:00)
        self.assertEqual(results[0]["uid"], "102")
        self.assertEqual(results[0]["folder"], "INBOX/Newsletters")
        self.assertEqual(results[1]["uid"], "101")
        self.assertEqual(results[1]["folder"], "INBOX")

    @patch("src.mailru_client.MailBox")
    def test_get_email_body(self, mock_mailbox_cls):
        mock_box = MagicMock()
        mock_mailbox_instance = mock_mailbox_cls.return_value
        mock_mailbox_instance.login.return_value = mock_mailbox_instance
        mock_mailbox_instance.__enter__.return_value = mock_box

        msg = MagicMock()
        msg.uid = "12345"
        msg.subject = "Test Message"
        msg.from_ = "sender@example.com"
        msg.to = ("test@mail.ru",)
        msg.date = datetime(2026, 9, 4, 10, 0, tzinfo=timezone.utc)
        msg.text = "Hello world body"
        msg.html = "<p>Hello world body</p>"
        mock_box.fetch.return_value = [msg]

        res = self.client.get_email_body(uid="12345", folder="INBOX")
        self.assertEqual(res["uid"], "12345")
        self.assertEqual(res["text"], "Hello world body")
        self.assertEqual(res["html"], "<p>Hello world body</p>")

    @patch("src.mailru_client.MailBox")
    def test_move_message(self, mock_mailbox_cls):
        mock_box = MagicMock()
        mock_mailbox_instance = mock_mailbox_cls.return_value
        mock_mailbox_instance.login.return_value = mock_mailbox_instance
        mock_mailbox_instance.__enter__.return_value = mock_box

        res = self.client.move_message(uid="555", to_folder="Archive", from_folder="INBOX")
        self.assertTrue(res)
        mock_box.move.assert_called_once_with("555", "Archive")

    @patch("src.mailru_client.MailBox")
    def test_save_draft(self, mock_mailbox_cls):
        mock_box = MagicMock()
        mock_mailbox_instance = mock_mailbox_cls.return_value
        mock_mailbox_instance.login.return_value = mock_mailbox_instance
        mock_mailbox_instance.__enter__.return_value = mock_box

        res = self.client.save_draft("recipient@example.com", "Draft Subject", "Draft Body")
        self.assertTrue(res)
        self.assertTrue(mock_box.append.called)


class TestServerHITL(unittest.TestCase):
    def test_hitl_lifecycle(self):
        # 1. Request HITL for mail_move_message
        token_msg = server.request_hitl("mail_move_message", {"uid": "999", "to_folder": "SPAM"})
        self.assertIn("ACTION BLOCKED (HITL REQUIRED)", token_msg)

        # Extract token
        token = token_msg.split("token: ")[1].strip()
        self.assertIn(token, server.PENDING_ACTIONS)

        # 2. Try executing with wrong token
        err = server.execute_pending_action("invalid_token")
        self.assertIn("Error: Invalid, expired, or already executed", err)

        # 3. Execute with valid token
        with patch.object(server.mail_client, "move_message", return_value=True) as mock_move:
            success = server.execute_pending_action(token)
            self.assertIn("Action Executed", success)
            mock_move.assert_called_once_with("999", "SPAM", "INBOX")

        # 4. Token cannot be reused (one-time use)
        reused = server.execute_pending_action(token)
        self.assertIn("Error: Invalid, expired, or already executed", reused)

    def test_hitl_ttl_expiration(self):
        # Stage action and manually age it past TTL
        token_msg = server.request_hitl("mail_send", {"to": "old@test.com", "subject": "S", "body": "B"})
        token = token_msg.split("token: ")[1].strip()
        self.assertIn(token, server.PENDING_ACTIONS)

        # Fast-forward time past TTL (3601 seconds)
        server.PENDING_ACTIONS[token]["created_at"] = time.time() - 3605

        res = server.execute_pending_action(token)
        self.assertIn("Error: Invalid, expired, or already executed", res)
        self.assertNotIn(token, server.PENDING_ACTIONS)

    def test_hitl_max_actions_eviction(self):
        server.PENDING_ACTIONS.clear()
        base_time = time.time()
        # Fill to capacity
        tokens = []
        for i in range(server.MAX_PENDING_ACTIONS):
            msg = server.request_hitl("mail_move_message", {"uid": str(i)})
            tok = msg.split("token: ")[1].strip()
            tokens.append(tok)
            # Ensure strictly incrementing recent timestamps (within TTL)
            server.PENDING_ACTIONS[tok]["created_at"] = base_time + i

        self.assertEqual(len(server.PENDING_ACTIONS), server.MAX_PENDING_ACTIONS)
        oldest_token = tokens[0]
        self.assertIn(oldest_token, server.PENDING_ACTIONS)

        # Trigger one more to force eviction of oldest
        msg_extra = server.request_hitl("mail_move_message", {"uid": "extra"})
        new_token = msg_extra.split("token: ")[1].strip()

        self.assertEqual(len(server.PENDING_ACTIONS), server.MAX_PENDING_ACTIONS)
        self.assertNotIn(oldest_token, server.PENDING_ACTIONS)
        self.assertIn(new_token, server.PENDING_ACTIONS)

    def test_path_traversal_dav_download(self):
        # 1. Disallowed path outside allowed roots
        res = server.dav_download_file("/remote/doc.pdf", "/etc/passwd")
        self.assertIn("Security Error: Path traversal detected", res)

        res2 = server.dav_download_file("/remote/doc.pdf", "/root/.ssh/id_rsa")
        self.assertIn("Security Error: Path traversal detected", res2)

        # 2. Allowed path under /tmp or /root/.agents
        with patch.object(server.dav_client, "download_file", return_value=True):
            ok_res = server.dav_download_file("/remote/doc.pdf", "/tmp/safe_doc.pdf")
            self.assertIn("downloaded to /tmp/safe_doc.pdf successfully", ok_res)


if __name__ == "__main__":
    unittest.main()
