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

## Support

- API Issues: Check `docker logs goimg-api`
- Database Issues: Check `docker logs goimg-postgres`
- k6 Issues: https://k6.io/docs/
- Project Issues: Open GitHub issue
