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

## 2026-04-06 - Trivy Action Resolution Failure
**Issue:** The security pipeline's Trivy vulnerability scan job consistently failed with the error `Unable to resolve action aquasecurity/setup-trivy@v0.2.1, unable to find version v0.2.1`.
**Root Cause:** The `aquasecurity/trivy-action` version being used (v0.28.0 pinned via SHA) contained internal references to a deprecated/removed version of a dependency (`setup-trivy@v0.2.1`), causing GitHub Actions to fail resolving the action steps entirely before execution began. Additionally, the requested Trivy binary version `v0.55.2` is outdated.
**Fix:** Update `aquasecurity/trivy-action` to a stable newer commit hash (`c1824fd6edce30d7ab345a9989de00bbd46ef284` corresponding to `v0.34.0`) that properly resolves its internal dependencies, and explicitly bump the requested Trivy binary `version` to `v0.69.3` to avoid download failures for deprecated binary releases.
