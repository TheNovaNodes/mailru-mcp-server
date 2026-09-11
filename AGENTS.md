Part 1: TheNovaNodes Core Invariants (Universal Standard)
1. Strict Git Flow (ПРАВИЛА КРОВИ):
   - NEVER push directly to main or master branches.
   - All changes must go through dedicated branches (feat/..., fix/..., docs/...) and Pull Requests.
   - NEVER merge PRs without explicit approval from ЗавЛаб.
   - No force-push on upstream branches.
2. Security & Credential Hygiene:
   - NEVER hardcode or log passwords, tokens, or credentials (especially MAILRU_PASSWORD, MAILRU_USERNAME).
   - All credentials must be loaded dynamically from environment variables.
3. Deadlock & Timeout Guardrails:
   - All network calls (IMAP, SMTP, WebDAV) must have explicit timeouts (configured via config.Timeout).
   - Auxiliary commands must use hard timeouts.
   - MCP stdio servers must redirect stdin (< /dev/null) during smoke tests.
   - Never loop endlessly without bounds.
4. Continuous Verification:
   - Never report a task complete without running local verification commands.
   - Use native project tools directly.

Part 2: Repository Profile & Specific Directives (mailru-mcp-server)
1. Project Overview & Tech Stack:
   - Go (1.22+ / 1.25 compatible).
   - Dependencies: github.com/mark3labs/mcp-go, github.com/emersion/go-imap/v2, WebDAV client.
   - Packages: internal/config, internal/hitl, internal/mail, internal/server, internal/webdav.
   - Entry point: main.go (starts stdio MCP server via mcpserver.ServeStdio).
2. The Golden Loop (Mandatory Verification Commands):
   - go vet ./...
   - go test -v -race -cover ./internal/...
   - go build ./...
3. Architectural Invariants & Taboos:
   - CRITICAL HITL (Human-in-the-Loop) INVARIANT: All destructive actions (sending email, moving email, deleting WebDAV files, uploading files) are strictly guarded by internal/hitl.Manager. Agents are FORBIDDEN from bypassing or relaxing HITL token checks!
   - Stdio Protocol Hygiene: os.Stdout is exclusively reserved for MCP JSON-RPC protocol. All application logs must go to os.Stderr.
   - Non-destructive read tools (mail_read_inbox, mail_get_body, mail_search_thread, dav_list_dir) execute directly without HITL tokens.
4. PR & Commit Conventions:
   - Conventional Commits (docs(agents): ...).
