# Sentinel's Journal

## 2024-05-22 - MIME Type Spoofing
**Vulnerability:** The application relied solely on the `Content-Type` header provided by the client to determine the file type of uploaded images.
**Learning:** Trusting client headers for file type validation allows attackers to bypass restrictions by simply spoofing the header (e.g., sending an executable with `Content-Type: image/jpeg`).
**Prevention:** Always use content-based detection (e.g., `http.DetectContentType` or magic bytes inspection) to verify the actual file type before processing or storing files.

## 2024-05-22 - Timing Attack in Invitation Token Comparison
**Vulnerability:** The `InvitationToken.Equals` method used standard string comparison (`==`) instead of constant-time comparison.
**Learning:** Even for randomly generated, sufficiently long tokens (like invitation tokens), using standard string comparison can expose the application to timing attacks where an attacker might deduce the token character by character.
**Prevention:** Always use `subtle.ConstantTimeCompare` when comparing sensitive tokens, secrets, or hashes to ensure comparison time does not leak information about the expected value.
