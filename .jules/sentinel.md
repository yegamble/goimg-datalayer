# Sentinel's Journal

## 2024-05-22 - MIME Type Spoofing
**Vulnerability:** The application relied solely on the `Content-Type` header provided by the client to determine the file type of uploaded images.
**Learning:** Trusting client headers for file type validation allows attackers to bypass restrictions by simply spoofing the header (e.g., sending an executable with `Content-Type: image/jpeg`).
**Prevention:** Always use content-based detection (e.g., `http.DetectContentType` or magic bytes inspection) to verify the actual file type before processing or storing files.

## 2025-05-23 - Rate Limiting Disabled by Default
**Vulnerability:** The application includes robust rate limiting middleware (global, auth, login, upload), but the main entry point (`cmd/api/main.go`) fails to initialize the `RateLimiterConfig` in the `MiddlewareConfig`. Consequently, `RateLimiterConfig` remains `nil`, and all rate limiting logic is skipped at runtime.
**Learning:** Security features implemented in middleware are useless if not wired up in the composition root. The presence of code does not imply active protection.
**Prevention:** Integration tests should verify not just that middleware *exists*, but that it is *active* in the production configuration. Verify configuration structs are fully populated.
