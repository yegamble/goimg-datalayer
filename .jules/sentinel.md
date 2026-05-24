# Sentinel's Journal

## 2024-05-22 - MIME Type Spoofing
**Vulnerability:** The application relied solely on the `Content-Type` header provided by the client to determine the file type of uploaded images.
**Learning:** Trusting client headers for file type validation allows attackers to bypass restrictions by simply spoofing the header (e.g., sending an executable with `Content-Type: image/jpeg`).
**Prevention:** Always use content-based detection (e.g., `http.DetectContentType` or magic bytes inspection) to verify the actual file type before processing or storing files.

## 2024-05-23 - JWT Algorithm Confusion Vulnerability
**Vulnerability:** The application was not restricting the accepted JWT signing methods during token parsing, allowing an attacker to modify the algorithm in the header (e.g., from RS256 to HS256) and sign the token using the application's public key as an HMAC secret.
**Learning:** `jwt.Parse` and `jwt.ParseWithClaims` alone only check that the algorithm in the token header matches *some* algorithm supported by the library. They do not enforce that it's the *expected* algorithm for the specific use case, unless explicitly configured to do so.
**Prevention:** Always restrict accepted signing methods by passing `jwt.WithValidMethods([]string{"RS256"})` (or the corresponding list of valid algorithms) as a `jwt.ParserOption` to explicitly enforce the expected algorithm.
