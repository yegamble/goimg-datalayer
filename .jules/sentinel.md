# Sentinel's Journal

## 2024-05-22 - MIME Type Spoofing
**Vulnerability:** The application relied solely on the `Content-Type` header provided by the client to determine the file type of uploaded images.
**Learning:** Trusting client headers for file type validation allows attackers to bypass restrictions by simply spoofing the header (e.g., sending an executable with `Content-Type: image/jpeg`).
**Prevention:** Always use content-based detection (e.g., `http.DetectContentType` or magic bytes inspection) to verify the actual file type before processing or storing files.
## 2026-07-15 - Constant-Time Comparison
**Vulnerability:** String comparison (==) was used to compare cryptographic invitation tokens.
**Learning:** Using simple string comparison for security tokens is vulnerable to timing attacks, as it allows an attacker to deduce the token length or content based on the comparison duration.
**Prevention:** Always use `crypto/subtle.ConstantTimeCompare` when comparing secrets, passwords, or tokens.
