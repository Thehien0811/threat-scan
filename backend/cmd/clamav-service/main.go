package main

import (
	"context"
	"encoding/json"
	"flag"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	nsq "github.com/nsqio/go-nsq"
	"github.com/threat-scan/service"
	"gopkg.in/yaml.v3"
)

type ConfigFile struct {
	NSQ struct {
		NSQDAddresses []string `yaml:"nsqd_addresses"`
		Topic         string   `yaml:"topic"`
		Channel       string   `yaml:"channel"`
		MaxInFlight   int      `yaml:"max_in_flight"`
	} `yaml:"nsq"`

	AVEngines struct {
		ClamAV struct {
			Enabled bool          `yaml:"enabled"`
			Host    string        `yaml:"host"`
			Port    int           `yaml:"port"`
			Timeout time.Duration `yaml:"timeout"`
		} `yaml:"clamav"`
	} `yaml:"av_engines"`

	Scanning struct {
		UploadPath  string `yaml:"upload_path"`
		MaxFileSize int64  `yaml:"max_file_size"`
		ScanTimeout int    `yaml:"scan_timeout"`
	} `yaml:"scanning"`

	Logging struct {
		Level  string `yaml:"level"`
		Format string `yaml:"format"`
	} `yaml:"logging"`
}

type ScanMessage struct {
	SHA256   string `json:"sha256"`
	FilePath string `json:"filepath"`
	FileName string `json:"filename"`
	ID       string `json:"id"`
}

type ScanResultMessage struct {
	ID        string               `json:"id"`
	SHA256    string               `json:"sha256"`
	FilePath  string               `json:"filepath"`
	FileName  string               `json:"filename"`
	Status    string               `json:"status"`
	Results   []*service.ScanResult `json:"results"`
	Error     string               `json:"error,omitempty"`
	Timestamp int64                `json:"timestamp"`
}

func main() {
	configPath := flag.String("config", "/etc/threat-scan/config.yaml", "Path to config file")
	flag.Parse()

	// Load configuration
	config, err := loadConfig(*configPath)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Create ClamAV scanner
	clamavScanner := service.NewClamAVScanner(
		config.AVEngines.ClamAV.Host,
		config.AVEngines.ClamAV.Port,
		config.AVEngines.ClamAV.Timeout,
	)

	// Check health
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	if err := clamavScanner.Health(ctx); err != nil {
		log.Printf("Warning: ClamAV health check failed: %v", err)
	} else {
		log.Println("ClamAV is healthy")
	}
	cancel()

	// Create NSQ consumer
	nsqConfig := nsq.NewConfig()
	nsqConfig.MaxInFlight = config.NSQ.MaxInFlight

	consumer, err := nsq.NewConsumer(config.NSQ.Topic, "clamav-consumer", nsqConfig)
	if err != nil {
		log.Fatalf("Failed to create NSQ consumer: %v", err)
	}

	// Create NSQ producer for results
	producer, err := nsq.NewProducer(config.NSQ.NSQDAddresses[0], nsq.NewConfig())
	if err != nil {
		log.Fatalf("Failed to create NSQ producer: %v", err)
	}

	handler := &ClamAVHandler{
		scanner:    clamavScanner,
		producer:   producer,
		uploadPath: config.Scanning.UploadPath,
	}

	consumer.AddHandler(handler)

	if err := consumer.ConnectToNSQDs(config.NSQ.NSQDAddresses); err != nil {
		log.Fatalf("Failed to connect to NSQDs: %v", err)
	}

	log.Println("ClamAV service started, listening for scan requests...")

	// Handle graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	<-sigChan
	log.Println("Shutting down...")

	consumer.Stop()
	<-consumer.StopChan

	if err := producer.Publish("scan-results", []byte(`{"status":"service_shutdown"}`)); err != nil {
		log.Printf("Failed to publish shutdown message: %v", err)
	}
	producer.Stop()
}

// ClamAVHandler implements nsq.Handler
type ClamAVHandler struct {
	scanner    *service.ClamAVScanner
	producer   *nsq.Producer
	uploadPath string
	mu         sync.Mutex
}

// HandleMessage processes scan messages from NSQ
func (h *ClamAVHandler) HandleMessage(msg *nsq.Message) error {
	var scanMsg ScanMessage
	if err := json.Unmarshal(msg.Body, &scanMsg); err != nil {
		log.Printf("Failed to unmarshal message: %v", err)
		return err
	}

	log.Printf("Processing scan request for file: %s (ID: %s)", scanMsg.FileName, scanMsg.ID)

	// Scan file
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	result, err := h.scanner.Scan(ctx, h.uploadPath+"/"+scanMsg.FilePath)
	cancel()

	// Prepare result message
	resultMsg := ScanResultMessage{
		ID:        scanMsg.ID,
		SHA256:    scanMsg.SHA256,
		FilePath:  scanMsg.FilePath,
		FileName:  scanMsg.FileName,
		Timestamp: time.Now().Unix(),
	}

	if err != nil {
		resultMsg.Status = "error"
		resultMsg.Error = err.Error()
		log.Printf("Scan error for %s: %v", scanMsg.FileName, err)
	} else {
		resultMsg.Results = []*service.ScanResult{result}
		resultMsg.Status = result.Status
		log.Printf("Scan completed for %s: %s", scanMsg.FileName, result.Status)
	}

	// Publish result to the correct topic and format for gRPC consumer
	resultJSON, err := json.Marshal(resultMsg)
	if err != nil {
		log.Printf("Failed to marshal result: %v", err)
		return err
	}

	if err := h.producer.Publish("threat_scan_results", resultJSON); err != nil {
		log.Printf("Failed to publish scan result: %v", err)
		return err
	}

	return nil
}

func loadConfig(path string) (*ConfigFile, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var config ConfigFile
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, err
	}

	return &config, nil
}
