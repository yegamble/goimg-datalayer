# Sentinel's Journal

## 2024-05-22 - MIME Type Spoofing
**Vulnerability:** The application relied solely on the `Content-Type` header provided by the client to determine the file type of uploaded images.
**Learning:** Trusting client headers for file type validation allows attackers to bypass restrictions by simply spoofing the header (e.g., sending an executable with `Content-Type: image/jpeg`).
**Prevention:** Always use content-based detection (e.g., `http.DetectContentType` or magic bytes inspection) to verify the actual file type before processing or storing files.

## 2025-05-25 - JWT Algorithm Confusion Vulnerability
**Vulnerability:** The application parsed JWT tokens using `jwt.ParseWithClaims` without restricting the allowed signing methods. It only manually checked if the parsed token's method was RSA *after* parsing. This allowed attackers to perform algorithm confusion attacks by signing an HS256 token using the server's RSA public key.
**Learning:** Checking the token method type after parsing is insufficient because the underlying library may still attempt to verify the signature using the wrong algorithm (e.g., using the RSA public key as an HMAC secret) if not explicitly restricted during parsing.
**Prevention:** Always restrict accepted signing methods when parsing JWTs by passing `jwt.WithValidMethods([]string{"RS256"})` (or the specific expected algorithm) to the parsing function.
