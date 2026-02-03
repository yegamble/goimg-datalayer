# Repository Cleanup Recommendations

**Date:** February 3, 2026  
**Analysis Summary:** 82 branches, 29 open PRs

## Executive Summary

This repository has accumulated many stale branches and PRs from automated agents (Claude, Copilot, Jules, Sentinel, Forge, Codex) and failed/duplicate attempts at fixes. 

**Recommended Actions:**
- Delete **79 stale branches** (keep `main` and current working branch)
- Close **24 stale PRs** (keep 5 dependabot PRs for review)

---

## Branches to Delete (79 total)

### 1. Duplicate Security Fix Branches (11 branches)
Same security fix attempted multiple times:

```bash
# MD5/SHA256 ETag fix duplicates (6 branches)
git push origin --delete security-fix-md5-etag-1375316705603946514
git push origin --delete security-fix-md5-etag-554331649697776233
git push origin --delete security/sha256-etag-8979186410250336942
git push origin --delete security/replace-md5-etag-4049267853982122024
git push origin --delete jules-security-md5-etag-replacement-7147065775503703952
git push origin --delete jules-security-replace-md5-etag-16851011127076795257

# Login rate limit fix duplicates (2 branches)
git push origin --delete security-fix-login-rate-limit-2408944352288426382
git push origin --delete security-fix-login-rate-limit-4368964247785590030

# Path traversal fix duplicates (3 branches)
git push origin --delete security-fix-path-traversal-9404772229033786952
git push origin --delete security/fix-local-storage-path-traversal-14670171853657262651
git push origin --delete security/fix-local-storage-traversal-zipslip-10587079317327877704
```

### 2. Duplicate CI/Docker Fix Branches (6 branches)
```bash
git push origin --delete ci/fix-docker-abi-15431493006667416070
git push origin --delete ci/fix-docker-compose-compatibility-18225642296448185779
git push origin --delete ci/fix-docker-compose-compatibility-2278436381825161805
git push origin --delete ci/fix-docker-compose-ports-and-versions-10520307194344139039
git push origin --delete fix/docker-libc-mismatch-1807783486627555365
git push origin --delete fix-docker-libc-mismatch-12719899614106104505
```

### 3. Duplicate Token Blacklist Test Branches (4 branches)
```bash
git push origin --delete enable-token-blacklist-tests-7060900152864331631
git push origin --delete enable-token-blacklist-tests-7645811878176344175
git push origin --delete jules-enable-token-blacklist-integration-tests-12858512467458308987
git push origin --delete test/enable-token-blacklist-integration-tests-14109797389539064421
```

### 4. Old Claude Agent Branches (27 branches)
Sprint planning, documentation, and linting branches that are no longer needed:

```bash
git push origin --delete claude/add-claude-documentation-YqRE9
git push origin --delete claude/analyze-test-coverage-XSmuN
git push origin --delete claude/check-github-workflows-yeyG7
git push origin --delete claude/check-test-coverage-VN3N7
git push origin --delete claude/document-and-update-roadmap-nsON7
git push origin --delete claude/fix-github-workflows-Tzkhb
git push origin --delete claude/fix-golangci-lint-016iUwN4izx5ifZkSRvtmdmu
git push origin --delete claude/fix-linting-errors-01BBkCofpXVDfTnraNhjkEtp
git push origin --delete claude/fix-linting-errors-01E4eGB3fjMfkSttJChGvTNT
git push origin --delete claude/fix-linting-errors-01Uxr6EX1L8q2xaaspJx9Xyo
git push origin --delete claude/improve-claude-md-files-011ynxyBxoHsA4M1Y3qkZLeR
git push origin --delete claude/launch-security-setup-013WiD9pcVyEuCm5QUxKMJgC
git push origin --delete claude/review-sprint-documentation-01VDZbE3jwywtYHU6fK5nfxt
git push origin --delete claude/sprint-1-tasks-011eSXkihmG9jtCi7RXCV5yL
git push origin --delete claude/sprint-2-planning-01LRQYb1zToeQXzn8BDkQ7z3
git push origin --delete claude/sprint-4-setup-01CvnDkFyieHJTUcS4SgK5M3
git push origin --delete claude/sprint-8-setup-01HLDzjMw3yWqaEe9p4dh9rr
git push origin --delete claude/sprint-planning-8-9-01HDGWaW67ZpX7Dx9Zhrf5XZ
git push origin --delete claude/update-readme-sprint-23-N3Jyy
git push origin --delete claude/update-roadmap-continue-sprints-C2txt
git push origin --delete claude/update-roadmap-phase3-FVbaS
git push origin --delete claude/update-roadmap-phase3-jQr3m
git push origin --delete claude/update-roadmap-phase-2-EpcQE
git push origin --delete claude/update-roadmap-sprint-17-6PkBB
git push origin --delete claude/update-roadmap-sprint-18-J9WGM
git push origin --delete claude/update-roadmap-sprint-18-d5SDF
git push origin --delete claude/update-sentinel-prompt-8qFRx
```

### 5. Stale Forge/Codex Agent Branches (8 branches)
```bash
git push origin --delete forge/pin-actions-14343198999953324075
git push origin --delete forge/pin-ci-dependencies-17334836860526225073
git push origin --delete forge/pin-ci-dependencies-7093034775483950354
git push origin --delete forge/pin-tool-versions-16396412678344479529
git push origin --delete forge/pin-upload-sarif-16094550786444948933
git push origin --delete forge-pin-actions-4942230978625776927
git push origin --delete codex/fix-failing-tests
git push origin --delete codex/improve-claude.md-and-reorganize-file-structure
```

### 6. Stale Feature Branches (17 branches)
```bash
git push origin --delete builder-moderation-tests-and-fixes-6943332295507740625
git push origin --delete fix/album-auth-filtering-15341512927078286186
git push origin --delete fix/album-auth-filtering-test-9503893918382737249
git push origin --delete fix/album-auth-filtering-verification-5999695645193353278
git push origin --delete task-album-auth-filtering-10945000699189417518
git push origin --delete fix/group-invitations-sprint20-5944539903468025430
git push origin --delete perf/group-membership-upsert-7381823249264549948
git push origin --delete perf/optimize-list-featured-images-verify-16262974457895754797
git push origin --delete perf/optimize-mark-notifications-read-13453079122065894093
git push origin --delete scribe/update-api-docs-groups-2801328044923629134
git push origin --delete sentinel/fix-mime-type-bypass-11348313214300286343
git push origin --delete sentinel-fix-image-upload-mime-spoofing-4999943321314220073
git push origin --delete infra/secure-tool-install-2186754529461026975
git push origin --delete update-image-scan-infected-status-17509551326609334860
git push origin --delete add-infected-file-counter-17936019205218820878
git push origin --delete jules-notify-malware-12291480927758410152
git push origin --delete jules-wiring-and-infrastructure-fixes-11184020412046105919
```

### 7. Stale CI/Jules Branches (5 branches)
```bash
git push origin --delete ci-optimize-build-artifacts-16859937477280289705
git push origin --delete ci/infrastructure-optimization-composite-action-7321679355279520626
git push origin --delete jules-ci-setup-go-env-6383912144870456015
git push origin --delete jules-enable-postman-e2e-tests-6304857716229548836
git push origin --delete jules-security-rate-limit-upload-10762845310385457464
```

### 8. Duplicate Copilot Branch (1 branch)
```bash
git push origin --delete copilot/delete-stale-branches
```

---

## PRs to Close (24 total)

### Stale/Duplicate PRs (Close without merging)

| PR # | Title | Reason to Close |
|------|-------|-----------------|
| #194 | [WIP] Remove stale and unused branches | Duplicate of this PR #195 |
| #170 | Enable Postman E2E tests in CI/CD | Stale, not progressing |
| #166 | security: add rate limiting to image upload endpoint | Review and close if superseded |
| #163 | ci: Add composite action for Go environment setup | Stale optimization |
| #162 | security: Replace MD5 with SHA-256 for ETag generation | Duplicate security fix |
| #161 | ci: Extract Go setup and libvips installation | Duplicate of #163 |
| #160 | security: Replace MD5 with SHA-256 for ETag generation | Duplicate security fix |
| #155 | Secure installation of security scanning tools | Stale |
| #153 | security: Fix weak hashing in local storage ETag | Duplicate security fix |
| #147 | security: Fix login rate limit bypass | Review - may be valuable |
| #145 | Enable and enhance TokenBlacklist integration tests | Stale test enhancement |
| #143 | security: Replace MD5 with SHA-256 for ETag generation | Duplicate security fix |
| #142 | ci: Fix Docker Compose command detection | Stale CI fix |
| #141 | security: Replace MD5 with SHA-256 for ETag generation | Duplicate security fix |
| #140 | ci: Fix Docker build libc mismatch | Stale CI fix |
| #139 | security: Replace MD5 with SHA-256 for ETag generation | Duplicate security fix |
| #138 | ci: Pin upload-sarif action and fix version comments | Stale CI fix |
| #137 | security: Add rate limiting to login endpoint | Duplicate of #147 |
| #131 | ci: Fix make docker-up failure | Stale CI fix |
| #130 | Fix ClamAV scan bypass in image upload | Review - security fix |
| #122 | Verify AlbumHandler Authorization Filtering | Stale verification |
| #120 | Verify and ensure album list authorization | Stale verification |
| #107 | Add infected file counter for users | Stale feature |

---

## PRs to Evaluate (5 - Dependabot Updates)

These are dependency updates that should be reviewed and either merged or closed:

| PR # | Title | Action |
|------|-------|--------|
| #177 | Bump actions/setup-node from 4.0.2 to 6.2.0 | Review & merge if passing |
| #176 | Bump actions/upload-artifact from 4.4.0 to 6.0.0 | Review & merge if passing |
| #175 | Bump golangci/golangci-lint-action from 6.1.1 to 9.2.0 | Review & merge if passing |
| #174 | Bump marocchino/sticky-pull-request-comment from 2.9.0 to 2.9.4 | Review & merge if passing |
| #173 | Bump aquasecurity/trivy-action from 0.28.0 to 0.33.1 | Review & merge if passing |

---

## Quick Cleanup Script

Run this script to delete all stale branches at once:

```bash
#!/bin/bash
# Save as cleanup-branches.sh and run with: bash cleanup-branches.sh

BRANCHES_TO_DELETE=(
  # Security duplicates
  "security-fix-md5-etag-1375316705603946514"
  "security-fix-md5-etag-554331649697776233"
  "security/sha256-etag-8979186410250336942"
  "security/replace-md5-etag-4049267853982122024"
  "jules-security-md5-etag-replacement-7147065775503703952"
  "jules-security-replace-md5-etag-16851011127076795257"
  "security-fix-login-rate-limit-2408944352288426382"
  "security-fix-login-rate-limit-4368964247785590030"
  "security-fix-path-traversal-9404772229033786952"
  "security/fix-local-storage-path-traversal-14670171853657262651"
  "security/fix-local-storage-traversal-zipslip-10587079317327877704"
  
  # CI/Docker duplicates
  "ci/fix-docker-abi-15431493006667416070"
  "ci/fix-docker-compose-compatibility-18225642296448185779"
  "ci/fix-docker-compose-compatibility-2278436381825161805"
  "ci/fix-docker-compose-ports-and-versions-10520307194344139039"
  "fix/docker-libc-mismatch-1807783486627555365"
  "fix-docker-libc-mismatch-12719899614106104505"
  
  # Token blacklist duplicates
  "enable-token-blacklist-tests-7060900152864331631"
  "enable-token-blacklist-tests-7645811878176344175"
  "jules-enable-token-blacklist-integration-tests-12858512467458308987"
  "test/enable-token-blacklist-integration-tests-14109797389539064421"
  
  # Claude branches
  "claude/add-claude-documentation-YqRE9"
  "claude/analyze-test-coverage-XSmuN"
  "claude/check-github-workflows-yeyG7"
  "claude/check-test-coverage-VN3N7"
  "claude/document-and-update-roadmap-nsON7"
  "claude/fix-github-workflows-Tzkhb"
  "claude/fix-golangci-lint-016iUwN4izx5ifZkSRvtmdmu"
  "claude/fix-linting-errors-01BBkCofpXVDfTnraNhjkEtp"
  "claude/fix-linting-errors-01E4eGB3fjMfkSttJChGvTNT"
  "claude/fix-linting-errors-01Uxr6EX1L8q2xaaspJx9Xyo"
  "claude/improve-claude-md-files-011ynxyBxoHsA4M1Y3qkZLeR"
  "claude/launch-security-setup-013WiD9pcVyEuCm5QUxKMJgC"
  "claude/review-sprint-documentation-01VDZbE3jwywtYHU6fK5nfxt"
  "claude/sprint-1-tasks-011eSXkihmG9jtCi7RXCV5yL"
  "claude/sprint-2-planning-01LRQYb1zToeQXzn8BDkQ7z3"
  "claude/sprint-4-setup-01CvnDkFyieHJTUcS4SgK5M3"
  "claude/sprint-8-setup-01HLDzjMw3yWqaEe9p4dh9rr"
  "claude/sprint-planning-8-9-01HDGWaW67ZpX7Dx9Zhrf5XZ"
  "claude/update-readme-sprint-23-N3Jyy"
  "claude/update-roadmap-continue-sprints-C2txt"
  "claude/update-roadmap-phase3-FVbaS"
  "claude/update-roadmap-phase3-jQr3m"
  "claude/update-roadmap-phase-2-EpcQE"
  "claude/update-roadmap-sprint-17-6PkBB"
  "claude/update-roadmap-sprint-18-J9WGM"
  "claude/update-roadmap-sprint-18-d5SDF"
  "claude/update-sentinel-prompt-8qFRx"
  
  # Forge/Codex branches
  "forge/pin-actions-14343198999953324075"
  "forge/pin-ci-dependencies-17334836860526225073"
  "forge/pin-ci-dependencies-7093034775483950354"
  "forge/pin-tool-versions-16396412678344479529"
  "forge/pin-upload-sarif-16094550786444948933"
  "forge-pin-actions-4942230978625776927"
  "codex/fix-failing-tests"
  "codex/improve-claude.md-and-reorganize-file-structure"
  
  # Feature branches
  "builder-moderation-tests-and-fixes-6943332295507740625"
  "fix/album-auth-filtering-15341512927078286186"
  "fix/album-auth-filtering-test-9503893918382737249"
  "fix/album-auth-filtering-verification-5999695645193353278"
  "task-album-auth-filtering-10945000699189417518"
  "fix/group-invitations-sprint20-5944539903468025430"
  "perf/group-membership-upsert-7381823249264549948"
  "perf/optimize-list-featured-images-verify-16262974457895754797"
  "perf/optimize-mark-notifications-read-13453079122065894093"
  "scribe/update-api-docs-groups-2801328044923629134"
  "sentinel/fix-mime-type-bypass-11348313214300286343"
  "sentinel-fix-image-upload-mime-spoofing-4999943321314220073"
  "infra/secure-tool-install-2186754529461026975"
  "update-image-scan-infected-status-17509551326609334860"
  "add-infected-file-counter-17936019205218820878"
  "jules-notify-malware-12291480927758410152"
  "jules-wiring-and-infrastructure-fixes-11184020412046105919"
  
  # CI/Jules branches
  "ci-optimize-build-artifacts-16859937477280289705"
  "ci/infrastructure-optimization-composite-action-7321679355279520626"
  "jules-ci-setup-go-env-6383912144870456015"
  "jules-enable-postman-e2e-tests-6304857716229548836"
  "jules-security-rate-limit-upload-10762845310385457464"
  
  # Duplicate copilot branch
  "copilot/delete-stale-branches"
)

echo "Deleting ${#BRANCHES_TO_DELETE[@]} stale branches..."

for branch in "${BRANCHES_TO_DELETE[@]}"; do
  echo "Deleting: $branch"
  git push origin --delete "$branch" 2>/dev/null || echo "  (may already be deleted)"
done

echo "Done! Deleted ${#BRANCHES_TO_DELETE[@]} branches."
```

---

## Post-Cleanup

After running the cleanup:
1. Verify the branch count reduced from 82 to ~3 (main + dependabot branches + current working)
2. Review and either merge or close the 5 dependabot PRs
3. Close PR #194 as duplicate
4. Consider this PR (#195) complete and merge it

---

## Notes

- **Dependabot branches** are kept because they represent legitimate dependency updates that should be reviewed
- **main** is the protected default branch
- The random numbers in branch names (e.g., `-17936019205218820878`) are typical of auto-generated agent branches
- Many security fixes were attempted multiple times, suggesting CI/build issues that prevented merging
