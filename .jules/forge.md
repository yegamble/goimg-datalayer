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

## 2026-05-27 - PostgreSQL Health Check User
**Issue:** The `pg_isready` health check in `ci.yml` was flooding PostgreSQL logs with `FATAL: role "root" does not exist` errors, and potentially risking initialization errors if the default user did not match the environment.
**Root Cause:** The `pg_isready` command in Docker runs as `root` by default and tries to authenticate as such unless explicitly told otherwise.
**Fix:** Explicitly added the `-U goimg_test` flag to the `pg_isready` commands in the CI workflow to authenticate correctly and silence the log noise.

## 2026-05-27 - CI Security Scan and Integration Test Fixes
**Issue:** CI failed due to multiple issues: `trivy` action failed to resolve (`setup-trivy@v0.2.1`), `gosec` flagged false positives for hardcoded credentials (G101) and path traversals (G304), and `integration` tests failed to compile due to missing arguments in `ReconstructUser`.
**Root Cause:** The `trivy-action` was pinned to an older version that internally referenced a missing or broken `setup-trivy` tag. `gosec` aggressively flagged SQL queries containing the word `token`. The `ReconstructUser` function signature was updated but the integration test mock data wasn't.
**Fix:** Pinned `trivy-action` to a stable commit (`c1824fd...` for `v0.34.0`) and updated the internal `version` flag to `v0.70.0`. Added explicit `// #nosec G101` to SQL queries and `// #nosec G304` + `filepath.Clean` to `os.ReadFile` calls. Added missing `emailVerified` (`false`) and `emailVerifiedAt` (`nil`) to `ReconstructUser` in tests.
