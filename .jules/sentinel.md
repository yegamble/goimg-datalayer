# Sentinel's Journal

## 2024-05-22 - MIME Type Spoofing
**Vulnerability:** The application relied solely on the `Content-Type` header provided by the client to determine the file type of uploaded images.
**Learning:** Trusting client headers for file type validation allows attackers to bypass restrictions by simply spoofing the header (e.g., sending an executable with `Content-Type: image/jpeg`).
**Prevention:** Always use content-based detection (e.g., `http.DetectContentType` or magic bytes inspection) to verify the actual file type before processing or storing files.
## 2024-05-22 - JWT Algorithm Confusion
**Vulnerability:** The `ValidateToken` function inside `jwt_service.go` was parsing JWT tokens and only validating the token signing method within the custom key function (`token.Method.(*jwt.SigningMethodRSA)`). It did not use the recommended library defense mechanism of enforcing the expected signing method inside the parser via `jwt.WithValidMethods()`.
**Learning:** This exposes the application to JWT algorithm confusion vulnerabilities if the underlying library or logic fails to properly reject tokens signed with a symmetric algorithm (e.g., HS256) instead of the expected asymmetric algorithm (RS256). An attacker could sign a payload with the public key (treated as a symmetric secret) and bypass authentication.
**Prevention:** Always use `jwt.WithValidMethods([]string{"RS256"})` when parsing JWT tokens to enforce strict validation of the accepted algorithm natively by the library before any custom verification logic executes.
