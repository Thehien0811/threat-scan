# Threat-Scan System

A multi-AV container-based file scanning system built with Go, gRPC, NSQ, ClamAV, and Comodo.

## Architecture

```
┌─────────────────────────────────────────────────────┐
│ Client (gRPC Request)                               │
└────────────────────┬────────────────────────────────┘
                     │
                     ▼
┌─────────────────────────────────────────────────────┐
│ Threat-Scan Service (Go)                            │
│ - gRPC Server (:50051)                              │
│ - NSQ Consumer (Topic: threat_scan)                 │
└────────────────────┬────────────────────────────────┘
                     │
          ┌──────────┴──────────┐
          │                     │
          ▼                     ▼
    ┌─────────────┐      ┌─────────────┐
    │  ClamAV     │      │   Comodo    │
    │  (Port 3310)│      │  (Port 3310)│
    └─────────────┘      └─────────────┘
          │                     │
          └──────────┬──────────┘
                     ▼
          ┌──────────────────────┐
          │ NSQ Results Topic    │
          │ (threat_scan_results)│
          └──────────────────────┘
```

## Features

- **Multi-AV Scanning**: Support for ClamAV and Comodo antivirus engines
- **gRPC Interface**: Modern gRPC API for synchronous scan requests
- **NSQ Integration**: Asynchronous message queue for background scanning
- **Docker Containerization**: All components run in isolated Docker containers
- **Concurrent Scanning**: Configurable max concurrent scans with semaphore
- **Health Checks**: Built-in health checks for all AV engines
- **File Validation**: SHA256 and filepath validation for requests
- **Timeout Management**: Configurable timeouts for scans and network operations

## Prerequisites

- Docker & Docker Compose
- Go 1.21+
- protoc (for gRPC code generation)
- grpcurl (optional, for testing)

## Project Structure

```
threat-scan/
├── cmd/
│   └── main.go                 # Service entry point
├── service/
│   ├── scanner.go              # Multi-scanner coordinator
│   ├── clamav.go               # ClamAV scanner implementation
│   ├── comodo.go               # Comodo scanner implementation
│   ├── nsq_consumer.go         # NSQ message consumer
│   └── grpc_server.go          # gRPC service implementation
├── proto/
│   └── scan.proto              # Protocol Buffer definitions
├── config/
│   └── config.yaml             # Service configuration
├── docker/
│   ├── Dockerfile.service      # Go service Docker image
│   ├── Dockerfile.clamav       # ClamAV Docker image
│   └── Dockerfile.comodo       # Comodo Docker image
├── docker-compose.yaml         # Docker Compose orchestration
├── go.mod                       # Go module file
└── Makefile                     # Build automation
```

## Quick Start

### 1. Build and Start Docker Containers

```bash
make docker-build
make docker-up
```

This will start:
- NSQLookupd (port 4160, 4161)
- NSQd (port 4150, 4151)
- NSQ Admin UI (port 4171)
- ClamAV (port 3310)
- Threat-Scan Service (port 50051)

### 2. Check Service Status

```bash
docker-compose ps
docker-compose logs threat-scan-service
```

### 3. Test with gRPC

```bash
# First, upload a file to the uploads volume
docker cp test-file.exe threat-scan-service:/uploads/

# Test the gRPC API
grpcurl -plaintext -d '{"sha256":"abcd1234...","filepath":"test-file.exe","filename":"test-file.exe"}' \
  localhost:50051 scan.ScanService/Scan
```

## Configuration

Edit `config/config.yaml` to customize:

```yaml
server:
  grpc_port: ":50051"
  max_concurrent_scans: 10

nsq:
  nsqd_addresses:
    - "nsqd:4150"
  topic: "threat_scan"
  channel: "scanner"

av_engines:
  clamav:
    enabled: true
    host: "clamav"
    port: 3310
    timeout: 60

scanning:
  upload_path: "/uploads"
  max_file_size: 104857600  # 100MB
```

## API Usage

### gRPC Request Format

```protobuf
message ScanRequest {
  string sha256 = 1;      // SHA256 hash of the file
  string filepath = 2;    // Path to the file (relative to /uploads)
  string filename = 3;    // Original filename
}
```

### gRPC Response Format

```protobuf
message ScanResponse {
  string status = 1;                // "clean", "infected", "error"
  repeated ScanResult results = 2;
  string error_message = 3;
}

message ScanResult {
  string engine = 1;      // "clamav", "comodo"
  string status = 2;      // "clean", "infected", "error"
  string detection = 3;   // Malware name if detected
  string details = 4;     // Additional details
}
```

### NSQ Message Format

**Publishing to `threat_scan` topic:**
```json
{
  "sha256": "hash123",
  "filepath": "suspicious.exe",
  "filename": "suspicious.exe",
  "id": "unique-request-id"
}
```

**Result on `threat_scan_results` topic:**
```json
{
  "id": "unique-request-id",
  "sha256": "hash123",
  "filepath": "suspicious.exe",
  "filename": "suspicious.exe",
  "status": "infected",
  "results": [
    {
      "engine": "clamav",
      "status": "infected",
      "detection": "Eicar-Test-File",
      "details": "stream: Eicar-Test-File FOUND"
    }
  ],
  "timestamp": 1234567890
}
```

## NSQ Admin UI

Access NSQ Admin UI at: http://localhost:4171

- View topics and channels
- Monitor message queues
- Debug consumer lag

## Building Locally

```bash
# Generate protobuf code
make generate

# Build the service
make build

# Run with local config
make run
```

## Logs

```bash
# View all logs
docker-compose logs

# View service logs only
docker-compose logs threat-scan-service

# Follow logs in real-time
docker-compose logs -f
```

## Testing

```bash
# Run unit tests
make test

# Test with eicar test file
docker cp /tmp/eicar.txt threat-scan-service:/uploads/

grpcurl -plaintext \
  -d '{"sha256":"0":"","filepath":"eicar.txt","filename":"eicar.txt"}' \
  localhost:50051 scan.ScanService/Scan
```

## Troubleshooting

### ClamAV not updating definitions

```bash
docker-compose exec clamav freshclam
```

### NSQ topics not created

Topics are created automatically when messages are published. Use NSQ Admin UI to verify.

### Service won't start

```bash
# Check logs
docker-compose logs threat-scan-service

# Verify ClamAV is healthy
docker-compose exec clamav clamdscan --version
```

### Increase verbosity

Set `LOG_LEVEL: debug` in docker-compose.yaml environment section.

## Extending with Comodo

The Comodo scanner is defined but not enabled by default. To enable:

1. Update `config/config.yaml` to enable Comodo
2. Implement `ComodoScanner` in `service/comodo.go`
3. Register it in `cmd/main.go`
4. Build and deploy

**Note**: Comodo requires appropriate licensing.

## Performance Tuning

### Increase concurrent scans

```yaml
server:
  max_concurrent_scans: 20  # Increase from 10
```

### Adjust timeouts

```yaml
av_engines:
  clamav:
    timeout: 120  # Increase from 60 for large files
```

### NSQ tuning

```yaml
nsq:
  max_in_flight: 200  # Process more messages in parallel
```

## Security Considerations

- Files are mounted read-only where possible
- gRPC can be secured with TLS/mTLS
- NSQ should run on isolated network
- Set appropriate file permissions on uploads volume
- Regularly update virus definitions

## Development

### Code Style

```bash
make fmt
make lint
```

### Adding New Scanners

1. Create new file `service/newav.go`
2. Implement `Scanner` interface
3. Create appropriate Docker image
4. Register in `cmd/main.go`
5. Update configuration

### Regenerating Protobuf

```bash
make generate
```

## License

To be determined

## Support

For issues and questions, please refer to the project documentation or contact the development team.
