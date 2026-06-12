# Sentinel's Journal

## 2024-05-22 - MIME Type Spoofing
**Vulnerability:** The application relied solely on the `Content-Type` header provided by the client to determine the file type of uploaded images.
**Learning:** Trusting client headers for file type validation allows attackers to bypass restrictions by simply spoofing the header (e.g., sending an executable with `Content-Type: image/jpeg`).
**Prevention:** Always use content-based detection (e.g., `http.DetectContentType` or magic bytes inspection) to verify the actual file type before processing or storing files.

## 2025-02-21 - JWT Algorithm Confusion
**Vulnerability:** The JWT validation didn't restrict the parsing algorithms, it only checked the header's algorithm *after* decoding. By providing a token signed with HMAC (HS256) where an RSA key was expected, an attacker could potentially forge tokens.
**Learning:** `jwt.ParseWithClaims` defaults to accepting any algorithm the parser supports. We must supply `jwt.WithValidMethods([]string{"RS256"})` explicitly to restrict parsing.
**Prevention:** Always restrict accepted signing methods for `golang-jwt/jwt/v5` parsing logic by passing `jwt.WithValidMethods(...)` as an option.
