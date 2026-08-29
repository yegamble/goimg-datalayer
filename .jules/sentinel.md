# Sentinel's Journal

## 2024-05-22 - MIME Type Spoofing
**Vulnerability:** The application relied solely on the `Content-Type` header provided by the client to determine the file type of uploaded images.
**Learning:** Trusting client headers for file type validation allows attackers to bypass restrictions by simply spoofing the header (e.g., sending an executable with `Content-Type: image/jpeg`).
**Prevention:** Always use content-based detection (e.g., `http.DetectContentType` or magic bytes inspection) to verify the actual file type before processing or storing files.

## 2023-11-09 - [JWT Algorithm Confusion]
**Vulnerability:** JWT Parsing used loose algorithm validation, potentially allowing HMAC algorithms instead of RS256.
**Learning:** Always use explicit `jwt.WithValidMethods` to strict validate accepted algorithms.
**Prevention:** Added `jwt.WithValidMethods([]string{"RS256"})` option in all `jwt.ParseWithClaims` calls.
