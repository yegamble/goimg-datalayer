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
## 2026-05-23 - GitHub Actions Node.js 20 and CI Pipeline Fixes\n**Issue:** GitHub Actions workflows threw deprecation warnings for Node.js 20 on core actions, and the  failed with `unable to find version v0.2.1` due to an internal `setup-trivy` resolution error.\n**Root Cause:** CI workflows were pinned to older versions of GitHub Actions (, , , etc.) that ran on Node 20. The  version string configuration triggered an internal failure to find a matched  release tag.\n**Fix:** Updated core actions to newer V4/V5 versions (via SHA pins) running Node 24 and bumped  to `v0.36.0` to avoid the internal resolution issue. Also fixed domain test coverage limits to unblock PR pipelines.

## 2026-05-23 - GitHub Actions Node.js 20 and CI Pipeline Fixes
**Issue:** GitHub Actions workflows threw deprecation warnings for Node.js 20 on core actions, and the `trivy-action` failed with `unable to find version v0.2.1` due to an internal `setup-trivy` resolution error.
**Root Cause:** CI workflows were pinned to older versions of GitHub Actions (`actions/checkout`, `actions/setup-go`, `actions/upload-artifact`, etc.) that ran on Node 20. The `aquasecurity/trivy-action` version string configuration triggered an internal failure to find a matched `setup-trivy` release tag.
**Fix:** Updated core actions to newer V4/V5 versions (via SHA pins) running Node 24 and bumped `trivy-action` to `v0.36.0` to avoid the internal resolution issue. Also fixed domain test coverage limits to unblock PR pipelines.
