# Sprint 10 Load Testing Strategy - Executive Summary

**Date**: 2026-01-06
**Sprint**: Sprint 10 - Security Enhancements
**Security Gates**: S10-PERF-001, S10-HIBP-003
**Status**: Ready for Execution

---

## Overview

This document provides an executive summary of the load testing strategy for Sprint 10 security features. For detailed instructions, see [SPRINT10_LOAD_TESTING_GUIDE.md](./SPRINT10_LOAD_TESTING_GUIDE.md).

---

## Features Under Test

### 1. Random Login Delay (Timing Attack Mitigation)

**Purpose**: Prevent credential enumeration via timing analysis by adding 100-300ms random delay to all login attempts.

**Implementation**:
- File: `/home/user/goimg-datalayer/internal/application/identity/timing.go`
- Delay: 100-300ms (cryptographically random)
- Coverage: ALL login attempts (success, failure, errors)

**Security Gate**: S10-PERF-001
- **Requirement**: p95 login latency < 500ms (including delay)
- **Risk**: Timing attacks could reveal valid usernames/emails

### 2. HIBP Password Check Integration

**Purpose**: Reject passwords found in data breaches using Have I Been Pwned API with k-anonymity privacy model.

**Implementation**:
- File: `/home/user/goimg-datalayer/internal/infrastructure/security/hibp_client.go`
- API: HIBP Passwords API (k-anonymity: only 5 chars of hash sent)
- Cache: Redis (24h TTL) + in-memory fallback
- Fail-open: Enabled (API failures don't block registration)

**Security Gate**: S10-HIBP-003
- **Requirement**: Reject compromised passwords, fail-open on API failure
- **Risk**: Weak/compromised passwords in production

---

## Test Scripts

All scripts located in `/home/user/goimg-datalayer/tests/load/`

| Script | Purpose | Duration | Key Metric |
|--------|---------|----------|------------|
| `sprint10-login-timing.js` | Verify p95 latency < 500ms | 13 min | Login p95 latency |
| `sprint10-hibp-registration.js` | Verify HIBP rejection/acceptance | 9 min | Rejection rate > 95% |
| `sprint10-hibp-failopen.js` | Verify fail-open behavior | 5 min | Success rate > 95% |

---

## Success Criteria

### Security Gate S10-PERF-001 (Login Timing)

| Metric | Target | Critical? |
|--------|--------|-----------|
| **Login p95 latency** | **< 500ms** | **YES** (Gate blocker) |
| Login p99 latency | < 750ms | Recommended |
| Timing consistency | > 95% | **YES** |
| Success vs. failure timing diff | < 50ms | **YES** |

**Pass Condition**: All critical metrics must meet targets under 100 concurrent users.

### Security Gate S10-HIBP-003 (Password Check)

| Metric | Target | Critical? |
|--------|--------|-----------|
| **Compromised password rejection** | **> 95%** | **YES** (Gate blocker) |
| **Strong password acceptance** | **> 95%** | **YES** (Gate blocker) |
| **Fail-open success rate** | **> 95%** | **YES** (Gate blocker) |
| Registration p95 latency | < 2s | Recommended |
| HIBP cache hit rate | > 90% | Recommended |

**Pass Condition**: All critical metrics must meet targets.

---

## Running Tests

### Quick Start (Local)

```bash
# 1. Start infrastructure and API
make docker-up
make migrate-up
make run

# 2. Run all Sprint 10 tests (~27 minutes total)
make test-load-sprint10-all
```

### Individual Tests

```bash
# Login timing test (13 minutes)
make test-load-sprint10-login

# HIBP registration test (9 minutes)
make test-load-sprint10-hibp

# HIBP fail-open test (5 minutes) - requires HIBP_ENABLED=false
export HIBP_ENABLED=false
make run
make test-load-sprint10-failopen
```

### CI/CD Integration

Add to `.github/workflows/load-tests.yml` (see guide for full example):

```yaml
- name: Run Sprint 10 Load Tests
  run: |
    k6 run tests/load/sprint10-login-timing.js --out json=results.json
    # Fail if p95 > 500ms
    LOGIN_P95=$(jq '.metrics.http_req_duration.values["p(95)"]' results.json)
    if (( $(echo "$LOGIN_P95 < 500" | bc -l) )); then
      echo "✅ S10-PERF-001 PASSED"
    else
      exit 1
    fi
```

---

## Expected Results

### Login Timing Test

**Healthy Output**:
```
✓ successful login: status 200                 98.5%
✓ successful login: delay in range             97.2%
✓ failed login: status 401                     98.8%
✓ failed login: delay in range                 97.1%

http_req_duration{endpoint:login_success}: p(95)=420ms  ← PASS
http_req_duration{endpoint:login_failure}: p(95)=415ms  ← PASS
login_delay_consistency: 97.2%                           ← PASS

Security Gate S10-PERF-001: PASS ✓
```

**Problem Indicators**:
- p95 > 500ms → Database/Redis performance issue
- Success vs. failure timing diff > 50ms → Timing leak detected
- Delay consistency < 95% → Random delay not consistently applied

### HIBP Registration Test

**Healthy Output**:
```
✓ compromised password: status 400             96.8%
✓ strong password: status 201                  98.1%

compromised_password_rejection_rate: 96.8%  ← PASS
strong_password_acceptance_rate: 98.1%      ← PASS
hibp_check_duration: p(50)=45ms p(95)=1200ms

Security Gate S10-HIBP-003: PASS ✓
```

**Problem Indicators**:
- Rejection rate < 95% → HIBP API issues or cache problems
- Acceptance rate < 95% → False positives or HIBP failures
- p50 > 100ms → Poor cache hit rate

### Fail-Open Test

**Healthy Output**:
```
✓ failopen: registration succeeds              99.2%

failopen_success_rate: 99.2%                ← PASS
failopen_registration_duration: p(95)=620ms ← PASS

Security Gate S10-HIBP-003 (Fail-Open): PASS ✓
```

**Check Server Logs**:
```
WARN HIBP API check failed fail_open=true
INFO user registered successfully (HIBP check skipped)
```

---

## Performance Baselines

### Login (100 VUs)

| Metric | Target | Acceptable | Poor |
|--------|--------|------------|------|
| p50 | 200-250ms | < 300ms | > 300ms |
| p95 | < 500ms | < 550ms | > 550ms |
| p99 | < 750ms | < 900ms | > 900ms |

### Registration with HIBP (20 req/s)

| Scenario | p95 Target | Acceptable | Poor |
|----------|------------|------------|------|
| Cache hit | < 200ms | < 300ms | > 300ms |
| Cache miss | < 2000ms | < 2500ms | > 2500ms |
| Fail-open | < 1000ms | < 1500ms | > 1500ms |

---

## Troubleshooting Quick Reference

| Problem | Symptom | Likely Cause | Fix |
|---------|---------|--------------|-----|
| **High latency** | p95 > 500ms | Database slow | Add indexes, scale DB |
| **Timing leak** | Success ≠ Failure timing | Delay not on all paths | Check defer placement |
| **Low rejection rate** | < 95% rejection | HIBP API issues | Check connectivity, logs |
| **Fail-open fails** | Errors when HIBP down | Fail-open = false | Set HIBP_FAIL_OPEN=true |
| **Cache misses** | p50 > 100ms | Redis not working | Check Redis connection |

---

## Next Steps After Testing

### 1. Document Results

Update `/home/user/goimg-datalayer/docs/sprints/sprint-10.md`:

```markdown
| S10-PERF-001 | <500ms p95 login latency | ✅ PASS | p95=420ms (load test) |
| S10-HIBP-003 | Fail-open behavior | ✅ PASS | 96.8% rejection (load test) |
```

### 2. Security Gate Sign-Off

Mark gates as complete in `/home/user/goimg-datalayer/claude/NEXT_STEPS.md`:

```markdown
| S10-PERF-001: <500ms p95 latency | ✅ VERIFIED | Load tests passed |
| S10-HIBP-003: Fail-open behavior | ✅ VERIFIED | All scenarios passed |
```

### 3. Production Monitoring

Set up Grafana dashboards for:
- `goimg_auth_login_delay_seconds` (histogram)
- `goimg_security_hibp_checks_total` (counter by result)
- `goimg_security_hibp_check_duration_seconds` (histogram)

Configure alerts:
- p95 login latency > 500ms
- HIBP cache hit rate < 90%
- HIBP error rate > 5%

### 4. CI/CD Integration

Add Sprint 10 load tests to nightly CI runs:
- Run on schedule (nightly at 2 AM)
- Fail build if thresholds not met
- Store results for trend analysis

---

## Resources

### Documentation
- **Detailed Guide**: [SPRINT10_LOAD_TESTING_GUIDE.md](./SPRINT10_LOAD_TESTING_GUIDE.md)
- **Sprint 10 Plan**: `/home/user/goimg-datalayer/claude/sprint_10_plan.md`
- **Security Gates**: `/home/user/goimg-datalayer/claude/security_gates.md`
- **General Load Testing**: `/home/user/goimg-datalayer/docs/performance/load-testing.md`

### Test Files
- Login timing: `/home/user/goimg-datalayer/tests/load/sprint10-login-timing.js`
- HIBP registration: `/home/user/goimg-datalayer/tests/load/sprint10-hibp-registration.js`
- Fail-open: `/home/user/goimg-datalayer/tests/load/sprint10-hibp-failopen.js`

### Implementation Files
- Timing delay: `/home/user/goimg-datalayer/internal/application/identity/timing.go`
- HIBP client: `/home/user/goimg-datalayer/internal/infrastructure/security/hibp_client.go`
- Password cache: `/home/user/goimg-datalayer/internal/infrastructure/security/password_cache.go`

### External References
- [k6 Documentation](https://k6.io/docs/)
- [HIBP API Documentation](https://haveibeenpwned.com/API/v3)
- [OWASP Timing Attack Guide](https://owasp.org/www-community/attacks/Timing_attack)

---

## Summary

Sprint 10 load testing verifies two critical security features work correctly under production load:

1. **Random Login Delay**: Prevents timing attacks by masking credential validity through consistent 100-300ms delays
2. **HIBP Password Check**: Rejects compromised passwords while maintaining privacy and availability

**Critical Success Criteria**:
- ✅ p95 login latency < 500ms (S10-PERF-001)
- ✅ Compromised password rejection > 95% (S10-HIBP-003)
- ✅ Fail-open success rate > 95% (S10-HIBP-003)

**Total Test Duration**: ~27 minutes (all tests combined)

**Commands**:
```bash
make test-load-sprint10-all    # Run all tests
make test-load-sprint10-login  # Login timing only
make test-load-sprint10-hibp   # HIBP registration only
```

For detailed instructions, troubleshooting, and CI/CD integration, see [SPRINT10_LOAD_TESTING_GUIDE.md](./SPRINT10_LOAD_TESTING_GUIDE.md).
