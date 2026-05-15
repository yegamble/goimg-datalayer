## 2026-05-15 - Unpinned External Service Tags
**Issue:** `latest` and `stable` tags were used for external services in production docker-compose.
**Root Cause:** The `docker-compose.prod.yml` inherited tags from local deployment templates rather than adhering to production best practices for external dependencies.
**Fix:** Pinned `certbot`, `clamav`, `ipfs`, `prometheus`, and `grafana` to concrete semantic versions discovered from Docker Hub APIs to prevent unexpected upstream breakages.
## 2026-05-15 - Unpinned External Service Tags
**Issue:** `latest` and `stable` tags were used for external services in production docker-compose.
**Root Cause:** The `docker-compose.prod.yml` inherited tags from local deployment templates rather than adhering to production best practices for external dependencies.
**Fix:** Pinned `certbot`, `clamav`, `ipfs`, `prometheus`, and `grafana` to concrete semantic versions discovered from Docker Hub APIs to prevent unexpected upstream breakages.
