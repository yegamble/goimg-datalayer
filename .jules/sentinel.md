# Sentinel's Journal

## 2024-05-22 - MIME Type Spoofing
**Vulnerability:** The application relied solely on the `Content-Type` header provided by the client to determine the file type of uploaded images.
**Learning:** Trusting client headers for file type validation allows attackers to bypass restrictions by simply spoofing the header (e.g., sending an executable with `Content-Type: image/jpeg`).
**Prevention:** Always use content-based detection (e.g., `http.DetectContentType` or magic bytes inspection) to verify the actual file type before processing or storing files.

## 2025-02-24 - Unwired Security Controls
**Vulnerability:** Rate limiting middleware was implemented but not applied to critical endpoints (Login, Upload), despite comments indicating it was.
**Learning:** The existence of security code (middleware) does not guarantee it is active. Configuration drift or incomplete implementation can leave "implemented" controls inactive.
**Prevention:** Implement integration tests that specifically verify the active enforcement of security controls (e.g., triggering the rate limit) rather than just unit testing the control logic in isolation.
