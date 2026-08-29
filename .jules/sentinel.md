# Sentinel's Journal

## 2024-05-22 - MIME Type Spoofing
**Vulnerability:** The application relied solely on the `Content-Type` header provided by the client to determine the file type of uploaded images.
**Learning:** Trusting client headers for file type validation allows attackers to bypass restrictions by simply spoofing the header (e.g., sending an executable with `Content-Type: image/jpeg`).
**Prevention:** Always use content-based detection (e.g., `http.DetectContentType` or magic bytes inspection) to verify the actual file type before processing or storing files.
## 2025-02-18 - JWT Algorithm Confusion Prevention
**Vulnerability:** The JWT validation in `jwt_service.go` checked the signing method type within the key validation function, but `jwt.ParseWithClaims` itself was not strictly restricted by passing parser options, making it vulnerable to certain variations of algorithm confusion.
**Learning:** Checking the token method inside the keyfunc isn't always enough to prevent the algorithm confusion attack early in the JWT parsing lifecycle, especially since `github.com/golang-jwt/jwt/v5` supports an explicit `jwt.WithValidMethods` option that completely rejects unsupported algorithms before key retrieval logic is executed.
**Prevention:** Always use `jwt.WithValidMethods([]string{"RS256"})` (or the specific expected algorithm) alongside checking the token method type when parsing tokens to provide defense-in-depth against algorithm confusion attacks.
