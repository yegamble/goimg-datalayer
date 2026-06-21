## 2026-01-20 - Pinned CI Tool Versions
**Issue:** `ci.yml` and `security.yml` were installing critical tools (`gosec`, `govulncheck`, `gocovmerge`, `grype`, `cyclonedx-gomod`) using `@latest` or unversioned scripts, leading to non-reproducible builds and potential supply chain vulnerabilities.
**Root Cause:** Tools were installed using `go install ...@latest` or `curl ... | sh` without version constraints.
**Fix:** Pinned all tools to specific versions or commit hashes (e.g., `gosec@v2.22.11`, `gocovmerge@b5bfa59`) to ensure stability and security.

## 2026-01-21 - Docker Runtime Libc Mismatch
**Issue:** `Dockerfile.api` and `Dockerfile.worker` were failing at runtime due to a segmentation fault or missing dynamic linker errors.
**Root Cause:** The build stage used `golang:1.25-alpine` (musl libc), but the runtime stage used `distroless/base-debian12` (glibc). Shared libraries (`libvips`) compiled against musl were manually copied to the glibc-based runtime, causing binary incompatibility.
**Fix:** Switched the runtime image to `alpine:3.21` to match the builder's libc (musl) and installed runtime dependencies (`vips`, `ca-certificates`) using `apk` instead of manually copying libraries.

## 2026-01-22 - Unpinned NPM Dependencies
**Issue:** `ci.yml` was installing `newman` and `newman-reporter-htmlextra` using `npm install -g ...` without version constraints, leading to potential breakage if new major versions are released (e.g., Newman v7).
**Root Cause:** CI pipeline configuration used default `latest` behavior for npm packages.
**Fix:** Pinned versions to `newman@6.2.2` and `newman-reporter-htmlextra@1.23.1` in `ci.yml` and updated `Makefile` guidance to match.

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
