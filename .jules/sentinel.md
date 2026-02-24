## 2025-05-23 - Rate Limiting Disabled by Default
**Vulnerability:** The application includes robust rate limiting middleware (global, auth, login, upload), but the main entry point (`cmd/api/main.go`) fails to initialize the `RateLimiterConfig` in the `MiddlewareConfig`. Consequently, `RateLimiterConfig` remains `nil`, and all rate limiting logic is skipped at runtime.
**Learning:** Security features implemented in middleware are useless if not wired up in the composition root. The presence of code does not imply active protection.
**Prevention:** Integration tests should verify not just that middleware *exists*, but that it is *active* in the production configuration. Verify configuration structs are fully populated.
