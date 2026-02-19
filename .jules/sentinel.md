# Sentinel's Journal

## 2024-05-22 - MIME Type Spoofing
**Vulnerability:** The application relied solely on the `Content-Type` header provided by the client to determine the file type of uploaded images.
**Learning:** Trusting client headers for file type validation allows attackers to bypass restrictions by simply spoofing the header (e.g., sending an executable with `Content-Type: image/jpeg`).
**Prevention:** Always use content-based detection (e.g., `http.DetectContentType` or magic bytes inspection) to verify the actual file type before processing or storing files.

## 2025-02-05 - Host Header Injection in QR Code Generation
**Vulnerability:** The `GetImageQRCode` endpoint used `inferBaseURLFromRequest` to construct absolute URLs, which blindly trusted the `X-Forwarded-Host` or `Host` header. This allowed attackers to inject malicious domains into generated QR codes, leading to potential phishing attacks.
**Learning:** Never trust client-provided headers like `Host` or `X-Forwarded-Host` for generating critical application links, especially those consumed by users (like QR codes or password reset links).
**Prevention:** Always rely on a securely configured `BASE_URL` environment variable and fail securely (e.g., return HTTP 500) if it is not configured, rather than falling back to insecure request headers.
