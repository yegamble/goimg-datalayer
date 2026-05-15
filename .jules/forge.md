## 2026-05-15 - Unpinned External Service Tags
**Issue:** `latest` and `stable` tags were used for external services in production docker-compose.
**Root Cause:** The `docker-compose.prod.yml` inherited tags from local deployment templates rather than adhering to production best practices for external dependencies.
**Fix:** Pinned `certbot`, `clamav`, `ipfs`, `prometheus`, and `grafana` to concrete semantic versions discovered from Docker Hub APIs to prevent unexpected upstream breakages.
## 2026-05-15 - Unpinned External Service Tags
**Issue:** `latest` and `stable` tags were used for external services in production docker-compose.
**Root Cause:** The `docker-compose.prod.yml` inherited tags from local deployment templates rather than adhering to production best practices for external dependencies.
**Fix:** Pinned `certbot`, `clamav`, `ipfs`, `prometheus`, and `grafana` to concrete semantic versions discovered from Docker Hub APIs to prevent unexpected upstream breakages.

## 2026-05-15 - Unpinned External Service Tags
**Issue:** `latest` and `stable` tags were used for external services in production docker-compose.
**Root Cause:** The `docker-compose.prod.yml` inherited tags from local deployment templates rather than adhering to production best practices for external dependencies.
**Fix:** Pinned `certbot`, `clamav`, `ipfs`, `prometheus`, and `grafana` to concrete semantic versions discovered from Docker Hub APIs to prevent unexpected upstream breakages.

## 2026-05-15 - GitHub Actions Node.js 20 Deprecation
**Issue:** GitHub Actions started throwing deprecation warnings for actions using Node.js 20.
**Root Cause:** Older versions of standard actions like `actions/checkout`, `actions/setup-go`, and `aquasecurity/trivy-action` relied on Node.js 20, which is scheduled for deprecation.
**Fix:** Updated the actions to their newer versions that use Node.js 24 (`checkout@v4.3.1`, `setup-go@v5.6.0`, `trivy-action@v0.34.0`, etc.) and pinned them with explicit hashes verified from the action repositories.
