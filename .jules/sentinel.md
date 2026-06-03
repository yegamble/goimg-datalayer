# Sentinel's Journal

## 2024-05-22 - MIME Type Spoofing
**Vulnerability:** The application relied solely on the `Content-Type` header provided by the client to determine the file type of uploaded images.
**Learning:** Trusting client headers for file type validation allows attackers to bypass restrictions by simply spoofing the header (e.g., sending an executable with `Content-Type: image/jpeg`).
**Prevention:** Always use content-based detection (e.g., `http.DetectContentType` or magic bytes inspection) to verify the actual file type before processing or storing files.
## 2026-06-03 - JWT Algorithm Confusion Vulnerability
**Vulnerability:** The application parsed JWT tokens using a custom `Keyfunc` that checked the token header algorithm but failed to whitelist the specific `ValidMethods` during the `jwt.Parse` process, which could allow an attacker to forge tokens by specifying `HS256` and using the exposed RSA public key as the HMAC secret.
**Learning:** `golang-jwt/jwt` v5 requires you to explicitly use `jwt.WithValidMethods([]string{"RS256"})` as an option to enforce strict algorithm whitelisting; merely inspecting `token.Method` inside the `Keyfunc` is insufficient defense in depth against algorithm confusion.
**Prevention:** Always strictly declare the expected algorithms during JWT parsing operations by supplying `jwt.WithValidMethods([]string{"ExpectedAlg"})` directly to the parser function.
