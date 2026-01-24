## 2026-01-20 - Pinned CI Tool Versions
**Issue:** `ci.yml` and `security.yml` were installing critical tools (`gosec`, `govulncheck`, `gocovmerge`, `grype`, `cyclonedx-gomod`) using `@latest` or unversioned scripts, leading to non-reproducible builds and potential supply chain vulnerabilities.
**Root Cause:** Tools were installed using `go install ...@latest` or `curl ... | sh` without version constraints.
**Fix:** Pinned all tools to specific versions or commit hashes (e.g., `gosec@v2.22.11`, `gocovmerge@b5bfa59`) to ensure stability and security.

## 2026-01-24 - Makefile Docker Compose Compatibility
**Issue:** `make docker-up` failed in environments without legacy `docker-compose` (Python-based), specifically on newer CI runners and developer machines.
**Root Cause:** The `Makefile` hardcoded the `docker-compose` command, but modern environments often provide only `docker compose` (V2 plugin).
**Fix:** Updated `Makefile` to dynamically detect availability of `docker compose` vs `docker-compose` and use the appropriate command, preferring V2.
