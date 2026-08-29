# Sentinel's Journal

## 2024-05-22 - MIME Type Spoofing
**Vulnerability:** The application relied solely on the `Content-Type` header provided by the client to determine the file type of uploaded images.
**Learning:** Trusting client headers for file type validation allows attackers to bypass restrictions by simply spoofing the header (e.g., sending an executable with `Content-Type: image/jpeg`).
**Prevention:** Always use content-based detection (e.g., `http.DetectContentType` or magic bytes inspection) to verify the actual file type before processing or storing files.

## 2025-02-20 - JWT Algorithm Confusion Prevention
**Vulnerability:** JWT parsing relied only on `Keyfunc` to validate the signing method.
**Learning:** To securely prevent JWT algorithm confusion vulnerabilities, `jwt.WithValidMethods` should be used to restrict parsing algorithms directly at the library level, adding defense-in-depth alongside manual `Keyfunc` checks.
**Prevention:** Always pass `jwt.WithValidMethods([]string{"RS256"})` (or the expected algorithms) when using `jwt.Parse` or `jwt.ParseWithClaims`.
