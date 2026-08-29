# Sentinel's Journal

## 2024-05-22 - MIME Type Spoofing
**Vulnerability:** The application relied solely on the `Content-Type` header provided by the client to determine the file type of uploaded images.
**Learning:** Trusting client headers for file type validation allows attackers to bypass restrictions by simply spoofing the header (e.g., sending an executable with `Content-Type: image/jpeg`).
**Prevention:** Always use content-based detection (e.g., `http.DetectContentType` or magic bytes inspection) to verify the actual file type before processing or storing files.

## 2025-02-28 - [Invitation Token Timing Attack]
**Vulnerability:** The `InvitationToken.Equals` method in `internal/domain/community/invitation_token.go` used basic string comparison (`==`) to check if a token equals another token. This introduces a vulnerability to timing attacks, as string comparison checks byte-by-byte and exits early if a mismatch is found.
**Learning:** Even if the token string length is long, all cryptographic or sensitive tokens should use constant-time comparison. We were doing this with other parts like `subtle.ConstantTimeCompare(expectedHash, actualHash)` but it was missing here.
**Prevention:** Always use `crypto/subtle.ConstantTimeCompare` after converting the values to byte slices when comparing secrets.
