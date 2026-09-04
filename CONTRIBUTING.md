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
     python -m py_compile src/*.py tests/*.py
     python -m unittest discover tests -v
     coverage run -m unittest discover tests && coverage report -m
     ```
     Target standard: **$\ge 95\%$ line coverage across all modules**.
   - **Post-PR (Cloud CI via GitHub CLI):** After opening a PR, monitor the actual cloud CI:
     ```bash
     gh pr checks <PR_NUMBER> --watch
     ```
     Never report readiness or attempt merging until all CI checks are green.

3. **Zero-Deadlock & Network Timeout Directive:**
   - Every network socket or HTTP connection **MUST** enforce an explicit timeout (`timeout=15` or `MAILRU_TIMEOUT`).
   - Infinite blocking calls on sockets or streaming endpoints are strictly prohibited.

4. **Radical Minimalism & Anti-Mock Doctrine:**
   - Never introduce phantom protocols or fake comfort through ungrounded mock tests.
   - Every integrated protocol must be verified against real-world protocol endpoints and specifications.

---

## 🛠️ Development Setup

1. **Clone repository and setup environment:**
   ```bash
   git clone https://github.com/TheNovaNodes/mailru-mcp-server.git
   cd mailru-mcp-server
   python3 -m venv .venv
   source .venv/bin/activate
   pip install -r requirements.txt
   ```

2. **Configure environment variables:**
   Credentials must be provided via environment variables (never committed to git):
   ```bash
   export MAILRU_USERNAME="user@mail.ru"
   export MAILRU_APP_PASS="app-specific-password"
   export MAILRU_TIMEOUT="15"
   ```

3. **Run tests:**
   ```bash
   python -m unittest discover tests -v
   ```

---

## 🛡️ Security & Human-In-The-Loop (HITL)

When adding or modifying MCP tools:
- **Read-only tools** (e.g. `mail_read_inbox`, `dav_list_dir`) may execute autonomously.
- **Mutating/Destructive tools** (e.g. `mail_send_reply`, `dav_delete_file`, `mail_move_message`) **MUST** use `request_hitl()` to enforce Two-Phase Commit with a 3600-second TTL token.
- **File downloads** **MUST** validate target destinations using `get_allowed_download_roots()` to prevent directory traversal outside designated workspaces.

---

## 📜 License

By contributing to this repository, you agree that your contributions will be licensed under the [MIT License](LICENSE).
