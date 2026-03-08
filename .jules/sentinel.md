# Sentinel's Journal

## 2024-05-22 - MIME Type Spoofing
**Vulnerability:** The application relied solely on the `Content-Type` header provided by the client to determine the file type of uploaded images.
**Learning:** Trusting client headers for file type validation allows attackers to bypass restrictions by simply spoofing the header (e.g., sending an executable with `Content-Type: image/jpeg`).
**Prevention:** Always use content-based detection (e.g., `http.DetectContentType` or magic bytes inspection) to verify the actual file type before processing or storing files.

## 2024-05-23 - Timing Attack Vulnerability in Token Comparison
**Vulnerability:** The `InvitationToken.Equals` method used standard string comparison (`==`) to compare cryptographic tokens.
**Learning:** Standard string comparisons terminate early upon finding the first differing character, revealing the position of the mismatch through execution time. This allows an attacker to brute-force the token character by character.
**Prevention:** Always use `crypto/subtle.ConstantTimeCompare` (after converting strings to byte slices) for comparing any sensitive data, such as tokens, passwords, or MACs.
