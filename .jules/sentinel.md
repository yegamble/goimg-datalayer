# Sentinel's Journal

## 2024-05-22 - MIME Type Spoofing
**Vulnerability:** The application relied solely on the `Content-Type` header provided by the client to determine the file type of uploaded images.
**Learning:** Trusting client headers for file type validation allows attackers to bypass restrictions by simply spoofing the header (e.g., sending an executable with `Content-Type: image/jpeg`).
**Prevention:** Always use content-based detection (e.g., `http.DetectContentType` or magic bytes inspection) to verify the actual file type before processing or storing files.

## 2025-02-05 - JWT Algorithm Confusion
**Vulnerability:** JWT algorithm confusion vulnerability allowing arbitrary signature verification by changing the `alg` header.
**Learning:** Relying solely on `token.Method.(*jwt.SigningMethodRSA)` type assertion during parsing is insufficient, as it occurs *after* the parsing library might have already accepted and validated a symmetric token (e.g., using `HS256`) against the public key file if not explicitly restricted, because the public key can sometimes be coerced into a valid HMAC secret.
**Prevention:** Always explicitly enforce the expected signing method(s) using `jwt.WithValidMethods(...)` directly in the `Parse` function call to prevent the library from even attempting to validate using an unexpected algorithm.
