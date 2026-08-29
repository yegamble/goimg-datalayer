# Sentinel's Journal

## 2024-05-22 - MIME Type Spoofing
**Vulnerability:** The application relied solely on the `Content-Type` header provided by the client to determine the file type of uploaded images.
**Learning:** Trusting client headers for file type validation allows attackers to bypass restrictions by simply spoofing the header (e.g., sending an executable with `Content-Type: image/jpeg`).
**Prevention:** Always use content-based detection (e.g., `http.DetectContentType` or magic bytes inspection) to verify the actual file type before processing or storing files.

## 2024-05-23 - JWT Algorithm Confusion
**Vulnerability:** The application was vulnerable to JWT algorithm confusion because it did not explicitly restrict the allowed signing methods during token parsing.
**Learning:** Checking the token's method type manually isn't enough; the `golang-jwt` library requires explicitly providing `jwt.WithValidMethods` to prevent attacks where a symmetric key is used as an asymmetric key.
**Prevention:** Always restrict accepted signing methods when parsing JWTs using `jwt.WithValidMethods([]string{"RS256"})` (or the expected algorithm).
