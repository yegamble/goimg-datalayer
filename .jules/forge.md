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

## 2026-07-11 - Fix Node.js 20 Deprecation and Trivy Action Errors
**Issue:** CI pipelines failed due to Node.js 20 deprecation causing caching service 400 errors and `aquasecurity/trivy-action` failing to resolve its dependent setup-trivy action.
**Root Cause:** The GitHub Actions runner deprecated Node.js 20, causing older versions of standard actions (`checkout`, `setup-go`, `upload-artifact`, `codeql-action/upload-sarif`) to fail during cache restoration. Additionally, `aquasecurity/trivy-action` `v0.28.0` referenced an invalid/deleted `setup-trivy` tag.
**Fix:** Pinned `actions/checkout` to `v4.2.2`, `actions/setup-go` to `v5.3.0`, `actions/upload-artifact` to `v4.6.0`, `github/codeql-action/upload-sarif` to `v4.36.2`, and `aquasecurity/trivy-action` to `v0.34.0` using their respective stable commit hashes.
