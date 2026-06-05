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

## 2026-06-05 - Trivy Action Version Fix
**Issue:** `security.yml` failing on Trivy scan step.
**Root Cause:** Using an old, potentially deprecated commit hash (`915b19bbe73b92a6cf82a1bc12b087c9a19a5fe2`) for `aquasecurity/trivy-action`.
**Fix:** Pinned `aquasecurity/trivy-action` to a stable commit hash `c1824fd6edce30d7ab345a9989de00bbd46ef284` (v0.34.0) to prevent binary download errors.

## 2026-06-05 - Node.js 20 Deprecation in GitHub Actions
**Issue:** Warnings for Node.js 20 deprecation in multiple workflows (`ci.yml`, `security.yml`).
**Root Cause:** Using older, pinned versions of actions (`actions/checkout`, `actions/setup-go`, `actions/upload-artifact`, `github/codeql-action/upload-sarif`) that still rely on Node.js 20 instead of Node.js 24.
**Fix:** Updated and pinned the actions to newer versions that are compatible with Node.js 24 (`actions/checkout@v4.2.2`, `actions/setup-go@v5.1.0`, `actions/upload-artifact@v4.6.0`, `github/codeql-action/upload-sarif@v3.32.2`).
