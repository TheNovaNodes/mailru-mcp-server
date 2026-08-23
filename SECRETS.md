# 🔐 Secrets Management

**CRITICAL LAW:**
Never store Mail.ru App Passwords or API keys in `.env` files or hardcoded configs.

## The `agent-vault` Protocol
This MCP server MUST be executed using the `with-secret` utility provided by TheNovaNodes ecosystem.

**Usage pattern for Agents:**
```bash
with-secret <VAULT_POINTER> --env MAILRU_APP_PASS -- <start_command>
```
The application will receive the `MAILRU_APP_PASS` securely in memory.
