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
## 2024-04-05 - Fix GitHub Actions System Path Shadowing & Trivy Version Pinning
**Issue:** `go: no such tool "covdata"` and `go.mod requires go >= 1.25` errors during CI composite actions, as well as pipeline failures on missing `trivy` versions (like `v0.55.2`).
**Root Cause:** In the `setup-go-env` composite action, the `Ensure node and system binary paths (Linux)` step appended paths like `/usr/bin` to `$GITHUB_PATH` *after* `actions/setup-go` was run. This caused the system's older Go binary to shadow the correct version installed by the setup action. Additionally, the `trivy-action` was unpinned from a stable release, causing missing version dependencies.
**Fix:** The `$GITHUB_PATH` manipulation must happen *before* running `actions/setup-go`. This ensures the setup action correctly places its installed Go binary ahead of the newly appended paths. The `trivy-action` uses `aquasecurity/trivy-action@c1824fd6edce30d7ab345a9989de00bbd46ef284 # v0.34.0` with `version: 'v0.69.3'` to avoid supply chain and versioning regressions.
