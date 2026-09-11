## 2024-05-24 - Path Traversal in AI-Driven File Downloads
**Vulnerability:** The `dav_download_file` MCP tool accepted an arbitrary `local_path` from the AI without validating if the path was outside the agent's expected working directory.
**Learning:** Even if the external service (WebDAV) is protected via HITL for write operations, local file write operations (like downloading a file) must be bounded. Without bounding, an AI agent (or an attacker influencing it) could overwrite critical system files, `.env` files, or the source code itself, bypassing the intended HITL safety for destructive actions by corrupting the local environment.
**Prevention:** Implement path validation using `os.path.commonpath([base_dir, target_path]) == base_dir` on all local file outputs derived from agent arguments to ensure they cannot escape the current working directory.

## 2026-09-06 - [Symlink Path Traversal in ValidateDownloadPath]
**Vulnerability:** Symlink path traversal via `filepath.Rel` without prior `filepath.EvalSymlinks` evaluation.
**Learning:** `filepath.Rel` alone is insufficient to prevent path traversal if the underlying filesystem uses symlinks that point outside the allowed root directories.
**Prevention:** Always use `filepath.EvalSymlinks` to resolve symlinks for both the target path and root directories before attempting to validate boundaries. For non-existent target files, symlinks in the parent directory path must also be evaluated.

## 2026-09-06 - [Insecure TLS Configuration for SMTP]
**Vulnerability:** SMTP client initialized without specifying a minimum TLS version in `tls.Config`.
**Learning:** Default TLS configuration can permit obsolete or weak protocols, exposing connections to TLS downgrade attacks.
**Prevention:** Explicitly set `MinVersion: tls.VersionTLS12` (or newer) in `tls.Config` when initializing secure connections.

## 2026-09-11 - [Insecure TLS Configuration for IMAP]
**Vulnerability:** IMAP client in `dialIMAP()` did not set `MinVersion: tls.VersionTLS12` in `tls.Config`.
**Learning:** Symmetric security posture across all protocol adapters (IMAP, SMTP, WebDAV) is required to prevent protocol downgrade.
**Prevention:** Explicitly set `MinVersion: tls.VersionTLS12` across all network TLS configurations.

## 2026-09-11 - [Path Traversal in AI-Driven WebDAV Uploads & Mail Attachments]
**Vulnerability:** `dav_upload_file` and `mail_send_with_attachment` accepted unbounded host file paths (`/etc/shadow`, `~/.ssh/id_rsa`).
**Learning:** Even when destructive operations are guarded by HITL, staging arbitrary system file paths allows potential exfiltration if human approval is social-engineered or accidentally given.
**Prevention:** Strictly enforce `webdav.ValidateDownloadPath()` on both staging and execution phases for all local file read/write arguments.
