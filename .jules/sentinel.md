# Sentinel's Journal

## 2024-05-22 - MIME Type Spoofing
**Vulnerability:** The application relied solely on the `Content-Type` header provided by the client to determine the file type of uploaded images.
**Learning:** Trusting client headers for file type validation allows attackers to bypass restrictions by simply spoofing the header (e.g., sending an executable with `Content-Type: image/jpeg`).
**Prevention:** Always use content-based detection (e.g., `http.DetectContentType` or magic bytes inspection) to verify the actual file type before processing or storing files.
## 2026-05-23 - Host Header Injection in QR Code Generation\n**Vulnerability:** The application inferred the base URL for QR codes dynamically using client-controlled HTTP headers (`X-Forwarded-Host`, `r.Host`) when `baseURL` was unconfigured, allowing Host Header Injection.\n**Learning:** Relying on client-provided headers to build absolute URLs (like QR codes pointing to the app) without validation allows attackers to generate links to malicious domains.\n**Prevention:** Always rely on explicitly configured base URL properties (e.g., from configuration files) to construct absolute URLs. If unconfigured, fail securely (500 Server Error) rather than guessing from headers.
