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

## 2026-05-02 - Trivy Action Failure
**Issue:** `aquasecurity/trivy-action` failed with "Unable to resolve action `aquasecurity/setup-trivy@v0.2.1`, unable to find version `v0.2.1`" during the security workflow.
**Root Cause:** An outdated, pre-0.28.0 version of `aquasecurity/trivy-action` (pinned to `v0.28.0` via `915b19b`) hardcoded a dependency on `aquasecurity/setup-trivy@v0.2.1`. That version (`v0.2.1` of the setup action) was apparently removed or became unresolvable, breaking the action. The `version: 'v0.55.2'` input controls the Trivy binary, but the internal action dependency failed.
**Fix:** Update `aquasecurity/trivy-action` to a recent, stable version (e.g., `v0.34.0` pinned to `c1824fd6edce30d7ab345a9989de00bbd46ef284`) which no longer has the broken `setup-trivy` dependency and explicitly set the `trivy-version` input to `v0.70.0` (as the action now requires it instead of `version`). Also updated `.github/workflows/security.yml` to remove the outdated `version` field.
