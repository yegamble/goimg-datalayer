# Sentinel's Journal

## 2024-05-22 - MIME Type Spoofing
**Vulnerability:** The application relied solely on the `Content-Type` header provided by the client to determine the file type of uploaded images.
**Learning:** Trusting client headers for file type validation allows attackers to bypass restrictions by simply spoofing the header (e.g., sending an executable with `Content-Type: image/jpeg`).
**Prevention:** Always use content-based detection (e.g., `http.DetectContentType` or magic bytes inspection) to verify the actual file type before processing or storing files.
## 2025-02-18 - Prevent JWT Algorithm Confusion
**Vulnerability:** The JWT parsing logic was vulnerable to algorithm confusion because it relied solely on checking the parsed token's method instead of explicitly restricting allowed methods during parsing using `jwt.WithValidMethods`.
**Learning:** The `jwt-go` / `golang-jwt` library allows specifying valid methods during parsing. If not specified, it may use an unexpected signing method (like HS256 with a public key as the symmetric secret) to validate the token before the custom Keyfunc runs.
**Prevention:** Always use `jwt.WithValidMethods([]string{"RS256"})` (or the specific expected algorithm) alongside `jwt.ParseWithClaims` or `jwt.Parse` to ensure the library rejects tokens signed with unexpected algorithms at the parsing stage.
