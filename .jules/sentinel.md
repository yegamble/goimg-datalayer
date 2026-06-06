# Sentinel's Journal

## 2024-05-22 - MIME Type Spoofing
**Vulnerability:** The application relied solely on the `Content-Type` header provided by the client to determine the file type of uploaded images.
**Learning:** Trusting client headers for file type validation allows attackers to bypass restrictions by simply spoofing the header (e.g., sending an executable with `Content-Type: image/jpeg`).
**Prevention:** Always use content-based detection (e.g., `http.DetectContentType` or magic bytes inspection) to verify the actual file type before processing or storing files.

## 2024-06-06 - JWT Algorithm Confusion Vulnerability
**Vulnerability:** The application was not enforcing the expected signing method (RS256) when parsing JWT tokens, making it vulnerable to algorithm confusion attacks where an attacker could use the public key as an HMAC secret.
**Learning:** Relying solely on the algorithm specified in the token header allows attackers to bypass signature validation.
**Prevention:** Always use `jwt.WithValidMethods` to explicitly define the allowed signing methods during token parsing.
