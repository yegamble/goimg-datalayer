# Sentinel's Journal

## 2024-05-22 - MIME Type Spoofing
**Vulnerability:** The application relied solely on the `Content-Type` header provided by the client to determine the file type of uploaded images.
**Learning:** Trusting client headers for file type validation allows attackers to bypass restrictions by simply spoofing the header (e.g., sending an executable with `Content-Type: image/jpeg`).
**Prevention:** Always use content-based detection (e.g., `http.DetectContentType` or magic bytes inspection) to verify the actual file type before processing or storing files.

## 2025-02-27 - JWT Algorithm Confusion Prevention
**Vulnerability:** The application was vulnerable to JWT algorithm confusion as `jwt.ParseWithClaims` did not explicitly restrict accepted signing methods using `jwt.WithValidMethods()`. Although it checked the `token.Method` inside the Keyfunc, an attacker could potentially bypass or complicate validation.
**Learning:** `golang-jwt/jwt` parsing logic must always explicitly restrict algorithms during parsing, otherwise the Keyfunc could be executed with unexpected algorithms. The error message changes from "unexpected signing method" to "token signature is invalid: signing method <alg> is invalid" when restricted properly.
**Prevention:** Always append `jwt.WithValidMethods([]string{"RS256"})` (or appropriate expected algorithm) as a parser option alongside the Keyfunc.
