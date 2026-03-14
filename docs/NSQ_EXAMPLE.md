# NSQ Consumer Example

Example of consuming scan results from NSQ.

```go
package main

import (
	"encoding/json"
	"log"

	nsq "github.com/nsqio/go-nsq"
)

type ScanResult struct {
	ID       string `json:"id"`
	Status   string `json:"status"`
	SHA256   string `json:"sha256"`
	Results  []interface{} `json:"results"`
}

func main() {
	config := nsq.NewConfig()
	consumer, _ := nsq.NewConsumer("threat_scan_results", "my_consumer", config)

	consumer.AddHandler(nsq.HandlerFunc(func(message *nsq.Message) error {
		var result ScanResult
		json.Unmarshal(message.Body, &result)
		
		log.Printf("Scan result for %s: %s", result.ID, result.Status)
		
		return nil
	}))

	consumer.ConnectToNSQLookupd("nsqlookupd:4161")
	
	select {}
}
```

## Publishing Scan Requests

```bash
# Publish a scan request to the threat_scan topic
curl -X POST http://localhost:4151/pub?topic=threat_scan \
  -d '{
    "sha256": "abc123...",
    "filepath": "file.exe",
    "filename": "file.exe",
    "id": "request-123"
  }'
```

## Monitoring with NSQ Admin

Open http://localhost:4171 to view:
- Active topics and channels
- Message rates
- Consumer lag
