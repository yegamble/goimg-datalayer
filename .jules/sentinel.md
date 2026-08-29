# Sentinel's Journal

## 2024-05-22 - MIME Type Spoofing
**Vulnerability:** The application relied solely on the `Content-Type` header provided by the client to determine the file type of uploaded images.
**Learning:** Trusting client headers for file type validation allows attackers to bypass restrictions by simply spoofing the header (e.g., sending an executable with `Content-Type: image/jpeg`).
**Prevention:** Always use content-based detection (e.g., `http.DetectContentType` or magic bytes inspection) to verify the actual file type before processing or storing files.

## 2025-02-23 - Timing Attack Vulnerability in Token Comparison
**Vulnerability:** The application used standard string comparison (`==`) to validate invitation tokens, which could theoretically allow an attacker to bypass validation via timing attacks.
**Learning:** Comparing sensitive secrets like tokens byte-by-byte exposes early-exit timing side channels, despite claims that token length "makes brute force infeasible".
**Prevention:** Always use `crypto/subtle.ConstantTimeCompare` when comparing tokens, passwords, hashes, or any sensitive cryptographic material.
