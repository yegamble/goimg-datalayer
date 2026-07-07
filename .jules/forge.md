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

## 2026-01-23 - Invalid golangci-lint version fallback
**Issue:** CI pipeline failed during the `golangci-lint` step with the error 'the Go language version used to build golangci-lint is lower than the targeted Go version'.
**Root Cause:** The `GOLANGCI_LINT_VERSION` environment variable in `ci.yml` was set to a non-existent `v2.x` version (`v2.6.2`), which caused the action to fallback to an invalid build.
**Fix:** Pinned `GOLANGCI_LINT_VERSION` to a valid `v1.x` version (`v1.64.5`) to ensure the correct build is used.

## 2026-01-23 - Trivy and ReconstructUser fixes
**Issue:** CI pipeline failed due to outdated `aquasecurity/trivy-action` and missing arguments in `identity.ReconstructUser` in `tests/integration/user_repository_test.go`.
**Root Cause:** Trivy action `v0.28.0` referenced an invalid/deleted `setup-trivy` tag. `identity.ReconstructUser` was updated with `emailVerified` and `emailVerifiedAt` arguments, but the integration test was not updated.
**Fix:** Pinned `aquasecurity/trivy-action` to `v0.34.0` (commit `c1824fd6edce30d7ab345a9989de00bbd46ef284`) and added `false, nil` to the `identity.ReconstructUser` call.

## 2026-01-23 - GoSec false positives and path traversal warnings
**Issue:** CI pipeline failed the GoSec security scan due to `G101` and `G304` rules.
**Root Cause:** `G101` flagged SQL query string constants containing the word 'Token' as potential hardcoded credentials. `G304` flagged `os.ReadFile(path)` calls in `loadPublicKey` and `loadPrivateKey` as potential file inclusion via variable.
**Fix:** Added `// #nosec G101` comments to the SQL queries and sanitized the paths using `filepath.Clean(path)` before calling `os.ReadFile`, suppressing the warning with `// #nosec G304`.
