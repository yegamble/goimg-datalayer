# Sentinel's Journal

## 2024-05-22 - MIME Type Spoofing
**Vulnerability:** The application relied solely on the `Content-Type` header provided by the client to determine the file type of uploaded images.
**Learning:** Trusting client headers for file type validation allows attackers to bypass restrictions by simply spoofing the header (e.g., sending an executable with `Content-Type: image/jpeg`).
**Prevention:** Always use content-based detection (e.g., `http.DetectContentType` or magic bytes inspection) to verify the actual file type before processing or storing files.

## 2024-05-23 - Timing Attack on Invitation Tokens
**Vulnerability:** The application used standard string comparison (`==`) for verifying invitation tokens.
**Learning:** String comparison short-circuits on the first mismatched character, allowing an attacker to iteratively guess a valid token by measuring the response time (timing attack).
**Prevention:** Always use `crypto/subtle.ConstantTimeCompare` for security-sensitive comparisons like tokens, hashes, or API keys to ensure comparison time is independent of the input contents.
