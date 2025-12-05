# Production Deployment Guide

## Quick Start

### Start Production Services

```bash
# From project root
cd /home/user/goimg-datalayer

# Set environment variables (required for production)
export DB_PASSWORD="your_secure_password_here"
export GRAFANA_ADMIN_PASSWORD="your_grafana_password_here"

# Start all services
docker-compose -f docker/docker-compose.prod.yml up -d

# Check service health
docker-compose -f docker/docker-compose.prod.yml ps

# View logs
docker-compose -f docker/docker-compose.prod.yml logs -f
```

### Stop Services

```bash
docker-compose -f docker/docker-compose.prod.yml down

# Stop and remove volumes (WARNING: destroys all data)
docker-compose -f docker/docker-compose.prod.yml down -v
```

## Service URLs

| Service | URL | Default Credentials |
|---------|-----|---------------------|
| Grafana | http://localhost:3000 | admin / admin |
| Prometheus | http://localhost:9091 | - |
| API | http://localhost:8080 | - |
| API Metrics | http://localhost:9090/metrics | - |

## Pre-requisites

1. **Docker Images**: Build your API and Worker images first
   ```bash
   docker build -t goimg-api:latest -f Dockerfile.api .
   docker build -t goimg-worker:latest -f Dockerfile.worker .
   ```

2. **Environment Variables**: Set production secrets
   ```bash
   export DB_PASSWORD="secure_password"
   export GRAFANA_ADMIN_PASSWORD="secure_password"
   ```

3. **Network Ports**: Ensure these ports are available:
   - 3000 (Grafana)
   - 8080 (API)
   - 9090 (API Metrics)
   - 9091 (Prometheus)
   - 5432 (PostgreSQL - optional, can be internal only)
   - 6379 (Redis - optional, can be internal only)

## Monitoring Setup

### Access Grafana Dashboards

1. Navigate to http://localhost:3000
2. Login with credentials (default: admin/admin)
3. Go to **Dashboards** → **goimg** folder
4. Available dashboards:
   - Application Overview
   - Image Gallery Metrics
   - Security Events
   - Infrastructure Health

### Configure Alerts

Grafana dashboards include pre-configured alerts:
- Malware Detection Alert (in Security Events dashboard)

To enable email notifications:
1. Edit `docker/docker-compose.prod.yml`
2. Add SMTP configuration to Grafana environment variables
3. Restart Grafana container

## Network Architecture

```
┌─────────────────────────────────────────────────┐
│ Frontend Network (172.20.0.0/24)                │
│  - API (port 8080, 9090)                        │
│  - Grafana (port 3000)                          │
│  - Prometheus (port 9091)                       │
└─────────────────────────────────────────────────┘
                      │
┌─────────────────────────────────────────────────┐
│ Backend Network (172.21.0.0/24)                 │
│  - API, Worker                                  │
│  - Redis, IPFS                                  │
│  - Prometheus                                   │
└─────────────────────────────────────────────────┘
                      │
┌─────────────────────────────────────────────────┐
│ Database Network (172.22.0.0/24) - INTERNAL     │
│  - PostgreSQL (isolated)                        │
└─────────────────────────────────────────────────┘
```

## Resource Requirements

Minimum recommended specs:
- **CPU**: 8 cores
- **RAM**: 16GB
- **Disk**: 100GB SSD

Service resource allocation:
- API: 2 CPU, 2GB RAM
- Worker: 4 CPU, 4GB RAM
- PostgreSQL: 2 CPU, 2GB RAM
- Redis: 1 CPU, 1GB RAM
- ClamAV: 2 CPU, 4GB RAM
- IPFS: 2 CPU, 2GB RAM
- Prometheus: 1 CPU, 2GB RAM
- Grafana: 1 CPU, 1GB RAM

## Health Monitoring

Check service health:
```bash
# All services
docker-compose -f docker/docker-compose.prod.yml ps

# Specific service
docker inspect --format='{{json .State.Health}}' goimg-api | jq

# API health endpoint
curl http://localhost:8080/health

# Prometheus health
curl http://localhost:9091/-/healthy

# Grafana health
curl http://localhost:3000/api/health
```

## Logs

View logs:
```bash
# All services
docker-compose -f docker/docker-compose.prod.yml logs

# Specific service
docker-compose -f docker/docker-compose.prod.yml logs api

# Follow logs
docker-compose -f docker/docker-compose.prod.yml logs -f api

# Last 100 lines
docker-compose -f docker/docker-compose.prod.yml logs --tail=100 api
```

## Backup and Recovery

### Database Backup
```bash
docker exec goimg-postgres pg_dump -U goimg goimg > backup_$(date +%Y%m%d_%H%M%S).sql
```

### Restore Database
```bash
docker exec -i goimg-postgres psql -U goimg goimg < backup_20231205_120000.sql
```

### Grafana Backup
```bash
docker exec goimg-grafana grafana-cli admin export-dashboards --outputDir=/tmp/backup
docker cp goimg-grafana:/tmp/backup ./grafana-backup-$(date +%Y%m%d)
```

### IPFS Backup
```bash
docker exec goimg-ipfs tar czf /tmp/ipfs-backup.tar.gz /data/ipfs
docker cp goimg-ipfs:/tmp/ipfs-backup.tar.gz ./ipfs-backup-$(date +%Y%m%d).tar.gz
```

## Troubleshooting

### Services won't start
```bash
# Check Docker daemon
systemctl status docker

# Check logs for errors
docker-compose -f docker/docker-compose.prod.yml logs

# Verify images exist
docker images | grep goimg

# Check disk space
df -h
```

### Database connection errors
```bash
# Verify PostgreSQL is healthy
docker exec goimg-postgres pg_isready -U goimg

# Check connection from API
docker exec goimg-api nc -zv postgres 5432

# View PostgreSQL logs
docker logs goimg-postgres
```

### Memory issues
```bash
# Check container memory usage
docker stats

# Verify resource limits
docker inspect goimg-api | jq '.[0].HostConfig.Memory'

# Check host memory
free -h
```

### Network connectivity issues
```bash
# List networks
docker network ls

# Inspect network
docker network inspect goimg_frontend

# Test connectivity between services
docker exec goimg-api ping -c 3 postgres
docker exec goimg-prometheus wget -O- http://api:9090/metrics
```

## Security Hardening

1. **Change default passwords**
   ```bash
   export GRAFANA_ADMIN_PASSWORD="strong_random_password"
   ```

2. **Use secrets management**
   - Use Docker secrets instead of environment variables
   - Store secrets in encrypted vault (HashiCorp Vault, AWS Secrets Manager)

3. **Enable TLS/HTTPS**
   - Use reverse proxy (Nginx, Traefik)
   - Obtain SSL certificates (Let's Encrypt)

4. **Restrict network access**
   - Use firewall rules
   - Limit exposed ports
   - Use VPN for internal services

5. **Regular updates**
   ```bash
   docker-compose -f docker/docker-compose.prod.yml pull
   docker-compose -f docker/docker-compose.prod.yml up -d
   ```

## Performance Tuning

### PostgreSQL
Edit `docker/postgres/postgresql.conf`:
```ini
shared_buffers = 512MB
effective_cache_size = 2GB
maintenance_work_mem = 128MB
max_connections = 100
```

### Redis
Adjust maxmemory in docker-compose.prod.yml:
```yaml
command: redis-server --appendonly yes --maxmemory 2gb --maxmemory-policy allkeys-lru
```

### Prometheus
Adjust retention in docker-compose.prod.yml:
```yaml
command:
  - '--storage.tsdb.retention.time=60d'  # Increase to 60 days
```

## Scaling

### Horizontal Scaling (API)
```yaml
# In docker-compose.prod.yml
api:
  deploy:
    replicas: 3  # Run 3 API instances
```

### Vertical Scaling (Resources)
```yaml
# In docker-compose.prod.yml
api:
  deploy:
    resources:
      limits:
        cpus: '4'      # Increase to 4 CPUs
        memory: 4G     # Increase to 4GB
```

## CI/CD Integration

GitHub Actions example:
```yaml
- name: Deploy to production
  run: |
    docker-compose -f docker/docker-compose.prod.yml pull
    docker-compose -f docker/docker-compose.prod.yml up -d
```

## Monitoring Checklist

- [ ] Grafana accessible and dashboards loading
- [ ] Prometheus scraping metrics successfully
- [ ] All services showing healthy status
- [ ] Database connection pool metrics visible
- [ ] Image upload metrics being recorded
- [ ] Security events being tracked
- [ ] Alerts configured and tested
- [ ] Log rotation working correctly
- [ ] Backup strategy implemented
- [ ] Resource usage within limits

## Support

For issues related to:
- **Infrastructure**: Check this deployment guide
- **Monitoring**: See `/home/user/goimg-datalayer/monitoring/README.md`
- **Application**: See project CLAUDE.md and documentation

## Next Steps

1. Build API and Worker Docker images
2. Set production environment variables
3. Start production stack
4. Verify all services are healthy
5. Access Grafana and confirm dashboards load
6. Implement Prometheus metrics in Go application
7. Configure alerting and notifications
8. Set up automated backups
9. Implement SSL/TLS with reverse proxy
10. Create runbook for common operations
