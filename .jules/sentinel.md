## 2024-05-22 - MIME Type Spoofing
**Vulnerability:** The application relied solely on the `Content-Type` header provided by the client to determine the file type of uploaded images.
**Learning:** Trusting client headers for file type validation allows attackers to bypass restrictions by simply spoofing the header (e.g., sending an executable with `Content-Type: image/jpeg`).
**Prevention:** Always use content-based detection (e.g., `http.DetectContentType` or magic bytes inspection) to verify the actual file type before processing or storing files.

## 2024-05-23 - Timing Attack in Token Comparison
**Vulnerability:** The `InvitationToken.Equals` method used standard string comparison (`==`) to verify tokens.
**Learning:** Standard string comparison in Go terminates early as soon as a mismatched character is found. This leaks information about how much of the token matched, allowing attackers to perform timing attacks and theoretically brute-force tokens byte by byte, even for 64-character hex strings.
**Prevention:** Always use `crypto/subtle.ConstantTimeCompare` when comparing sensitive secrets (tokens, hashes, API keys). Convert strings to byte slices first: `subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1`.
