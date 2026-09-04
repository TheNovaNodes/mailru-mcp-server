import unittest
from unittest.mock import patch, MagicMock
import os
import sys

sys.path.insert(0, os.path.abspath(os.path.join(os.path.dirname(__file__), "..")))

os.environ.setdefault("MAILRU_USERNAME", "test@mail.ru")
os.environ.setdefault("MAILRU_APP_PASS", "test_pass")

from src.caldav_client import CalDAVClient
from src.carddav_client import CardDAVClient
from src import server


class TestCalDAVClient(unittest.TestCase):
    def test_init_raises(self):
        with patch.dict(os.environ, {}, clear=True):
            with self.assertRaises(ValueError):
                CalDAVClient()

    @patch("requests.request")
    def test_list_events_found(self, mock_req):
        mock_resp = MagicMock()
        mock_resp.status_code = 207
        mock_resp.text = (
            "<d:multistatus>\n"
            "BEGIN:VEVENT\nUID:123\nSUMMARY:Meeting\nEND:VEVENT\n"
            "BEGIN:VEVENT\nUID:456\nSUMMARY:Standup\nEND:VEVENT\n"
            "</d:multistatus>"
        )
        mock_req.return_value = mock_resp

        client = CalDAVClient()
        events = client.list_events(days_ahead=3)
        self.assertEqual(len(events), 2)
        self.assertIn("SUMMARY:Meeting", events[0])
        self.assertIn("SUMMARY:Standup", events[1])

    @patch("requests.request")
    def test_list_events_empty(self, mock_req):
        mock_resp = MagicMock()
        mock_resp.status_code = 200
        mock_resp.text = "<d:multistatus></d:multistatus>"
        mock_req.return_value = mock_resp

        client = CalDAVClient()
        events = client.list_events(days_ahead=1)
        self.assertEqual(events, ["No upcoming events found."])

    @patch("requests.request")
    def test_list_events_http_error(self, mock_req):
        mock_resp = MagicMock()
        mock_resp.status_code = 500
        mock_req.return_value = mock_resp

        client = CalDAVClient()
        events = client.list_events()
        self.assertIn("Failed to fetch events: HTTP 500", events[0])

    @patch("requests.put")
    def test_create_event_success(self, mock_put):
        mock_resp = MagicMock()
        mock_resp.status_code = 201
        mock_put.return_value = mock_resp

        client = CalDAVClient()
        res = client.create_event("Sprint Planning", "2026-09-05T10:00:00", "2026-09-05T11:00:00")
        self.assertTrue(res)
        self.assertTrue(mock_put.called)

    @patch("requests.put")
    def test_create_event_error(self, mock_put):
        mock_resp = MagicMock()
        mock_resp.status_code = 400
        mock_resp.text = "Bad payload"
        mock_put.return_value = mock_resp

        client = CalDAVClient()
        with self.assertRaises(RuntimeError):
            client.create_event("Sprint Planning", "2026-09-05T10:00:00Z", "2026-09-05T11:00:00Z")


class TestCardDAVClient(unittest.TestCase):
    def test_init_raises(self):
        with patch.dict(os.environ, {}, clear=True):
            with self.assertRaises(ValueError):
                CardDAVClient()

    @patch("requests.request")
    def test_search_contacts_found(self, mock_req):
        mock_resp = MagicMock()
        mock_resp.status_code = 207
        mock_resp.text = (
            "<d:multistatus>\n"
            "BEGIN:VCARD\nFN:John Doe\nEMAIL:john@example.com\nEND:VCARD\n"
            "</d:multistatus>"
        )
        mock_req.return_value = mock_resp

        client = CardDAVClient()
        res = client.search_contacts("John")
        self.assertEqual(len(res), 1)
        self.assertIn("FN:John Doe", res[0]["vcard"])

    @patch("requests.request")
    def test_search_contacts_empty(self, mock_req):
        mock_resp = MagicMock()
        mock_resp.status_code = 200
        mock_resp.text = "<d:multistatus></d:multistatus>"
        mock_req.return_value = mock_resp

        client = CardDAVClient()
        res = client.search_contacts("Unknown")
        self.assertIn("info", res[0])

    @patch("requests.request")
    def test_search_contacts_http_error(self, mock_req):
        mock_resp = MagicMock()
        mock_resp.status_code = 403
        mock_req.return_value = mock_resp

        client = CardDAVClient()
        res = client.search_contacts("query")
        self.assertIn("error", res[0])

    @patch("requests.put")
    def test_create_contact_with_phone(self, mock_put):
        mock_resp = MagicMock()
        mock_resp.status_code = 201
        mock_put.return_value = mock_resp

        client = CardDAVClient()
        res = client.create_contact("Alice", "alice@example.com", "+123456789")
        self.assertTrue(res)
        call_args = mock_put.call_args[1]
        self.assertIn(b"TEL;TYPE=CELL:+123456789", call_args["data"])

    @patch("requests.put")
    def test_create_contact_error(self, mock_put):
        mock_resp = MagicMock()
        mock_resp.status_code = 500
        mock_resp.text = "Internal error"
        mock_put.return_value = mock_resp

        client = CardDAVClient()
        with self.assertRaises(RuntimeError):
            client.create_contact("Bob", "bob@example.com")


class TestServerCalendarAndContactTools(unittest.TestCase):
    def test_calendar_list_events(self):
        with patch.object(server.caldav_client, "list_events", return_value=["Event 1", "Event 2"]):
            res = server.calendar_list_events(5)
            self.assertEqual(res, "Event 1\n\nEvent 2")

        with patch.object(server.caldav_client, "list_events", side_effect=Exception("Timeout")):
            res = server.calendar_list_events()
            self.assertIn("Failed to fetch calendar events: Timeout", res)

    def test_calendar_create_event_hitl(self):
        token_msg = server.calendar_create_event("Review", "2026-09-04T12:00:00Z", "2026-09-04T13:00:00Z")
        self.assertIn("ACTION BLOCKED (HITL REQUIRED)", token_msg)
        token = token_msg.split("token: ")[1].strip()

        with patch.object(server.caldav_client, "create_event", return_value=True) as mock_create:
            res = server.execute_pending_action(token)
            self.assertIn("Event 'Review' scheduled", res)
            mock_create.assert_called_once_with("Review", "2026-09-04T12:00:00Z", "2026-09-04T13:00:00Z")

    def test_contact_search(self):
        with patch.object(server.carddav_client, "search_contacts", return_value=[
            {"vcard": "VCARD1"},
            {"info": "Info message"},
            {"error": "Some error"}
        ]):
            res = server.contact_search("test")
            self.assertIn("VCARD1", res)
            self.assertIn("Info message", res)
            self.assertIn("Error: Some error", res)

        with patch.object(server.carddav_client, "search_contacts", side_effect=Exception("Failed")):
            res = server.contact_search("test")
            self.assertIn("Failed to search contacts: Failed", res)

    def test_contact_create_hitl(self):
        token_msg = server.contact_create("Bob", "bob@example.com", "+79990001122")
        self.assertIn("ACTION BLOCKED (HITL REQUIRED)", token_msg)
        token = token_msg.split("token: ")[1].strip()

        with patch.object(server.carddav_client, "create_contact", return_value=True) as mock_create:
            res = server.execute_pending_action(token)
            self.assertIn("Contact 'Bob' saved to Address Book", res)
            mock_create.assert_called_once_with("Bob", "bob@example.com", "+79990001122")


if __name__ == "__main__":
    unittest.main()
