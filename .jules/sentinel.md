# Sentinel's Journal

## 2024-05-22 - MIME Type Spoofing
**Vulnerability:** The application relied solely on the `Content-Type` header provided by the client to determine the file type of uploaded images.
**Learning:** Trusting client headers for file type validation allows attackers to bypass restrictions by simply spoofing the header (e.g., sending an executable with `Content-Type: image/jpeg`).
**Prevention:** Always use content-based detection (e.g., `http.DetectContentType` or magic bytes inspection) to verify the actual file type before processing or storing files.

## 2025-04-01 - Timing Attacks in Invitation Tokens
**Vulnerability:** String comparison (e.g., `==`) for sensitive token validations, such as `InvitationToken.Equals`.
**Learning:** Standard string comparisons terminate early upon encountering the first mismatched character. Attackers can measure the response time variations to infer the correct characters sequentially, completely bypassing random hex token protections.
**Prevention:** Always convert sensitive strings to byte slices and compare them using `crypto/subtle.ConstantTimeCompare` (verifying the return value is exactly `1`) to ensure comparisons execute in constant time, regardless of the input values.
