## 2024-05-22 - MIME Type Spoofing
**Vulnerability:** The application relied solely on the `Content-Type` header provided by the client to determine the file type of uploaded images.
**Learning:** Trusting client headers for file type validation allows attackers to bypass restrictions by simply spoofing the header (e.g., sending an executable with `Content-Type: image/jpeg`).
**Prevention:** Always use content-based detection (e.g., `http.DetectContentType` or magic bytes inspection) to verify the actual file type before processing or storing files.

## 2025-03-09 - JWT Algorithm Confusion Vulnerability
**Vulnerability:** The `ValidateToken` function parsed JWT tokens using `jwt.ParseWithClaims` without restricting the allowed signing methods.
**Learning:** Even though the callback checked if the parsed method was an instance of `*jwt.SigningMethodRSA`, a well-known vulnerability exists where an attacker signs the token using the application's public key but with an HMAC algorithm (e.g., HS256). If the library accepts it, it may be validated against the public key treated as an HMAC secret, leading to authentication bypass.
**Prevention:** Always explicitly enforce the expected signing algorithms when parsing JWT tokens. In `golang-jwt/jwt/v5`, use the `jwt.WithValidMethods([]string{"RS256"})` parser option.
