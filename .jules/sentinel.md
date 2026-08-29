# Sentinel's Journal

## 2024-05-22 - MIME Type Spoofing
**Vulnerability:** The application relied solely on the `Content-Type` header provided by the client to determine the file type of uploaded images.
**Learning:** Trusting client headers for file type validation allows attackers to bypass restrictions by simply spoofing the header (e.g., sending an executable with `Content-Type: image/jpeg`).
**Prevention:** Always use content-based detection (e.g., `http.DetectContentType` or magic bytes inspection) to verify the actual file type before processing or storing files.

## 2025-03-31 - InvitationToken Timing Attack Vulnerability
**Vulnerability:** The application was using simple string equality `t.value == other.value` to verify invitation tokens, leaving it vulnerable to timing attacks.
**Learning:** Comparing sensitive tokens (like invitation codes, session tokens, or API keys) character by character can reveal partial matches based on execution time differences, allowing attackers to incrementally brute force tokens.
**Prevention:** Always convert strings to byte slices and use `crypto/subtle.ConstantTimeCompare` when evaluating secrets, even if they are lengthy, to ensure operations take a uniform amount of time regardless of match correctness.
