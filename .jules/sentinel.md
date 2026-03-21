# Sentinel's Journal

## 2024-05-22 - MIME Type Spoofing
**Vulnerability:** The application relied solely on the `Content-Type` header provided by the client to determine the file type of uploaded images.
**Learning:** Trusting client headers for file type validation allows attackers to bypass restrictions by simply spoofing the header (e.g., sending an executable with `Content-Type: image/jpeg`).
**Prevention:** Always use content-based detection (e.g., `http.DetectContentType` or magic bytes inspection) to verify the actual file type before processing or storing files.

## 2026-03-21 - Timing Attack in Token Comparison
**Vulnerability:** The `InvitationToken.Equals` method used standard string comparison (`==`) which leaks information through timing variations, allowing an attacker to theoretically deduce a token character by character.
**Learning:** In the domain layer, sensitive string comparisons (like token validation) must be resistant to timing attacks, regardless of the token length.
**Prevention:** Always use `crypto/subtle.ConstantTimeCompare` after converting strings to byte slices for any sensitive token or secret comparison.
