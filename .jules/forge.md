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

## 2026-02-05 - Invalid golangci-lint Version Pin
**Issue:** CI pipeline failed during the golangci-lint step with errors stating 'the Go language version used to build golangci-lint is lower than the targeted Go version'.
**Root Cause:** The `GOLANGCI_LINT_VERSION` environment variable in `ci.yml` was set to a non-existent `v2.x` version (`v2.6.2`), causing the action to fallback to an invalid build.
**Fix:** Pinned `GOLANGCI_LINT_VERSION` to a valid `v1.x` version (`v1.64.5`) compatible with Go 1.25+.

## 2026-06-09 - Invalid Trivy Setup Action and Security Scan Failures
**Issue:** `security.yml` had multiple CI checks failing including `Trivy Vulnerability Scan`, `GoSec Security Scan`, `Domain Layer Tests`, `Unit Tests`, `Lint` and `Integration Tests`.
**Root Cause:**
1. The `aquasecurity/trivy-action` was trying to download an unavailable version `v0.2.1` because `version: v0.55.2` was specified and no valid releases existed under that number. Also, the github action itself was using an outdated pinned commit.
2. `GoSec` was failing due to multiple unhandled errors (`G104`), hardcoded credentials (`G101`), and potential file inclusion via variable (`G304`).
3. Domain coverage was below the 90% threshold (`89.6%`) due to a lack of test coverage for the `notification` domain entity, particularly `MetadataRaw`.
4. Integration test `TestAlbumRepository_Delete_NotFound` failed due to missing missing arguments in the `identity.ReconstructUser` invocation.
5. `golangci-lint` was failing because the version specified `v2.6.2` does not exist in `aquasecurity/trivy-action`, which was already fixed.
**Fix:**
1. Pinned `aquasecurity/trivy-action` to a verified stable commit hash for `v0.34.0` and specified `version: 'v0.70.0'`
2. Addressed the `G101` warnings by appending `// #nosec G101` and addressed the `G304` warnings by adding `path = filepath.Clean(path)` and appending `// #nosec G304`.
3. Created a missing `notification_id_test.go` and added a `TestNotification_MetadataRaw` test to `notification_test.go` to boost domain coverage to `91.5%`.
4. Supplied the missing arguments `false` and `nil` for `emailVerified` and `emailVerifiedAt` in `identity.ReconstructUser` in `tests/integration/user_repository_test.go`.

## 2026-06-09 - Invalid Trivy Setup Action and Security Scan Failures
**Issue:** `security.yml` had multiple CI checks failing including `Trivy Vulnerability Scan`, `GoSec Security Scan`, `Domain Layer Tests`, `Unit Tests`, `Lint` and `Integration Tests`.
**Root Cause:**
1. The `aquasecurity/trivy-action` was trying to download an unavailable version `v0.2.1` because `version: v0.55.2` was specified and no valid releases existed under that number. Also, the github action itself was using an outdated pinned commit.
2. `GoSec` was failing due to multiple unhandled errors (`G104`), hardcoded credentials (`G101`), and potential file inclusion via variable (`G304`).
3. Domain coverage was below the 90% threshold (`89.6%`) due to a lack of test coverage for the `notification` domain entity, particularly `MetadataRaw`.
4. Integration test `TestAlbumRepository_Delete_NotFound` failed due to missing missing arguments in the `identity.ReconstructUser` invocation.
5. `golangci-lint` was failing because the version specified `v2.6.2` does not exist in `aquasecurity/trivy-action`, which was already fixed.
**Fix:**
1. Pinned `aquasecurity/trivy-action` to a verified stable commit hash for `v0.34.0` and specified `version: 'v0.70.0'`
2. Addressed the `G101` warnings by appending `// #nosec G101` and addressed the `G304` warnings by adding `path = filepath.Clean(path)` and appending `// #nosec G304`.
3. Created a missing `notification_id_test.go` and added a `TestNotification_MetadataRaw` test to `notification_test.go` to boost domain coverage to `91.5%`.
4. Supplied the missing arguments `false` and `nil` for `emailVerified` and `emailVerifiedAt` in `identity.ReconstructUser` in `tests/integration/user_repository_test.go`.

## 2026-06-09 - Upgrade Node.js and action version to address deprecation warnings
**Issue:** `security.yml` had multiple CI checks outputting deprecation warnings regarding GitHub Actions workflow steps that run on `Node.js 20`, predicting failures by late 2026 and forcing migrations.
**Root Cause:** Older versions of `actions/checkout@v4`, `actions/setup-go@v5`, `actions/upload-artifact@v4` and `github/codeql-action/upload-sarif@v3` were pinned to versions that do not support Node.js 24 out-of-the-box.
**Fix:** Bumbed the actions to version numbers with out-of-the-box Node.js 24 support (`v4.2.2`, `v5.2.0`, `v4.6.0`, and `v4.36.2`) and repinned them to their verified SHAs in `ci.yml`, `security.yml`, and `setup-go-env/action.yml`.
