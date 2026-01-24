# Sentinel's Journal

## 2024-05-22 - MIME Type Spoofing
**Vulnerability:** The application relied solely on the `Content-Type` header provided by the client to determine the file type of uploaded images.
**Learning:** Trusting client headers for file type validation allows attackers to bypass restrictions by simply spoofing the header (e.g., sending an executable with `Content-Type: image/jpeg`).
**Prevention:** Always use content-based detection (e.g., `http.DetectContentType` or magic bytes inspection) to verify the actual file type before processing or storing files.

## 2025-05-23 - ClamAV Scan Bypass
**Vulnerability:** The `UploadImageHandler` enqueued image processing jobs but failed to enqueue the corresponding malware scanning job, effectively bypassing the antivirus check despite the existence of the scanning logic.
**Learning:** Having security controls (like `ImageScanHandler`) implemented in the codebase is insufficient if they are not explicitly invoked in the business workflow.
**Prevention:** Ensure that security checkpoints (scanning, validation) are mandatory steps in the workflow orchestration and consider using a "secure by default" design where processing cannot start until scanning passes (e.g., via job chaining).
