# Sentinel's Journal

## 2024-05-22 - MIME Type Spoofing
**Vulnerability:** The application relied solely on the `Content-Type` header provided by the client to determine the file type of uploaded images.
**Learning:** Trusting client headers for file type validation allows attackers to bypass restrictions by simply spoofing the header (e.g., sending an executable with `Content-Type: image/jpeg`).
**Prevention:** Always use content-based detection (e.g., `http.DetectContentType` or magic bytes inspection) to verify the actual file type before processing or storing files.

## 2025-02-18 - JWT Algorithm Confusion Prevention
**Vulnerability:** JWT `ParseWithClaims` relied solely on the Keyfunc type assertion to check the signing method, which can be insufficient if algorithms are not strictly limited.
**Learning:** Explicitly restricting allowed signing methods using `jwt.WithValidMethods` at the parser level provides defense-in-depth and prevents algorithm confusion attacks before the Keyfunc is even evaluated.
**Prevention:** Always pass `jwt.WithValidMethods([]string{"RS256"})` (or the expected algorithm) as an option to `jwt.ParseWithClaims`.
