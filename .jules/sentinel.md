# Sentinel's Journal

## 2024-05-22 - MIME Type Spoofing
**Vulnerability:** The application relied solely on the `Content-Type` header provided by the client to determine the file type of uploaded images.
**Learning:** Trusting client headers for file type validation allows attackers to bypass restrictions by simply spoofing the header (e.g., sending an executable with `Content-Type: image/jpeg`).
**Prevention:** Always use content-based detection (e.g., `http.DetectContentType` or magic bytes inspection) to verify the actual file type before processing or storing files.

## 2026-05-12 - Timing Attack on Invitation Tokens
**Vulnerability:** The application used standard string comparison (`t.value == other.value`) to verify invitation tokens.
**Learning:** String comparison fails early when it encounters the first non-matching character, allowing an attacker to determine if a token is valid byte-by-byte using timing measurements.
**Prevention:** Always use `crypto/subtle.ConstantTimeCompare` when comparing tokens, secrets, hashes, or any sensitive data against user input.
