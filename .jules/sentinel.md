# Sentinel's Journal

## 2024-05-22 - MIME Type Spoofing
**Vulnerability:** The application relied solely on the `Content-Type` header provided by the client to determine the file type of uploaded images.
**Learning:** Trusting client headers for file type validation allows attackers to bypass restrictions by simply spoofing the header (e.g., sending an executable with `Content-Type: image/jpeg`).
**Prevention:** Always use content-based detection (e.g., `http.DetectContentType` or magic bytes inspection) to verify the actual file type before processing or storing files.

## 2025-12-05 - JWT Algorithm Confusion Attack
**Vulnerability:** The application was vulnerable to a JWT algorithm confusion attack because the JWT parsing logic did not enforce a specific signing algorithm (`RS256`). An attacker could alter the JWT header to use HMAC (`HS256`) and sign the token with the application's public key, effectively bypassing authentication.
**Learning:** The `jwt-go` / `golang-jwt` library parses the algorithm specified in the token header, which should not be implicitly trusted by the server without strict verification against the expected algorithm.
**Prevention:** Always restrict accepted algorithms explicitly using `jwt.WithValidMethods([]string{"RS256"})` (or the specific algorithm expected) when using `jwt.ParseWithClaims`.
