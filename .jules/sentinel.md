# Sentinel's Journal

## 2024-05-22 - MIME Type Spoofing
**Vulnerability:** The application relied solely on the `Content-Type` header provided by the client to determine the file type of uploaded images.
**Learning:** Trusting client headers for file type validation allows attackers to bypass restrictions by simply spoofing the header (e.g., sending an executable with `Content-Type: image/jpeg`).
**Prevention:** Always use content-based detection (e.g., `http.DetectContentType` or magic bytes inspection) to verify the actual file type before processing or storing files.

## 2025-03-09 - JWT Algorithm Confusion Vulnerability
**Vulnerability:** The application parsed JWTs without explicitly restricting the valid signing methods via options, relying only on checking the parsed token method. This made it vulnerable to algorithm confusion attacks where an attacker could sign a token with HMAC (HS256) using the public key as the secret.
**Learning:** The `jwt-go` / `golang-jwt` library by default accepts any signing algorithm specified in the token header if valid methods are not restricted via parsing options.
**Prevention:** Always restrict accepted signing methods when parsing tokens using `jwt.WithValidMethods([]string{"RS256"})` alongside checking the token method type.
