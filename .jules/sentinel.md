# Sentinel's Journal

## 2024-05-22 - MIME Type Spoofing
**Vulnerability:** The application relied solely on the `Content-Type` header provided by the client to determine the file type of uploaded images.
**Learning:** Trusting client headers for file type validation allows attackers to bypass restrictions by simply spoofing the header (e.g., sending an executable with `Content-Type: image/jpeg`).
**Prevention:** Always use content-based detection (e.g., `http.DetectContentType` or magic bytes inspection) to verify the actual file type before processing or storing files.

## 2025-06-08 - [JWT Algorithm Confusion Fix]
**Vulnerability:** The application parsed JWTs without enforcing the expected signing algorithm (RS256) at the parsing library level (`jwt.ParseWithClaims`), making it vulnerable to algorithm confusion attacks where an attacker could forge a token by changing the header to HS256.
**Learning:** Always use `jwt.WithValidMethods` in `github.com/golang-jwt/jwt` to restrict accepted signing algorithms at the parser level, rather than relying solely on type assertions later in the callback.
**Prevention:** Ensure all `jwt.Parse` or `jwt.ParseWithClaims` calls include `jwt.WithValidMethods([]string{"<EXPECTED_ALGO>"})`.
