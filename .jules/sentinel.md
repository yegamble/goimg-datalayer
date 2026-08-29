# Sentinel's Journal

## 2024-05-22 - MIME Type Spoofing
**Vulnerability:** The application relied solely on the `Content-Type` header provided by the client to determine the file type of uploaded images.
**Learning:** Trusting client headers for file type validation allows attackers to bypass restrictions by simply spoofing the header (e.g., sending an executable with `Content-Type: image/jpeg`).
**Prevention:** Always use content-based detection (e.g., `http.DetectContentType` or magic bytes inspection) to verify the actual file type before processing or storing files.

## 2025-02-27 - JWT Algorithm Confusion Vulnerability
**Vulnerability:** JWT parsing logic allowed potential algorithm confusion by not explicitly restricting the accepted signing algorithms via `jwt.WithValidMethods(...)`. While the method type was checked inline, an explicit restriction adds defense in depth against issues like HMAC being accepted as RSA ("None" attack).
**Learning:** Checking the token method type inline is insufficient protection. The underlying JWT library should be explicitly constrained to only accept expected algorithms, completely rejecting tokens with spoofed headers early in the parsing phase.
**Prevention:** Always use `jwt.WithValidMethods([]string{"RS256"})` (or the specific expected algorithm) when using `jwt.Parse` or `jwt.ParseWithClaims`.
