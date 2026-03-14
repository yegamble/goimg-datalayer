# Sentinel's Journal

## 2024-05-22 - MIME Type Spoofing
**Vulnerability:** The application relied solely on the `Content-Type` header provided by the client to determine the file type of uploaded images.
**Learning:** Trusting client headers for file type validation allows attackers to bypass restrictions by simply spoofing the header (e.g., sending an executable with `Content-Type: image/jpeg`).
**Prevention:** Always use content-based detection (e.g., `http.DetectContentType` or magic bytes inspection) to verify the actual file type before processing or storing files.

## 2024-05-23 - IDOR in Moderation Endpoints
**Vulnerability:** The `GetUserBanStatus` endpoint retrieved the requested user's ban status via the `userID` path parameter without validating if the authenticated user matches the `userID` or has appropriate admin/moderator privileges.
**Learning:** Endpoints retrieving user-specific PII/status via parameters are prone to Insecure Direct Object Reference (IDOR). Relying solely on the presence of an authentication token is insufficient if authorization (role or resource ownership) isn't explicitly validated in the handler.
**Prevention:** Always extract `userCtx` (via `GetUserFromContext(ctx)`) for user-specific endpoints and enforce role or ownership validations before executing the query. Return a generic 403 Forbidden without leaking the data.
