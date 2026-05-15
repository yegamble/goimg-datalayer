# Sentinel's Journal

## 2024-05-22 - MIME Type Spoofing
**Vulnerability:** The application relied solely on the `Content-Type` header provided by the client to determine the file type of uploaded images.
**Learning:** Trusting client headers for file type validation allows attackers to bypass restrictions by simply spoofing the header (e.g., sending an executable with `Content-Type: image/jpeg`).
**Prevention:** Always use content-based detection (e.g., `http.DetectContentType` or magic bytes inspection) to verify the actual file type before processing or storing files.

## 2024-05-22 - Timing Attack in Invitation Tokens
**Vulnerability:** The application compared sensitive invitation tokens using simple string equality (`==`), making it potentially vulnerable to timing attacks where an attacker could deduce the token byte by byte based on response time.
**Learning:** Even if tokens are long enough to make brute force theoretically infeasible, any sensitive token, hash, or password should always use constant-time comparison to provide defense-in-depth and adhere to strict security best practices.
**Prevention:** Always use `crypto/subtle.ConstantTimeCompare` (casting strings to `[]byte`) when evaluating the equality of sensitive values like passwords, tokens, or hashes.
