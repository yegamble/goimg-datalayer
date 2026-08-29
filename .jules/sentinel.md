# Sentinel's Journal

## 2024-05-22 - MIME Type Spoofing
**Vulnerability:** The application relied solely on the `Content-Type` header provided by the client to determine the file type of uploaded images.
**Learning:** Trusting client headers for file type validation allows attackers to bypass restrictions by simply spoofing the header (e.g., sending an executable with `Content-Type: image/jpeg`).
**Prevention:** Always use content-based detection (e.g., `http.DetectContentType` or magic bytes inspection) to verify the actual file type before processing or storing files.

## 2024-05-31 - JWT Algorithm Confusion Vulnerability
**Vulnerability:** The application was vulnerable to JWT algorithm confusion because `jwt.ParseWithClaims` did not explicitly restrict accepted signing methods using `jwt.WithValidMethods`.
**Learning:** Relying only on checking the token method type inside the key lookup function (`func(token *jwt.Token) (interface{}, error)`) is insufficient and error-prone. The underlying library can still attempt to parse or process it insecurely if not explicitly restricted at the parser level.
**Prevention:** Always restrict accepted signing methods when parsing JWT tokens by passing `jwt.WithValidMethods([]string{"RS256"})` (or the specific expected algorithm) to the parsing function.
