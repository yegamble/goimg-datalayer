# Sentinel's Journal

## 2024-05-22 - MIME Type Spoofing
**Vulnerability:** The application relied solely on the `Content-Type` header provided by the client to determine the file type of uploaded images.
**Learning:** Trusting client headers for file type validation allows attackers to bypass restrictions by simply spoofing the header (e.g., sending an executable with `Content-Type: image/jpeg`).
**Prevention:** Always use content-based detection (e.g., `http.DetectContentType` or magic bytes inspection) to verify the actual file type before processing or storing files.

## 2026-05-30 - JWT Algorithm Confusion Vulnerability
**Vulnerability:** The application was not restricting the accepted signing methods when parsing JWT tokens, potentially allowing attackers to forge tokens by changing the algorithm to HS256 (symmetric) and signing them with the application's public key (if known).
**Learning:** Trusting the `alg` header in a JWT without strict validation of the expected algorithm allows for algorithm confusion attacks.
**Prevention:** Always restrict accepted signing methods when parsing JWTs using `jwt.WithValidMethods([]string{"RS256"})` (or the specific expected algorithm).
