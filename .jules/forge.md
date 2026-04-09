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

## 2026-01-23 - PostgreSQL Health Check User
**Issue:** `ci.yml` was using `pg_isready` without a specified user for the `postgres` service health check, causing 'role root does not exist' fatal errors during database initialization.
**Root Cause:** The health check command was running as the root user instead of the expected `goimg_test` user.
**Fix:** Explicitly specified the user using `pg_isready -U goimg_test` in the health check command within the `options` block of the `postgres` service in `.github/workflows/ci.yml`.

## 2026-01-23 - Trivy Action Version Resolution
**Issue:** `security.yml` was failing to resolve `aquasecurity/trivy-action` due to an incompatible action version (`0.28.0`) combined with a removed upstream binary version (`v0.55.2`).
**Root Cause:** The Trivy action was using a deprecated binary version and a commit hash that no longer reliably resolved the action dependencies.
**Fix:** Pinned `aquasecurity/trivy-action` to a stable commit hash (`c1824fd6edce30d7ab345a9989de00bbd46ef284` -> `v0.34.0`) and updated the `version` field to explicitly use a working Trivy binary release (`v0.69.3`) to ensure stable execution.
