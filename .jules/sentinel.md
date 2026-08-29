# Sentinel's Journal

## 2024-05-22 - MIME Type Spoofing
**Vulnerability:** The application relied solely on the `Content-Type` header provided by the client to determine the file type of uploaded images.
**Learning:** Trusting client headers for file type validation allows attackers to bypass restrictions by simply spoofing the header (e.g., sending an executable with `Content-Type: image/jpeg`).
**Prevention:** Always use content-based detection (e.g., `http.DetectContentType` or magic bytes inspection) to verify the actual file type before processing or storing files.

## 2025-05-24 - Timing attack vulnerability in invitation token equality check
**Vulnerability:** The equality check in `InvitationToken.Equals` (`internal/domain/community/invitation_token.go`) was using standard string comparison `==`.
**Learning:** Even though the token is reasonably long, standard string comparison is theoretically vulnerable to timing attacks and should not be used.
**Prevention:** Used `crypto/subtle.ConstantTimeCompare([]byte(t.value), []byte(other.value)) == 1` to perform the comparison securely.
