# Sentinel's Journal

## 2024-11-20 - Timing Attacks on Invitation Tokens
**Vulnerability:** The `Equals` method of `InvitationToken` used a standard string comparison `==` operator.
**Learning:** Using `==` allows attackers to infer portions of a secret token by measuring the time the comparison takes (since it stops at the first mismatch).
**Prevention:** To prevent timing attacks, sensitive string comparisons (such as token validation in the domain layer, e.g., `InvitationToken.Equals`) must use `crypto/subtle.ConstantTimeCompare` after converting strings to byte slices.

## 2024-05-22 - MIME Type Spoofing
**Vulnerability:** The application relied solely on the `Content-Type` header provided by the client to determine the file type of uploaded images.
**Learning:** Trusting client headers for file type validation allows attackers to bypass restrictions by simply spoofing the header (e.g., sending an executable with `Content-Type: image/jpeg`).
**Prevention:** Always use content-based detection (e.g., `http.DetectContentType` or magic bytes inspection) to verify the actual file type before processing or storing files.
