# Architecture Overview

## System Components

### 1. Go Service (threat-scan-service)
- **Language**: Go 1.21
- **Port**: 50051 (gRPC)
- **Responsibilities**:
  - gRPC API endpoint for synchronous scanning
  - NSQ message consumer for async scanning
  - Multi-AV scanner coordination
  - Request validation and file management

### 2. ClamAV Container
- **Image**: clamav/clamav:latest
- **Port**: 3310 (CLAMD protocol)
- **Responsibilities**:
  - File signature-based malware scanning
  - Virus definition database management
  - Real-time threat detection

### 3. NSQ Infrastructure
- **NSQLookupd**: Service discovery
- **NSQd**: Message queue broker
- **NSQ Admin UI**: Monitoring dashboard

### 4. Shared Storage
- **Volume**: /uploads
- **Purpose**: Store files for scanning
- **Mounted in**: Service and ClamAV containers

## Data Flow

### Synchronous gRPC Flow

```
┌─────────────────────────┐
│  Client gRPC Request    │
│ {sha256, filepath, ...} │
└────────────┬────────────┘
             │
             ▼
┌─────────────────────────────────────────┐
│  threat-scan-service                    │
│  1. Validate request                    │
│  2. Verify file exists                  │
│  3. Spawn scan tasks                    │
└────────────┬────────────────────────────┘
             │
    ┌────────┴────────┐
    │                 │
    ▼                 ▼
┌──────────┐   ┌──────────────┐
│ ClamAV   │   │ Comodo (TBD) │
│ Scan     │   │ Scan         │
└────┬─────┘   └──────┬───────┘
     │                │
     └────────┬───────┘
              │
              ▼
    ┌─────────────────────┐
    │ Aggregate Results   │
    │ - Status (clean/    │
    │   infected/error)   │
    │ - Per-engine details│
    └────────┬────────────┘
             │
             ▼
    ┌──────────────────────┐
    │ gRPC Response        │
    │ Return to Client     │
    └──────────────────────┘
```

### Asynchronous NSQ Flow

```
┌──────────────────────────┐
│ Publisher                │
│ Publishes ScanMessage    │
│ to threat_scan topic     │
└────────────┬─────────────┘
             │
             ▼
    ┌────────────────────────┐
    │    NSQd (Topic)        │
    │   threat_scan topic    │
    └────────────┬───────────┘
                 │
    ┌────────────▼────────────┐
    │   NSQ Consumer          │
    │ threat-scan-service     │
    │ channel: scanner        │
    └────────┬────────────────┘
             │
             ├─ Validate
             ├─ Scan File
             └─ Aggregate Results
             │
             ▼
    ┌─────────────────────────┐
    │ NSQd (Results Topic)    │
    │threat_scan_results topic│
    └─────────────────────────┘
             │
             ▼
    ┌─────────────────────────┐
    │   Result Consumer       │
    │  (Your Application)     │
    │ Processes Results       │
    └─────────────────────────┘
```

## Container Communication

```
┌──────────────────────────────────────────────────────────────┐
│                     Docker Network                            │
│                   (threat-scan-net)                           │
│                                                               │
│  ┌──────────────────────┐    ┌──────────────────────┐       │
│  │  threat-scan-service │    │     clamav           │       │
│  │    :50051            │◄───►│     :3310            │       │
│  │ (gRPC)               │    │  (CLAMD Protocol)    │       │
│  │ (NSQ Consumer)       │    │                      │       │
│  └──────┬───────────────┘    └──────────────────────┘       │
│         │                                                    │
│  ┌──────▼──────────────────────────────────────────┐        │
│  │           NSQ Infrastructure                    │        │
│  │  ┌──────────────┐  ┌────────────┐              │        │
│  │  │ nsqlookupd   │  │   nsqd     │              │        │
│  │  │  :4160/4161  │  │ :4150/4151 │              │        │
│  │  └──────────────┘  └────────────┘              │        │
│  └───────────────────────────────────────────────┘        │
│                                                               │
│  ┌──────────────────────────────────────────────┐           │
│  │    Shared Volume: /uploads                   │           │
│  │  ┌────────────────────────────────────────┐ │           │
│  │  │ File Storage                           │ │           │
│  │  │ - Mounted to service                   │ │           │
│  │  │ - Mounted to ClamAV                    │ │           │
│  │  └────────────────────────────────────────┘ │           │
│  └──────────────────────────────────────────────┘           │
│                                                               │
└──────────────────────────────────────────────────────────────┘
```

## Configuration Hierarchy

```
config/config.yaml
├── Server Config
│   ├── gRPC Port
│   └── Max Concurrent Scans
├── NSQ Config
│   ├── NSQd Addresses
│   ├── Topic/Channel
│   └── Max In Flight
├── AV Engines Config
│   ├── ClamAV
│   │   ├── Enabled
│   │   ├── Host/Port
│   │   └── Timeout
│   └── Comodo
│       ├── Enabled
│       ├── Host/Port
│       └── Timeout
├── Scanning Config
│   ├── Upload Path
│   ├── Max File Size
│   └── Scan Timeout
└── Logging Config
    ├── Level
    └── Format
```

## Request Processing Pipeline

```
1. Client Request (gRPC)
   ↓
2. Validate Request
   - SHA256 not empty
   - FilePath not empty
   - Filename provided
   ↓
3. Validate File
   - File exists
   - File size < maxSize
   - Not a directory
   ↓
4. Acquire Semaphore
   - Wait if at max concurrent scans
   ↓
5. Spawn Scan Tasks
   - ClamAV scan (async)
   - Comodo scan (async) [if enabled]
   - Each with timeout
   ↓
6. Collect Results
   - Wait for all engines
   - Timeout enforcement
   ↓
7. Aggregate Results
   - Determine overall status
   - clean: All engines clean
   - infected: Any engine infected
   - error: All scans failed
   ↓
8. Return Response
   - Status + engine results
   - Error details if applicable
```

## Failure Scenarios

### ClamAV Down
- gRPC requests fail with error status
- Result shows ClamAV unavailable
- NSQ messages requeued/retried

### NSQ Down
- gRPC API works normally
- Async scanning queues pause
- Messages preserved in queue
- Resumes when NSQ available

### File Not Found
- Validation fails immediately
- Returns error response
- No scan attempted

### Scan Timeout
- Per-engine timeout (60s configurable)
- Overall timeout (300s configurable)
- Partial results returned if available

### Disk Full
- File read fails
- Returns error status
- No partial results

## Scaling Considerations

### Horizontal Scaling
- Run multiple service replicas
- Each connects to same NSQd
- Load balance gRPC requests
- NSQ handles message distribution

### Vertical Scaling
- Increase max_concurrent_scans
- Increase NSQ max_in_flight
- Allocate more resources to ClamAV

### Bottlenecks
- ClamAV CPU: Limit concurrent scans
- Disk I/O: Use fast storage
- Network: Ensure sufficient bandwidth
- Memory: Monitor ClamAV process

## Security Architecture

```
┌─────────────────────────────────┐
│  External Client                │
│  (Untrusted)                    │
└────────────┬────────────────────┘
             │
             │ gRPC (TLS optional)
             ▼
┌─────────────────────────────────┐
│  Network Boundary               │
└────────────┬────────────────────┘
             │
             ▼
┌─────────────────────────────────┐
│  gRPC Server                    │
│  - Input validation             │
│  - Request rate limiting        │
│  - Authentication (future)      │
└────────────┬────────────────────┘
             │
             │ Docker Network (isolated)
             ▼
┌─────────────────────────────────┐
│  Container Sandbox              │
│  - No-new-privileges            │
│  - Read-only root               │
│  - Limited capabilities         │
└────────────┬────────────────────┘
             │
             ▼
┌─────────────────────────────────┐
│  File Scanner                   │
│  - ClamAV read-only access      │
│  - File validation              │
│  - Sandboxed execution          │
└─────────────────────────────────┘
```

## Monitoring Strategy

```
┌─────────────────────────────────────────┐
│     Application Metrics                 │
│  - Scan duration (histogram)            │
│  - Scan errors (counter)                │
│  - File scanned (counter)               │
│  - AV engine status (gauge)             │
└────────────┬────────────────────────────┘
             │
    ┌────────▼────────┐
    │                 │
    ▼                 ▼
┌────────────┐   ┌────────────────┐
│ Prometheus │   │  Custom Logs   │
│  (Metrics) │   │  (JSON format) │
└──────┬─────┘   └────────┬───────┘
       │                  │
       └────────┬─────────┘
                ▼
    ┌─────────────────────┐
    │    Observability    │
    │  - Dashboards       │
    │  - Alerting         │
    │  - Tracing          │
    └─────────────────────┘
```

## Technology Stack

| Layer | Technology | Purpose |
|-------|-----------|---------|
| API | gRPC | Sync scanning requests |
| Messaging | NSQ | Async queue processing |
| Language | Go 1.21 | Service implementation |
| Scanning | ClamAV | Malware detection |
| Scanning | Comodo | Alternative AV engine |
| Containerization | Docker | Service isolation |
| Orchestration | Docker Compose | Container management |
| Configuration | YAML | Service configuration |
| Protocol Buffers | protoc | Message serialization |
| Storage | Volume Mount | File access |

## Performance Characteristics

- **Throughput**: 10-50 files/sec (depends on file size)
- **Latency**: 100ms-5s per file (ClamAV timeout dependent)
- **Concurrency**: 10 concurrent scans (configurable)
- **Memory**: ~200MB idle, ~2GB active
- **CPU**: 1-4 cores (scales with concurrency)
