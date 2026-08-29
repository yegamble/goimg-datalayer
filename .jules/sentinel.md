# Sentinel's Journal

## 2024-05-22 - MIME Type Spoofing
**Vulnerability:** The application relied solely on the `Content-Type` header provided by the client to determine the file type of uploaded images.
**Learning:** Trusting client headers for file type validation allows attackers to bypass restrictions by simply spoofing the header (e.g., sending an executable with `Content-Type: image/jpeg`).
**Prevention:** Always use content-based detection (e.g., `http.DetectContentType` or magic bytes inspection) to verify the actual file type before processing or storing files.

## 2025-05-29 - [Preventing JWT Algorithm Confusion]
**Vulnerability:** JWT `ParseWithClaims` relies only on inside-the-closure method type assertion `token.Method.(*jwt.SigningMethodRSA)`, which doesn't configure the parser itself to reject unauthorized algorithms early.
**Learning:** Always use `jwt.WithValidMethods([]string{...})` alongside the manual method type check to enforce strict algorithm acceptance at the parsing level and prevent algorithm confusion attacks where the attacker signs the public key using a symmetric algorithm.
**Prevention:** Explicitly configure `jwt.WithValidMethods([]string{"RS256"})` when parsing JWTs.
