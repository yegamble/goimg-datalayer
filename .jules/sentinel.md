# Sentinel's Journal

## 2024-05-22 - MIME Type Spoofing
**Vulnerability:** The application relied solely on the `Content-Type` header provided by the client to determine the file type of uploaded images.
**Learning:** Trusting client headers for file type validation allows attackers to bypass restrictions by simply spoofing the header (e.g., sending an executable with `Content-Type: image/jpeg`).
**Prevention:** Always use content-based detection (e.g., `http.DetectContentType` or magic bytes inspection) to verify the actual file type before processing or storing files.

## 2025-02-05 - JWT Algorithm Confusion Prevention
**Vulnerability:** The JWT validation logic checked the token signature algorithm dynamically based on the token header (e.g. `token.Method.(*jwt.SigningMethodRSA)`) instead of strictly requiring the expected signature algorithm at the parser level, allowing algorithm confusion vulnerabilities.
**Learning:** The `golang-jwt/jwt/v5` package introduced the `jwt.WithValidMethods` option to strictly enforce allowed signing algorithms when parsing tokens. By default, checking the algorithm type dynamically inside the key function does not prevent the library from parsing the token using the algorithm provided in the token header.
**Prevention:** Always use `jwt.WithValidMethods([]string{"RS256"})` (or the specific expected algorithm) when parsing tokens with `golang-jwt/jwt/v5`, rather than solely relying on dynamic type checking of the parsed `token.Method`.
