# Sentinel's Journal

## 2024-05-22 - MIME Type Spoofing
**Vulnerability:** The application relied solely on the `Content-Type` header provided by the client to determine the file type of uploaded images.
**Learning:** Trusting client headers for file type validation allows attackers to bypass restrictions by simply spoofing the header (e.g., sending an executable with `Content-Type: image/jpeg`).
**Prevention:** Always use content-based detection (e.g., `http.DetectContentType` or magic bytes inspection) to verify the actual file type before processing or storing files.

## 2026-05-22 - JWT Algorithm Confusion Vulnerability
**Vulnerability:** The application was not explicitly restricting accepted JWT signing methods in the `jwt.ParseWithClaims` configuration, potentially allowing attackers to supply tokens signed with a different algorithm (e.g., HMAC instead of RSA), which is known as JWT Algorithm Confusion.
**Learning:** Only relying on type assertions in the key callback (e.g., `_, ok := token.Method.(*jwt.SigningMethodRSA)`) can sometimes be bypassed or fail safely depending on library implementations and edge cases.
**Prevention:** Always restrict accepted signing methods by appending `jwt.WithValidMethods([]string{"RS256"})` (or the specific expected algorithm) to `jwt.Parse` or `jwt.ParseWithClaims`.
