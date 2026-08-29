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

## 2026-05-21 - golangci-lint Version Mismatch
**Issue:** `make lint` and CI pipeline were failing with "can't load config: the Go language version used to build golangci-lint is lower than the targeted Go version".
**Root Cause:** `GOLANGCI_LINT_VERSION` was set to a non-existent version (`v2.6.2`) causing fallback/failure, and the deprecated `gomodguard` linter caused additional warnings with the corrected version.
**Fix:** Pinned `GOLANGCI_LINT_VERSION` to `v1.64.5` (compatible with Go 1.25) and replaced `gomodguard` with `gomodguard_v2` in `.golangci.yml`. Note that while this surfaces existing lint issues in the codebase, the CI is configured with `only-new-issues: true` on PRs, so it correctly fails only if new code violates rules, while allowing the pipeline to proceed otherwise.

## 2026-05-21 - Deprecated Actions and Setup-Trivy Version Resolution
**Issue:** CI pipelines failed because `aquasecurity/trivy-action` was trying to resolve an invalid underlying version of `setup-trivy@v0.2.1`. Also, actions like checkout and setup-go generated Node 20 deprecation warnings.
**Root Cause:** `aquasecurity/trivy-action` uses a composite action that had a hardcoded/broken setup step in earlier versions. Several github actions were pinned to older versions that didn't support Node 24.
**Fix:** Pinned `aquasecurity/trivy-action` to commit `c1824fd6edce30d7ab345a9989de00bbd46ef284` (v0.34.0) with explicit version `v0.70.0`. Updated checkout, setup-go, upload-artifact, and upload-sarif actions to their Node 24-compatible major versions.

## 2026-05-21 - Deprecated Actions and download-artifact Action Update
**Issue:** `download-artifact` action generated Node 20 deprecation warnings.
**Fix:** Upgraded `actions/download-artifact` in `.github/workflows/ci.yml` from `v4.1.8` to `v4.3.0` which is compatible with Node 24.
