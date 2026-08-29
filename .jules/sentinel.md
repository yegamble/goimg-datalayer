# Sentinel's Journal

## 2024-05-22 - MIME Type Spoofing
**Vulnerability:** The application relied solely on the `Content-Type` header provided by the client to determine the file type of uploaded images.
**Learning:** Trusting client headers for file type validation allows attackers to bypass restrictions by simply spoofing the header (e.g., sending an executable with `Content-Type: image/jpeg`).
**Prevention:** Always use content-based detection (e.g., `http.DetectContentType` or magic bytes inspection) to verify the actual file type before processing or storing files.

## 2024-11-05 - Timing Attacks on Tokens
**Vulnerability:** The `Equals` method of `InvitationToken` used regular string comparison (`==`) which short-circuits on the first mismatched character.
**Learning:** Regular string comparison leaks timing information. An attacker could measure the time taken for comparisons and guess the token character by character, bypassing its cryptographic security, despite its length.
**Prevention:** Always use `crypto/subtle.ConstantTimeCompare` for comparing secrets or cryptographic tokens to prevent timing attacks.