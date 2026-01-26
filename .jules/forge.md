## 2026-01-20 - Pinned CI Tool Versions
**Issue:** `ci.yml` and `security.yml` were installing critical tools (`gosec`, `govulncheck`, `gocovmerge`, `grype`, `cyclonedx-gomod`) using `@latest` or unversioned scripts, leading to non-reproducible builds and potential supply chain vulnerabilities.
**Root Cause:** Tools were installed using `go install ...@latest` or `curl ... | sh` without version constraints.
**Fix:** Pinned all tools to specific versions or commit hashes (e.g., `gosec@v2.22.11`, `gocovmerge@b5bfa59`) to ensure stability and security.

## 2026-03-09 - Docker Libc Incompatibility
**Issue:** `Dockerfile.api` and `Dockerfile.worker` mixed Alpine builder (musl) with Debian runtime (glibc) while copying `libvips` shared libraries. This creates a binary incompatibility causing crashes.
**Root Cause:** The `golang:alpine` image is musl-based, but `distroless/base-debian12` is glibc-based. Copied C libraries (vips, glib) from Alpine are not compatible with Debian.
**Fix:** Switched runtime to `alpine:3.21` to match the builder's libc (musl) and installed dependencies (`vips`) via package manager (`apk`) instead of manual copying.
