# Sentinel's Journal

## 2024-05-22 - MIME Type Spoofing
**Vulnerability:** The application relied solely on the `Content-Type` header provided by the client to determine the file type of uploaded images.
**Learning:** Trusting client headers for file type validation allows attackers to bypass restrictions by simply spoofing the header (e.g., sending an executable with `Content-Type: image/jpeg`).
**Prevention:** Always use content-based detection (e.g., `http.DetectContentType` or magic bytes inspection) to verify the actual file type before processing or storing files.
## 2025-07-04 - Fix JWT Algorithm Confusion
**Vulnerability:** JWT algorithm confusion vulnerability due to missing jwt.WithValidMethods in jwt.ParseWithClaims.
**Learning:** Relying solely on token.Method.(*jwt.SigningMethodRSA) check inside the Keyfunc callback might not be robust enough; strictly enforcing allowed methods explicitly during parsing using standard library options provides better defense-in-depth.
**Prevention:** Always restrict accepted signing methods using jwt.WithValidMethods([]string{"RS256"}) alongside checking the token method type.
