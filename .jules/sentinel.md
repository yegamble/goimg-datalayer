# Sentinel's Journal

## 2024-05-22 - MIME Type Spoofing
**Vulnerability:** The application relied solely on the `Content-Type` header provided by the client to determine the file type of uploaded images.
**Learning:** Trusting client headers for file type validation allows attackers to bypass restrictions by simply spoofing the header (e.g., sending an executable with `Content-Type: image/jpeg`).
**Prevention:** Always use content-based detection (e.g., `http.DetectContentType` or magic bytes inspection) to verify the actual file type before processing or storing files.
## 2026-04-18 - Prevent Timing Attacks in Token Comparisons
**Vulnerability:** Timing attack vulnerability in `InvitationToken.Equals` where sensitive tokens were compared using standard string equality (`==`).
**Learning:** Even if tokens are long and assumed to make brute force infeasible, standard string comparison leaks timing information and violates defense-in-depth security principles for sensitive cryptographic material.
**Prevention:** Always use `crypto/subtle.ConstantTimeCompare` instead of `==` when comparing hashes, tokens, or other sensitive secrets. Convert strings to `[]byte` and explicitly verify the result is `1`.
