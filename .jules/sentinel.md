# Sentinel's Journal

## 2024-05-22 - MIME Type Spoofing
**Vulnerability:** The application relied solely on the `Content-Type` header provided by the client to determine the file type of uploaded images.
**Learning:** Trusting client headers for file type validation allows attackers to bypass restrictions by simply spoofing the header (e.g., sending an executable with `Content-Type: image/jpeg`).
**Prevention:** Always use content-based detection (e.g., `http.DetectContentType` or magic bytes inspection) to verify the actual file type before processing or storing files.

## 2025-03-09 - JWT Algorithm Confusion Prevention
**Vulnerability:** The JWT parsing logic checked the signing method type within the key function but did not explicitly restrict the allowed algorithms using `jwt.WithValidMethods`.
**Learning:** Relying solely on manual type checks in the key function for JWT validation is less robust than using the library's built-in algorithm restriction (`jwt.WithValidMethods`), which enforces the algorithm at the parsing level and provides an extra layer of defense against algorithm confusion attacks.
**Prevention:** Always use `jwt.WithValidMethods([]string{"RS256"})` (or the appropriate expected algorithm) when parsing JWTs to strictly define accepted signing methods.
