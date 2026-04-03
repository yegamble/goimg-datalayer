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

## 2026-01-23 - GitHub Actions $GITHUB_PATH Shadowing
**Issue:** Pipeline jobs relying on `setup-go-env` were failing with "Go 1.25+ required, but found Go 1.24" despite using `actions/setup-go@v5` configured to install Go 1.25.5.
**Root Cause:** The `setup-go-env` composite action was appending system paths (like `/usr/bin`) to `$GITHUB_PATH` *after* the `actions/setup-go` step. Since GitHub Actions prepends new `$GITHUB_PATH` entries, the system's older Go binary in `/usr/bin/go` took precedence over the newly downloaded Go 1.25 binary.
**Fix:** Reordered the steps in `setup-go-env/action.yml` so that system paths are added *before* `actions/setup-go`. This ensures the downloaded Go version is prepended last and correctly overrides any system binaries.
