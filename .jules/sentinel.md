# Sentinel's Journal

## 2024-05-22 - MIME Type Spoofing
**Vulnerability:** The application relied solely on the `Content-Type` header provided by the client to determine the file type of uploaded images.
**Learning:** Trusting client headers for file type validation allows attackers to bypass restrictions by simply spoofing the header (e.g., sending an executable with `Content-Type: image/jpeg`).
**Prevention:** Always use content-based detection (e.g., `http.DetectContentType` or magic bytes inspection) to verify the actual file type before processing or storing files.

## 2026-05-06 - Timing Attacks in Invitation Tokens
**Vulnerability:** Comparing invitation tokens using simple string equality (`==`) allows attackers to use timing analysis to guess valid tokens character by character.
**Learning:** Any sensitive token, hash, or password must use constant-time comparison to protect against side-channel timing attacks.
**Prevention:** Always use `crypto/subtle.ConstantTimeCompare` when comparing tokens, password hashes, or API keys. Ensure arguments are cast to `[]byte` and explicitly check that the function returns `1` for a match.
