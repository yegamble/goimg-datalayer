# k6 Load Testing Implementation Summary

## Overview

A comprehensive k6 load testing suite has been implemented for the goimg-datalayer API. This suite includes realistic user flow simulations, performance thresholds, and detailed documentation.

## What Was Created

### Test Scenarios (5 files)

1. **auth-flow.js** - User Authentication Flow
   - Tests: Register → Login → Fetch profile → Logout
   - VUs: 50, Duration: 5 minutes
   - Think time: 1-3 seconds
   - Thresholds: p(95) < 200ms, errors < 1%

2. **browse-flow.js** - Image Browsing Flow
   - Tests: List images → View details → Load variants → Fetch comments
   - VUs: 100, Duration: 10 minutes
   - Think time: 2-5 seconds
   - Mix: 70% authenticated, 30% anonymous users

3. **upload-flow.js** - Image Upload Flow
   - Tests: Login → Upload → Poll status → Verify variants
   - VUs: 20, Duration: 10 minutes
   - Think time: 5-10 seconds (respects rate limits)
   - Thresholds: p(95) < 5000ms, errors < 2%

4. **social-flow.js** - Social Interactions Flow
   - Tests: Browse → Like → Comment → View activity → Unlike
   - VUs: 75, Duration: 10 minutes
   - Think time: 1-4 seconds

5. **mixed-traffic.js** - Realistic Mixed Traffic
   - Traffic mix: 60% browse, 25% social, 10% upload, 5% auth
   - VUs: 150, Duration: 15 minutes
   - Simulates production-like load

### Helper Modules (3 files)

1. **helpers/config.js**
   - Centralized configuration
   - Environment variable support
   - Base URL management
   - Think time utilities
   - Query parameter builder

2. **helpers/data.js**
   - Test data generation (usernames, emails, passwords)
   - Realistic image metadata generation
   - Tag generation
   - Comment generation
   - Multipart form data creation for uploads

3. **helpers/auth.js**
   - User registration
   - Login/logout
   - Token management
   - Batch user creation for setup
   - Authorization header helpers

### Documentation (3 files)

1. **docs/performance/load-testing.md** (15KB)
   - Comprehensive guide covering:
     - Prerequisites and installation
     - All test scenarios in detail
     - Running tests
     - Interpreting results
     - Performance baselines
     - Optimization recommendations
     - Troubleshooting
     - Advanced usage

2. **tests/load/README.md**
   - Quick reference guide
   - Test scenario summary table
   - Common commands
   - Performance thresholds
   - Troubleshooting quick tips

3. **tests/load/IMPLEMENTATION_SUMMARY.md** (this file)
   - Implementation overview
   - File listing and descriptions
   - Quick start guide

### Supporting Files (2 files)

1. **tests/load/.gitignore**
   - Ignores test output files (JSON, HTML, logs)
   - Prevents committing temporary artifacts

2. **tests/load/example-custom-test.js.template**
   - Template for creating custom tests
   - Documented examples
   - Best practices

### Makefile Targets (6 new targets)

Added to `/home/user/goimg-datalayer/Makefile`:

- `make load-test` - Run all k6 load tests sequentially
- `make load-test-quick` - Quick 1-minute smoke test
- `make load-test-auth` - Run authentication flow test
- `make load-test-browse` - Run browsing flow test
- `make load-test-upload` - Run upload flow test
- `make load-test-social` - Run social interactions test

## Directory Structure

```
/home/user/goimg-datalayer/
├── tests/load/
│   ├── README.md                          # Quick reference
│   ├── IMPLEMENTATION_SUMMARY.md          # This file
│   ├── .gitignore                         # Ignore test outputs
│   ├── auth-flow.js                       # Auth flow test
│   ├── browse-flow.js                     # Browse flow test
│   ├── upload-flow.js                     # Upload flow test
│   ├── social-flow.js                     # Social flow test
│   ├── mixed-traffic.js                   # Mixed traffic test
│   ├── example-custom-test.js.template    # Template for custom tests
│   └── helpers/
│       ├── config.js                      # Configuration
│       ├── data.js                        # Test data generation
│       └── auth.js                        # Authentication helpers
├── docs/performance/
│   └── load-testing.md                    # Comprehensive guide
└── Makefile                               # Updated with load test targets
```

## Quick Start

### 1. Install k6

**macOS**:
```bash
brew install k6
```

**Linux (Ubuntu/Debian)**:
```bash
sudo gpg --no-default-keyring --keyring /usr/share/keyrings/k6-archive-keyring.gpg --keyserver hkp://keyserver.ubuntu.com:80 --recv-keys C5AD17C747E3415A3642D57D77C6C491D6AC1D69
echo "deb [signed-by=/usr/share/keyrings/k6-archive-keyring.gpg] https://dl.k6.io/deb stable main" | sudo tee /etc/apt/sources.list.d/k6.list
sudo apt-get update
sudo apt-get install k6
```

**Using Docker** (no installation needed):
```bash
alias k6='docker run --rm -i --network host -v $(pwd):/workspace grafana/k6'
```

### 2. Start API Server

```bash
# Start infrastructure
make docker-up

# Run migrations
make migrate-up

# Start API
make run
```

### 3. Run Tests

```bash
# Quick smoke test (1 minute)
make load-test-quick

# Run specific scenario
make load-test-browse

# Run all tests (~50 minutes)
make load-test
```

## Features

### Realistic Load Patterns

- **Staged ramp-up**: Gradual increase from 0 → 25% → 100% VUs
- **Think time**: Random delays simulating real user behavior
- **Mixed user types**: Authenticated and anonymous users
- **Realistic data**: Generated usernames, emails, images, comments

### Comprehensive Coverage

- **Authentication**: Registration, login, logout, token refresh
- **Browsing**: Image listing, details, variants, comments
- **Uploads**: Multipart form data, metadata, variant verification
- **Social**: Likes, comments, user activity
- **Error handling**: Invalid tokens, rate limiting, edge cases

### Performance Monitoring

- **HTTP metrics**: Request duration, error rates, throughput
- **Custom metrics**: Flow durations, success counts, user actions
- **Thresholds**: Automatic pass/fail based on performance targets
- **Tags**: Endpoint-level metric breakdowns

### Developer-Friendly

- **Modular helpers**: Reusable authentication, data generation
- **Environment configuration**: Override base URL and settings
- **Clear documentation**: Quick references and comprehensive guides
- **Template**: Example for creating custom tests

## Performance Targets

### Standard Endpoints (Read/Write)

| Metric | Target | Excellent | Acceptable | Poor |
|--------|--------|-----------|------------|------|
| p(95) | < 200ms | < 100ms | 200-500ms | > 500ms |
| p(99) | < 500ms | < 250ms | 500-1000ms | > 1000ms |
| Error Rate | < 1% | < 0.1% | 1-2% | > 2% |

### Upload Endpoints

| Metric | Target | Excellent | Acceptable | Poor |
|--------|--------|-----------|------------|------|
| p(95) | < 5s | < 3s | 5-10s | > 10s |
| p(99) | < 10s | < 7s | 10-20s | > 20s |
| Error Rate | < 2% | < 0.5% | 2-5% | > 5% |

## Example Output

```
scenarios: (100.00%) 1 scenario, 100 max VUs, 13m30s max duration
default: 100 VUs in stages (10m30s)

✓ explore recent successful
✓ images returned
✓ image details loaded
✓ variant loaded
✓ comments loaded

checks.........................: 98.50% ✓ 14775      ✗ 225
data_received..................: 145 MB 24 MB/s
data_sent......................: 12 MB  2.0 MB/s
http_req_duration..............: avg=85ms  min=10ms med=75ms max=450ms p(95)=180ms p(99)=350ms
  { endpoint:explore_recent }...: avg=72ms  min=15ms med=65ms max=200ms p(95)=150ms p(99)=180ms
  { endpoint:get_image }........: avg=95ms  min=20ms med=85ms max=350ms p(95)=200ms p(99)=280ms
  { endpoint:get_variant }......: avg=45ms  min=10ms med=40ms max=150ms p(95)=90ms  p(99)=120ms
http_req_failed................: 0.50%  ✓ 75         ✗ 14925
http_reqs......................: 15000  2500/s
iteration_duration.............: avg=3.2s  min=1.5s med=3.0s max=8.5s  p(95)=5.2s  p(99)=7.1s
iterations.....................: 5000   833.33/s
vus............................: 100    min=0        max=100
vus_max........................: 100    min=100      max=100

✓ http_req_duration{endpoint:explore_recent}: p(95)<200
✓ http_req_duration{endpoint:get_image}: p(95)<200
✓ http_req_failed: rate<0.01
```

## Common Use Cases

### Development Testing

Quick smoke test during development:
```bash
make load-test-quick
```

### CI/CD Integration

Add to GitHub Actions:
```yaml
- name: Load Test
  run: make load-test-quick
```

### Performance Regression Testing

Run specific scenario after changes:
```bash
make load-test-browse
```

### Capacity Planning

Run mixed traffic to understand limits:
```bash
k6 run tests/load/mixed-traffic.js
```

### Stress Testing

Find breaking point:
```bash
k6 run --vus 500 --duration 30m tests/load/mixed-traffic.js
```

## Next Steps

1. **Establish Baselines**
   - Run tests against current system
   - Document actual performance metrics
   - Set realistic targets based on results

2. **Integrate with CI/CD**
   - Add quick smoke tests to PR checks
   - Run full suite nightly
   - Alert on performance regressions

3. **Monitor in Production**
   - Use Grafana + InfluxDB for visualization
   - Export k6 metrics: `k6 run --out influxdb=http://localhost:8086/k6`
   - Set up performance dashboards

4. **Customize for Your Needs**
   - Create scenario-specific tests
   - Adjust VUs and duration based on expected traffic
   - Add custom metrics for business-critical flows

## Support and Documentation

### Quick References
- **Quick Start**: `/tests/load/README.md`
- **Full Guide**: `/docs/performance/load-testing.md`
- **Template**: `/tests/load/example-custom-test.js.template`

### External Resources
- k6 Documentation: https://k6.io/docs/
- k6 Examples: https://k6.io/docs/examples/
- Performance Testing Guide: https://k6.io/docs/testing-guides/

### Troubleshooting

**API not running**:
```bash
make run
curl http://localhost:8080/api/v1/health
```

**k6 not installed**:
```bash
# Use Docker alternative
docker run --rm -i --network host -v $(pwd):/workspace grafana/k6 run /workspace/tests/load/browse-flow.js
```

**High error rates**:
```bash
# Check API logs
docker logs goimg-api

# Reduce load
k6 run --vus 25 --duration 5m tests/load/browse-flow.js
```

## Summary

The k6 load testing suite is now fully implemented and ready to use. It provides:

- 5 comprehensive test scenarios covering all major API flows
- Realistic user behavior simulation with think times
- Modular helper functions for reusability
- Performance thresholds aligned with API requirements
- Extensive documentation for all skill levels
- Easy integration with existing build pipeline

Run `make load-test-quick` to get started!
