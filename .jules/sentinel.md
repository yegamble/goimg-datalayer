# Sentinel's Journal

## 2024-05-22 - MIME Type Spoofing
**Vulnerability:** The application relied solely on the `Content-Type` header provided by the client to determine the file type of uploaded images.
**Learning:** Trusting client headers for file type validation allows attackers to bypass restrictions by simply spoofing the header (e.g., sending an executable with `Content-Type: image/jpeg`).
**Prevention:** Always use content-based detection (e.g., `http.DetectContentType` or magic bytes inspection) to verify the actual file type before processing or storing files.

## 2025-03-09 - JWT Algorithm Confusion Vulnerability
**Vulnerability:** The JWT validation logic did not explicitly restrict the allowed signing methods during parsing, relying only on a check inside the keyfunc.
**Learning:** Attackers can exploit this by changing the algorithm header to HS256 and signing the token symmetrically using the server's public key, bypassing the intended RSA verification.
**Prevention:** Always use `jwt.NewParser(jwt.WithValidMethods([]string{"RS256"})).Parse...` to explicitly enforce allowed signing algorithms at the parser level before the keyfunc is executed.
