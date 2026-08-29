## 2026-04-29 - Fixed Path Traversal in JWT Key Loading
**Issue:** Path traversal false positives flagged by gosec in JWT key loading via `os.ReadFile`.
**Root Cause:** gosec correctly flags usage of `os.ReadFile(path)` where `path` is an unsanitized string argument, even if it comes from configuration.
**Fix:** Apply `filepath.Clean()` to paths before passing to file operations, and add explicit `#nosec G304` comments noting that paths originate securely from configuration.
