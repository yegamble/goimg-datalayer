# Sentinel's Journal

## 2024-05-22 - MIME Type Spoofing
**Vulnerability:** The application relied solely on the `Content-Type` header provided by the client to determine the file type of uploaded images.
**Learning:** Trusting client headers for file type validation allows attackers to bypass restrictions by simply spoofing the header (e.g., sending an executable with `Content-Type: image/jpeg`).
**Prevention:** Always use content-based detection (e.g., `http.DetectContentType` or magic bytes inspection) to verify the actual file type before processing or storing files.

## 2026-04-11 - IDOR in User Ban Status
**Vulnerability:** Any authenticated user could view the ban status of any other user due to missing role and resource ownership validation in the `GetUserBanStatus` endpoint.
**Learning:** Endpoints that retrieve user-specific status information must enforce strict access controls. Without role or ownership checks, attackers can enumerate and access sensitive user state data.
**Prevention:** Enforce IDOR protection by validating that the requester is an admin, moderator, or the owner of the resource being requested.
