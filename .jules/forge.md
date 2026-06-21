## 2026-06-21 - Invalid setup-trivy Action Download
**Issue:** `security.yml` CI pipeline failed to download the `setup-trivy` binary during the vulnerability scanning step.
**Root Cause:** The `aquasecurity/trivy-action` was pinned to `0.28.0` (`915b19bbe...`), an older version that references a deleted/invalid `setup-trivy` tag, leading to a 404 download error.
**Fix:** Updated the `aquasecurity/trivy-action` uses statement to pin a verified stable version (e.g., `v0.34.0` via commit hash `c1824fd6e...`) which points to valid release assets.

## 2026-06-21 - Node.js 20 Deprecations and CI Flakiness
**Issue:** GitHub Actions CI pipelines were emitting deprecation warnings for Node.js 20 and Trivy/GoSec were failing due to missing updates or caching issues. Additionally, an integration test (`TestUserRepository_FindExpiredGuests`) was missing arguments to `identity.ReconstructUser`.
**Root Cause:**
1.  Multiple GitHub Actions (e.g., `checkout@v4.1.1`, `setup-go@v5.0.2`, `upload-artifact@v4.4.0`, `cache@v4.0.0`, `setup-node@v4.0.2`, `codeql-action/upload-sarif@v3.31.10`) were pinned to older SHAs that still utilized Node.js 20 runtime, which GitHub is deprecating.
2.  `golangci-lint-action` was running on an old version `v6.1.1`.
3.  `TestUserRepository_FindExpiredGuests` had missing parameters for `emailVerified` and `emailVerifiedAt` in its call to `identity.ReconstructUser`.
**Fix:**
1.  Updated and pinned actions to the latest SHAs (e.g., `checkout@v4.2.2`, `setup-go@v5.3.0`, `upload-artifact@v4.6.1`, `cache@v4.2.2`, `setup-node@v4.2.0`, `codeql-action/upload-sarif@v4.36.2`) which run on Node.js 24.
2.  Updated `golangci-lint-action` to `v6.5.0` (`1b6671e...`).
3.  Added the missing `false` and `nil` arguments to the `ReconstructUser` call in `tests/integration/user_repository_test.go`.
