# Sentinel's Journal

## 2024-05-22 - MIME Type Spoofing
**Vulnerability:** The application relied solely on the `Content-Type` header provided by the client to determine the file type of uploaded images.
**Learning:** Trusting client headers for file type validation allows attackers to bypass restrictions by simply spoofing the header (e.g., sending an executable with `Content-Type: image/jpeg`).
**Prevention:** Always use content-based detection (e.g., `http.DetectContentType` or magic bytes inspection) to verify the actual file type before processing or storing files.

## 2024-05-22 - JWT Algorithm Confusion Vulnerability
**Vulnerability:** The application was vulnerable to JWT algorithm confusion because `jwt.ParseWithClaims` did not restrict the accepted signing methods, allowing an attacker to bypass signature validation by signing a token with HMAC (e.g. HS256) using the RSA public key.
**Learning:** Always explicitly restrict accepted signing methods when parsing JWT tokens to prevent algorithm confusion attacks.
**Prevention:** Use `jwt.WithValidMethods([]string{"RS256"})` (or the specific expected algorithms) alongside checking the token method type when parsing tokens.
