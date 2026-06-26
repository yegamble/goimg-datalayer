# Sentinel's Journal

## 2024-05-22 - MIME Type Spoofing
**Vulnerability:** The application relied solely on the `Content-Type` header provided by the client to determine the file type of uploaded images.
**Learning:** Trusting client headers for file type validation allows attackers to bypass restrictions by simply spoofing the header (e.g., sending an executable with `Content-Type: image/jpeg`).
**Prevention:** Always use content-based detection (e.g., `http.DetectContentType` or magic bytes inspection) to verify the actual file type before processing or storing files.

## 2025-02-25 - JWT Algorithm Confusion Vulnerability
**Vulnerability:** JWT parsing logic relied solely on manual type assertion (`token.Method.(*jwt.SigningMethodRSA)`) instead of using the library's built-in algorithm validation, which can lead to algorithm confusion vulnerabilities if not strictly enforced.
**Learning:** When using `golang-jwt/jwt/v5`, manual type checking inside the key function is insufficient and error-prone. The library provides `jwt.WithValidMethods` specifically to strictly enforce accepted algorithms during token parsing.
**Prevention:** Always use `jwt.WithValidMethods([]string{"RS256"})` (or the specific expected algorithms) alongside checking the token method type when parsing tokens to explicitly restrict accepted signing methods.
