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

## 2026-01-23 - Trivy Scanner Configuration Issues
**Issue:** `security.yml`'s `trivy` job was failing due to git missing references or silent failure in trivy version parsing.
**Root Cause:** Trivy requires full git history to scan properly (`fetch-depth: 0`). Additionally, `aquasecurity/trivy-action@0.28.0` ignores the generic `version` parameter and expects `trivy-version` explicitly.
**Fix:** Added `fetch-depth: 0` to the checkout step in the trivy job and specified `trivy-version: '0.55.2'` in the `trivy-action` usage blocks.

## 2026-01-24 - Setup Go Env Error Handling
**Issue:** The `.github/actions/setup-go-env` composite action was failing during the "Ensure node and system binary paths" step on runners where `ls` matches no files. This occurred because `pipefail` and `errexit` caused `ls -d` on non-existent paths to fail the job before `sort` or `tail` executed.
**Root Cause:** Bash was running with `set -e -o pipefail`. `ls -d /opt/acttoolcache/node/*/x64/bin` failed and immediately stopped execution since the path did not exist on GitHub-hosted runners.
**Fix:** Wrapped the command in `set +e` and `set -e` to prevent failure on missing directories, and added `continue-on-error: true` as an additional safeguard.
