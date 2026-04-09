# Sentinel's Journal

## 2024-05-22 - MIME Type Spoofing
**Vulnerability:** The application relied solely on the `Content-Type` header provided by the client to determine the file type of uploaded images.
**Learning:** Trusting client headers for file type validation allows attackers to bypass restrictions by simply spoofing the header (e.g., sending an executable with `Content-Type: image/jpeg`).
**Prevention:** Always use content-based detection (e.g., `http.DetectContentType` or magic bytes inspection) to verify the actual file type before processing or storing files.

## 2026-04-09 - Prevent Timing Attack in InvitationToken Comparison
**Vulnerability:** The `InvitationToken.Equals` method used a naive string comparison (`==`) to compare tokens, making it vulnerable to timing attacks. An attacker could theoretically deduce the token character-by-character by observing the time taken to reject incorrect tokens.
**Learning:** Sensitive strings like invitation tokens, session IDs, and hashes should never be compared using `==`. The comment in the file literally acknowledged `crypto/subtle.ConstantTimeCompare would be ideal here` but dismissed it for simplicity, demonstrating a classic prioritization of convenience over security.
**Prevention:** Always use `crypto/subtle.ConstantTimeCompare` when comparing sensitive tokens, even if they are long enough to make brute force computationally difficult.
