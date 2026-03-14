# Building and Deploying

## Prerequisites

- Docker Desktop or Docker Engine
- docker-compose v1.27+
- At least 4GB free disk space
- 2+ CPU cores available

## Development Build

### 1. Setup Development Environment

```bash
chmod +x setup.sh
./setup.sh
```

This will:
- Verify prerequisites (Go, Docker, protoc)
- Download Go dependencies
- Generate protobuf code
- Build the Go service binary
- Build Docker images

### 2. Start the System

```bash
docker-compose up -d
```

Monitor startup:
```bash
docker-compose logs -f
```

Wait for all services to be healthy:
```bash
docker-compose ps
```

### 3. Verify Health

```bash
# Check service health
docker-compose exec threat-scan-service curl http://localhost:50051/healthz

# Check ClamAV
docker-compose exec clamav clamdscan --version

# Check NSQ
curl http://localhost:4161/stats.json | jq
```

## Production Build

### 1. Optimize Docker Images

Edit Dockerfiles to:
- Use specific alpine/golang versions (no 'latest')
- Add security scanning
- Minimize layer count
- Set resource limits

### 2. Use Docker Multi-stage Build

The Dockerfile.service already uses multi-stage builds to reduce image size.

### 3. Configure Resource Limits

In `docker-compose.yaml`, add:

```yaml
services:
  threat-scan-service:
    deploy:
      resources:
        limits:
          cpus: '2'
          memory: 2G
        reservations:
          cpus: '1'
          memory: 1G
```

### 4. Security Hardening

```yaml
services:
  threat-scan-service:
    security_opt:
      - no-new-privileges:true
    read_only: true
    tmpfs: /tmp
```

### 5. Setup Persistence

```yaml
volumes:
  uploads:
    driver: local
    driver_opts:
      type: nfs
      o: addr=${NFS_SERVER},vers=4,soft,timeo=180,bg,tcp,rw
      device: ":/exports/threat-scan-uploads"
```

## Scaling

### Horizontal Scaling

Run multiple instances of threat-scan-service:

```bash
docker-compose scale threat-scan-service=3
```

With load balancing (Nginx example):
```nginx
upstream scanners {
    server threat-scan-service:50051;
    server threat-scan-service-2:50051;
    server threat-scan-service-3:50051;
}

grpc//call upstream scanners;
```

### NSQ Scaling

Add more NSQd nodes:

```yaml
nsqd-2:
  image: nsqio/nsq:latest
  command: /nsqd --lookupd-tcp-address=nsqlookupd:4160
```

## Monitoring & Logging

### ELK Stack Integration

```yaml
elasticsearch:
  image: docker.elastic.co/elasticsearch/elasticsearch:8.0.0
  
logstash:
  image: docker.elastic.co/logstash/logstash:8.0.0
  volumes:
    - ./logstash.conf:/usr/share/logstash/pipeline/logstash.conf

kibana:
  image: docker.elastic.co/kibana/kibana:8.0.0
  ports:
    - "5601:5601"
```

### Prometheus Metrics

Add to service:
```go
import "github.com/prometheus/client_golang/prometheus"

var scanDuration = prometheus.NewHistogramVec(
    prometheus.HistogramOpts{
        Name: "scan_duration_seconds",
    },
    []string{"engine", "status"},
)
```

## Troubleshooting Build Issues

### OutOfMemory during build

Increase Docker memory limit and retry:
```bash
docker build --memory=4g -f docker/Dockerfile.service .
```

### Protobuf generation fails

Regenerate with verbose output:
```bash
protoc -v --go_out=. proto/scan.proto
```

### ClamAV database download fails

The ClamAV image downloads virus definitions on startup. If behind a proxy:

```dockerfile
RUN freshclam --http-proxy=proxy.example.com:8080
```

## Rolling Updates

### Blue-Green Deployment

```bash
# Deploy new version to "green" environment
docker-compose -f docker-compose.green.yaml up -d

# Test health checks pass
docker-compose -f docker-compose.green.yaml exec threat-scan-service healthcheck

# Switch traffic
# (requires load balancer modification)

# Take down old version
docker-compose down
mv docker-compose.green.yaml docker-compose.yaml
```

## Backing Up Data

```bash
# Backup ClamAV databases
docker run --rm \
  -v threat-scan_clamav-db:/data \
  -v $(pwd)/backups:/backup \
  alpine tar czf /backup/clamav-db.tar.gz /data

# Backup uploads
docker run --rm \
  -v threat-scan_uploads:/data \
  -v $(pwd)/backups:/backup \
  alpine tar czf /backup/uploads.tar.gz /data
```

## Performance Tuning

### ClamAV Configuration

```dockerfile
RUN cat >> /etc/clamav/clamd.conf << EOF
MaxScanSize 200M
MaxFileSize 150M
MaxRecursion 16
MaxFiles 5000
EOF
```

### NSQ Configuration

Adjust in docker-compose:
```yaml
command: /nsqd --max-bytes-per-file=268435456 --sync-interval=5s
```

### Service Configuration

In `config/config.yaml`:
```yaml
server:
  max_concurrent_scans: 50  # Increase for more throughput
nsq:
  max_in_flight: 500  # Higher = more parallel processing
av_engines:
  clamav:
    timeout: 300  # Increase for large files
```
