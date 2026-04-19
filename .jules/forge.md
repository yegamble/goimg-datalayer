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

## 2026-01-23 - GitHub Actions PATH Shadowing
**Issue:** `go: no such tool "covdata"` and other version mismatch errors in local GitHub Actions runners (like `act`).
**Root Cause:** The `.github/actions/setup-go-env` action appended `/usr/bin` to `$GITHUB_PATH` *after* running `actions/setup-go`. Because GitHub Actions prepends `$GITHUB_PATH` entries to the environment's `PATH`, the older system `go` binary located in `/usr/bin` shadowed the correct Go version downloaded by the setup action.
**Fix:** Moved the step that modifies `$GITHUB_PATH` to run *before* `actions/setup-go`.

## 2026-01-23 - PostgreSQL Healthcheck User Error
**Issue:** PostgreSQL service healthchecks in `ci.yml` were failing with `role "root" does not exist` and delaying job execution.
**Root Cause:** The `--health-cmd pg_isready` command was executed as the default user (root) inside the postgres container, but the container was configured with `POSTGRES_USER: goimg_test`.
**Fix:** Modified the healthcheck command to explicitly specify the user: `--health-cmd "pg_isready -U goimg_test"`.
