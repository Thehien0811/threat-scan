package main

import (
	"flag"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"

	nsq "github.com/nsqio/go-nsq"
	"github.com/threat-scan/service"
	"gopkg.in/yaml.v3"
)

type ConfigFile struct {
	Server struct {
		GRPCPort           string `yaml:"grpc_port"`
		MaxConcurrentScans int    `yaml:"max_concurrent_scans"`
	} `yaml:"server"`

	NSQ struct {
		NSQDAddresses []string `yaml:"nsqd_addresses"`
		Topic         string   `yaml:"topic"`
		Channel       string   `yaml:"channel"`
		MaxInFlight   int      `yaml:"max_in_flight"`
	} `yaml:"nsq"`

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

func main() {
	configPath := flag.String("config", "/etc/threat-scan/config.yaml", "Path to config file")
	flag.Parse()

	// Load configuration
	config, err := loadConfig(*configPath)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Create NSQ producer
	producer, err := nsq.NewProducer(config.NSQ.NSQDAddresses[0], nsq.NewConfig())
	if err != nil {
		log.Fatalf("Failed to create NSQ producer: %v", err)
	}
	defer producer.Stop()

	log.Printf("Connected to NSQ at %s", config.NSQ.NSQDAddresses[0])

	// Create gRPC server with NSQ producer
	grpcServer := service.NewGRPCServer(
		producer,
		config.NSQ.NSQDAddresses[0],
		config.Scanning.UploadPath,
		config.Server.MaxConcurrentScans,
	)

	// Start gRPC server in goroutine
	var wg sync.WaitGroup
	var serverErr error

	wg.Add(1)
	go func() {
		defer wg.Done()
		serverErr = service.StartGRPCServer(config.Server.GRPCPort, grpcServer)
	}()

	// Setup signal handler
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGTERM, syscall.SIGINT)

	log.Println("Threat-scan service started successfully")
	log.Printf("gRPC server listening on %s", config.Server.GRPCPort)
	log.Printf("Publishing scan requests to NSQ topic: %s", config.NSQ.Topic)

	// Wait for signal or error
	select {
	case sig := <-sigChan:
		log.Printf("Received signal: %v, shutting down...", sig)
	case err := <-func() <-chan error {
		ch := make(chan error)
		go func() { ch <- serverErr }()
		return ch
	}():
		if err != nil {
			log.Fatalf("Server error: %v", err)
		}
	}

	wg.Wait()
	log.Println("Threat-scan service stopped")
}

func loadConfig(path string) (*ConfigFile, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var config ConfigFile
	err = yaml.Unmarshal(data, &config)
	if err != nil {
		return nil, err
	}

	return &config, nil
}
