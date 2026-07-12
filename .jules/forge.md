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

## 2026-07-12 - Trivy Action Resolution Failure
**Issue:** Security scanning workflows failed with 'Unable to resolve action aquasecurity/setup-trivy' or binary download errors.
**Root Cause:** Older versions of aquasecurity/trivy-action (e.g., v0.28.0) reference an invalid or deleted setup-trivy tag, breaking the CI pipeline.
**Fix:** Updated and pinned aquasecurity/trivy-action to a stable commit hash for v0.34.0 (c1824fd6edce30d7ab345a9989de00bbd46ef284) to resolve the invalid tag reference.

## 2026-07-12 - Node.js 20 Deprecation Warnings Fix
**Issue:** GitHub Actions workflows were failing or logging warnings due to Node.js 20 deprecation, causing cache service errors and potential tool download breaks.
**Root Cause:** The workflows used outdated actions that depended on Node.js 20, which is deprecated on GitHub Actions runners.
**Fix:** Updated standard actions (e.g., actions/checkout, actions/setup-go) to their modern versions and securely pinned them to their respective stable commit hashes.

## 2026-07-12 - Fix GoSec false positives for SQL constants and File Inclusion
**Issue:** GoSec failed CI with G101 (Potential hardcoded credentials) on SQL strings containing 'token' and G304 (Potential file inclusion via variable) on os.ReadFile calls using config variables.
**Root Cause:** GoSec's string pattern matching flag 'token' in SQL statements as a credential. G304 flags dynamic file paths in 'os.ReadFile'.
**Fix:** Used 'filepath.Clean' and '#nosec G304' comments for path usage based on application config, and suppressed G101 false positives with '#nosec G101' above the SQL constant blocks.
