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

## 2026-02-05 - GitHub Action Node.js 20 Deprecations and Invalid Linter Version
**Issue:** CI workflows were generating Node.js 20 deprecation warnings causing 'Cache service responded with 400' errors, and `golangci-lint-action` could fail due to specifying a non-existent version `v2.6.2`.
**Root Cause:** Using older Action versions (like `actions/checkout@v4.1.1` and `aquasecurity/trivy-action@0.28.0`) that relied on deprecated Node environments, and an invalid `GOLANGCI_LINT_VERSION` env variable.
**Fix:** Updated standard actions (e.g., checkout, upload-artifact, setup-node) and `aquasecurity/trivy-action` to modern versions securely pinned to their stable commit hashes, and corrected `GOLANGCI_LINT_VERSION` to `v1.64.5`.

## 2026-02-05 - Trivy Action Version Format
**Issue:** `aquasecurity/trivy-action` failed with `aquasecurity/trivy info checking GitHub for tag 'v0.55.2'` when `version: 'v0.55.2'` was specified.
**Root Cause:** The action expects the version string without the 'v' prefix (e.g., `0.55.2`) when downloading the binary via the installer script. The `setup-trivy` action correctly parsed it, but the main action failed.
**Fix:** Changed `version: 'v0.55.2'` to `version: '0.55.2'` in `.github/workflows/security.yml`.
