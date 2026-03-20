# Sentinel's Journal

## 2024-05-23 - Timing Attack on Invitation Tokens
**Vulnerability:** The application was using simple string comparison (`==`) to validate sensitive invitation tokens in `internal/domain/community/invitation_token.go`.
**Learning:** Standard string comparison operators fail fast (short-circuiting on the first non-matching byte), creating a timing attack vector where an attacker can systematically guess the token by measuring the time taken for comparison.
**Prevention:** Always use `crypto/subtle.ConstantTimeCompare` to validate sensitive data like tokens, passwords, and hashes. Note that the inputs must be converted to `[]byte` first, and the result checked explicitly against `1`.

## 2024-05-22 - MIME Type Spoofing
**Vulnerability:** The application relied solely on the `Content-Type` header provided by the client to determine the file type of uploaded images.
**Learning:** Trusting client headers for file type validation allows attackers to bypass restrictions by simply spoofing the header (e.g., sending an executable with `Content-Type: image/jpeg`).
**Prevention:** Always use content-based detection (e.g., `http.DetectContentType` or magic bytes inspection) to verify the actual file type before processing or storing files.
