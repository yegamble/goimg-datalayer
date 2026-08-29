# Sentinel's Journal

## 2024-05-22 - MIME Type Spoofing
**Vulnerability:** The application relied solely on the `Content-Type` header provided by the client to determine the file type of uploaded images.
**Learning:** Trusting client headers for file type validation allows attackers to bypass restrictions by simply spoofing the header (e.g., sending an executable with `Content-Type: image/jpeg`).
**Prevention:** Always use content-based detection (e.g., `http.DetectContentType` or magic bytes inspection) to verify the actual file type before processing or storing files.

## 2024-05-28 - JWT Algorithm Confusion
**Vulnerability:** The application was vulnerable to JWT algorithm confusion because it relied solely on checking `token.Method.(*jwt.SigningMethodRSA)` within the key function without explicitly restricting valid signing methods during parsing.
**Learning:** Attackers can potentially bypass authentication by generating an HMAC-signed token using the application's public RSA key as the HMAC secret, which `jwt-go` might accept if `ValidMethods` are not strictly enforced.
**Prevention:** Always restrict accepted signing methods when parsing JWT tokens using `jwt.WithValidMethods([]string{"RS256"})` alongside verifying the token method type.
