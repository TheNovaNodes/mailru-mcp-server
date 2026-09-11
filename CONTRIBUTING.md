# Contributing to mailru-mcp-server


Thank you for your interest in contributing to `mailru-mcp-server`, a critical component of the **TheNovaNodes** ecosystem providing Model Context Protocol (MCP) integration with Mail.ru services.

Both human engineers and autonomous AI agents are welcome to contribute under the collective directives outlined below.

---

## 🩸 Core Engineering Directives ("Правила Крови")

1. **Strict Git Flow:**
   - **NEVER** push directly to `master` or `main` branches.
   - All bugfixes, refactorings, and features **MUST** go through dedicated feature branches (`feat/*`, `fix/*`, `chore/*`) and Pull Requests (PR).
   - **NEVER** merge PRs without explicit approval from **ЗавЛаб**.

2. **Verification Without Absurdity:**
   - **Pre-commit / Pre-push (Native Minimum):** Verify changes locally before committing:
     ```bash
     make lint
     make test
     make coverage
     ```
     Ensure all unit tests pass with the race detector enabled (`-race`).
   - **Post-PR (Cloud CI via GitHub CLI):** After opening a PR, monitor the actual cloud CI:
     ```bash
     gh pr checks <PR_NUMBER> --watch
     ```
     Never report readiness or attempt merging until all CI checks are green.

3. **Zero-Deadlock & Network Timeout Directive:**
   - Every network socket or HTTP connection **MUST** enforce an explicit timeout (`15s` or `MAILRU_TIMEOUT`).
   - Infinite blocking calls on sockets or streaming endpoints are strictly prohibited.

4. **Documentation Synchronization Invariant:**
   - Code changes, refactorings, or new tool registrations **MUST** be accompanied by synchronized updates to `README.md`, `ARCHITECTURE.md`, `CHANGELOG.md`, and `SECURITY.md` in the same commit / PR.

---

## 🛠️ Development Setup

1. **Prerequisites:**
   - Go `1.22` or higher.
   - `make` utility.

2. **Clone repository and build:**
   ```bash
   git clone https://github.com/TheNovaNodes/mailru-mcp-server.git
   cd mailru-mcp-server
   make build
   ```

3. **Configure environment variables:**
   ```bash
   export MAILRU_USERNAME="user@mail.ru"
   export MAILRU_APP_PASS="app-specific-password"
   export MAILRU_TIMEOUT="15"
   ```

4. **Run test suite:**
   ```bash
   make test
   ```

---

## 🛡️ Security & Human-In-The-Loop (HITL)

When adding or modifying MCP tools:
- **Read-only tools** (e.g. `mail_read_inbox`, `dav_list_dir`) may execute autonomously.
- **Mutating/Destructive tools** (e.g. `mail_send_reply`, `dav_delete_file`, `mail_move_message`) **MUST** use `hitl.Manager.Request()` to enforce Two-Phase Commit with a 3600-second TTL token.
- **File downloads** **MUST** validate target destinations using `webdav.ValidateDownloadPath()` to prevent directory traversal outside designated workspaces.

---

## 📜 License

By contributing to this repository, you agree that your contributions will be licensed under the [MIT License](LICENSE).
