# Sentinel's Journal

## 2024-05-22 - MIME Type Spoofing
**Vulnerability:** The application relied solely on the `Content-Type` header provided by the client to determine the file type of uploaded images.
**Learning:** Trusting client headers for file type validation allows attackers to bypass restrictions by simply spoofing the header (e.g., sending an executable with `Content-Type: image/jpeg`).
**Prevention:** Always use content-based detection (e.g., `http.DetectContentType` or magic bytes inspection) to verify the actual file type before processing or storing files.

## 2025-05-28 - [JWT Algorithm Confusion]
**Vulnerability:** JWT `ParseWithClaims` lacked explicit whitelisting of the expected signing algorithm (`RS256`).
**Learning:** Checking the token method type inside the `Keyfunc` callback (`if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok`) is insufficient if `WithValidMethods` isn't used, as the library might still evaluate the signature improperly.
**Prevention:** Always use `jwt.WithValidMethods([]string{"RS256"})` (or the specific expected algorithm) alongside checking the token method type.
