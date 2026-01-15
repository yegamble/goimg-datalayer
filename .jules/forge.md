## 2025-01-20 - Pinned GitHub Actions and Updated Dockerfiles
**Issue:** GitHub Actions workflows were using unpinned versions (tags) for `trivy-action` and `codeql-action/upload-sarif`, which poses a supply chain security risk and could break if tags are updated with breaking changes. Also, Dockerfiles were using an outdated Go version (1.22) while the project requires 1.25.
**Root Cause:** Initial setup likely used convenient tags and didn't update the Docker base image when the project upgraded Go versions.
**Fix:** Pinned all actions to specific commit SHAs (found via release tags) and updated Dockerfiles to use `golang:1.25-alpine`.
