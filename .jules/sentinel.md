# Sentinel's Journal

## 2024-05-22 - MIME Type Spoofing
**Vulnerability:** The application relied solely on the `Content-Type` header provided by the client to determine the file type of uploaded images.
**Learning:** Trusting client headers for file type validation allows attackers to bypass restrictions by simply spoofing the header (e.g., sending an executable with `Content-Type: image/jpeg`).
**Prevention:** Always use content-based detection (e.g., `http.DetectContentType` or magic bytes inspection) to verify the actual file type before processing or storing files.

## 2024-03-18 - [HIGH] Fix IDOR in GetUserBanStatus
**Vulnerability:** The `GetUserBanStatus` endpoint was lacking proper authorization checks, allowing any authenticated user to view the ban status of any other user via an Insecure Direct Object Reference (IDOR).
**Learning:** Proper extraction of the `UserContext` and validation of roles/ownership are critical to prevent unauthorized access to sensitive user data.
**Prevention:** Always extract the user context via `GetUserFromContext(ctx)` and enforce role or ownership validations before executing queries on sensitive endpoints.
