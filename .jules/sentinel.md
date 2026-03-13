# Sentinel's Journal

## 2024-05-22 - MIME Type Spoofing
**Vulnerability:** The application relied solely on the `Content-Type` header provided by the client to determine the file type of uploaded images.
**Learning:** Trusting client headers for file type validation allows attackers to bypass restrictions by simply spoofing the header (e.g., sending an executable with `Content-Type: image/jpeg`).
**Prevention:** Always use content-based detection (e.g., `http.DetectContentType` or magic bytes inspection) to verify the actual file type before processing or storing files.
## 2024-03-24 - Invitation Token Timing Attack Vulnerability
**Vulnerability:** The `InvitationToken.Equals` method in `internal/domain/community/invitation_token.go` used a standard string comparison (`==`) for comparing cryptographically secure tokens. This introduced a timing attack vulnerability, potentially allowing an attacker to brute-force or deduce tokens by observing subtle differences in response times during comparison.
**Learning:** Even when tokens are long (e.g., 64 characters) and brute-forcing is considered infeasible, any cryptographically sensitive string comparison should use constant-time operations to adhere to defense-in-depth principles. The comment in the codebase itself acknowledged that `crypto/subtle.ConstantTimeCompare` would be ideal but opted for string comparison for "simplicity", which is a dangerous trade-off in security-sensitive domains.
**Prevention:** Always use `subtle.ConstantTimeCompare` for comparing sensitive tokens, secrets, or hashes. Ensure strings are converted to byte slices (`[]byte`) before passing them to the function, and explicitly check if the result equals `1` to confirm a match.
