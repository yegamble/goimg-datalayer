# Sentinel's Journal

## 2024-05-22 - MIME Type Spoofing
**Vulnerability:** The application relied solely on the `Content-Type` header provided by the client to determine the file type of uploaded images.
**Learning:** Trusting client headers for file type validation allows attackers to bypass restrictions by simply spoofing the header (e.g., sending an executable with `Content-Type: image/jpeg`).
**Prevention:** Always use content-based detection (e.g., `http.DetectContentType` or magic bytes inspection) to verify the actual file type before processing or storing files.
## 2024-05-23 - Host Header Injection in QR Code Generation
**Vulnerability:** The application was using `X-Forwarded-Host` or `Host` header to generate absolute URLs for QR codes when `BASE_URL` was not configured. This allowed attackers to generate QR codes pointing to malicious domains.
**Learning:** Fallback mechanisms that rely on user-controlled headers for URL generation are dangerous. Always require explicit configuration for base URLs.
**Prevention:** Removed the fallback logic `inferBaseURLFromRequest` and made `BASE_URL` mandatory for this feature.
