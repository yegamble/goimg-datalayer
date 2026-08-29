# Sentinel's Journal

## 2024-05-22 - MIME Type Spoofing
**Vulnerability:** The application relied solely on the `Content-Type` header provided by the client to determine the file type of uploaded images.
**Learning:** Trusting client headers for file type validation allows attackers to bypass restrictions by simply spoofing the header (e.g., sending an executable with `Content-Type: image/jpeg`).
**Prevention:** Always use content-based detection (e.g., `http.DetectContentType` or magic bytes inspection) to verify the actual file type before processing or storing files.

## 2024-05-22 - Timing Attack in Invitation Token Comparison
**Vulnerability:** The `InvitationToken.Equals` method used standard string comparison (`==`) to check if a provided token matched the expected token. This allows an attacker to perform a timing attack by measuring how long the comparison takes, potentially guessing the token byte-by-byte.
**Learning:** Comparing sensitive values like tokens or hashes using standard equality checks is vulnerable to timing attacks, as string comparison returns immediately upon the first non-matching byte.
**Prevention:** Always use `crypto/subtle.ConstantTimeCompare` when comparing sensitive data like tokens, passwords, or hashes to ensure the comparison time is independent of the input.
