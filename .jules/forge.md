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

## 2026-02-05 - Trivy Installation and Git Reference Failures
**Issue:** `security.yml` Trivy jobs were failing to find the git commit or failing to install the scanner.
**Root Cause:** The `actions/checkout` action didn't fetch enough git history by default (`fetch-depth: 1`), breaking Trivy's git repository scanning. In addition, the Trivy GitHub Action required the explicit parameter `version` instead of `trivy-version` which is an invalid input for this action version and will cause the check to fail.
**Fix:** Added `fetch-depth: 0` to the checkout step before Trivy and explicitly specified `version: '0.55.2'`.

## 2026-02-05 - Bash Subshell Pipeline Failures in GitHub Actions
**Issue:** The `setup-go-env` composite action failed when executing a wildcard `ls` inside command substitution `$(ls ... 2>/dev/null | ...)` because no matching files existed.
**Root Cause:** GitHub Actions defaults to `set -e -o pipefail`. If `ls` returns a non-zero exit code because it finds no files, the entire script terminates immediately, even inside a variable assignment.
**Fix:** Temporarily disabled exit-on-error with `set +e` before the command substitution and re-enabled it with `set -e` immediately after to allow graceful handling of empty matches.
