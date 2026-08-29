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

## 2026-02-10 - Silent Failure in Code Generation Drift Check
**Issue:** The CI job `openapi-validation` was passing successfully even when `make generate` failed due to missing tools, masking code drift between the OpenAPI spec and generated code.
**Root Cause:** The pipeline used `make generate || echo ...` to suppress failures, and `oapi-codegen` was not installed in the CI environment.
**Fix:** Installed `oapi-codegen@v2.5.1` in the CI job and removed error suppression for `make generate` to ensure the job fails on generation errors. Also updated `Makefile` to use `$(go env GOPATH)/bin` for better portability.

## 2026-02-17 - Security Scan Resource Exhaustion
**Issue:** Security scanning jobs were failing with billing/resource exhaustion errors ("spending limit") because they were running on every push, even for non-code changes.
**Root Cause:** The `security.yml` workflow triggers were too broad (`branches: "**"`), triggering heavy scans (Trivy, GoSec, SBOM) unnecessarily.
**Fix:** Added `paths-ignore` to `security.yml` to skip scans for documentation files (`**.md`, `docs/**`, `LICENSE`, `**.txt`). Kept triggers strict for PRs to main/develop but reduced noise from doc-only commits.

## 2026-02-17 - CI Build Job Cost Optimization
**Issue:** The main `ci.yml` workflow was failing due to spending limits, exacerbated by running builds on both `ubuntu-latest` and `macos-latest`.
**Root Cause:** The `build` job used a matrix strategy including `macos-latest`, which consumes GitHub Actions minutes at a 10x rate compared to Linux runners.
**Fix:** Removed the `matrix` strategy and `macos-latest` from the `build` job, standardizing on `ubuntu-latest`. This significantly reduces the billing impact while maintaining verification for the primary target environment (Linux containers).
