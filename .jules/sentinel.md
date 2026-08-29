# Sentinel's Journal

## 2024-05-22 - MIME Type Spoofing
**Vulnerability:** The application relied solely on the `Content-Type` header provided by the client to determine the file type of uploaded images.
**Learning:** Trusting client headers for file type validation allows attackers to bypass restrictions by simply spoofing the header (e.g., sending an executable with `Content-Type: image/jpeg`).
**Prevention:** Always use content-based detection (e.g., `http.DetectContentType` or magic bytes inspection) to verify the actual file type before processing or storing files.

## 2026-02-05 - Timing Attacks in Invitation Token Comparison
**Vulnerability:** The `InvitationToken.Equals` method in `internal/domain/community/invitation_token.go` used a simple string comparison (`t.value == other.value`) to compare cryptographically secure tokens.
**Learning:** Using standard string comparison operators (`==`) on sensitive data like tokens or passwords introduces timing attack vulnerabilities. Attackers can deduce the correct token character by character by measuring the exact time the comparison takes, as standard comparisons fail immediately upon finding the first mismatched character.
**Prevention:** To prevent timing attacks, always use `crypto/subtle.ConstantTimeCompare` after converting the sensitive strings to byte slices when validating or comparing sensitive tokens, secrets, or passwords. This ensures the comparison takes the same amount of time regardless of how many characters match.
