package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"path/filepath"
	"time"

	nsq "github.com/nsqio/go-nsq"
)

// ScanMessage represents a scan request from NSQ
type ScanMessage struct {
	SHA256   string `json:"sha256"`
	FilePath string `json:"filepath"`
	FileName string `json:"filename"`
	ID       string `json:"id"`
}

// ScanResultMessage represents a scan result to be published
type ScanResultMessage struct {
	ID        string       `json:"id"`
	SHA256    string       `json:"sha256"`
	FilePath  string       `json:"filepath"`
	FileName  string       `json:"filename"`
	Status    string       `json:"status"`
	Results   []ScanResult `json:"results"`
	Error     string       `json:"error,omitempty"`
	Timestamp int64        `json:"timestamp"`
}

// NSQConsumer handles NSQ message consumption
type NSQConsumer struct {
	consumer   *nsq.Consumer
	scanner    *MultiScanner
	producer   *nsq.Producer
	uploadPath string
}

// NewNSQConsumer creates a new NSQ consumer
func NewNSQConsumer(nsqdAddrs []string, topic, channel string, scanner *MultiScanner, uploadPath string) (*NSQConsumer, error) {
	config := nsq.NewConfig()
	config.MaxInFlight = 100

	consumer, err := nsq.NewConsumer(topic, channel, config)
	if err != nil {
		return nil, fmt.Errorf("failed to create NSQ consumer: %w", err)
	}

	producer, err := nsq.NewProducer(nsqdAddrs[0], nsq.NewConfig())
	if err != nil {
		return nil, fmt.Errorf("failed to create NSQ producer: %w", err)
	}

	nc := &NSQConsumer{
		consumer:   consumer,
		scanner:    scanner,
		producer:   producer,
		uploadPath: uploadPath,
	}

	consumer.AddHandler(nc)

	if err := consumer.ConnectToNSQDs(nsqdAddrs); err != nil {
		return nil, fmt.Errorf("failed to connect to NSQDs: %w", err)
	}

	return nc, nil
}

// HandleMessage processes incoming NSQ messages
func (nc *NSQConsumer) HandleMessage(msg *nsq.Message) error {
	var scanMsg ScanMessage
	if err := json.Unmarshal(msg.Body, &scanMsg); err != nil {
		log.Printf("Failed to unmarshal message: %v", err)
		return err
	}

	log.Printf("Processing scan request: %s for file: %s", scanMsg.ID, scanMsg.FilePath)

	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Second)
	defer cancel()

	// Construct full file path
	fullPath := filepath.Join(nc.uploadPath, scanMsg.FilePath)

	// Validate file
	if err := ValidateFile(fullPath, 100*1024*1024); err != nil {
		ncresultMsg := ScanResultMessage{
			ID:        scanMsg.ID,
			SHA256:    scanMsg.SHA256,
			FilePath:  scanMsg.FilePath,
			FileName:  scanMsg.FileName,
			Status:    "error",
			Error:     err.Error(),
			Timestamp: time.Now().Unix(),
		}
		nc.publishResult(resultMsg)
		return err
	}

	// Scan file
	results, err := nc.scanner.Scan(ctx, fullPath)
	if err != nil {
		resultMsg := ScanResultMessage{
			ID:        scanMsg.ID,
			SHA256:    scanMsg.SHA256,
			FilePath:  scanMsg.FilePath,
			FileName:  scanMsg.FileName,
			Status:    "error",
			Error:     err.Error(),
			Timestamp: time.Now().Unix(),
		}
		_ = nc.publishResult(resultMsg)
		return nil
	}

	// Determine overall status
	status := "clean"
	for _, result := range results {
		if result.Status == "infected" {
			status = "infected"
			break
		}
	}

	resultMsg := ScanResultMessage{
		ID:        scanMsg.ID,
		SHA256:    scanMsg.SHA256,
		FilePath:  scanMsg.FilePath,
		FileName:  scanMsg.FileName,
		Status:    status,
		Results:   results,
		Timestamp: time.Now().Unix(),
	}

	_ = nc.publishResult(resultMsg)
	log.Printf("Scan completed for %s: %s", scanMsg.ID, status)

	return nil
}

// publishResult publishes result to NSQ
func (nc *NSQConsumer) publishResult(resultMsg ScanResultMessage) error {
	body, err := json.Marshal(resultMsg)
	if err != nil {
		log.Printf("Failed to marshal result: %v", err)
		return err
	}

	err = nc.producer.Publish("threat_scan_results", body)
	if err != nil {
		log.Printf("Failed to publish result: %v", err)
		return err
	}

	return nil
}

// Close closes the consumer and producer
func (nc *NSQConsumer) Close() error {
	nc.consumer.Stop()
	nc.producer.Stop()
	return nil
}
