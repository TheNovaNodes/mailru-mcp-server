## 2024-08-31 - Missing Connection Pooling in API Clients
**Learning:** Found an anti-pattern in `CalDAVClient` and `CardDAVClient` where standalone `requests.request` and `requests.put` calls were used for API communication. This is extremely inefficient because it establishes a new TCP connection and TLS session for every single request, missing out on HTTP Keep-Alive.
**Action:** Always use `requests.Session()` in API clients to enable connection pooling. Bind the authentication and common headers to the session to reduce per-request latency.
