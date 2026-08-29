# Sentinel's Journal

## 2024-05-22 - MIME Type Spoofing
**Vulnerability:** The application relied solely on the `Content-Type` header provided by the client to determine the file type of uploaded images.
**Learning:** Trusting client headers for file type validation allows attackers to bypass restrictions by simply spoofing the header (e.g., sending an executable with `Content-Type: image/jpeg`).
**Prevention:** Always use content-based detection (e.g., `http.DetectContentType` or magic bytes inspection) to verify the actual file type before processing or storing files.

## 2024-05-22 - JWT Algorithm Confusion
**Vulnerability:** The `ValidateToken` function used a custom `Keyfunc` to assert the token method was RSA but didn't restrict the parsing algorithms, allowing attackers to sign tokens with HMAC (HS256) using the application's public key.
**Learning:** `golang-jwt` requires explicit algorithm restriction using `jwt.WithValidMethods` to prevent algorithm confusion attacks where the parser falls back to symmetric validation with asymmetric keys.
**Prevention:** Always supply `jwt.WithValidMethods([]string{"RS256"})` (or the expected algorithm) to `jwt.ParseWithClaims` to strictly enforce the signing method.
