# CDN Configuration Guide

> Content Delivery Network setup for optimal image delivery performance
>
> **Sprint 9 - Task 3.4**: CDN Configuration for Image Variants

This guide covers CDN configuration for the goimg-datalayer application to deliver image variants with optimal performance, caching, and global availability.

## Table of Contents

- [Overview](#overview)
- [CDN Benefits](#cdn-benefits)
- [CloudFlare Setup](#cloudflare-setup-recommended)
- [AWS CloudFront Setup](#aws-cloudfront-setup-alternative)
- [Cache Header Configuration](#cache-header-configuration)
- [CDN Purge Procedures](#cdn-purge-procedures)
- [Performance Optimization](#performance-optimization)
- [Monitoring and Analytics](#monitoring-and-analytics)
- [Troubleshooting](#troubleshooting)
- [Cost Estimation](#cost-estimation)

## Overview

The goimg-datalayer application generates multiple image variants (thumbnail, small, medium, large, original) that benefit significantly from CDN caching:

**Image Variant Endpoints**:
```
GET /api/v1/images/{imageID}/variants/{size}
```

**Variant Sizes**:
- `thumbnail` - 150x150px (profile pictures, thumbnails)
- `small` - 400x400px (mobile preview)
- `medium` - 800x800px (desktop preview)
- `large` - 1600x1600px (full-screen viewing)
- `original` - Original uploaded file (download only)

**Why CDN is Critical**:
- Image variants are **immutable** (never change after creation)
- High bandwidth requirements (images are large)
- Global user base requires low-latency delivery
- Reduces load on origin server (API)

## CDN Benefits

### Performance Improvements

| Metric | Without CDN | With CDN (CloudFlare) | Improvement |
|--------|-------------|----------------------|-------------|
| **TTFB (Time to First Byte)** | 300-500ms | 20-50ms | **85-90% faster** |
| **Image load time** (1MB) | 2-5s | 200-500ms | **75-90% faster** |
| **Origin server load** | 100% | 5-15% | **85-95% reduction** |
| **Bandwidth cost** | $0.09/GB | $0.01/GB | **90% cheaper** |

### Global Availability

**Edge Locations**:
- **CloudFlare**: 300+ locations worldwide
- **AWS CloudFront**: 450+ locations worldwide

**Latency Reduction**:
- Asia → US origin: 250-400ms → **20-50ms** (CDN edge)
- Europe → US origin: 100-200ms → **15-30ms** (CDN edge)
- Local → US origin: 50-100ms → **10-20ms** (CDN edge)

## CloudFlare Setup (Recommended)

CloudFlare is recommended for ease of setup, generous free tier, and excellent performance.

### Why CloudFlare?

**Advantages**:
- **Free tier**: Unlimited bandwidth, free SSL
- **Easy setup**: DNS-based configuration (no code changes)
- **WAF (Web Application Firewall)**: DDoS protection included
- **Analytics**: Real-time traffic insights
- **Image Optimization**: Automatic format conversion (AVIF, WebP)
- **Cache purge**: Instant global purge

**Free Tier Includes**:
- Unlimited bandwidth
- Free SSL/TLS certificates
- DDoS protection
- Basic WAF rules
- Analytics
- 100% uptime SLA

### Step 1: Create CloudFlare Account

1. Sign up at [https://dash.cloudflare.com/sign-up](https://dash.cloudflare.com/sign-up)
2. Add your domain (e.g., `example.com`)
3. CloudFlare will scan existing DNS records
4. Verify DNS records are correct

### Step 2: Update Nameservers

**Change nameservers at your domain registrar**:

```
Old nameservers (example):
ns1.digitalocean.com
ns2.digitalocean.com

New CloudFlare nameservers (provided by CloudFlare):
amber.ns.cloudflare.com
brad.ns.cloudflare.com
```

**Wait for DNS propagation** (5 minutes to 48 hours, typically < 1 hour):
```bash
# Check nameservers
dig NS example.com +short
# Should show CloudFlare nameservers
```

### Step 3: SSL/TLS Configuration

**Navigate to**: SSL/TLS → Overview

**Select SSL/TLS Mode**: **Full (strict)**

| Mode | Description | Recommended |
|------|-------------|-------------|
| Off | No HTTPS | ❌ Never use |
| Flexible | CloudFlare → Browser: HTTPS, CloudFlare → Origin: HTTP | ❌ Not secure |
| Full | End-to-end encryption but no certificate validation | ⚠️ Development only |
| **Full (strict)** | End-to-end encryption with valid certificate | ✅ **Production** |

**Requirements for Full (strict)**:
- Origin server must have valid SSL certificate (Let's Encrypt)
- Certificate must match domain
- See [docs/deployment/ssl.md](./ssl.md) for origin SSL setup

**Verify SSL/TLS**:
```bash
curl -I https://api.example.com/health
# HTTP/2 200 (HTTP/2 indicates CloudFlare is active)
```

### Step 4: Configure Page Rules for Image Caching

**Navigate to**: Rules → Page Rules → Create Page Rule

**Rule 1: Cache Image Variants (High TTL)**

| Setting | Value |
|---------|-------|
| **URL Pattern** | `api.example.com/api/v1/images/*/variants/*` |
| **Cache Level** | Cache Everything |
| **Edge Cache TTL** | 1 month (2592000 seconds) |
| **Browser Cache TTL** | 1 year (31536000 seconds) |

**Configuration**:
```
URL: api.example.com/api/v1/images/*/variants/*

Settings:
- Cache Level: Cache Everything
- Edge Cache TTL: 1 month
- Browser Cache TTL: 1 year
```

**Rule 2: Bypass Cache for API Endpoints**

| Setting | Value |
|---------|-------|
| **URL Pattern** | `api.example.com/api/v1/*` |
| **Cache Level** | Bypass |

**Configuration**:
```
URL: api.example.com/api/v1/*

Settings:
- Cache Level: Bypass
```

**Rule Priority**:
1. **Rule 1** (Image variants) - **Higher priority** (more specific pattern)
2. **Rule 2** (API bypass) - **Lower priority** (general pattern)

**Free Tier Limit**: 3 page rules (sufficient for image caching)

### Step 5: Caching Configuration

**Navigate to**: Caching → Configuration

**Cache Settings**:

| Setting | Value | Description |
|---------|-------|-------------|
| **Caching Level** | Standard | Cache static files automatically |
| **Browser Cache TTL** | Respect Existing Headers | Use origin `Cache-Control` headers |
| **Always Online** | On | Serve cached content if origin is down |

**Query String Handling**:
- **Respect Query String**: On (cache different variants separately)

**Advanced Settings**:

| Setting | Value |
|---------|-------|
| **Tiered Cache** | On (CloudFlare Pro+) | Multi-tier caching for better cache hit ratio |
| **Argo Smart Routing** | On (CloudFlare Pro+) | Intelligent routing for faster delivery |

### Step 6: Security Settings

**Navigate to**: Security → Settings

**Recommended Settings**:

| Setting | Value | Description |
|---------|-------|-------------|
| **Security Level** | Medium | Balance security and accessibility |
| **Challenge Passage** | 30 minutes | Time before re-challenging visitors |
| **Browser Integrity Check** | On | Block known malicious browsers |
| **Hotlink Protection** | Off | Allow image embedding (or enable if needed) |

**WAF (Web Application Firewall)**:

**Navigate to**: Security → WAF

**Managed Rules**:
- **OWASP Core Ruleset**: On
- **CloudFlare Managed Ruleset**: On

### Step 7: Performance Optimization

**Navigate to**: Speed → Optimization

**Recommended Settings**:

| Setting | Value | Description |
|---------|-------|-------------|
| **Auto Minify** | JavaScript, CSS, HTML | Reduce file sizes |
| **Brotli Compression** | On | Better compression than gzip |
| **Early Hints** | On | Speed up page loads |
| **HTTP/2** | On (default) | Faster multiplexed connections |
| **HTTP/3 (QUIC)** | On | Next-gen HTTP protocol |

**Image Optimization** (CloudFlare Pro+):

**Polish**: Lossless or Lossy
- Automatic WebP/AVIF conversion
- Reduces image size by 20-50%

**Mirage**: On (CloudFlare Pro+)
- Lazy loading for images
- Adaptive image quality based on connection speed

### Step 8: Verify CDN is Working

**Test cache hit**:
```bash
# First request (cache MISS)
curl -I https://api.example.com/api/v1/images/123e4567-e89b-12d3-a456-426614174000/variants/medium
# Expected: CF-Cache-Status: MISS

# Second request (cache HIT)
curl -I https://api.example.com/api/v1/images/123e4567-e89b-12d3-a456-426614174000/variants/medium
# Expected: CF-Cache-Status: HIT
```

**Check CloudFlare headers**:
```bash
curl -I https://api.example.com/api/v1/images/123e4567-e89b-12d3-a456-426614174000/variants/medium

# Expected headers:
# CF-Cache-Status: HIT
# CF-RAY: xxxxx-SJC (edge location)
# Age: 3600 (seconds in cache)
# Cache-Control: public, max-age=31536000, immutable
```

## AWS CloudFront Setup (Alternative)

AWS CloudFront is an alternative CDN with deep AWS integration.

### Why CloudFront?

**Advantages**:
- Deep AWS integration (S3, Lambda@Edge)
- More edge locations (450+ vs 300+)
- Advanced features (Lambda@Edge, CloudFront Functions)
- Fine-grained access control (signed URLs)

**Disadvantages**:
- More complex setup
- Higher cost than CloudFlare (no free tier)
- Requires AWS account

### Step 1: Create CloudFront Distribution

**Navigate to**: AWS CloudFront Console → Create Distribution

**Origin Settings**:

| Setting | Value | Example |
|---------|-------|---------|
| **Origin Domain** | Your origin domain | `api.example.com` |
| **Protocol** | HTTPS only | HTTPS only |
| **Minimum Origin SSL Protocol** | TLSv1.2 | TLSv1.2 |
| **Origin Path** | (empty) | - |
| **Custom Headers** | (optional) | `X-Origin-Verify: secret-token` |

**Default Cache Behavior**:

| Setting | Value |
|---------|-------|
| **Path Pattern** | Default (*) |
| **Viewer Protocol Policy** | Redirect HTTP to HTTPS |
| **Allowed HTTP Methods** | GET, HEAD, OPTIONS |
| **Cached HTTP Methods** | GET, HEAD, OPTIONS |
| **Cache Policy** | Create custom policy (see below) |
| **Origin Request Policy** | CORS-CustomOrigin |

**Create Custom Cache Policy**:

**Name**: `goimg-image-variants-cache`

| Setting | Value |
|---------|-------|
| **TTL Settings** | - |
| - Minimum TTL | 1 second |
| - Maximum TTL | 31536000 (1 year) |
| - Default TTL | 2592000 (30 days) |
| **Cache Key Settings** | - |
| - Headers | Include whitelist: `Accept`, `Accept-Encoding` |
| - Query Strings | All |
| - Cookies | None |
| **Compression** | Brotli and Gzip |

### Step 2: Distribution Settings

| Setting | Value | Example |
|---------|-------|---------|
| **Price Class** | Use All Edge Locations | All (or specific regions) |
| **AWS WAF** | (optional) | Enable if needed |
| **Alternate Domain Names (CNAMEs)** | Your domain | `cdn.example.com`, `api.example.com` |
| **SSL Certificate** | Custom SSL Certificate | Request ACM certificate |
| **Supported HTTP Versions** | HTTP/2, HTTP/3 | HTTP/2 and HTTP/3 |
| **Default Root Object** | (empty) | - |
| **Logging** | On (recommended) | S3 bucket for logs |

### Step 3: Request ACM Certificate

**Navigate to**: AWS Certificate Manager → Request Certificate

**Domain Names**:
```
api.example.com
cdn.example.com (optional)
```

**Validation**: DNS validation (recommended)

**Add CNAME records** to your DNS:
```
_xxxxx.api.example.com CNAME _xxxxx.acm-validations.aws.
```

**Wait for validation** (5-30 minutes).

### Step 4: Create Behavior for Image Variants

**Navigate to**: CloudFront Distribution → Behaviors → Create Behavior

**Path Pattern**: `/api/v1/images/*/variants/*`

| Setting | Value |
|---------|-------|
| **Origin** | api.example.com |
| **Viewer Protocol Policy** | HTTPS only |
| **Allowed HTTP Methods** | GET, HEAD, OPTIONS |
| **Cache Policy** | goimg-image-variants-cache |
| **Origin Request Policy** | CORS-CustomOrigin |
| **Response Headers Policy** | Create custom policy (see below) |

**Create Response Headers Policy**:

**Name**: `goimg-security-headers`

| Header | Value |
|--------|-------|
| **Strict-Transport-Security** | `max-age=31536000; includeSubDomains; preload` |
| **X-Content-Type-Options** | `nosniff` |
| **X-Frame-Options** | `SAMEORIGIN` |
| **Referrer-Policy** | `strict-origin-when-cross-origin` |
| **CORS Allow-Origin** | `*` (or specific origins) |
| **CORS Allow-Methods** | `GET, HEAD, OPTIONS` |
| **CORS Allow-Headers** | `Accept, Accept-Encoding` |

### Step 5: Update DNS

**Create CNAME record** pointing to CloudFront:

```
Type: CNAME
Name: cdn
Value: d123456789abcd.cloudfront.net (from CloudFront distribution)
TTL: 300
```

**Or use Route 53 Alias** (recommended if using Route 53):
```
Type: A (Alias)
Name: cdn
Alias Target: d123456789abcd.cloudfront.net
```

### Step 6: Verify CloudFront

**Test cache**:
```bash
curl -I https://cdn.example.com/api/v1/images/123/variants/medium

# Expected headers:
# X-Cache: Hit from cloudfront
# Age: 3600
# Cache-Control: public, max-age=31536000, immutable
```

## Cache Header Configuration

The goimg-datalayer API already sends optimal cache headers for image variants.

### Application Cache Headers

**Implemented in**: `internal/interfaces/http/handlers/image_handler.go` (line 753)

```go
// GetImageVariant sends cache headers for image variants
func (h *ImageHandler) GetImageVariant(w http.ResponseWriter, r *http.Request) {
    // ... (variant retrieval logic)

    // Set cache headers for immutable image variants
    w.Header().Set("Content-Type", contentType)
    w.Header().Set("Content-Length", strconv.FormatInt(variantDTO.FileSize, 10))
    w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")

    // ... (stream image data)
}
```

### Cache-Control Directives Explained

**`Cache-Control: public, max-age=31536000, immutable`**

| Directive | Meaning | Value |
|-----------|---------|-------|
| **public** | Can be cached by any cache (CDN, browser, proxy) | - |
| **max-age=31536000** | Cache for 31,536,000 seconds (1 year) | 1 year |
| **immutable** | Content never changes, no need to revalidate | - |

**Why 1 Year?**
- Image variants are **immutable** (never change after creation)
- New image uploads get new UUIDs (different URLs)
- Maximum allowed cache duration

### Additional Headers

**Implemented in nginx** (`docker/nginx/conf.d/api.conf`):

```nginx
# Cache headers for image variants
location ~* ^/api/v1/images/.+/variants/.+$ {
    proxy_pass http://api:8080;

    # Cache headers (set by application)
    # Cache-Control: public, max-age=31536000, immutable

    # ETag and Last-Modified for conditional requests
    add_header ETag $upstream_http_etag;
    add_header Last-Modified $upstream_http_last_modified;

    # CORS headers
    add_header Access-Control-Allow-Origin * always;
    add_header Access-Control-Allow-Methods "GET, HEAD, OPTIONS" always;

    # Security headers
    add_header X-Content-Type-Options nosniff always;
}
```

### Conditional Requests (304 Not Modified)

**ETag and Last-Modified** enable conditional requests:

```bash
# First request
curl -I https://api.example.com/api/v1/images/123/variants/medium
# Response:
# HTTP/2 200
# ETag: "abc123"
# Last-Modified: Mon, 01 Jan 2024 00:00:00 GMT
# Cache-Control: public, max-age=31536000, immutable

# Conditional request (with If-None-Match)
curl -I https://api.example.com/api/v1/images/123/variants/medium \
  -H "If-None-Match: \"abc123\""
# Response:
# HTTP/2 304 Not Modified (no body sent, saves bandwidth)
```

### Cache TTL Recommendations

| Content Type | TTL | Reason |
|-------------|-----|--------|
| **Image variants** | 1 year (31536000s) | Immutable, never changes |
| **Thumbnails** | 1 year (31536000s) | Immutable, never changes |
| **Original images** | 1 year (31536000s) | Immutable, never changes |
| **API responses** (JSON) | No cache (0s) | Dynamic, user-specific |
| **Health check** | No cache (0s) | Real-time status |

## CDN Purge Procedures

### When to Purge Cache

**Purge is needed when**:
- Image is **deleted** (remove from cache to prevent access)
- Image is **updated** (rare - variants are immutable)
- Security incident (immediate cache eviction)

**Purge is NOT needed for**:
- New image upload (new URL, won't conflict with cache)
- Metadata update (title, description) - doesn't affect variants

### CloudFlare Purge

**Purge Single File**:

```bash
# Using CloudFlare API
curl -X POST "https://api.cloudflare.com/client/v4/zones/ZONE_ID/purge_cache" \
  -H "Authorization: Bearer YOUR_API_TOKEN" \
  -H "Content-Type: application/json" \
  --data '{
    "files": [
      "https://api.example.com/api/v1/images/123e4567-e89b-12d3-a456-426614174000/variants/medium",
      "https://api.example.com/api/v1/images/123e4567-e89b-12d3-a456-426614174000/variants/thumbnail"
    ]
  }'
```

**Purge All Image Variants for a Single Image**:

```bash
# Purge all variants for image ID
IMAGE_ID="123e4567-e89b-12d3-a456-426614174000"

curl -X POST "https://api.cloudflare.com/client/v4/zones/ZONE_ID/purge_cache" \
  -H "Authorization: Bearer YOUR_API_TOKEN" \
  -H "Content-Type: application/json" \
  --data "{
    \"files\": [
      \"https://api.example.com/api/v1/images/$IMAGE_ID/variants/thumbnail\",
      \"https://api.example.com/api/v1/images/$IMAGE_ID/variants/small\",
      \"https://api.example.com/api/v1/images/$IMAGE_ID/variants/medium\",
      \"https://api.example.com/api/v1/images/$IMAGE_ID/variants/large\",
      \"https://api.example.com/api/v1/images/$IMAGE_ID/variants/original\"
    ]
  }"
```

**Purge Everything** (use sparingly):

```bash
# WARNING: Purges all cached content
curl -X POST "https://api.cloudflare.com/client/v4/zones/ZONE_ID/purge_cache" \
  -H "Authorization: Bearer YOUR_API_TOKEN" \
  -H "Content-Type: application/json" \
  --data '{"purge_everything":true}'
```

**Get CloudFlare Zone ID**:
```bash
curl -X GET "https://api.cloudflare.com/client/v4/zones?name=example.com" \
  -H "Authorization: Bearer YOUR_API_TOKEN" | jq -r '.result[0].id'
```

**CloudFlare Dashboard Purge**:
1. Navigate to: Caching → Configuration
2. Click "Purge Cache"
3. Select "Custom Purge" → "Purge by URL"
4. Enter URLs to purge

### AWS CloudFront Purge

**Purge (Invalidation)**:

```bash
# Invalidate single file
aws cloudfront create-invalidation \
  --distribution-id E1234567890ABC \
  --paths "/api/v1/images/123e4567-e89b-12d3-a456-426614174000/variants/medium"

# Invalidate all variants for an image
aws cloudfront create-invalidation \
  --distribution-id E1234567890ABC \
  --paths "/api/v1/images/123e4567-e89b-12d3-a456-426614174000/variants/*"

# Invalidate all images (use sparingly)
aws cloudfront create-invalidation \
  --distribution-id E1234567890ABC \
  --paths "/api/v1/images/*/variants/*"
```

**Check invalidation status**:
```bash
aws cloudfront get-invalidation \
  --distribution-id E1234567890ABC \
  --id INVALIDATION_ID
```

**CloudFront Console**:
1. Navigate to: CloudFront → Distributions → Your Distribution
2. Click "Invalidations" tab
3. Click "Create Invalidation"
4. Enter paths: `/api/v1/images/*/variants/*`

**Cost**: 1,000 free invalidations per month, $0.005 per path after

### Automated Purge Integration

**Integrate purge into application**:

```go
// internal/application/gallery/commands/delete_image.go

func (h *DeleteImageHandler) Handle(ctx context.Context, cmd DeleteImageCommand) error {
    // 1. Delete image from database
    if err := h.repo.Delete(ctx, cmd.ImageID); err != nil {
        return err
    }

    // 2. Delete from storage
    if err := h.storage.Delete(ctx, imageKey); err != nil {
        return err
    }

    // 3. Purge from CDN
    if err := h.cdn.PurgeImageVariants(ctx, cmd.ImageID); err != nil {
        h.logger.Warn().Err(err).Msg("Failed to purge CDN cache (non-critical)")
        // Don't fail the delete operation if purge fails
    }

    return nil
}
```

## Performance Optimization

### 1. Preload Critical Images

**HTTP/2 Server Push** (CloudFlare, CloudFront):

```nginx
# Nginx configuration
location / {
    http2_push /api/v1/images/featured/variants/medium;
}
```

**Preload Header**:
```nginx
add_header Link "</api/v1/images/featured/variants/medium>; rel=preload; as=image" always;
```

### 2. Image Format Optimization

**CloudFlare Polish** (Pro plan):
- Automatically converts JPEG/PNG to WebP or AVIF
- Reduces file size by 20-50%
- Transparent to client (content negotiation via `Accept` header)

**Example**:
```bash
# Browser sends
Accept: image/webp,image/apng,image/*,*/*;q=0.8

# CloudFlare returns WebP if Polish is enabled
Content-Type: image/webp
```

### 3. Lazy Loading

**Mirage (CloudFlare Pro)**:
- Lazy loads images below the fold
- Adaptive quality based on connection speed
- Automatic `loading="lazy"` attribute

### 4. Tiered Caching

**CloudFlare Tiered Cache** (Pro plan):
- Multiple cache tiers (origin → regional → edge)
- Higher cache hit ratio
- Reduces origin requests by 95%+

**CloudFront with Regional Edge Caches**:
- Automatically enabled
- 13 regional edge caches worldwide

## Monitoring and Analytics

### CloudFlare Analytics

**Navigate to**: Analytics & Logs → Traffic

**Key Metrics**:
- **Requests**: Total requests, cached vs uncached
- **Bandwidth**: Total bandwidth, saved bandwidth
- **Cache Hit Ratio**: Percentage of cached requests
- **Top Endpoints**: Most accessed URLs

**Expected Cache Hit Ratio**:
- **Image variants**: 95-99% (immutable content)
- **API endpoints**: 0% (bypassed)

**GraphQL Analytics** (CloudFlare Pro+):
```graphql
query {
  viewer {
    zones(filter: {zoneTag: "ZONE_ID"}) {
      httpRequests1dGroups(
        limit: 30
        filter: {
          date_geq: "2024-12-01"
          date_lt: "2024-12-31"
        }
      ) {
        dimensions {
          date
        }
        sum {
          requests
          cachedRequests
          bytes
          cachedBytes
        }
      }
    }
  }
}
```

### AWS CloudFront Monitoring

**CloudWatch Metrics**:

| Metric | Description | Target |
|--------|-------------|--------|
| **CacheHitRate** | Percentage of cached requests | > 95% |
| **OriginLatency** | Time to origin | < 100ms |
| **4xxErrorRate** | Client errors | < 1% |
| **5xxErrorRate** | Server errors | < 0.1% |

**Access Logs**:
```bash
# Enable logging to S3
aws cloudfront update-distribution \
  --id E1234567890ABC \
  --logging-config \
    Enabled=true,\
    Bucket=my-cloudfront-logs.s3.amazonaws.com,\
    Prefix=cdn/
```

**Analyze logs with Athena**:
```sql
SELECT
  uri,
  COUNT(*) as request_count,
  SUM(CASE WHEN result_type = 'Hit' THEN 1 ELSE 0 END) as cache_hits,
  ROUND(SUM(CASE WHEN result_type = 'Hit' THEN 1 ELSE 0 END) * 100.0 / COUNT(*), 2) as hit_rate
FROM cloudfront_logs
WHERE uri LIKE '%/variants/%'
GROUP BY uri
ORDER BY request_count DESC
LIMIT 100;
```

## Troubleshooting

### Cache Not Working

**Symptom**: `CF-Cache-Status: MISS` or `X-Cache: Miss from cloudfront` on every request

**Solutions**:

1. **Check cache headers from origin**:
   ```bash
   curl -I https://api.example.com/api/v1/images/123/variants/medium
   # Must include: Cache-Control: public, max-age=31536000
   ```

2. **Verify page rule is active** (CloudFlare):
   - Navigate to: Rules → Page Rules
   - Check rule order (more specific rules first)
   - Verify URL pattern matches

3. **Check query strings**:
   ```bash
   # Different query strings = different cache entries
   /variants/medium        # Cached separately
   /variants/medium?v=1    # Cached separately
   ```

4. **Verify origin is reachable**:
   ```bash
   curl -I https://api.example.com/health
   # Should return 200 OK
   ```

### High Cache Miss Rate

**Symptom**: Cache hit ratio < 80%

**Solutions**:

1. **Check URL variations**:
   ```bash
   # Inconsistent URLs reduce cache hit rate
   /api/v1/images/123/variants/medium     # Good
   /api/v1/images/123/variants/medium/    # Different (trailing slash)
   ```

2. **Query string caching**:
   - Enable "Query String Sort" (CloudFlare)
   - Normalize query strings in application

3. **Cookie stripping**:
   - Remove cookies for image requests
   - CloudFlare: Page Rules → "Disable Performance" → Off

### Stale Content After Update

**Symptom**: Old image still served after deletion

**Solutions**:

1. **Purge cache manually**:
   ```bash
   # CloudFlare
   curl -X POST "https://api.cloudflare.com/client/v4/zones/ZONE_ID/purge_cache" \
     -H "Authorization: Bearer YOUR_API_TOKEN" \
     --data '{"files":["https://api.example.com/api/v1/images/123/variants/medium"]}'
   ```

2. **Automatic purge on delete**:
   - Integrate CDN purge into delete workflow
   - Use webhook or API integration

3. **Shorter TTL for testing**:
   ```nginx
   # Temporarily reduce TTL for testing
   Cache-Control: public, max-age=300  # 5 minutes
   ```

## Cost Estimation

### CloudFlare Pricing

**Free Tier**:
- Unlimited bandwidth (free forever)
- Free SSL/TLS certificates
- Basic DDoS protection
- 3 page rules

**Pro ($20/month)**:
- All Free features
- Image optimization (Polish, Mirage)
- Advanced DDoS protection
- 20 page rules
- Tiered caching

**Business ($200/month)**:
- All Pro features
- Advanced WAF
- Custom SSL
- Dedicated support

**Enterprise (custom pricing)**:
- Custom features
- 24/7 support
- SLA guarantees

### AWS CloudFront Pricing

**Data Transfer Out** (to internet):

| Region | First 10TB/month | Next 40TB/month | Next 100TB/month |
|--------|------------------|-----------------|------------------|
| **US, Europe, Israel** | $0.085/GB | $0.080/GB | $0.060/GB |
| **Asia Pacific** | $0.140/GB | $0.135/GB | $0.120/GB |
| **South America** | $0.250/GB | $0.240/GB | $0.220/GB |

**HTTP/HTTPS Requests**:
- First 10M requests: $0.0075 per 10,000 requests
- Next 90M requests: $0.0070 per 10,000 requests

**Invalidations**:
- First 1,000 paths/month: Free
- After 1,000: $0.005 per path

**Example (100TB/month, US region)**:
- Data transfer: ~$6,000/month
- Requests (10B): ~$70/month
- **Total**: ~$6,070/month

**Comparison**: CloudFlare Pro ($20/month) vs CloudFront (~$6,000/month for 100TB)

### Cost Optimization

**CloudFlare (Free Tier)**:
- No data transfer charges
- Unlimited bandwidth
- **Best for**: Most use cases

**CloudFront**:
- Pay per GB transferred
- Good for AWS-native deployments
- **Best for**: Deep AWS integration needed

**Optimization Tips**:
1. **Enable compression** (Brotli, gzip) - reduces data transfer by 50-70%
2. **Image optimization** (CloudFlare Polish) - reduces size by 20-50%
3. **High cache TTL** - reduces origin requests
4. **Purge selectively** - avoid "purge everything"

## References

- [CloudFlare Documentation](https://developers.cloudflare.com/)
- [AWS CloudFront Documentation](https://docs.aws.amazon.com/cloudfront/)
- [HTTP Caching (MDN)](https://developer.mozilla.org/en-US/docs/Web/HTTP/Caching)
- [Cache-Control Header Spec (RFC 7234)](https://httpwg.org/specs/rfc7234.html)
- [Production Deployment Guide](./production.md)
- [Environment Variables Reference](./environment_variables.md)

---

**Document Version**: 1.0
**Last Updated**: 2025-12-07 (Sprint 9 - Task 3.4)
**Next Review**: After CDN setup in production
