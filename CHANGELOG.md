# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

---

## [2.0.1] - 2026-09-11

### Security & Hardening
- **IMAP TLS Downgrade Protection:** Enforced `MinVersion: tls.VersionTLS12` on `dialIMAP()`.
- **Universal Filesystem Containment:** Extended `ValidateDownloadPath()` to `dav_upload_file` and `mail_send_with_attachment`.
- **Configurable Operational Roots:** Added `MAILRU_ALLOWED_ROOTS` support and automatic resolution of `os.UserHomeDir()` and `os.TempDir()`.
- **Configurable Operator Name:** Added `MAILRU_OPERATOR_NAME` support in HITL prompt templates.

### Changed
- **CI/CD FinOps Optimization:** Added `concurrency`, `timeout-minutes: 10`, `fetch-depth: 1`, `paths-ignore`, and Go caching.
- **Documentation Anonymization:** Generalized internal system paths across `README.md`, `ARCHITECTURE.md`, and `SECURITY.md`.

---

## [2.0.0] - 2026-09-04

### Added
- **Full Go Rewrite:** Complete ground-up rewrite in Go 1.22+ using `github.com/mark3labs/mcp-go`.
- **Static Binary Architecture:** Single standalone `13MB` binary eliminating Python `.venv` and interpreter runtime overhead.
- **High Performance & Low Memory:** Reduced RSS RAM consumption from ~85MB to ~15MB; cold-start latency reduced from ~1200ms to <10ms.
- **Robust IMAP/SMTP Engine:** Implemented with `github.com/emersion/go-imap/v2` with full MIME multipart parsing, UTC date normalization, smart folder aggregation, and TLS socket deadlines.
- **Native WebDAV Client:** Built with `net/http`, PROPFIND XML decoding, MKCOL, PUT, GET, DELETE, and path traversal validation.
- **Thread-Safe HITL Manager:** Mutex-protected Two-Phase Commit token manager with sliding 3600-second TTL and capacity bounds.
- **Go Project Infrastructure:** `go.mod`, `go.sum`, `Makefile`, and updated GitHub Actions CI running `go vet` and `go test -race`.

### Changed
- Migrated CI workflow to GitHub Actions Go 1.22 environment.
- Updated governance and architecture documentation to reflect Go standards (ADR-6).

### Removed
- Retired legacy Python codebase (`src/`, `.venv`, `requirements.txt`, `pyproject.toml`).

---

## [1.2.0] - 2026-09-04

### Added
- **Fail-Safe Socket Timeouts:** Explicit 15-second deadline across IMAP, SMTP, and WebDAV calls.
- **HITL Two-Phase Commit Hardening:** Sliding 3600-second TTL on staged actions, capacity bound (`MAX_PENDING_ACTIONS = 100`).
- **Workspace Containment Defense:** Path traversal validation in `dav_download_file` utilizing `get_allowed_download_roots()`.
- **Governance Documentation:** Official MIT License, CONTRIBUTING.md, CHANGELOG.md, SECURITY.md.

### Removed
- **Phantom CalDAV/CardDAV Modules:** Amputated `caldav_client` and `carddav_client` following audit.

---

## [1.1.0] - 2026-09-04

### Added
- **Multi-Folder IMAP Aggregation:** Support for Mail.ru smart subfolders.
- **Timezone Normalization:** Safe UTC normalization on mixed headers.
- **Message Body Extraction & Draft Staging:** Added `mail_get_body` and `mail_send_draft`.

---

## [1.0.0] - 2026-08-23

### Added
- Initial release of `mailru-mcp-server` in Python.
- FastMCP stdio transport, IMAP inbox retrieval, SMTP dispatch, WebDAV storage.
