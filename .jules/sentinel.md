# Sentinel's Journal

## 2024-05-22 - MIME Type Spoofing
**Vulnerability:** The application relied solely on the `Content-Type` header provided by the client to determine the file type of uploaded images.
**Learning:** Trusting client headers for file type validation allows attackers to bypass restrictions by simply spoofing the header (e.g., sending an executable with `Content-Type: image/jpeg`).
**Prevention:** Always use content-based detection (e.g., `http.DetectContentType` or magic bytes inspection) to verify the actual file type before processing or storing files.

## 2024-06-28 - JWT Algorithm Confusion Vulnerability
**Vulnerability:** The JWT parsing logic in `jwt_service.go` did not strictly enforce the allowed signing methods, making it susceptible to algorithm confusion attacks where an attacker could forge tokens using symmetric algorithms (e.g., HS256) instead of the expected asymmetric algorithm (RS256).
**Learning:** Relying solely on type assertions inside the Keyfunc (e.g., `token.Method.(*jwt.SigningMethodRSA)`) is insufficient in `golang-jwt/jwt` because it doesn't prevent the library from attempting to verify the token with an unexpected algorithm before or during the Parse phase if options aren't strictly provided.
**Prevention:** Always restrict accepted signing methods when parsing JWTs by explicitly providing `jwt.WithValidMethods([]string{"RS256"})` (or the specific expected algorithms) to the `jwt.Parse` or `jwt.ParseWithClaims` functions.
