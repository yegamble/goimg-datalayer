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

## 2026-01-23 - PostgreSQL Health Check Failure
**Issue:** The PostgreSQL service health check in `ci.yml` was failing with `role "root" does not exist` errors.
**Root Cause:** The health check command (`pg_isready`) was running without explicitly specifying the user, causing it to default to the user running the health check (root).
**Fix:** Updated the `--health-cmd` to explicitly specify the test user with `pg_isready -U goimg_test`.

## 2026-01-23 - Trivy Action Failures
**Issue:** The `aquasecurity/trivy-action` in `security.yml` was failing due to resolution or binary download errors.
**Root Cause:** The action version wasn't pinned to a stable commit hash (like `v0.34.0` at `c1824fd6edce30d7ab345a9989de00bbd46ef284`) and the requested Trivy release version (`v0.55.2`) had upstream download issues.
**Fix:** Pinned `aquasecurity/trivy-action` to a stable hash (`c1824fd6edce30d7ab345a9989de00bbd46ef284`) and updated the explicit Trivy release version to `v0.69.3`.
