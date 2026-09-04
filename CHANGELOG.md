# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

---

## [1.2.0] - 2026-09-04

### Added
- **Fail-Safe Socket Timeouts:** Explicit 15-second deadline (`MAILRU_TIMEOUT=15`) across all IMAP (`MailBox`), SMTP (`smtplib.SMTP_SSL`), and WebDAV (`webdavclient3`) network calls to eliminate deadlock hazards.
- **HITL Two-Phase Commit Hardening:** Sliding 3600-second TTL on staged actions, capacity bound (`MAX_PENDING_ACTIONS = 100`) with oldest-entry eviction, and automatic token pruning.
- **Workspace Containment Defense:** Path traversal validation in `dav_download_file` utilizing `get_allowed_download_roots()` (`/root/.agents`, `/root/projects`, `/tmp`, and CWD).
- **Professional Governance Documentation:** Official [MIT License](LICENSE), [CONTRIBUTING.md](CONTRIBUTING.md), [CHANGELOG.md](CHANGELOG.md), [SECURITY.md](SECURITY.md), and [pyproject.toml](pyproject.toml).
- **Comprehensive Unit Testing:** 47 automated tests achieving **97% overall line coverage**.

### Removed
- **Phantom CalDAV/CardDAV Modules:** Amputated `src/caldav_client.py`, `src/carddav_client.py`, and fake mock tests `tests/test_caldav_and_carddav.py` following network audits confirming Mail.ru lacks public CalDAV and rejects CardDAV PUT contact creation. Calendar and address book management are natively delegated to Nextcloud.
- **Amputated Phantom Tools:** Removed `calendar_list_events`, `calendar_create_event`, `contact_search`, and `contact_create`. Tool registry consolidated to 13 verified, operational tools.
- **Unused Dependency:** Removed unneeded direct `requests` dependency from `requirements.txt`.

---

## [1.1.0] - 2026-09-04

### Added
- **Multi-Folder IMAP Aggregation:** Support for automatic scanning of Mail.ru smart subfolders (`INBOX/Newsletters`, `INBOX/Social`, `INBOX/News`, `INBOX/Receipts`).
- **Timezone Normalization:** Safe UTC timestamp normalization (`_normalize_date`) to prevent sorting comparison failures on mixed offset headers.
- **Message Body Extraction:** Added `mail_get_body` for complete text/HTML parsing by UID.
- **Draft Staging:** Added `mail_send_draft` appending directly to `Черновики` / `Drafts`.

---

## [1.0.0] - 2026-08-23

### Added
- Initial release of `mailru-mcp-server`.
- Core FastMCP integration for stdio transport.
- IMAP inbox retrieval, SMTP message dispatch, and WebDAV cloud storage management.
- Initial Human-in-the-Loop two-phase commit structure.
