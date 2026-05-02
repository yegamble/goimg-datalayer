# Sentinel's Journal

## 2024-05-22 - MIME Type Spoofing
**Vulnerability:** The application relied solely on the `Content-Type` header provided by the client to determine the file type of uploaded images.
**Learning:** Trusting client headers for file type validation allows attackers to bypass restrictions by simply spoofing the header (e.g., sending an executable with `Content-Type: image/jpeg`).
**Prevention:** Always use content-based detection (e.g., `http.DetectContentType` or magic bytes inspection) to verify the actual file type before processing or storing files.

## 2024-05-23 - Timing Attack on Invitation Token
**Vulnerability:** The application used string comparison `==` to compare `InvitationToken` equality in `internal/domain/community/invitation_token.go`.
**Learning:** Comparing tokens or hashes with `==` can leak information about the match length via timing variations (since string comparison aborts early). This could allow attackers to infer valid tokens through timing analysis over many requests.
**Prevention:** Always use `crypto/subtle.ConstantTimeCompare` when comparing sensitive tokens, secrets, or hashes.
