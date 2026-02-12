# Sentinel's Journal

## 2024-05-22 - MIME Type Spoofing
**Vulnerability:** The application relied solely on the `Content-Type` header provided by the client to determine the file type of uploaded images.
**Learning:** Trusting client headers for file type validation allows attackers to bypass restrictions by simply spoofing the header (e.g., sending an executable with `Content-Type: image/jpeg`).
**Prevention:** Always use content-based detection (e.g., `http.DetectContentType` or magic bytes inspection) to verify the actual file type before processing or storing files.

## 2024-05-23 - Rate Limiting Activation
**Vulnerability:** Rate limiting implementation existed in middleware but was not activated in the application entry point (`main.go`), leaving sensitive endpoints (login, upload) vulnerable to brute-force and DoS attacks.
**Learning:** Having security features implemented in libraries/middleware is insufficient if they are not explicitly wired up in the composition root. Configuration structs (like `MiddlewareConfig`) must be populated correctly.
**Prevention:** Add integration tests that verify the presence of security headers (like `X-RateLimit-Limit`) on sensitive endpoints to ensure the protection is active in the deployed application.
