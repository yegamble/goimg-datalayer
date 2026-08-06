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

## 2026-01-23 - golangci-lint Invalid Version Fallback
**Issue:** The CI pipeline was failing during the golangci-lint step with a Go language version mismatch error.
**Root Cause:** The GOLANGCI_LINT_VERSION in ci.yml was set to a non-existent version (v2.6.2), causing the action to fallback to an invalid build.
**Fix:** Pinned GOLANGCI_LINT_VERSION to a valid version (v1.64.5) to ensure the correct binary is downloaded.

## 2026-01-23 - Trivy Setup Action Resolution
**Issue:** Security scanning jobs failed with 'Unable to resolve action aquasecurity/setup-trivy'.
**Root Cause:** The `aquasecurity/trivy-action` was pinned to an older version (`0.28.0`) that relied on a deprecated or missing `setup-trivy` action.
**Fix:** Updated `aquasecurity/trivy-action` to a newer, stable version (`c1824fd6edce30d7ab345a9989de00bbd46ef284` / `v0.34.0`).
