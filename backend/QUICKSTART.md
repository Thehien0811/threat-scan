# Quick Reference

## Starting the System

```bash
# First time setup
chmod +x setup.sh
./setup.sh

# Start containers
docker-compose up -d

# View logs
docker-compose logs -f
```

## Testing

### Test with gRPC

```bash
# Copy test file to uploads volume
docker cp /path/to/test.exe threat-scan-service:/uploads/

# Scan via gRPC
grpcurl -plaintext \
  -d '{"sha256":"test123","filepath":"test.exe","filename":"test.exe"}' \
  localhost:50051 \
  scan.ScanService/Scan
```

### Test with NSQ

```bash
# Publish scan request
curl -X POST http://localhost:4151/pub?topic=threat_scan \
  -d '{"sha256":"test","filepath":"file.exe","filename":"file.exe","id":"req1"}'

# View results in NSQ Admin
# http://localhost:4171
```

## Common Commands

| Command | Purpose |
|---------|---------|
| `docker-compose up -d` | Start all services |
| `docker-compose down` | Stop all services |
| `docker-compose ps` | Check service status |
| `docker-compose logs service_name` | View service logs |
| `docker-compose exec service_name sh` | SSH into container |
| `make build` | Build Go service |
| `make docker-build` | Build Docker images |
| `make clean` | Clean everything |

## Port Mapping

| Service | Port | Purpose |
|---------|------|---------|
| NSQLookupd | 4160/4161 | NSQ coordination |
| NSQd | 4150/4151 | Message queue |
| NSQ Admin | 4171 | Admin UI |
| ClamAV | 3310 | AV scanning |
| gRPC Service | 50051 | API endpoint |

## Configuration Files

- **config/config.yaml** - Service configuration
- **docker-compose.yaml** - Container orchestration
- **Dockerfile.service** - Go service image
- **Dockerfile.clamav** - ClamAV image
- **Makefile** - Build automation

## Key Directories

- **cmd/** - Application entry point
- **service/** - Core scanning logic
- **proto/** - Protocol buffer definitions
- **docker/** - Dockerfile definitions
- **config/** - Configuration files
- **docs/** - Documentation

## File Upload

Files must be placed in the `/uploads` mounted volume:

```bash
# From host
docker cp local-file.exe threat-scan-service:/uploads/

# Filepath in request
filepath: "local-file.exe"  # (relative to /uploads)
```

## Monitoring NSQ

- Admin UI: http://localhost:4171
- View topics, channels, message rates
- Monitor consumer lag
- Inspect messages

## Health Checks

```bash
# Service health
docker-compose ps

# Individual component health
docker-compose exec clamav clamdscan --version
docker-compose exec nsqd curl http://localhost:4151/api/stats
curl http://localhost:4161/api/nodes
```

## Stop and Clean

```bash
# Stop containers
docker-compose stop

# Remove containers
docker-compose down

# Remove volumes (WARNING: deletes data)
docker-compose down -v

# Full cleanup
make clean
```

## Viewing Logs

```bash
# All services
docker-compose logs

# Specific service
docker-compose logs threat-scan-service

# Follow logs
docker-compose logs -f

# Last 100 lines
docker-compose logs --tail=100
```

## Building Locally (Without Docker)

```bash
# Generate protobuf
make generate

# Build binary
make build

# Binary location
./bin/threat-scan-service
```

## Debugging

### Check if port is accessible
```bash
nc -zv localhost 50051
```

### Check Docker network
```bash
docker network inspect threat-scan_threat-scan-net
```

### View container IP
```bash
docker inspect threat-scan-service | grep IPAddress
```

### Execute command in container
```bash
docker-compose exec threat-scan-service ps aux
```

## Performance Tips

1. **Increase concurrent scans** - Edit config.yaml `max_concurrent_scans`
2. **Adjust timeouts** - Longer timeouts for large files
3. **NSQ tuning** - Increase `max_in_flight` for more parallelism
4. **ClamAV updates** - Run `docker-compose exec clamav freshclam`

## Error Messages

| Error | Solution |
|-------|----------|
| "Connection refused" | Ensure docker-compose up was run |
| "File not found" | Ensure file is in /uploads volume |
| "Timeout" | Increase timeout in config.yaml |
| "ClamAV unhealthy" | Run `freshclam` to update definitions |
| "NSQ connection error" | Check NSQ services are running |

## Tips & Tricks

### Re-update ClamAV definitions
```bash
docker-compose exec clamav freshclam
```

### Create test file
```bash
# EICAR test file (safe to use for testing)
echo 'X5O!P%@AP[4\PZX54(P^)7CC)7}$EICAR-STANDARD-ANTIVIRUS-TEST-FILE!$H+H*' > eicar.txt
docker cp eicar.txt threat-scan-service:/uploads/
```

### Monitor disk usage
```bash
docker system df
docker exec threat-scan-service df -h
```

### Check memory usage
```bash
docker stats threat-scan-service
```

## Deployment Quick Links

- **Quick Start**: See README.md
- **API Docs**: See docs/API.md
- **Deployment**: See docs/DEPLOYMENT.md
- **Build/Deploy**: See docs/BUILD_DEPLOY.md
- **NSQ Examples**: See docs/NSQ_EXAMPLE.md
