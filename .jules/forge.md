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

## 2026-08-09 - Fix CI Tool Version Failures
**Issue:** The CI pipeline was failing because of invalid action/tool versions (Trivy action failing to download and golangci-lint failing with requested version doesn't exist).
**Root Cause:** `aquasecurity/trivy-action` was pinned to an old version (`0.28.0`) referencing deleted tags, and `GOLANGCI_LINT_VERSION` was set to a non-existent version (`v2.6.2`).
**Fix:** Pinned `aquasecurity/trivy-action` to a verified stable commit hash for `v0.36.0` (`a9c7b0f06e461e9d4b4d1711f154ee024b8d7ab8`) and updated `GOLANGCI_LINT_VERSION` to a valid release version (`v1.64.5`).

## 2026-08-09 - Resolve Node 20 Deprecation Warnings & Test Signature Mismatch
**Issue:** CI failed due to Node 20 deprecation warnings breaking cache/tool downloads, and integration tests failing due to an outdated domain signature call in `user_repository_test.go`.
**Root Cause:** Several standard GitHub Actions were running outdated versions (e.g., checkout v4.1.1, setup-go v5.0.2), causing Node 20 warnings which led to 400 errors from the cache service. Furthermore, an integration test had not been updated after `identity.ReconstructUser` was modified to require `emailVerified` and `emailVerifiedAt`.
**Fix:** Pinned all relevant GitHub Actions (`checkout`, `setup-go`, `setup-node`, `upload-artifact`, `golangci-lint-action`, `codeql-action/upload-sarif`) to modern versions resolving the Node 20 issue. Updated `tests/integration/user_repository_test.go` to provide the required boolean and pointer fields to `identity.ReconstructUser`.
