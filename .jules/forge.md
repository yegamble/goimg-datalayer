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

## 2025-02-21 - CI Pipeline Fixes
**Issue:** Various CI checks failed due to outdated Node.js versions in GitHub Actions, missing arguments in test helpers, and G101/G304 security warnings.
**Root Cause:**
1. Actions `@v3`/`@v4` dependencies triggered Node.js 20 deprecation warnings.
2. `aquasecurity/trivy-action` was pointing to an unresolvable version or commit.
3. `identity.ReconstructUser` lacked newly added fields in tests.
4. `gosec` flagged SQL statements containing "Token" as hardcoded credentials (G101) and `os.ReadFile` usage as potential path traversal (G304).
**Fix:**
1. Updated action versions (`actions/checkout`, `actions/setup-go`, `actions/upload-artifact`, `github/codeql-action`, `golangci/golangci-lint-action`) to newer SHAs.
2. Pinned `aquasecurity/trivy-action` to a valid `v0.34.0` hash (`c1824fd...`) and updated `trivy-version` if necessary.
3. Added missing boolean and time pointer parameters to `identity.ReconstructUser` in `user_repository_test.go`.
4. Appended `// #nosec G101` and `// #nosec G304` and used `filepath.Clean(path)` in `token_repository.go` and `jwt_service.go` respectively.
