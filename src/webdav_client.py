import os
from webdav3.client import Client
from typing import List, Dict

DEFAULT_TIMEOUT = 15

class WebDAVClient:
    def __init__(self):
        self.username = os.environ.get("MAILRU_USERNAME")
        self.password = os.environ.get("MAILRU_APP_PASS")
        self.host = os.environ.get("MAILRU_WEBDAV_HOST", "https://webdav.cloud.mail.ru")
        self.timeout = int(os.environ.get("MAILRU_TIMEOUT", str(DEFAULT_TIMEOUT)))

        if not self.username or not self.password:
            raise ValueError("MAILRU_USERNAME and MAILRU_APP_PASS must be set in environment.")

        options = {
            'webdav_hostname': self.host,
            'webdav_login':    self.username,
            'webdav_password': self.password,
            'webdav_timeout':  self.timeout
        }
        self.client = Client(options)

    def list_directory(self, path: str = "/") -> List[str]:
        """List contents of a directory in WebDAV."""
        if not path.startswith("/"):
            path = "/" + path
        try:
            return self.client.list(path)
        except Exception as e:
            raise RuntimeError(f"Failed to list directory {path}: {str(e)}")

    def create_directory(self, path: str) -> bool:
        """Create a new directory in WebDAV."""
        if not path.startswith("/"):
            path = "/" + path
        try:
            self.client.mkdir(path)
            return True
        except Exception as e:
            raise RuntimeError(f"Failed to create directory {path}: {str(e)}")

    def upload_file(self, local_path: str, remote_path: str) -> bool:
        """Upload a file to WebDAV."""
        if not remote_path.startswith("/"):
            remote_path = "/" + remote_path
        try:
            self.client.upload_sync(remote_path=remote_path, local_path=local_path)
            return True
        except Exception as e:
            raise RuntimeError(f"Failed to upload {local_path} to {remote_path}: {str(e)}")

    def download_file(self, remote_path: str, local_path: str) -> bool:
        """Download a file from WebDAV."""
        if not remote_path.startswith("/"):
            remote_path = "/" + remote_path
        try:
            self.client.download_sync(remote_path=remote_path, local_path=local_path)
            return True
        except Exception as e:
            raise RuntimeError(f"Failed to download {remote_path} to {local_path}: {str(e)}")

    def delete(self, path: str) -> bool:
        """Delete a file or directory."""
        if not path.startswith("/"):
            path = "/" + path
        try:
            self.client.clean(path)
            return True
        except Exception as e:
            raise RuntimeError(f"Failed to delete {path}: {str(e)}")
