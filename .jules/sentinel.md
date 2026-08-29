# Sentinel's Journal

## 2024-05-22 - MIME Type Spoofing
**Vulnerability:** The application relied solely on the `Content-Type` header provided by the client to determine the file type of uploaded images.
**Learning:** Trusting client headers for file type validation allows attackers to bypass restrictions by simply spoofing the header (e.g., sending an executable with `Content-Type: image/jpeg`).
**Prevention:** Always use content-based detection (e.g., `http.DetectContentType` or magic bytes inspection) to verify the actual file type before processing or storing files.

## 2024-06-12 - Timing Attack via Non-Constant Time Comparison
**Vulnerability:** The application used simple string comparison (`==`) for verifying sensitive tokens such as InvitationTokens and DeviceFingerprints.
**Learning:** Standard string comparison evaluates character by character and returns early on a mismatch, allowing attackers to infer the expected token byte-by-byte by observing the time taken to respond.
**Prevention:** Always use `crypto/subtle.ConstantTimeCompare` when comparing sensitive data like tokens, hashes, or passwords to prevent timing attacks. Ensure both arguments are cast to `[]byte` and explicitly check that the function returns `1` for a successful match.
