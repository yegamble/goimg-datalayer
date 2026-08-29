# Sentinel's Journal

## 2024-05-22 - MIME Type Spoofing
**Vulnerability:** The application relied solely on the `Content-Type` header provided by the client to determine the file type of uploaded images.
**Learning:** Trusting client headers for file type validation allows attackers to bypass restrictions by simply spoofing the header (e.g., sending an executable with `Content-Type: image/jpeg`).
**Prevention:** Always use content-based detection (e.g., `http.DetectContentType` or magic bytes inspection) to verify the actual file type before processing or storing files.
## 2025-06-04 - Prevent JWT Algorithm Confusion
**Vulnerability:** The application parsed JWT tokens without explicitly restricting the allowed signing methods, relying only on checking `token.Method` inside the key function.
**Learning:** Relying solely on `token.Method` checks inside the `Keyfunc` is insufficient to prevent all JWT algorithm confusion attacks, as the parsing library might change behavior.
**Prevention:** Always use `jwt.WithValidMethods([]string{"RS256"})` (or the specific expected algorithm) when calling `jwt.ParseWithClaims` to restrict accepted signing methods at the library level.
