# Sentinel's Journal

## 2024-05-22 - MIME Type Spoofing
**Vulnerability:** The application relied solely on the `Content-Type` header provided by the client to determine the file type of uploaded images.
**Learning:** Trusting client headers for file type validation allows attackers to bypass restrictions by simply spoofing the header (e.g., sending an executable with `Content-Type: image/jpeg`).
**Prevention:** Always use content-based detection (e.g., `http.DetectContentType` or magic bytes inspection) to verify the actual file type before processing or storing files.

## 2025-02-14 - JWT Algorithm Confusion Vulnerability
**Vulnerability:** The application parsed JWTs using `jwt.ParseWithClaims` without restricting the valid signing algorithms to RS256, allowing potential attackers to sign tokens with symmetric algorithms using the application's public key.
**Learning:** Relying solely on `token.Method.(*jwt.SigningMethodRSA)` type assertion within the `Keyfunc` callback is insufficient as an attacker can provide a valid HMAC token using the public key as the secret, which passes the `jwt-go` parsing checks but creates an algorithm confusion attack. Explicitly restricting the expected algorithm is necessary.
**Prevention:** Always use `jwt.WithValidMethods([]string{"RS256"})` when parsing JWTs to strictly enforce the expected algorithm at the library level, rejecting any algorithm confusion attempts before the `Keyfunc` is evaluated.
