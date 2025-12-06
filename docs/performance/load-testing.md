# Load Testing Guide

This document provides comprehensive information about load testing the goimg-datalayer API using k6.

## Table of Contents

- [Overview](#overview)
- [Prerequisites](#prerequisites)
- [Test Scenarios](#test-scenarios)
- [Running Tests](#running-tests)
- [Interpreting Results](#interpreting-results)
- [Performance Baselines](#performance-baselines)
- [Optimization Recommendations](#optimization-recommendations)
- [Troubleshooting](#troubleshooting)

## Overview

The goimg-datalayer project includes comprehensive k6 load tests that simulate realistic user behavior patterns. These tests help identify performance bottlenecks, validate system capacity, and ensure the API can handle expected production traffic.

### Test Coverage

- **Authentication Flow**: User registration, login, profile access, logout
- **Browsing Flow**: Image discovery, viewing, variant loading, comment reading
- **Upload Flow**: Image upload with metadata, processing verification, variant generation
- **Social Flow**: Liking images, commenting, viewing user activity
- **Mixed Traffic**: Realistic combination of all flows with weighted distribution

## Prerequisites

### 1. Install k6

**macOS (Homebrew)**:
```bash
brew install k6
```

**Linux (Debian/Ubuntu)**:
```bash
sudo gpg -k
sudo gpg --no-default-keyring --keyring /usr/share/keyrings/k6-archive-keyring.gpg --keyserver hkp://keyserver.ubuntu.com:80 --recv-keys C5AD17C747E3415A3642D57D77C6C491D6AC1D69
echo "deb [signed-by=/usr/share/keyrings/k6-archive-keyring.gpg] https://dl.k6.io/deb stable main" | sudo tee /etc/apt/sources.list.d/k6.list
sudo apt-get update
sudo apt-get install k6
```

**Windows (Chocolatey)**:
```powershell
choco install k6
```

**Docker** (alternative):
```bash
docker pull grafana/k6:latest
```

For more installation options, see: https://k6.io/docs/getting-started/installation/

### 2. Start API Server

Before running load tests, ensure the API server is running:

```bash
# Start infrastructure
make docker-up

# Run migrations
make migrate-up

# Start API server
make run
```

The API should be accessible at `http://localhost:8080/api/v1`.

### 3. Verify API Health

```bash
curl http://localhost:8080/api/v1/health
```

Expected response:
```json
{
  "status": "ok",
  "timestamp": "2024-12-05T10:30:00Z"
}
```

## Test Scenarios

### 1. Authentication Flow (`auth-flow.js`)

**Purpose**: Tests user authentication lifecycle

**Flow**:
1. Register new user
2. Login with credentials
3. Fetch user profile
4. Logout

**Configuration**:
- Virtual Users (VUs): 50
- Duration: 5 minutes
- Think Time: 1-3 seconds between requests

**Run**:
```bash
make load-test-auth
# or
k6 run tests/load/auth-flow.js
```

### 2. Browsing Flow (`browse-flow.js`)

**Purpose**: Tests read-heavy image browsing patterns

**Flow**:
1. List recent public images
2. View image details (3-5 images)
3. Load image variants (thumbnail, medium)
4. Fetch comments

**Configuration**:
- Virtual Users (VUs): 100
- Duration: 10 minutes
- Think Time: 2-5 seconds between requests
- Mix: 70% authenticated, 30% anonymous users

**Run**:
```bash
make load-test-browse
# or
k6 run tests/load/browse-flow.js
```

### 3. Upload Flow (`upload-flow.js`)

**Purpose**: Tests write-heavy image upload operations

**Flow**:
1. Login
2. Upload image (JPEG with metadata)
3. Poll processing status
4. Verify variants generated (thumbnail, small, medium, large, original)

**Configuration**:
- Virtual Users (VUs): 20
- Duration: 10 minutes
- Think Time: 5-10 seconds (respects rate limits)

**Run**:
```bash
make load-test-upload
# or
k6 run tests/load/upload-flow.js
```

### 4. Social Flow (`social-flow.js`)

**Purpose**: Tests social interaction features

**Flow**:
1. Browse images
2. Like 2-4 images
3. Comment on 1-2 images
4. View own liked images
5. Unlike one image

**Configuration**:
- Virtual Users (VUs): 75
- Duration: 10 minutes
- Think Time: 1-4 seconds

**Run**:
```bash
make load-test-social
# or
k6 run tests/load/social-flow.js
```

### 5. Mixed Traffic (`mixed-traffic.js`)

**Purpose**: Simulates realistic production traffic with multiple user behaviors

**Traffic Distribution**:
- 60% Browsing
- 25% Social interactions
- 10% Uploading content
- 5% Authentication

**Configuration**:
- Virtual Users (VUs): 150
- Duration: 15 minutes

**Run**:
```bash
k6 run tests/load/mixed-traffic.js
```

### Quick Smoke Test

For rapid testing during development:

```bash
make load-test-quick
```

This runs a 1-minute browsing test with 10 VUs.

## Running Tests

### Run All Tests

```bash
make load-test
```

This sequentially runs all test scenarios. Total duration: ~50 minutes.

### Run Individual Tests

```bash
make load-test-auth      # Authentication flow
make load-test-browse    # Browsing flow
make load-test-upload    # Upload flow
make load-test-social    # Social flow
```

### Custom Configuration

Override default settings using environment variables:

```bash
# Change base URL
API_BASE_URL=http://staging.example.com:8080/api/v1 k6 run tests/load/browse-flow.js

# Adjust user pool size
USER_POOL_SIZE=100 k6 run tests/load/social-flow.js
```

### Docker-based Execution

```bash
docker run --rm -i \
  -v $(pwd)/tests/load:/scripts \
  --network host \
  grafana/k6 run /scripts/browse-flow.js
```

## Interpreting Results

### Key Metrics

k6 outputs several important metrics:

#### HTTP Metrics

- **http_req_duration**: Total request duration (sending + waiting + receiving)
  - **p(95)**: 95th percentile - 95% of requests complete within this time
  - **p(99)**: 99th percentile - 99% of requests complete within this time
  - **med**: Median response time
  - **avg**: Average response time

- **http_req_failed**: Percentage of failed HTTP requests
  - Target: < 1%

- **http_reqs**: Total number of HTTP requests made
  - **rate**: Requests per second (throughput)

#### Custom Metrics

Each test scenario includes custom metrics:

- **auth_flow_duration**: Total time for complete auth flow
- **browse_flow_duration**: Total time for browsing session
- **upload_flow_duration**: Total time from upload to variant verification
- **images_viewed_count**: Number of images viewed
- **uploads_success_count**: Number of successful uploads
- **likes_count**: Number of likes performed

### Example Output

```
     ✓ explore recent successful
     ✓ images returned
     ✓ image details loaded
     ✓ variant loaded

     checks.........................: 98.50% ✓ 14775      ✗ 225
     data_received..................: 145 MB 24 MB/s
     data_sent......................: 12 MB  2.0 MB/s
     http_req_duration..............: avg=85ms  min=10ms med=75ms max=450ms p(95)=180ms p(99)=350ms
     http_req_failed................: 0.50%  ✓ 75         ✗ 14925
     http_reqs......................: 15000  2500/s
     iteration_duration.............: avg=3.2s  min=1.5s med=3.0s max=8.5s  p(95)=5.2s  p(99)=7.1s
     iterations.....................: 5000   833.33/s
     vus............................: 100    min=0        max=100
     vus_max........................: 100    min=100      max=100
```

### Threshold Evaluation

At the end of each test, k6 evaluates thresholds:

```
✓ http_req_duration{endpoint:explore_recent}: p(95)<200    ← PASS
✓ http_req_failed: rate<0.01                               ← PASS
✗ http_req_duration{endpoint:upload}: p(95)<5000          ← FAIL (p95=6200ms)
```

## Performance Baselines

These are target performance baselines for the goimg-datalayer API:

### Non-Upload Endpoints

| Metric | Target | Excellent | Acceptable | Poor |
|--------|--------|-----------|------------|------|
| p(95) response time | < 200ms | < 100ms | 200-500ms | > 500ms |
| p(99) response time | < 500ms | < 250ms | 500-1000ms | > 1000ms |
| Error rate | < 1% | < 0.1% | 1-2% | > 2% |

### Upload Endpoints

| Metric | Target | Excellent | Acceptable | Poor |
|--------|--------|-----------|------------|------|
| p(95) response time | < 5s | < 3s | 5-10s | > 10s |
| p(99) response time | < 10s | < 7s | 10-20s | > 20s |
| Error rate | < 2% | < 0.5% | 2-5% | > 5% |

### Throughput Targets

| Operation | Target RPS | Peak RPS |
|-----------|-----------|----------|
| Image browsing | 500/s | 1000/s |
| Image details | 300/s | 600/s |
| Social actions (likes/comments) | 100/s | 250/s |
| Image uploads | 10/s | 25/s |

### Concurrent Users

| Scenario | Target VUs | Peak VUs |
|----------|-----------|----------|
| Mixed traffic | 150 | 300 |
| Browsing only | 100 | 200 |
| Social interactions | 75 | 150 |
| Uploads | 20 | 50 |

## Optimization Recommendations

### When p(95) > 200ms for Read Endpoints

1. **Database Query Optimization**
   - Add missing indexes
   - Review query plans with `EXPLAIN ANALYZE`
   - Implement database connection pooling
   - Consider query result caching

2. **Caching Strategy**
   - Implement Redis caching for frequently accessed data
   - Cache image metadata and variants
   - Set appropriate TTLs (Time To Live)

3. **Application-Level Optimization**
   - Reduce database round trips
   - Implement batch operations
   - Optimize JSON serialization
   - Use connection pooling

### When Upload p(95) > 5s

1. **Asynchronous Processing**
   - Move variant generation to background workers
   - Return 201 immediately after initial validation
   - Implement status polling endpoint

2. **Image Processing Optimization**
   - Optimize libvips settings
   - Generate variants in parallel
   - Consider progressive upload for large files

3. **Storage Optimization**
   - Use faster storage backend (SSD vs HDD)
   - Implement CDN for variant delivery
   - Consider image upload to edge locations

### When Error Rate > 1%

1. **Investigate Error Patterns**
   ```bash
   # Run test with verbose logging
   k6 run --http-debug tests/load/browse-flow.js
   ```

2. **Common Issues**
   - Database connection pool exhaustion
   - Rate limiting configuration
   - Timeout settings too aggressive
   - Resource limits (file descriptors, memory)

3. **Monitoring**
   - Check application logs
   - Monitor database connection pool
   - Review system resource usage (CPU, memory, I/O)

### General Best Practices

1. **Database**
   - Ensure proper indexing on frequently queried columns
   - Monitor slow query log
   - Use read replicas for read-heavy operations
   - Implement connection pooling (max connections: 25-50 per instance)

2. **Caching**
   - Cache image metadata (TTL: 5 minutes)
   - Cache user profiles (TTL: 10 minutes)
   - Cache popular images list (TTL: 1 minute)
   - Implement cache warming for popular content

3. **Application**
   - Use goroutine pools to limit concurrency
   - Implement graceful degradation
   - Set appropriate timeouts (read: 5s, write: 30s)
   - Monitor memory allocations and GC pressure

4. **Infrastructure**
   - Horizontal scaling: Add more API instances
   - Load balancing: Distribute traffic evenly
   - CDN: Serve static content and image variants
   - Database read replicas for scaling reads

## Troubleshooting

### Test Fails to Start

**Error**: `API health check failed - is the server running?`

**Solution**:
```bash
# Verify API is running
curl http://localhost:8080/api/v1/health

# If not, start the API
make run
```

### High Error Rates During Test

**Error**: `http_req_failed: rate>10%`

**Possible Causes**:
1. Rate limiting triggered
2. Database connection pool exhausted
3. System resource limits reached

**Solutions**:
```bash
# Check API logs
docker logs goimg-api

# Check database connections
docker exec -it goimg-postgres psql -U postgres -d goimg -c "SELECT count(*) FROM pg_stat_activity;"

# Check system resources
top
iostat
```

### k6 Installation Issues

**Docker Alternative**:
```bash
alias k6='docker run --rm -i --network host -v $(pwd):/workspace grafana/k6'
k6 run tests/load/browse-flow.js
```

### Timeout Errors

**Error**: `request timeout`

**Solutions**:
1. Increase timeout in test:
   ```javascript
   http.get(url, {
     timeout: '30s'  // Increase timeout
   });
   ```

2. Check API server health:
   ```bash
   curl -w "@curl-format.txt" http://localhost:8080/api/v1/images
   ```

### Memory Issues During Load Test

**Error**: k6 crashes or system runs out of memory

**Solutions**:
1. Reduce number of VUs
2. Reduce test duration
3. Run tests sequentially instead of in parallel
4. Use `--compatibility-mode=base` for lower memory usage

### Database Connection Pool Exhausted

**Symptoms**: Errors like "pq: sorry, too many clients already"

**Solutions**:
1. Increase database max connections:
   ```sql
   ALTER SYSTEM SET max_connections = 200;
   SELECT pg_reload_conf();
   ```

2. Optimize application connection pool:
   ```go
   db.SetMaxOpenConns(50)
   db.SetMaxIdleConns(25)
   db.SetConnMaxLifetime(5 * time.Minute)
   ```

3. Reduce VU count in load test

## Advanced Usage

### Distributed Load Testing

For testing at scale, use k6 Cloud or distributed execution:

```bash
# Using k6 Cloud
k6 cloud tests/load/mixed-traffic.js

# Using multiple k6 instances (manual)
# Instance 1:
k6 run --vus 50 tests/load/browse-flow.js

# Instance 2 (on different machine):
k6 run --vus 50 tests/load/social-flow.js
```

### Integrating with CI/CD

```yaml
# .github/workflows/load-test.yml
name: Load Tests
on:
  schedule:
    - cron: '0 2 * * *'  # Daily at 2 AM

jobs:
  load-test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v2
      - name: Install k6
        run: |
          sudo gpg --no-default-keyring --keyring /usr/share/keyrings/k6-archive-keyring.gpg --keyserver hkp://keyserver.ubuntu.com:80 --recv-keys C5AD17C747E3415A3642D57D77C6C491D6AC1D69
          echo "deb [signed-by=/usr/share/keyrings/k6-archive-keyring.gpg] https://dl.k6.io/deb stable main" | sudo tee /etc/apt/sources.list.d/k6.list
          sudo apt-get update
          sudo apt-get install k6
      - name: Run load tests
        run: make load-test-quick
```

### Custom Metrics and Trends

Add custom tracking to test scenarios:

```javascript
import { Trend } from 'k6/metrics';

const customTrend = new Trend('custom_metric');

export default function() {
  const start = Date.now();
  // ... perform operations
  customTrend.add(Date.now() - start);
}
```

## Resources

- [k6 Documentation](https://k6.io/docs/)
- [k6 Examples](https://k6.io/docs/examples/)
- [Performance Testing Best Practices](https://k6.io/docs/testing-guides/running-large-tests/)
- [Grafana Dashboard for k6](https://grafana.com/grafana/dashboards/2587)

## Support

For issues or questions:
- Check API server logs: `docker logs goimg-api`
- Review test output for specific errors
- Consult k6 documentation: https://k6.io/docs/
- Open an issue on the project repository
