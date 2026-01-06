# Sprint 10 Load Testing Guide

> Comprehensive load testing strategy for Sprint 10 security features: Random Login Delay and HIBP Password Check

**Sprint**: Sprint 10 - Security Enhancements
**Security Gate**: S10-PERF-001
**Last Updated**: 2026-01-06

---

## Table of Contents

- [Overview](#overview)
- [Features Under Test](#features-under-test)
- [Test Scripts](#test-scripts)
- [Success Criteria](#success-criteria)
- [Running Tests Locally](#running-tests-locally)
- [Running Tests in CI](#running-tests-in-ci)
- [Interpreting Results](#interpreting-results)
- [Troubleshooting](#troubleshooting)
- [Performance Baselines](#performance-baselines)

---

## Overview

Sprint 10 introduced two critical security features that impact API performance:

1. **Random Login Delay (100-300ms)**: Mitigates timing attacks by adding cryptographically random delays to all login attempts
2. **HIBP Password Check**: Integrates with Have I Been Pwned API to reject compromised passwords

This guide provides comprehensive load testing strategies to verify these features work correctly under production-like load and meet Security Gate S10-PERF-001 requirements.

---

## Features Under Test

### Feature 1: Random Login Delay (Timing Attack Mitigation)

**Implementation**:
- File: `/home/user/goimg-datalayer/internal/application/identity/timing.go`
- Delay range: 100-300ms (cryptographically random using `crypto/rand`)
- Applied to: **ALL** login attempts (success, failure, errors)
- Purpose: Prevent credential enumeration via timing analysis

**Security Properties**:
- Consistent timing for valid vs. invalid credentials
- No timing information leaked in logs or error messages
- Random distribution prevents averaging attacks

**Performance Requirement (S10-PERF-001)**:
- **p95 login latency < 500ms** (including the 100-300ms delay)

### Feature 2: HIBP Password Check

**Implementation**:
- File: `/home/user/goimg-datalayer/internal/infrastructure/security/hibp_client.go`
- API: Have I Been Pwned Passwords API (k-anonymity model)
- Cache: Redis + in-memory fallback (24h TTL for negative results)
- Fail-open: API failures do not block registration

**Security Properties**:
- Privacy: Only first 5 chars of SHA-1 hash sent to HIBP
- Availability: Fail-open configuration allows registration if API is down
- Performance: Redis caching reduces API calls (>90% cache hit rate expected)

**Performance Characteristics**:
- Cache hit: <100ms
- Cache miss (API call): 100-500ms
- Fail-open (API unavailable): <100ms (no external call)

---

## Test Scripts

All test scripts are located in `/home/user/goimg-datalayer/tests/load/`

### 1. Login Timing Test (`sprint10-login-timing.js`)

**Purpose**: Verify random login delay is consistently applied and meets p95 latency requirement.

**Scenarios**:
1. **Timing Verification** (ramping-vus):
   - 0 → 10 → 50 → 100 → 10 → 0 VUs over 13 minutes
   - Tests p95 latency under increasing load
   - Measures both successful and failed login timing

2. **Consistency Check** (constant-vus):
   - 20 VUs for 10 minutes
   - Statistical analysis of timing distribution
   - Compares success vs. failure timing profiles

**Key Metrics**:
- `login_success_duration`: Timing for valid credentials
- `login_failure_duration`: Timing for invalid credentials
- `login_delay_consistency`: Percentage of requests within 100-500ms range

**Success Criteria**:
- p95 login latency < 500ms (both success and failure)
- Delay consistency > 95%
- Timing difference between success/failure < 50ms

**Run Command**:
```bash
make test-load-sprint10-login
# OR
k6 run tests/load/sprint10-login-timing.js
```

### 2. HIBP Registration Test (`sprint10-hibp-registration.js`)

**Purpose**: Verify HIBP password checking rejects compromised passwords and accepts strong passwords.

**Scenarios**:
1. **Compromised Password Test** (constant-arrival-rate):
   - 5 registrations/second for 5 minutes
   - Uses known compromised passwords (password123, admin123, etc.)
   - Expects 400 Bad Request with breach error message

2. **Strong Password Test** (constant-arrival-rate):
   - 10 registrations/second for 5 minutes
   - Uses cryptographically random 16-character passwords
   - Expects 201 Created with user + tokens

3. **Mixed Registration** (ramping-arrival-rate):
   - 5 → 10 → 20 → 5 registrations/second over 9 minutes
   - 80% strong passwords, 20% compromised (realistic ratio)
   - Tests system behavior under mixed load

**Key Metrics**:
- `compromised_password_rejection_rate`: % of compromised passwords rejected
- `strong_password_acceptance_rate`: % of strong passwords accepted
- `hibp_check_duration`: HIBP API + cache latency

**Success Criteria**:
- Compromised password rejection rate > 95%
- Strong password acceptance rate > 95%
- p95 registration latency < 2s (includes HIBP API call)
- p50 HIBP check < 100ms (indicates good cache hit rate)

**Run Command**:
```bash
make test-load-sprint10-hibp
# OR
k6 run tests/load/sprint10-hibp-registration.js
```

### 3. HIBP Fail-Open Test (`sprint10-hibp-failopen.js`)

**Purpose**: Verify system continues to work when HIBP API is unavailable (fail-open behavior).

**Scenarios**:
1. **Fail-Open Verification** (constant-vus):
   - 20 VUs for 5 minutes
   - Registration with HIBP disabled or unreachable
   - Expects successful registration despite HIBP unavailable

**Key Metrics**:
- `failopen_success_rate`: % of registrations that succeed
- `failopen_registration_duration`: Latency without HIBP

**Success Criteria**:
- Registration success rate > 95% (despite HIBP unavailable)
- p95 latency < 1s (faster without HIBP API call)
- Server logs show HIBP warnings (not errors)

**Prerequisites**:
Set environment variable before starting API server:
```bash
export HIBP_ENABLED=false
make run
```

OR block HIBP API network access:
```bash
# Add to /etc/hosts
127.0.0.1 api.pwnedpasswords.com
```

**Run Command**:
```bash
make test-load-sprint10-failopen
# OR
k6 run tests/load/sprint10-hibp-failopen.js
```

---

## Success Criteria

### Security Gate S10-PERF-001

**Requirement**: p95 login latency < 500ms (including 100-300ms random delay)

**Pass Criteria**:

| Metric | Target | Critical? |
|--------|--------|-----------|
| Login p95 latency | < 500ms | **YES** (Security Gate) |
| Login p99 latency | < 750ms | Recommended |
| Timing consistency | > 95% | **YES** |
| Success vs. failure timing diff | < 50ms | **YES** |

### HIBP Functionality (S10-HIBP-003)

| Metric | Target | Critical? |
|--------|--------|-----------|
| Compromised password rejection rate | > 95% | **YES** |
| Strong password acceptance rate | > 95% | **YES** |
| Registration p95 latency | < 2s | Recommended |
| HIBP cache hit rate | > 90% | Recommended |
| Fail-open success rate | > 95% | **YES** (Security Gate) |

### Error Rate Thresholds

| Endpoint | Max Error Rate | Notes |
|----------|----------------|-------|
| `/auth/login` | < 1% | Excluding intentional failures |
| `/auth/register` (strong password) | < 5% | Network variability |
| `/auth/register` (fail-open) | < 5% | Should succeed despite HIBP down |

---

## Running Tests Locally

### Prerequisites

1. **Install k6**:
   ```bash
   # macOS
   brew install k6

   # Linux (Ubuntu/Debian)
   sudo apt-get install k6

   # Docker alternative
   docker pull grafana/k6:latest
   ```

2. **Start Infrastructure**:
   ```bash
   # Start PostgreSQL, Redis, ClamAV
   make docker-up

   # Run migrations
   make migrate-up
   ```

3. **Start API Server**:
   ```bash
   make run
   ```

4. **Verify Health**:
   ```bash
   curl http://localhost:8080/api/v1/health
   # Expected: {"status":"ok"}
   ```

### Running Individual Tests

**Login Timing Test**:
```bash
make test-load-sprint10-login
```

Expected output:
```
     ✓ successful login: status 200
     ✓ successful login: has access token
     ✓ successful login: delay in range (100-500ms)
     ✓ failed login: status 401
     ✓ failed login: delay in range (100-500ms)

     checks.........................: 98.50% ✓ 14775    ✗ 225
     http_req_duration{endpoint:login_success}: avg=250ms p95=420ms p99=680ms
     http_req_duration{endpoint:login_failure}: avg=245ms p95=415ms p99=670ms
     login_delay_consistency........: 97.2%
```

**HIBP Registration Test**:
```bash
make test-load-sprint10-hibp
```

Expected output:
```
     ✓ compromised password: status 400
     ✓ compromised password: error mentions breach
     ✓ strong password: status 201
     ✓ strong password: user created

     compromised_password_rejection_rate: 96.8%
     strong_password_acceptance_rate....: 98.1%
     hibp_check_duration................: p50=45ms p95=1200ms
```

**Fail-Open Test**:
```bash
# 1. Disable HIBP
export HIBP_ENABLED=false
make run

# 2. Run test
make test-load-sprint10-failopen
```

Expected output:
```
     ✓ failopen: registration succeeds
     ✓ failopen: user created
     ✓ failopen: response time < 1s

     failopen_success_rate: 99.2%
     failopen_registration_duration: p95=620ms
```

### Running All Sprint 10 Tests

```bash
make test-load-sprint10-all
```

This runs all three tests sequentially (total duration: ~30 minutes).

---

## Running Tests in CI

### GitHub Actions Integration

Add to `.github/workflows/load-tests.yml`:

```yaml
name: Sprint 10 Load Tests

on:
  push:
    branches: [main, develop]
  pull_request:
    branches: [main]
  schedule:
    - cron: '0 2 * * *'  # Daily at 2 AM UTC

jobs:
  load-test-sprint10:
    runs-on: ubuntu-latest
    timeout-minutes: 60

    services:
      postgres:
        image: postgres:16-alpine
        env:
          POSTGRES_PASSWORD: postgres
          POSTGRES_DB: goimg_test
        options: >-
          --health-cmd pg_isready
          --health-interval 10s
          --health-timeout 5s
          --health-retries 5
        ports:
          - 5432:5432

      redis:
        image: redis:7-alpine
        options: >-
          --health-cmd "redis-cli ping"
          --health-interval 10s
          --health-timeout 5s
          --health-retries 5
        ports:
          - 6379:6379

    steps:
      - name: Checkout code
        uses: actions/checkout@v3

      - name: Set up Go
        uses: actions/setup-go@v4
        with:
          go-version: '1.21'

      - name: Install k6
        run: |
          sudo gpg --no-default-keyring --keyring /usr/share/keyrings/k6-archive-keyring.gpg \
            --keyserver hkp://keyserver.ubuntu.com:80 \
            --recv-keys C5AD17C747E3415A3642D57D77C6C491D6AC1D69
          echo "deb [signed-by=/usr/share/keyrings/k6-archive-keyring.gpg] https://dl.k6.io/deb stable main" \
            | sudo tee /etc/apt/sources.list.d/k6.list
          sudo apt-get update
          sudo apt-get install k6

      - name: Run database migrations
        run: make migrate-up
        env:
          DATABASE_URL: postgres://postgres:postgres@localhost:5432/goimg_test?sslmode=disable

      - name: Start API server
        run: |
          make run &
          sleep 10
        env:
          DATABASE_URL: postgres://postgres:postgres@localhost:5432/goimg_test?sslmode=disable
          REDIS_ADDR: localhost:6379

      - name: Run Sprint 10 Login Timing Test
        run: k6 run tests/load/sprint10-login-timing.js --out json=login-timing-results.json
        continue-on-error: false

      - name: Run Sprint 10 HIBP Registration Test
        run: k6 run tests/load/sprint10-hibp-registration.js --out json=hibp-results.json
        continue-on-error: false

      - name: Run Sprint 10 Fail-Open Test
        run: HIBP_ENABLED=false k6 run tests/load/sprint10-hibp-failopen.js --out json=failopen-results.json
        continue-on-error: false

      - name: Upload test results
        uses: actions/upload-artifact@v3
        if: always()
        with:
          name: sprint10-load-test-results
          path: |
            login-timing-results.json
            hibp-results.json
            failopen-results.json
          retention-days: 30

      - name: Evaluate Security Gate S10-PERF-001
        run: |
          # Extract p95 from results and verify < 500ms
          LOGIN_P95=$(jq '.metrics.http_req_duration.values["p(95)"]' login-timing-results.json)
          if (( $(echo "$LOGIN_P95 < 500" | bc -l) )); then
            echo "✅ Security Gate S10-PERF-001 PASSED: p95=$LOGIN_P95ms"
          else
            echo "❌ Security Gate S10-PERF-001 FAILED: p95=$LOGIN_P95ms (requirement: <500ms)"
            exit 1
          fi
```

### Scheduled Load Tests

Run nightly to catch performance regressions:

```yaml
on:
  schedule:
    - cron: '0 2 * * *'  # 2 AM UTC daily
```

### Performance Budgets

Fail CI if performance degrades:

```bash
# In CI script
k6 run --quiet tests/load/sprint10-login-timing.js || exit 1
```

k6 will exit with code 1 if any threshold fails.

---

## Interpreting Results

### Understanding k6 Output

**Metrics Explained**:

```
✓ successful login: status 200        14750/15000 (98.3%)
✗ successful login: delay in range     14000/15000 (93.3%)  ← PROBLEM

http_req_duration{endpoint:login_success}:
  avg=250ms min=120ms med=240ms max=680ms p(95)=420ms p(99)=650ms
                                                  ↑
                                        This must be < 500ms
```

**Status**:
- ✓ = Check passed
- ✗ = Check failed

**Percentiles**:
- `p(95)=420ms`: 95% of requests completed in ≤ 420ms
- `p(99)=650ms`: 99% of requests completed in ≤ 650ms

**Custom Metrics**:
```
login_success_duration.....: avg=245ms p(50)=235ms p(95)=415ms
login_failure_duration.....: avg=242ms p(50)=232ms p(95)=410ms
login_delay_consistency....: 97.2%  ← 97.2% of requests had delay in 100-500ms range
```

### Analyzing Timing Consistency

**Expected Behavior**:
- Success and failure latencies should be nearly identical
- Both should have p50 around 200-250ms (100-300ms delay + processing)
- Consistency rate > 95%

**Problem Indicators**:
```
login_success_duration: p(95)=320ms
login_failure_duration: p(95)=180ms  ← TIMING LEAK!
```

If failure is consistently faster, timing attack mitigation is not working.

**Investigation Steps**:
1. Check if delay is applied to ALL code paths (including errors)
2. Review `internal/application/identity/commands/login.go` for defer placement
3. Check logs for exceptions or early returns bypassing delay

### Analyzing HIBP Performance

**Cache Hit Rate Calculation**:
```
hibp_check_duration: p(50)=45ms p(95)=1200ms

p(50) < 100ms → Good cache hit rate (most requests cached)
p(95) > 1s    → Some API calls happening (expected on cache miss)
```

**Cache warming**: First few requests will be slow (cache misses). After warm-up, p50 should drop below 100ms.

**Problem Indicators**:
```
hibp_check_duration: p(50)=800ms  ← BAD: Most requests hitting API
```

Possible causes:
- Redis not running
- Cache TTL too short
- Cache keys not matching

### Fail-Open Verification

**Expected Behavior**:
```
failopen_success_rate: 99.2%  ← Registrations succeed despite HIBP down
failopen_registration_duration: p(95)=620ms  ← Faster (no HIBP API call)
```

**Check Server Logs**:
```
WARN HIBP API check failed fail_open=true
INFO user registered successfully (HIBP check skipped)
```

Logs should show warnings (not errors) and confirm fail-open behavior.

---

## Troubleshooting

### Problem: p95 Login Latency > 500ms

**Symptoms**:
```
http_req_duration{endpoint:login_success}: p(95)=680ms  ← FAIL
```

**Causes**:
1. **Database slow**: Check PostgreSQL query performance
   ```bash
   # Check slow queries
   docker exec goimg-postgres psql -U postgres -d goimg \
     -c "SELECT query, mean_exec_time FROM pg_stat_statements ORDER BY mean_exec_time DESC LIMIT 10;"
   ```

2. **Redis slow**: Check Redis latency
   ```bash
   docker exec goimg-redis redis-cli --latency
   ```

3. **High CPU/memory**: Check resource usage
   ```bash
   docker stats goimg-api
   ```

**Solutions**:
- Add database indexes for frequently queried columns
- Increase database connection pool size
- Scale API server horizontally
- Use Redis for session caching

### Problem: Timing Leak Detected

**Symptoms**:
```
login_success_duration: p(95)=320ms
login_failure_duration: p(95)=180ms  ← Consistent difference
```

**Causes**:
1. Delay not applied to all code paths
2. Early return before defer executes
3. Error handling bypasses delay

**Investigation**:
```bash
# Review login handler
cat internal/application/identity/commands/login.go

# Check timing.go implementation
cat internal/application/identity/timing.go
```

**Solutions**:
- Ensure defer is used for delay (executes on all returns)
- Apply delay even on errors
- Verify crypto/rand is used (not math/rand)

### Problem: HIBP Rejections < 95%

**Symptoms**:
```
compromised_password_rejection_rate: 78.2%  ← FAIL
```

**Causes**:
1. HIBP API returning unexpected responses
2. Cache serving stale negative results for compromised passwords
3. Network issues to HIBP API

**Investigation**:
```bash
# Check HIBP API manually
curl -A "goimg-test" https://api.pwnedpasswords.com/range/5BAA6  # Hash prefix for "password123"

# Check server logs for HIBP errors
docker logs goimg-api | grep -i hibp
```

**Solutions**:
- Verify HIBP API is reachable
- Check HIBP client timeout settings
- Clear Redis cache for testing: `docker exec goimg-redis redis-cli FLUSHDB`

### Problem: Fail-Open Not Working

**Symptoms**:
```
failopen_success_rate: 12.3%  ← FAIL
http_req_failed: 87.7%
```

**Causes**:
1. HIBP_ENABLED still set to `true`
2. HIBP fail-open config set to `false`
3. API not handling HIBP errors gracefully

**Investigation**:
```bash
# Check environment variables
docker exec goimg-api env | grep HIBP

# Check server logs
docker logs goimg-api | tail -100
```

**Solutions**:
- Set `HIBP_ENABLED=false` in environment
- Set `HIBP_FAIL_OPEN=true` in config
- Review error handling in `hibp_client.go`

### Problem: k6 Crashes or High Memory

**Symptoms**:
```
ERRO[0245] thresholds on metrics 'http_req_duration' were crossed
panic: runtime error: out of memory
```

**Causes**:
- Too many VUs for available memory
- Test duration too long
- Large response bodies

**Solutions**:
```bash
# Reduce VUs
k6 run --vus 20 tests/load/sprint10-login-timing.js

# Reduce duration
k6 run --duration 2m tests/load/sprint10-login-timing.js

# Use lower memory mode
k6 run --compatibility-mode=base tests/load/sprint10-login-timing.js
```

### Problem: Rate Limiting Triggered

**Symptoms**:
```
http_req_failed: 45.2%
Status 429: Too Many Requests
```

**Causes**:
- Load test exceeds rate limits
- Rate limiting configuration too strict

**Solutions**:
```bash
# Reduce arrival rate
# Edit test script, lower rate value:
rate: 5,  // Reduced from 20

# Adjust rate limits for testing
export RATE_LIMIT_LOGIN=100  # Increase limit
make run
```

---

## Performance Baselines

### Login Endpoint (With Random Delay)

| Load Level | Target p50 | Target p95 | Target p99 |
|------------|------------|------------|------------|
| 10 VUs     | 200ms      | 350ms      | 500ms      |
| 50 VUs     | 220ms      | 400ms      | 550ms      |
| 100 VUs    | 250ms      | **480ms**  | 650ms      |

**Critical**: p95 must remain below 500ms at all load levels.

### Registration Endpoint (With HIBP Check)

| Load Level | Cache Hit p95 | Cache Miss p95 | Fail-Open p95 |
|------------|---------------|----------------|---------------|
| 5 req/s    | 150ms         | 1500ms         | 500ms         |
| 10 req/s   | 180ms         | 1800ms         | 600ms         |
| 20 req/s   | 220ms         | **2000ms**     | 700ms         |

**Notes**:
- Cache hits should be < 200ms at all load levels
- Cache misses can be up to 2s (includes HIBP API call)
- Fail-open should be < 1s at all load levels

### HIBP Cache Performance

| Metric | Target | Notes |
|--------|--------|-------|
| Cache hit rate | > 90% | After warm-up period |
| Cache hit latency (p95) | < 100ms | Redis lookup |
| Cache miss latency (p95) | < 2s | Includes API call |
| API timeout | 5s | Configurable |

---

## Next Steps

After running tests and verifying all pass:

1. **Document Results**:
   - Update `/docs/sprints/sprint-10.md` with test results
   - Include screenshots of k6 output
   - Note any performance insights

2. **Update Security Gate**:
   ```markdown
   | S10-PERF-001 | <500ms p95 login latency | ✅ PASS | Load tests |
   ```

3. **Create Performance Dashboard**:
   - Set up Grafana dashboard for ongoing monitoring
   - Track login latency, HIBP cache hit rate, fail-open incidents
   - Configure alerts for threshold violations

4. **Integrate into CI/CD**:
   - Add Sprint 10 load tests to nightly CI runs
   - Fail builds if performance degrades
   - Track performance trends over time

5. **Production Monitoring**:
   - Enable Prometheus metrics for `goimg_auth_login_delay_seconds`
   - Enable Prometheus metrics for `goimg_security_hibp_*`
   - Set up alerts for p95 > 500ms or cache hit rate < 90%

---

## References

### Internal Documentation
- `/home/user/goimg-datalayer/claude/sprint_10_plan.md` - Sprint 10 implementation plan
- `/home/user/goimg-datalayer/docs/performance/load-testing.md` - General load testing guide
- `/home/user/goimg-datalayer/claude/security_gates.md` - Security gate definitions

### Sprint 10 Implementation Files
- `/home/user/goimg-datalayer/internal/application/identity/timing.go` - Random delay implementation
- `/home/user/goimg-datalayer/internal/infrastructure/security/hibp_client.go` - HIBP integration
- `/home/user/goimg-datalayer/internal/infrastructure/security/password_cache.go` - Redis caching

### External Resources
- [k6 Documentation](https://k6.io/docs/)
- [k6 Thresholds](https://k6.io/docs/using-k6/thresholds/)
- [k6 Metrics](https://k6.io/docs/using-k6/metrics/)
- [HIBP API Documentation](https://haveibeenpwned.com/API/v3)

---

## Support

For questions or issues:
- Check this guide first
- Review k6 output and server logs
- Consult `/docs/performance/load-testing.md` for general k6 usage
- Open an issue with test results attached

**Security Gate Approval**: All Sprint 10 load tests must pass before Security Gate S10 can be marked complete.
