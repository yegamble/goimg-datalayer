# Sentinel's Journal

## 2024-05-22 - MIME Type Spoofing
**Vulnerability:** The application relied solely on the `Content-Type` header provided by the client to determine the file type of uploaded images.
**Learning:** Trusting client headers for file type validation allows attackers to bypass restrictions by simply spoofing the header (e.g., sending an executable with `Content-Type: image/jpeg`).
**Prevention:** Always use content-based detection (e.g., `http.DetectContentType` or magic bytes inspection) to verify the actual file type before processing or storing files.
## 2026-05-05 - Constant-time comparison for token equality
**Vulnerability:** The InvitationToken Equals method used standard string equality (==) to compare secure tokens.
**Learning:** Standard string equality checks return as soon as a mismatch is found, enabling timing attacks where an attacker can determine the token character by character by measuring the response time.
**Prevention:** Always use `crypto/subtle.ConstantTimeCompare` when comparing security-sensitive strings, tokens, or hashes to ensure the comparison time is independent of the input values.
