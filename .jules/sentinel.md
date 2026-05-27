# Sentinel's Journal

## 2024-05-22 - MIME Type Spoofing
**Vulnerability:** The application relied solely on the `Content-Type` header provided by the client to determine the file type of uploaded images.
**Learning:** Trusting client headers for file type validation allows attackers to bypass restrictions by simply spoofing the header (e.g., sending an executable with `Content-Type: image/jpeg`).
**Prevention:** Always use content-based detection (e.g., `http.DetectContentType` or magic bytes inspection) to verify the actual file type before processing or storing files.
## 2024-05-27 - JWT Algorithm Confusion Vulnerability
**Vulnerability:** The application parsed JWT tokens without explicitly verifying that the signing method used matched the expected signing method (`RS256`).
**Learning:** Relying solely on the `token.Method.(*jwt.SigningMethodRSA)` typecast inside the `Keyfunc` callback leaves a small but theoretical risk depending on the underlying JWT library's implementation details. Attackers could potentially spoof tokens by altering the header (e.g., using `HS256` and forcing the server to evaluate a public key as an HMAC secret).
**Prevention:** Always use `jwt.WithValidMethods([]string{"RS256"})` to strictly define the acceptable signing algorithms when parsing JWT tokens.
