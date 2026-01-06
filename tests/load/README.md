# k6 Load Tests

Comprehensive load testing suite for the goimg-datalayer API.

## Quick Start

### Prerequisites

1. Install k6: https://k6.io/docs/getting-started/installation/
2. Start the API server: `make run`

### Run Tests

```bash
# Quick smoke test (1 minute)
make load-test-quick

# Run all tests (~50 minutes)
make load-test

# Run individual test scenarios
make load-test-auth      # Authentication flow (5 min)
make load-test-browse    # Image browsing (10 min)
make load-test-upload    # Image uploads (10 min)
make load-test-social    # Social interactions (10 min)
```

## Test Scenarios

| Test | VUs | Duration | Purpose |
|------|-----|----------|---------|
| `auth-flow.js` | 50 | 5 min | Register → Login → Profile → Logout |
| `browse-flow.js` | 100 | 10 min | List images → View details → Load variants → Comments |
| `upload-flow.js` | 20 | 10 min | Upload → Verify processing → Check variants |
| `social-flow.js` | 75 | 10 min | Like → Comment → View activity → Unlike |
| `mixed-traffic.js` | 150 | 15 min | Realistic mix (60% browse, 25% social, 10% upload, 5% auth) |

## Directory Structure

```
tests/load/
├── README.md              # This file
├── auth-flow.js           # Authentication flow test
├── browse-flow.js         # Browsing flow test
├── upload-flow.js         # Upload flow test
├── social-flow.js         # Social interaction test
├── mixed-traffic.js       # Mixed traffic simulation
└── helpers/
    ├── config.js          # Configuration and environment settings
    ├── data.js            # Test data generation
    └── auth.js            # Authentication helpers
```

## Performance Thresholds

### Standard Endpoints (Read/Write)
- **p(95)**: < 200ms
- **Error rate**: < 1%

### Upload Endpoints
- **p(95)**: < 5000ms (5 seconds)
- **Error rate**: < 2%

## Configuration

### Environment Variables

Override defaults by setting environment variables:

```bash
# Change API URL
export API_BASE_URL=http://staging.example.com:8080/api/v1

# Adjust user pool size
export USER_POOL_SIZE=100

# Run test
k6 run browse-flow.js
```

### Available Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `API_BASE_URL` | `http://localhost:8080/api/v1` | API base URL |
| `USER_POOL_SIZE` | `50` | Number of users to pre-create |

## Understanding Results

### Key Metrics

```
http_req_duration..............: avg=85ms  med=75ms max=450ms p(95)=180ms p(99)=350ms
http_req_failed................: 0.50%  ✓ 75         ✗ 14925
http_reqs......................: 15000  2500/s
```

- **http_req_duration**: Response time distribution
  - `p(95)`: 95% of requests completed within this time
  - `avg`: Average response time
  - `med`: Median response time

- **http_req_failed**: Percentage of failed requests
  - Target: < 1%

- **http_reqs**: Total requests and throughput (requests/second)

### Status Indicators

- ✓ **Green check**: Threshold passed
- ✗ **Red X**: Threshold failed

## Common Issues

### API Not Running

**Error**: `API health check failed`

**Fix**:
```bash
make run
curl http://localhost:8080/api/v1/health
```

### High Error Rate

**Error**: `http_req_failed: rate>10%`

**Causes**:
- Rate limiting
- Database connection pool exhausted
- Server resource limits

**Fix**:
```bash
# Check API logs
docker logs goimg-api

# Reduce VUs or duration
k6 run --vus 25 --duration 5m browse-flow.js
```

### k6 Not Installed

**Fix using Docker**:
```bash
docker run --rm -i \
  -v $(pwd):/scripts \
  --network host \
  grafana/k6 run /scripts/browse-flow.js
```

## Advanced Usage

### Custom VUs and Duration

```bash
# Override test configuration
k6 run --vus 200 --duration 30m mixed-traffic.js
```

### Output to InfluxDB

```bash
k6 run --out influxdb=http://localhost:8086/k6 browse-flow.js
```

### JSON Summary Export

```bash
k6 run --summary-export=results.json browse-flow.js
```

### Debug Mode

```bash
k6 run --http-debug browse-flow.js
```

## Documentation

For detailed documentation, see:
- [Load Testing Guide](/home/user/goimg-datalayer/docs/performance/load-testing.md)
- [k6 Documentation](https://k6.io/docs/)

## Test Helpers

The `helpers/` directory contains reusable modules:

### config.js

Centralizes configuration:
```javascript
import { config, endpoint } from './helpers/config.js';

// Get full URL
const url = endpoint('/images');
```

### data.js

Generates test data:
```javascript
import { generateUsername, generateImageTitle } from './helpers/data.js';

const username = generateUsername();
const title = generateImageTitle();
```

### auth.js

Authentication utilities:
```javascript
import { createAuthenticatedUser, login } from './helpers/auth.js';

// Create and login user
const user = createAuthenticatedUser();

// Or login existing
const tokens = login(email, password);
```

## Performance Baselines

| Metric | Target | Current |
|--------|--------|---------|
| Browse p(95) | < 200ms | TBD |
| Upload p(95) | < 5s | TBD |
| Social p(95) | < 200ms | TBD |
| Error Rate | < 1% | TBD |
| Throughput | 500 RPS | TBD |

Run tests and update baselines as system evolves.

## Contributing

When adding new test scenarios:

1. Follow existing test structure
2. Use helper modules for common operations
3. Set realistic think times
4. Define appropriate thresholds
5. Document the test flow
6. Update this README

---

## Sprint 10 Security Feature Tests

Sprint 10 introduced security-focused load tests for timing attack mitigation and HIBP password checking.

### Quick Start (Sprint 10)

```bash
# Run all Sprint 10 tests (~27 minutes)
make test-load-sprint10-all

# Or run individually
make test-load-sprint10-login     # Login timing (13 min)
make test-load-sprint10-hibp      # HIBP registration (9 min)
make test-load-sprint10-failopen  # Fail-open test (5 min)
```

### Sprint 10 Test Scenarios

| Test | Duration | Security Gate | Purpose |
|------|----------|---------------|---------|
| `sprint10-login-timing.js` | 13 min | S10-PERF-001 | Verify p95 login latency < 500ms with random delay |
| `sprint10-hibp-registration.js` | 9 min | S10-HIBP-003 | Verify compromised password rejection |
| `sprint10-hibp-failopen.js` | 5 min | S10-HIBP-003 | Verify fail-open behavior when HIBP unavailable |

### Login Timing Test (`sprint10-login-timing.js`)

**Purpose**: Verify random login delay (100-300ms) for timing attack mitigation

**Security Gate**: S10-PERF-001 - p95 login latency < 500ms

**Success Criteria**:
- ✅ p95 login latency < 500ms (CRITICAL)
- ✅ Timing consistency > 95%
- ✅ Success vs. failure timing diff < 50ms

**Run**:
```bash
make test-load-sprint10-login
```

**Expected Output**:
```
✓ successful login: delay in range    97.2%
✓ failed login: delay in range        97.1%

http_req_duration{endpoint:login_success}: p(95)=420ms  ← PASS
http_req_duration{endpoint:login_failure}: p(95)=415ms  ← PASS

Security Gate S10-PERF-001: PASS ✓
```

### HIBP Registration Test (`sprint10-hibp-registration.js`)

**Purpose**: Verify HIBP password checking rejects compromised passwords

**Security Gate**: S10-HIBP-003 - Compromised password rejection

**Success Criteria**:
- ✅ Compromised password rejection rate > 95% (CRITICAL)
- ✅ Strong password acceptance rate > 95% (CRITICAL)
- ✅ p95 registration latency < 2s

**Run**:
```bash
make test-load-sprint10-hibp
```

**Expected Output**:
```
✓ compromised password: status 400    96.8%
✓ strong password: status 201         98.1%

compromised_password_rejection_rate: 96.8%  ← PASS
strong_password_acceptance_rate: 98.1%      ← PASS

Security Gate S10-HIBP-003: PASS ✓
```

### HIBP Fail-Open Test (`sprint10-hibp-failopen.js`)

**Purpose**: Verify system continues working when HIBP API is unavailable

**Security Gate**: S10-HIBP-003 - Fail-open behavior

**Success Criteria**:
- ✅ Registration success rate > 95% (CRITICAL)
- ✅ p95 latency < 1s

**Prerequisites**:
```bash
export HIBP_ENABLED=false
make run
make test-load-sprint10-failopen
```

**Expected Output**:
```
✓ failopen: registration succeeds     99.2%

failopen_success_rate: 99.2%              ← PASS
failopen_registration_duration: p(95)=620ms ← PASS

Security Gate S10-HIBP-003 (Fail-Open): PASS ✓
```

**Server Logs Should Show**:
```
WARN HIBP API check failed fail_open=true
INFO user registered successfully (HIBP check skipped)
```

### Sprint 10 Documentation

For comprehensive Sprint 10 load testing documentation:
- **Summary**: `/home/user/goimg-datalayer/docs/testing/SPRINT10_LOAD_TESTING_SUMMARY.md`
- **Detailed Guide**: `/home/user/goimg-datalayer/docs/testing/SPRINT10_LOAD_TESTING_GUIDE.md`
- **Implementation Plan**: `/home/user/goimg-datalayer/claude/sprint_10_plan.md`

### Sprint 10 Performance Baselines

| Metric | Target | Notes |
|--------|--------|-------|
| Login p95 (100 VUs) | < 500ms | Includes 100-300ms random delay |
| Registration p95 (cache hit) | < 200ms | Redis cache lookup |
| Registration p95 (cache miss) | < 2000ms | Includes HIBP API call |
| Fail-open p95 | < 1000ms | No external API call |
| HIBP cache hit rate | > 90% | After warm-up period |

---

## Support

- API Issues: Check `docker logs goimg-api`
- Database Issues: Check `docker logs goimg-postgres`
- k6 Issues: https://k6.io/docs/
- Sprint 10 Tests: See `/home/user/goimg-datalayer/docs/testing/SPRINT10_LOAD_TESTING_GUIDE.md`
- Project Issues: Open GitHub issue
