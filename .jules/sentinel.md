# Sentinel's Journal

## 2024-05-22 - MIME Type Spoofing
**Vulnerability:** The application relied solely on the `Content-Type` header provided by the client to determine the file type of uploaded images.
**Learning:** Trusting client headers for file type validation allows attackers to bypass restrictions by simply spoofing the header (e.g., sending an executable with `Content-Type: image/jpeg`).
**Prevention:** Always use content-based detection (e.g., `http.DetectContentType` or magic bytes inspection) to verify the actual file type before processing or storing files.

## 2024-10-24 - Timing Attack in Token Comparison
**Vulnerability:** The `Equals` method of `InvitationToken` in the domain layer used the standard `==` string comparison operator instead of constant-time comparison.
**Learning:** Standard string comparison operators (`==`) in Go short-circuit on the first mismatched byte, which means the comparison time varies based on how many characters match. Attackers can measure these small timing differences to guess a token character by character (timing attack).
**Prevention:** Always use `crypto/subtle.ConstantTimeCompare` after converting strings to byte slices for sensitive value comparisons (like tokens, passwords, hashes, MACs) to prevent timing attacks.
