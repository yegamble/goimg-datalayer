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
## 2026-05-03 - CI/CD PostgreSQL Health Check Failures
**Issue:** PostgreSQL service health checks in GitHub Actions workflows (ci.yml) were failing or logging errors during container initialization.
**Root Cause:** The `pg_isready` command in the health check `options` block was missing the specific database user argument (`-U goimg_test`), causing it to default to the root user, which does not exist in the configured test environment.
**Fix:** Explicitly appended the username flag to the command: `pg_isready -U goimg_test`.

## 2026-05-03 - CI/CD golangci-lint Version Mismatch
**Issue:** The linting job in GitHub Actions was failing because the requested `golangci-lint` version (`v2.6.2`) could not be resolved or downloaded.
**Root Cause:** The environment variable `GOLANGCI_LINT_VERSION` was set to `v2.6.2`, but the `golangci-lint` project is still on major version `1.x` (e.g., `v1.64.x`), making `v2.x` non-existent.
**Fix:** Pinned `GOLANGCI_LINT_VERSION` to a valid, stable release (`v1.64.5`) in `ci.yml` and the documentation.
