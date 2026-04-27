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

## 2025-04-27 - PostgreSQL Service Health Checks
**Issue:** PostgreSQL container health checks can fail during initialization if the DB role is not specified.
**Root Cause:** The `pg_isready` command runs as the `root` user by default in the health check, leading to 'role "root" does not exist' fatal errors.
**Fix:** Specify the expected database user in the health check command (e.g., `pg_isready -U goimg_test`) to ensure it runs correctly and accurately reflects readiness.

## 2025-04-27 - GitHub Actions Node.js 20 Deprecation
**Issue:** GitHub Actions running on Node.js 20 are deprecated and will be removed from the runner.
**Root Cause:** Using older versions of common actions like `actions/checkout@v4.1.1` and `github/codeql-action/upload-sarif@v3`.
**Fix:** Upgrade all occurrences of deprecated actions to their latest versions that support Node.js 24 (e.g., `actions/checkout@v4.3.1`, `github/codeql-action/upload-sarif@v4`).

## 2025-04-27 - Trivy Setup Action Missing Version
**Issue:** `aquasecurity/setup-trivy@v0.2.1` fails to resolve due to non-existent version tag.
**Root Cause:** Using invalid or non-existent action versions. (Note: The CI failure was for `aquasecurity/setup-trivy@v0.2.1`, but the fix updated `aquasecurity/trivy-action` instead, but the error actually occurs because `v0.55.2` is incorrect for trivy-action and causes an internal setup failure or is outdated).
**Fix:** Upgrade `aquasecurity/trivy-action` to a valid pinned version (e.g. `v0.34.0`) and use a valid trivy version inside the config `v0.70.0`.
