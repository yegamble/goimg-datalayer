# Sentinel's Journal

## 2024-05-22 - MIME Type Spoofing
**Vulnerability:** The application relied solely on the `Content-Type` header provided by the client to determine the file type of uploaded images.
**Learning:** Trusting client headers for file type validation allows attackers to bypass restrictions by simply spoofing the header (e.g., sending an executable with `Content-Type: image/jpeg`).
**Prevention:** Always use content-based detection (e.g., `http.DetectContentType` or magic bytes inspection) to verify the actual file type before processing or storing files.
## 2025-02-14 - Prevent Directory Traversal via Prefix Bypass
**Vulnerability:** Path Traversal using partial path matches.
**Learning:** `strings.HasPrefix(cleanPath, basePath)` is insufficient because `cleanPath` could start with `basePath` without being a sub-directory. E.g., if `basePath` is `/var/storage`, `cleanPath` of `/var/storage_secrets/file.txt` would match the prefix.
**Prevention:** To prevent partial directory name bypasses, always check with `filepath.Separator` appended: `strings.HasPrefix(cleanPath, filepath.Clean(baseDir) + string(filepath.Separator))` or evaluate using `filepath.Rel`.
