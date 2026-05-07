# Sentinel's Journal

## 2024-05-22 - MIME Type Spoofing
**Vulnerability:** The application relied solely on the `Content-Type` header provided by the client to determine the file type of uploaded images.
**Learning:** Trusting client headers for file type validation allows attackers to bypass restrictions by simply spoofing the header (e.g., sending an executable with `Content-Type: image/jpeg`).
**Prevention:** Always use content-based detection (e.g., `http.DetectContentType` or magic bytes inspection) to verify the actual file type before processing or storing files.
## 2026-05-07 - Storage Path Traversal Bypass
**Vulnerability:** The application attempted to prevent path traversal in `internal/infrastructure/storage/local/local.go` by checking for exact strings like `..` or null bytes.
**Learning:** String-based checks are insufficient because paths can be manipulated to bypass exact matching checks, leaving file inclusion and overwrite attacks possible.
**Prevention:** Always use the path module (`filepath.Clean` and `filepath.Join`) to canonicalize user-provided inputs, then check whether the resulting string still originates within the expected `basePath` bounds using `strings.HasPrefix`.
