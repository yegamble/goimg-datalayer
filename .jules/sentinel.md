# Sentinel's Journal

## 2024-05-22 - MIME Type Spoofing
**Vulnerability:** The application relied solely on the `Content-Type` header provided by the client to determine the file type of uploaded images.
**Learning:** Trusting client headers for file type validation allows attackers to bypass restrictions by simply spoofing the header (e.g., sending an executable with `Content-Type: image/jpeg`).
**Prevention:** Always use content-based detection (e.g., `http.DetectContentType` or magic bytes inspection) to verify the actual file type before processing or storing files.

## 2025-02-12 - JWT Algorithm Confusion Prevention
**Vulnerability:** The JWT parsing logic was susceptible to algorithm confusion (e.g., accepting HMAC signatures when RSA was expected) despite checking token.Method inside the Keyfunc.
**Learning:** Relying solely on type assertion inside the Keyfunc is less secure than explicitly restricting the algorithms accepted by the parser.
**Prevention:** Always restrict accepted signing methods when parsing tokens using `jwt.WithValidMethods([]string{"RS256"})` (or the specific expected algorithm).
