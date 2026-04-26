# Sentinel's Journal

## 2024-05-22 - MIME Type Spoofing
**Vulnerability:** The application relied solely on the `Content-Type` header provided by the client to determine the file type of uploaded images.
**Learning:** Trusting client headers for file type validation allows attackers to bypass restrictions by simply spoofing the header (e.g., sending an executable with `Content-Type: image/jpeg`).
**Prevention:** Always use content-based detection (e.g., `http.DetectContentType` or magic bytes inspection) to verify the actual file type before processing or storing files.

## 2026-04-26 - Token Timing Attacks
**Vulnerability:** The application used simple string comparison (`==`) to compare sensitive invitation tokens in `internal/domain/community/invitation_token.go`.
**Learning:** Comparing sensitive tokens using standard string equality can be vulnerable to timing attacks, as the comparison might terminate early upon finding the first mismatch, leaking information about the token character-by-character.
**Prevention:** Always use `crypto/subtle.ConstantTimeCompare` (with the string cast to `[]byte`) to compare sensitive tokens, hashes, or credentials to prevent timing side-channels. Ensure the result is checked against `1`.
