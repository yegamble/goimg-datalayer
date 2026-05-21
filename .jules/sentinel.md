# Sentinel's Journal

## 2024-05-22 - MIME Type Spoofing
**Vulnerability:** The application relied solely on the `Content-Type` header provided by the client to determine the file type of uploaded images.
**Learning:** Trusting client headers for file type validation allows attackers to bypass restrictions by simply spoofing the header (e.g., sending an executable with `Content-Type: image/jpeg`).
**Prevention:** Always use content-based detection (e.g., `http.DetectContentType` or magic bytes inspection) to verify the actual file type before processing or storing files.

## 2026-05-21 - GoSec G101 and G304 False Positives in CI
**Vulnerability:** GoSec was flagging SQL queries containing the word "token" as hardcoded credentials (G101), and the RSA key loaders as file inclusion vulnerabilities (G304).
**Learning:** For SQL queries where "token" is just part of the variable/query name, a `#nosec G101` comment immediately above the constant is required. For file inclusion, standard `filepath.Clean` and `#nosec G304` are needed when the paths are loaded securely from application configuration.
**Prevention:** Always sanitize dynamic filepaths before calling `os.ReadFile` and carefully label false positives in constants to ensure security scans pass cleanly in CI.
