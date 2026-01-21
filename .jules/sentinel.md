## 2024-05-23 - Path Traversal in Local Storage
**Vulnerability:** The `local` storage provider used `filepath.Join` without validating that the resulting path remained within the configured base directory. This could allow an attacker to write or read files outside the storage root (Zip Slip / Path Traversal) if they could bypass the initial `validateKey` check (e.g., via race conditions or if `validateKey` was relaxed/modified in the future).
**Learning:** In Go, `filepath.Join` cleans the path but does not prevent directory traversal if the input contains `..`. A check like `strings.Contains(key, "..")` is good, but a robust defense-in-depth approach requires resolving the absolute path and verifying the prefix matches the expected base directory.
**Prevention:** Always use a `resolvePath` helper that:
1. Joins the base path and user input.
2. Cleans the path (`filepath.Clean`).
3. Verifies `strings.HasPrefix(cleanPath, basePath + separator)`.
