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

## 2026-04-14 - PostgreSQL Health Check User Mismatch
**Issue:** Integration tests running in GitHub Actions and locally were failing due to flaky PostgreSQL container startups.
**Root Cause:** The `health-cmd pg_isready` in `ci.yml` executed as the `root` user within the PostgreSQL container. However, since the database was initialized with `POSTGRES_USER: goimg_test`, the root user role did not exist, causing the readiness check to fail and timeout or the test execution to commence prematurely.
**Fix:** Updated the health check command to specify the initialized user explicitly: `--health-cmd "pg_isready -U goimg_test"`.

## 2026-04-14 - Trivy Action Binary Resolution Failures
**Issue:** The security pipeline's Trivy vulnerability scan was failing to resolve and execute.
**Root Cause:** The GitHub action `aquasecurity/trivy-action` was pinned to an outdated version (`0.28.0`), and it explicitly requested a Trivy binary release (`v0.55.2`) that could not be consistently retrieved or was unsupported by the older action wrapper.
**Fix:** Pinned `aquasecurity/trivy-action` to a newer, stable commit hash (`c1824fd6edce30d7ab345a9989de00bbd46ef284` which resolves to `v0.34.0`) and updated the required Trivy binary version in the `with:` block to `v0.69.3`.
