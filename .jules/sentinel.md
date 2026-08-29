# Sentinel's Journal

## 2024-05-22 - MIME Type Spoofing
**Vulnerability:** The application relied solely on the `Content-Type` header provided by the client to determine the file type of uploaded images.
**Learning:** Trusting client headers for file type validation allows attackers to bypass restrictions by simply spoofing the header (e.g., sending an executable with `Content-Type: image/jpeg`).
**Prevention:** Always use content-based detection (e.g., `http.DetectContentType` or magic bytes inspection) to verify the actual file type before processing or storing files.

## 2026-05-01 - Path Traversal Prevention in Local Storage
**Vulnerability:** Path traversal was possible because the validation only checked for the ".." pattern in string components, which could potentially be bypassed or mismanaged, leading to accessing files outside the intended base directory.
**Learning:** Checking for substrings like ".." is insufficient for securing file paths, especially against complex directory traversal attacks or partial directory name bypasses.
**Prevention:** Always combine `filepath.Join` and `filepath.Clean` to resolve the full absolute path. Ensure the final path stays within bounds by checking `strings.HasPrefix(cleanPath, filepath.Clean(baseDir) + string(filepath.Separator))` to prevent partial directory name bypasses (e.g., `/var/lib/storage2` bypassing `/var/lib/storage`).
