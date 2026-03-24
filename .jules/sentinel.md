# Sentinel's Journal

## 2024-05-22 - MIME Type Spoofing
**Vulnerability:** The application relied solely on the `Content-Type` header provided by the client to determine the file type of uploaded images.
**Learning:** Trusting client headers for file type validation allows attackers to bypass restrictions by simply spoofing the header (e.g., sending an executable with `Content-Type: image/jpeg`).
**Prevention:** Always use content-based detection (e.g., `http.DetectContentType` or magic bytes inspection) to verify the actual file type before processing or storing files.

## 2026-03-24 - Timing Attack on Invitation Token Comparison
**Vulnerability:** The application used standard string comparison (`==`) for validating sensitive invitation tokens, making it vulnerable to timing attacks.
**Learning:** Standard string comparisons short-circuit as soon as they find a differing character, leaking the length of the matching prefix and allowing attackers to guess secrets byte-by-byte.
**Prevention:** Always use `crypto/subtle.ConstantTimeCompare` after converting strings to byte slices for any sensitive string comparisons (like tokens, hashes, passwords) to ensure constant execution time regardless of input correctness.
