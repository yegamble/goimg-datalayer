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

## 2026-08-01 - CI Tools and Github Actions Versions Pinned Properly
**Issue:** Some GitHub Actions, such as `aquasecurity/trivy-action`, `actions/checkout`, `actions/setup-go`, `actions/setup-node`, `actions/upload-artifact`, `github/codeql-action/upload-sarif`, and `golangci/golangci-lint-action`, were pinned to old versions, some of which had incorrect tags or produced Node.js 20 deprecation warnings. Additionally, `GOLANGCI_LINT_VERSION` was set to a non-existent `v2.6.2`.
**Root Cause:** The pipeline configuration had not been updated to use the latest stable versions and correctly resolved SHAs for these tools.
**Fix:** Updated these GitHub Actions to their modern versions pinned by SHAs and updated `GOLANGCI_LINT_VERSION` to `v1.64.5`.

## 2026-08-01 - CI Tools Configuration
**Issue:** The pipeline failed due to `GOLANGCI_LINT_VERSION` being set to a non-existent `v2.6.2` version and trivy action referencing a missing tag `0.28.0` which caused failure in `security.yml`.
**Root Cause:** The versions configuration and tags had not been accurately set and pinned, blocking pipeline executions.
**Fix:** Fixed `GOLANGCI_LINT_VERSION` to `v1.64.5` and `aquasecurity/trivy-action` to use a stable tag `v0.36.0`.
