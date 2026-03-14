# API Documentation

## gRPC API

The threat-scan service provides a gRPC API for file scanning.

### Service Definition

```protobuf
service ScanService {
  rpc Scan(ScanRequest) returns (ScanResponse);
}
```

### Messages

#### ScanRequest

```protobuf
message ScanRequest {
  string sha256 = 1;      // SHA256 hash of the file
  string filepath = 2;    // Path to the file (relative to mounted volume)
  string filename = 3;    // Original filename
}
```

#### ScanResponse

```protobuf
message ScanResponse {
  string status = 1;                // "clean", "infected", "error"
  repeated ScanResult results = 2;  // Results from each AV engine
  string error_message = 3;         // Error details if status is "error"
}
```

#### ScanResult

```protobuf
message ScanResult {
  string engine = 1;      // "clamav", "comodo"
  string status = 2;      // "clean", "infected", "error"
  string detection = 3;   // Malware name if infected
  string details = 4;     // Additional details
}
```

### Status Codes

- **clean**: File is clean according to all AV engines
- **infected**: At least one AV engine detected a threat
- **error**: Scanning failed or file validation failed

### Example Usage

#### Using grpcurl

```bash
# Simple test
grpcurl -plaintext \
  -d '{"sha256":"test123","filepath":"malware.exe","filename":"malware.exe"}' \
  localhost:50051 \
  scan.ScanService/Scan

# Pretty print result
grpcurl -plaintext \
  -d '{"sha256":"test123","filepath":"file.exe","filename":"file.exe"}' \
  localhost:50051 \
  scan.ScanService/Scan | jq .
```

#### Using Go Client

```go
package main

import (
	"context"
	"log"

	pb "github.com/threat-scan/pb/scan"
	"google.golang.org/grpc"
)

func main() {
	conn, _ := grpc.Dial("localhost:50051", grpc.WithInsecure())
	defer conn.Close()

	client := pb.NewScanServiceClient(conn)

	reply, err := client.Scan(context.Background(), &pb.ScanRequest{
		Sha256:   "abc123",
		Filepath: "file.exe",
		Filename: "file.exe",
	})

	if err != nil {
		log.Fatalf("Error: %v", err)
	}

	log.Printf("Status: %s\n", reply.Status)
	for _, result := range reply.Results {
		log.Printf("  %s: %s\n", result.Engine, result.Status)
	}
}
```

#### Using Python Client (with grpcio-tools)

```python
import grpc
from scan_pb2 import ScanRequest
from scan_pb2_grpc import ScanServiceStub

channel = grpc.insecure_channel('localhost:50051')
stub = ScanServiceStub(channel)

request = ScanRequest(
    sha256='abc123',
    filepath='file.exe',
    filename='file.exe'
)

response = stub.Scan(request)
print(f"Status: {response.status}")
for result in response.results:
    print(f"  {result.engine}: {result.status}")
```

## NSQ API

### Publishing Scan Requests

**Topic**: `threat_scan`

**Message Format**:
```json
{
  "sha256": "hash_value",
  "filepath": "path/to/file.exe",
  "filename": "file.exe",
  "id": "unique-request-id"
}
```

**Example with curl**:
```bash
curl -X POST http://localhost:4151/pub?topic=threat_scan \
  -d @- << 'EOF'
{
  "sha256": "abc123",
  "filepath": "malware.exe",
  "filename": "malware.exe",
  "id": "request-001"
}
EOF
```

### Consuming Scan Results

**Topic**: `threat_scan_results`

**Result Format**:
```json
{
  "id": "request-001",
  "sha256": "abc123",
  "filepath": "malware.exe",
  "filename": "malware.exe",
  "status": "infected",
  "results": [
    {
      "engine": "clamav",
      "status": "infected",
      "detection": "Win.Malware.Generic",
      "details": "Detected malware pattern"
    }
  ],
  "timestamp": 1234567890
}
```

**Go Consumer Example**:
```go
package main

import (
	"encoding/json"
	"log"

	nsq "github.com/nsqio/go-nsq"
)

type ResultMessage struct {
	ID       string `json:"id"`
	SHA256   string `json:"sha256"`
	Status   string `json:"status"`
	Results  []interface{} `json:"results"`
}

func main() {
	config := nsq.NewConfig()
	consumer, _ := nsq.NewConsumer("threat_scan_results", "my_handler", config)

	consumer.AddHandler(nsq.HandlerFunc(func(message *nsq.Message) error {
		var result ResultMessage
		json.Unmarshal(message.Body, &result)
		
		log.Printf("ID: %s, Status: %s", result.ID, result.Status)
		return nil
	}))

	consumer.ConnectToNSQLookupd("nsqlookupd:4161")
	select {}
}
```

## Error Handling

### gRPC Errors

The service returns appropriate gRPC status codes:

```
Code.OK (0)              - Success
Code.INVALID_ARGUMENT    - Invalid request (missing required fields)
Code.NOT_FOUND           - File not found
Code.INTERNAL            - Internal server error
Code.UNAVAILABLE         - Service unavailable (AV engines down)
Code.DEADLINE_EXCEEDED   - Scan timeout
```

Example error response:
```json
{
  "status": "error",
  "error_message": "file not found: /uploads/malware.exe"
}
```

### NSQ Error Handling

Failed messages are retried based on NSQ configuration. If still failing, check:

1. File exists in upload volume
2. AV engine is healthy
3. Disk space available
4. Look at service logs

## Rate Limiting

### gRPC Concurrency

Maximum concurrent scans controlled by:
```yaml
server:
  max_concurrent_scans: 10  # Use semaphore
```

Requests beyond limit will wait. No rejection occurs.

### NSQ Flow Control

Message flow controlled by:
```yaml
nsq:
  max_in_flight: 100  # Messages in process
```

## Timeouts

### Request Timeouts

Set via context in client:
```go
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
defer cancel()
resp, err := client.Scan(ctx, req)
```

### Service Timeouts

Configured in config.yaml:
```yaml
av_engines:
  clamav:
    timeout: 60  # seconds per engine
scanning:
  scan_timeout: 300  # seconds total
```

## Monitoring

### Health Check

```bash
# Check if service is responding
grpcurl -plaintext localhost:50051 grpc.health.v1.Health/Check

# List available services
grpcurl -plaintext localhost:50051 list
```

### Metrics

Access Prometheus metrics (when integrated):
```
http://localhost:9090/metrics
```

Key metrics:
- `scan_duration_seconds` - Scan execution time
- `scan_errors_total` - Total scan errors
- `file_scanned_total` - Total files scanned
- `av_engine_status` - AV engine health

### NSQ Monitoring

Admin UI: http://localhost:4171

View in UI:
- Message rates
- Consumer lag
- Queue depth
- Topic/channel stats
