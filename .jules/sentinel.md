# Sentinel's Journal

## 2024-05-22 - MIME Type Spoofing
**Vulnerability:** The application relied solely on the `Content-Type` header provided by the client to determine the file type of uploaded images.
**Learning:** Trusting client headers for file type validation allows attackers to bypass restrictions by simply spoofing the header (e.g., sending an executable with `Content-Type: image/jpeg`).
**Prevention:** Always use content-based detection (e.g., `http.DetectContentType` or magic bytes inspection) to verify the actual file type before processing or storing files.

## 2026-02-05 - Unprotected Sensitive Endpoints
**Vulnerability:** Login and Image Upload endpoints were missing rate limiting middleware, exposing the system to brute force (login) and DoS/storage abuse (upload).
**Learning:** Relying on optional configuration structs (`MiddlewareConfig`) without strict enforcement can lead to security features being silently disabled (defaulting to nil).
**Prevention:** Use functional options pattern or builder pattern that enforces security defaults, or explicitly check for nil configurations and warn/fail on startup for critical security components.
