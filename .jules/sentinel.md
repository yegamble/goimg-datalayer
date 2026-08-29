# Sentinel's Journal

## 2024-05-22 - MIME Type Spoofing
**Vulnerability:** The application relied solely on the `Content-Type` header provided by the client to determine the file type of uploaded images.
**Learning:** Trusting client headers for file type validation allows attackers to bypass restrictions by simply spoofing the header (e.g., sending an executable with `Content-Type: image/jpeg`).
**Prevention:** Always use content-based detection (e.g., `http.DetectContentType` or magic bytes inspection) to verify the actual file type before processing or storing files.

## 2025-06-25 - JWT Algorithm Confusion
**Vulnerability:** The application was vulnerable to JWT algorithm confusion as it did not restrict the accepted signing methods when parsing tokens.
**Learning:** Relying solely on the token's method type check inside the key function is insufficient, as the parser might still process the token with an unexpected algorithm before the check, or the parser's behavior might change.
**Prevention:** Always explicitly restrict the accepted signing methods using `jwt.WithValidMethods([]string{"RS256"})` (or the specific expected algorithm) when parsing tokens.
