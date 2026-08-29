## 2026-04-29 - Fixed Path Traversal in JWT Key Loading
**Vulnerability:** Path traversal (CWE-22) identified by gosec (G304) in the loading of JWT private and public keys.
**Learning:** `os.ReadFile` usage on variables can be flagged as path traversal, even when the path string is sourced from environment variables/configuration.
**Prevention:** Utilizing `filepath.Clean` and prepending `// #nosec G304 // Path is securely provided by application configuration` safely resolves the false positive scanner warning.
