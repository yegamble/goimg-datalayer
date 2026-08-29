# Sentinel's Journal

## 2024-05-22 - MIME Type Spoofing
**Vulnerability:** The application relied solely on the `Content-Type` header provided by the client to determine the file type of uploaded images.
**Learning:** Trusting client headers for file type validation allows attackers to bypass restrictions by simply spoofing the header (e.g., sending an executable with `Content-Type: image/jpeg`).
**Prevention:** Always use content-based detection (e.g., `http.DetectContentType` or magic bytes inspection) to verify the actual file type before processing or storing files.

## 2026-05-16 - Timing attack on invitation token comparison
**Vulnerability:** The `InvitationToken.Equals` function used a simple `==` string comparison instead of `crypto/subtle.ConstantTimeCompare`, opening the potential for timing attacks.
**Learning:** Even if tokens are long (e.g. 64 hex characters), using regular string equality exposes the system to theoretical timing attacks because standard string comparison fails fast.
**Prevention:** Always use `crypto/subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1` when comparing sensitive tokens, hashes, or passwords to prevent information leakage through execution time differences.

## 2026-05-16 - JWT Algorithm Confusion Vulnerability
**Vulnerability:** The `ValidateToken` function in `jwt_service.go` checked if the method was RSA, but the JWT parser was not restricted by `jwt.WithValidMethods`, allowing potential algorithm confusion attacks (e.g., using HMAC to sign a token but forcing the server to verify it with a public RSA key acting as an HMAC secret).
**Learning:** Relying solely on `token.Method.(*jwt.SigningMethodRSA)` during parsing is insufficient if `ParseWithClaims` accepts multiple signing methods by default, as the `keyFunc` might be abused before type assertion.
**Prevention:** Always restrict accepted signing methods explicitly using `jwt.WithValidMethods([]string{"RS256"})` when parsing tokens.
