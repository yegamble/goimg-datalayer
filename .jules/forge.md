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
## 2025-06-24 - Trivy Action Cache Issues
**Issue:** Updating `trivy-action` sequentially (`v0.28.0` -> `v0.34.0` -> `v0.36.0`) revealed that `v0.34.0` used an older `setup-trivy` action that sometimes failed to cache correctly on GitHub runners, throwing "No files were found with the provided path: trivy-fs-results.txt" due to binary installation exit code 1 failures.
**Root Cause:** The `version` parameter in the action's `with:` block (`version: 'v0.55.2'`) was causing exit codes during binary fetch inside `setup-trivy`.
**Fix:** Bumping `trivy-action` to `v0.36.0` (commit `a9c7b0f06e461e9d4b4d1711f154ee024b8d7ab8`) and strictly removing the explicit `version: 'v0.55.2'` constraint from the configuration allowed the action to gracefully fall back to its internal defaults, resolving the exit code failures and allowing artifacts to generate properly.

**Issue:** Updating `trivy-action` sequentially (`v0.28.0` -> `v0.34.0` -> `v0.36.0`) revealed that `v0.36.0` default behaviors on the runner resulted in Trivy version `v0.70.0` pulling its vulnerability DB and failing silently (exit code 1).
**Root Cause:** The `version` parameter in the action's `with:` block (`version: 'v0.55.2'`) was mistakenly removed during the action version bump, causing it to use a newer version (`v0.70.0`) which caused errors during the scan.
**Fix:** Restored the `version: 'v0.55.2'` constraint in both GitHub action steps (scan and format).
