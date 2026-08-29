# Sentinel's Journal

## 2024-05-22 - MIME Type Spoofing
**Vulnerability:** The application relied solely on the `Content-Type` header provided by the client to determine the file type of uploaded images.
**Learning:** Trusting client headers for file type validation allows attackers to bypass restrictions by simply spoofing the header (e.g., sending an executable with `Content-Type: image/jpeg`).
**Prevention:** Always use content-based detection (e.g., `http.DetectContentType` or magic bytes inspection) to verify the actual file type before processing or storing files.

## 2024-02-23 - JWT Algorithm Confusion Prevention
**Vulnerability:** JWT `ValidateToken` lacked `WithValidMethods` algorithm validation, leading to potential algorithm confusion if signature methods are incorrectly set to symmetric algorithms like HS256 when expecting asymmetric ones like RS256.
**Learning:** It's insufficient to only check `token.Method.(*jwt.SigningMethodRSA)`. We must also configure `jwt.ParseWithClaims` to strictly only accept `RS256` or the exact expected signing methods.
**Prevention:** Always use `jwt.WithValidMethods([]string{"RS256"})` (or appropriate expected algorithm) as a parsing option during `jwt.ParseWithClaims` or similar functions to lock down valid signature algorithms.
