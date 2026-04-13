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

## 2026-04-13 - Pinned Production Docker Image Tags
**Issue:** `docker/docker-compose.prod.yml` was using `:latest` or `:stable` tags for critical external services (`certbot`, `clamav`, `ipfs`, `prometheus`, `grafana`), violating the repository requirement for deterministic and reproducible deployments. Unpinned tags can lead to unexpected breakages during production deployments if upstream images introduce breaking changes.
**Root Cause:** External service definitions in the production compose file relied on default floating tags instead of explicit semantic versions.
**Fix:** Pinned all external service images in `docker/docker-compose.prod.yml` to specific, stable semantic versions (`certbot:v2.11.0`, `clamav:1.4.3`, `ipfs:v0.31.0`, `prometheus:v2.55.1`, `grafana:12.4.2`) discovered via Docker Hub or the local `docker-compose.yml`. Internal `goimg-*` images were left as `:latest` to preserve local deployment workflows.
