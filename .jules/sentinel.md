# Sentinel's Journal

## 2024-05-22 - MIME Type Spoofing
**Vulnerability:** The application relied solely on the `Content-Type` header provided by the client to determine the file type of uploaded images.
**Learning:** Trusting client headers for file type validation allows attackers to bypass restrictions by simply spoofing the header (e.g., sending an executable with `Content-Type: image/jpeg`).
**Prevention:** Always use content-based detection (e.g., `http.DetectContentType` or magic bytes inspection) to verify the actual file type before processing or storing files.
## 2025-02-05 - Fix JWT Algorithm Confusion Vulnerability
**Vulnerability:** The application accepted JWT tokens signed with a symmetric algorithm (HS256) instead of enforcing the required asymmetric algorithm (RS256).
**Learning:** This occurred because `jwt.ParseWithClaims` wasn't explicitly restricted to only accept the expected algorithm, allowing a malicious actor to bypass signature validation by changing the header to HS256 and signing the token using the public key as the symmetric secret.
**Prevention:** Always restrict accepted signing methods when parsing JWTs using `jwt.WithValidMethods([]string{"RS256"})` alongside checking the token method type.
