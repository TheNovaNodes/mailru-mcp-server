import unittest
from unittest.mock import patch, MagicMock
import os
import sys

sys.path.insert(0, os.path.abspath(os.path.join(os.path.dirname(__file__), "..")))

# Set mock env vars
os.environ.setdefault("MAILRU_USERNAME", "test@mail.ru")
os.environ.setdefault("MAILRU_APP_PASS", "test_pass")

from src.webdav_client import WebDAVClient
from src import server


class TestWebDAVClient(unittest.TestCase):
    @patch("src.webdav_client.Client")
    def test_client_init(self, mock_client_cls):
        client = WebDAVClient()
        self.assertEqual(client.username, "test@mail.ru")
        self.assertEqual(client.password, "test_pass")
        self.assertEqual(client.timeout, 15)
        mock_client_cls.assert_called_once()
        options = mock_client_cls.call_args[0][0]
        self.assertEqual(options.get("webdav_timeout"), 15)

    def test_client_init_raises(self):
        with patch.dict(os.environ, {}, clear=True):
            with self.assertRaises(ValueError):
                WebDAVClient()

    @patch("src.webdav_client.Client")
    def test_list_directory(self, mock_client_cls):
        mock_instance = mock_client_cls.return_value
        mock_instance.list.return_value = ["file1.txt", "file2.pdf"]
        client = WebDAVClient()

        res = client.list_directory("docs")
        self.assertEqual(res, ["file1.txt", "file2.pdf"])
        mock_instance.list.assert_called_with("/docs")

    @patch("src.webdav_client.Client")
    def test_list_directory_error(self, mock_client_cls):
        mock_instance = mock_client_cls.return_value
        mock_instance.list.side_effect = Exception("Connection lost")
        client = WebDAVClient()

        with self.assertRaises(RuntimeError):
            client.list_directory("/invalid")

    @patch("src.webdav_client.Client")
    def test_create_directory(self, mock_client_cls):
        mock_instance = mock_client_cls.return_value
        client = WebDAVClient()

        self.assertTrue(client.create_directory("new_folder"))
        mock_instance.mkdir.assert_called_with("/new_folder")

    @patch("src.webdav_client.Client")
    def test_create_directory_error(self, mock_client_cls):
        mock_instance = mock_client_cls.return_value
        mock_instance.mkdir.side_effect = Exception("Permission denied")
        client = WebDAVClient()

        with self.assertRaises(RuntimeError):
            client.create_directory("/new_folder")

    @patch("src.webdav_client.Client")
    def test_upload_file(self, mock_client_cls):
        mock_instance = mock_client_cls.return_value
        client = WebDAVClient()

        self.assertTrue(client.upload_file("/tmp/local.txt", "remote.txt"))
        mock_instance.upload_sync.assert_called_with(remote_path="/remote.txt", local_path="/tmp/local.txt")

    @patch("src.webdav_client.Client")
    def test_upload_file_error(self, mock_client_cls):
        mock_instance = mock_client_cls.return_value
        mock_instance.upload_sync.side_effect = Exception("Disk full")
        client = WebDAVClient()

        with self.assertRaises(RuntimeError):
            client.upload_file("/tmp/local.txt", "/remote.txt")

    @patch("src.webdav_client.Client")
    def test_download_file(self, mock_client_cls):
        mock_instance = mock_client_cls.return_value
        client = WebDAVClient()

        self.assertTrue(client.download_file("remote.txt", "/tmp/local.txt"))
        mock_instance.download_sync.assert_called_with(remote_path="/remote.txt", local_path="/tmp/local.txt")

    @patch("src.webdav_client.Client")
    def test_download_file_error(self, mock_client_cls):
        mock_instance = mock_client_cls.return_value
        mock_instance.download_sync.side_effect = Exception("Not found")
        client = WebDAVClient()

        with self.assertRaises(RuntimeError):
            client.download_file("/remote.txt", "/tmp/local.txt")

    @patch("src.webdav_client.Client")
    def test_delete(self, mock_client_cls):
        mock_instance = mock_client_cls.return_value
        client = WebDAVClient()

        self.assertTrue(client.delete("trash.txt"))
        mock_instance.clean.assert_called_with("/trash.txt")

    @patch("src.webdav_client.Client")
    def test_delete_error(self, mock_client_cls):
        mock_instance = mock_client_cls.return_value
        mock_instance.clean.side_effect = Exception("Locked")
        client = WebDAVClient()

        with self.assertRaises(RuntimeError):
            client.delete("/trash.txt")


class TestServerWebDAVTools(unittest.TestCase):
    def test_dav_list_dir_empty(self):
        with patch.object(server.dav_client, "list_directory", return_value=[]):
            res = server.dav_list_dir("/")
            self.assertIn("Directory is empty", res)

    def test_dav_list_dir_paginated(self):
        items = [f"item_{i}.txt" for i in range(100)]
        with patch.object(server.dav_client, "list_directory", return_value=items):
            res = server.dav_list_dir("/", offset=10, limit=20)
            self.assertIn("Showing 10 to 30 of 100 total items", res)
            self.assertIn("item_10.txt", res)
            self.assertIn("item_29.txt", res)
            self.assertNotIn("item_35.txt", res)

    def test_dav_list_dir_error(self):
        with patch.object(server.dav_client, "list_directory", side_effect=Exception("Timeout")):
            res = server.dav_list_dir("/")
            self.assertIn("WebDAV list failed: Timeout", res)

    def test_dav_create_folder_hitl(self):
        token_msg = server.dav_create_folder("/new_dir")
        self.assertIn("ACTION BLOCKED (HITL REQUIRED)", token_msg)
        token = token_msg.split("token: ")[1].strip()

        with patch.object(server.dav_client, "create_directory", return_value=True) as mock_mkdir:
            res = server.execute_pending_action(token)
            self.assertIn("Folder /new_dir created", res)
            mock_mkdir.assert_called_once_with("/new_dir")

    def test_dav_upload_file_hitl(self):
        token_msg = server.dav_upload_file("/tmp/in.txt", "/remote/out.txt")
        self.assertIn("ACTION BLOCKED (HITL REQUIRED)", token_msg)
        token = token_msg.split("token: ")[1].strip()

        with patch.object(server.dav_client, "upload_file", return_value=True) as mock_upload:
            res = server.execute_pending_action(token)
            self.assertIn("File uploaded to /remote/out.txt", res)
            mock_upload.assert_called_once_with("/tmp/in.txt", "/remote/out.txt")

    def test_dav_download_file_safe_and_error(self):
        # Safe local path in current dir
        safe_path = os.path.join(os.getcwd(), "test_download.tmp")
        with patch.object(server.dav_client, "download_file", return_value=True):
            res = server.dav_download_file("/remote.pdf", safe_path)
            self.assertIn("downloaded to", res)

        # Download failure
        with patch.object(server.dav_client, "download_file", side_effect=Exception("Network error")):
            res = server.dav_download_file("/remote.pdf", safe_path)
            self.assertIn("Download failed: Network error", res)

    def test_dav_delete_file_hitl(self):
        token_msg = server.dav_delete_file("/del.txt")
        self.assertIn("ACTION BLOCKED (HITL REQUIRED)", token_msg)
        token = token_msg.split("token: ")[1].strip()

        with patch.object(server.dav_client, "delete", return_value=True) as mock_del:
            res = server.execute_pending_action(token)
            self.assertIn("/del.txt deleted", res)
            mock_del.assert_called_once_with("/del.txt")


if __name__ == "__main__":
    unittest.main()
