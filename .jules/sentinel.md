# Sentinel's Journal

## 2024-05-22 - MIME Type Spoofing
**Vulnerability:** The application relied solely on the `Content-Type` header provided by the client to determine the file type of uploaded images.
**Learning:** Trusting client headers for file type validation allows attackers to bypass restrictions by simply spoofing the header (e.g., sending an executable with `Content-Type: image/jpeg`).
**Prevention:** Always use content-based detection (e.g., `http.DetectContentType` or magic bytes inspection) to verify the actual file type before processing or storing files.

## 2024-05-18 - [JWT Algorithm Confusion]
**Vulnerability:** JWT tokens were parsed without explicitly restricting the allowed signing methods via `jwt.WithValidMethods`. An attacker could sign a token with a symmetric algorithm (HS256) using the public key, bypassing validation if the key type isn't strictly verified by the underlying library logic.
**Learning:** While the `keyFunc` in `jwt.ParseWithClaims` checked the `token.Method` type, this check happens *after* the parsing phase. Relying solely on `keyFunc` checks is insufficient as it is a defense mechanism executed later in the process. Explicitly providing `jwt.WithValidMethods` enforces algorithm restrictions at the parser level, preventing algorithm confusion earlier in the validation pipeline.
**Prevention:** Always use `jwt.WithValidMethods([]string{...})` alongside `jwt.Parse` or `jwt.ParseWithClaims` to strictly define the expected signing algorithms.
