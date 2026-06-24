# Sentinel's Journal

## 2024-05-22 - MIME Type Spoofing
**Vulnerability:** The application relied solely on the `Content-Type` header provided by the client to determine the file type of uploaded images.
**Learning:** Trusting client headers for file type validation allows attackers to bypass restrictions by simply spoofing the header (e.g., sending an executable with `Content-Type: image/jpeg`).
**Prevention:** Always use content-based detection (e.g., `http.DetectContentType` or magic bytes inspection) to verify the actual file type before processing or storing files.

## 2025-06-24 - JWT Algorithm Confusion Vulnerability
**Vulnerability:** The application parsed JWTs using `jwt.ParseWithClaims` without restricting the allowed signing methods, making it susceptible to accepting tokens signed with a symmetric algorithm (like HS256) when an asymmetric algorithm (RS256) was expected.
**Learning:** If the library allows fallback to symmetric signing dynamically based on the token's header `alg`, an attacker could forge a token by signing it symmetrically using the application's *public* key as the secret.
**Prevention:** Always explicitly define the allowed signing methods during token validation by supplying `jwt.WithValidMethods([]string{"RS256"})` (or the appropriate algorithm) to the parsing function.
