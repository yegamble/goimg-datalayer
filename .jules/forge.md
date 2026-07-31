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

## 2026-07-31 - Pinned Trivy Action version to v0.36.0
**Issue:** Security workflow was failing due to 'Unable to resolve action aquasecurity/setup-trivy' or binary download errors.
**Root Cause:** Older versions of `aquasecurity/trivy-action` (like 0.28.0) reference invalid or deleted `setup-trivy` tags.
**Fix:** Pinned `aquasecurity/trivy-action` to stable commit `a9c7b0f06e461e9d4b4d1711f154ee024b8d7ab8` (v0.36.0) to ensure reliable resolution.

## 2026-07-31 - CI Tool Action Update
**Issue:** GitHub Actions workflows were generating warnings due to the impending deprecation of Node.js 20 on GitHub runners and the CodeQL Action v3 deprecation.
**Root Cause:** Workflow files contained pinned actions using old versions (like `actions/checkout@v4.1.1`, `actions/upload-artifact@v4.4.0`) which relied on deprecated Node.js versions and an outdated CodeQL action.
**Fix:** Updated all standard GitHub Actions (`checkout`, `setup-go`, `setup-node`, `upload-artifact`, `download-artifact`, `codeql-action/upload-sarif`, `golangci-lint-action`) to their modern versions targeting newer runtime dependencies, preserving SHA pinning.

## 2026-07-31 - CI Tool Version Updates
**Issue:** GitHub Actions workflows were generating warnings due to the impending deprecation of Node.js 20 on GitHub runners and the CodeQL Action v3 deprecation.
**Root Cause:** Workflow files contained pinned actions using old versions (like `actions/checkout@v4.1.1`, `actions/upload-artifact@v4.4.0`) which relied on deprecated Node.js versions and an outdated CodeQL action.
