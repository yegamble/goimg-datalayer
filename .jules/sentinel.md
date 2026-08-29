# Sentinel's Journal

## 2024-05-22 - MIME Type Spoofing
**Vulnerability:** The application relied solely on the `Content-Type` header provided by the client to determine the file type of uploaded images.
**Learning:** Trusting client headers for file type validation allows attackers to bypass restrictions by simply spoofing the header (e.g., sending an executable with `Content-Type: image/jpeg`).
**Prevention:** Always use content-based detection (e.g., `http.DetectContentType` or magic bytes inspection) to verify the actual file type before processing or storing files.

## 2025-02-18 - Timing Attack Vulnerability in Invitation Tokens
**Vulnerability:** The `Equals` method in `InvitationToken` used standard string equality (`==`), making it vulnerable to timing attacks as the comparison fails early upon finding the first differing character.
**Learning:** Comparing sensitive strings (like cryptographic tokens, passwords, or hashes) using standard string comparison operators allows an attacker to deduce the expected string character by character by measuring the exact time taken to reject an invalid attempt.
**Prevention:** To prevent timing attacks, sensitive string comparisons must use `crypto/subtle.ConstantTimeCompare` after converting the strings to byte slices.
