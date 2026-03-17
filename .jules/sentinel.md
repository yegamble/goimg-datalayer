# Sentinel's Journal

## 2024-05-22 - MIME Type Spoofing
**Vulnerability:** The application relied solely on the `Content-Type` header provided by the client to determine the file type of uploaded images.
**Learning:** Trusting client headers for file type validation allows attackers to bypass restrictions by simply spoofing the header (e.g., sending an executable with `Content-Type: image/jpeg`).
**Prevention:** Always use content-based detection (e.g., `http.DetectContentType` or magic bytes inspection) to verify the actual file type before processing or storing files.

## 2025-05-22 - Timing attacks in string comparisons
**Vulnerability:** The domain layer's string comparison for sensitive data such as tokens (e.g., `InvitationToken.Equals`) used the standard `==` operator, allowing for potential timing attacks.
**Learning:** Using standard string comparison operators on sensitive data like tokens introduces timing attack vulnerabilities, where attackers can deduce the string based on the time it takes to process the comparison.
**Prevention:** Always use `crypto/subtle.ConstantTimeCompare` after converting strings to byte slices for sensitive data comparison. Avoid standard equality `==` or inequality `!=` operators.
