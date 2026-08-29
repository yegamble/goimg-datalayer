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

## 2026-01-23 - Trivy Action Resolution and Binary Download Error
**Issue:** Security workflow failing due to 'Unable to resolve action aquasecurity/setup-trivy' or binary download errors when running `aquasecurity/trivy-action`.
**Root Cause:** The action was pinned to an older version (`0.28.0` / `915b...`) which failed to resolve dependencies or download the specified Trivy binary version (`v0.55.2`, which may have been removed upstream or unsupported by the older action).
**Fix:** Pinned `aquasecurity/trivy-action` to a stable commit hash (`c1824fd6edce30d7ab345a9989de00bbd46ef284` for `v0.34.0`) and explicitly provided a valid release tag like `version: 'v0.69.3'` in the `with:` block to explicitly pin a working Trivy binary release and avoid failing on missing/removed upstream versions.
