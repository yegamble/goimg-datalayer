# Sentinel's Journal

## 2024-05-22 - MIME Type Spoofing
**Vulnerability:** The application relied solely on the `Content-Type` header provided by the client to determine the file type of uploaded images.
**Learning:** Trusting client headers for file type validation allows attackers to bypass restrictions by simply spoofing the header (e.g., sending an executable with `Content-Type: image/jpeg`).
**Prevention:** Always use content-based detection (e.g., `http.DetectContentType` or magic bytes inspection) to verify the actual file type before processing or storing files.

## 2024-05-23 - Timing Attacks in Token Comparison
**Vulnerability:** Comparing sensitive tokens (like invitation tokens or session IDs) using standard string comparison operators (`==` or `!=`) creates a timing attack vulnerability. An attacker can measure the time it takes for the comparison to fail and use that information to deduce the token character by character.
**Learning:** Standard string comparison stops at the first mismatched character. This allows brute-force attacks against tokens to be significantly faster by measuring timing differences.
**Prevention:** Always use `crypto/subtle.ConstantTimeCompare([]byte(token1), []byte(token2)) == 1` when comparing any security-sensitive strings or byte slices to ensure the comparison time is independent of the input contents.
