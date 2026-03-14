package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

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

	AVEngines struct {
		ClamAV struct {
			Enabled bool          `yaml:"enabled"`
			Host    string        `yaml:"host"`
			Port    int           `yaml:"port"`
			Timeout time.Duration `yaml:"timeout"`
		} `yaml:"clamav"`
		Comodo struct {
			Enabled bool          `yaml:"enabled"`
			Host    string        `yaml:"host"`
			Port    int           `yaml:"port"`
			Timeout time.Duration `yaml:"timeout"`
		} `yaml:"comodo"`
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

func main() {
	configPath := flag.String("config", "/etc/threat-scan/config.yaml", "Path to config file")
	flag.Parse()

	// Load configuration
	config, err := loadConfig(*configPath)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Create multi-scanner
	scanner := service.NewMultiScanner()

	// Register ClamAV scanner
	if config.AVEngines.ClamAV.Enabled {
		clamav := service.NewClamAVScanner(
			config.AVEngines.ClamAV.Host,
			config.AVEngines.ClamAV.Port,
			config.AVEngines.ClamAV.Timeout,
		)
		scanner.RegisterScanner("clamav", clamav)
		log.Println("ClamAV scanner registered")
	}

	// TODO: Register Comodo scanner when implemented
	// if config.AVEngines.Comodo.Enabled {
	//     comodo := service.NewComodoScanner(...)
	//     scanner.RegisterScanner("comodo", comodo)
	// }

	// Check scanner health
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	health := scanner.CheckHealth(ctx)
	cancel()

	for name, err := range health {
		if err != nil {
			log.Printf("Warning: %s health check failed: %v", name, err)
		} else {
			log.Printf("%s is healthy", name)
		}
	}

	// Create gRPC server
	grpcServer := service.NewGRPCServer(
		scanner,
		config.Scanning.UploadPath,
		config.Server.MaxConcurrentScans,
	)

	// Create NSQ consumer
	nsqConsumer, err := service.NewNSQConsumer(
		config.NSQ.NSQDAddresses,
		config.NSQ.Topic,
		config.NSQ.Channel,
		scanner,
		config.Scanning.UploadPath,
	)
	if err != nil {
		log.Fatalf("Failed to create NSQ consumer: %v", err)
	}
	defer nsqConsumer.Close()
	log.Println("NSQ consumer started")

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
	log.Printf("NSQ topic: %s, channel: %s", config.NSQ.Topic, config.NSQ.Channel)

	// Wait for signal or error
	select {
	case sig := <-sigChan:
		log.Printf("Received signal: %v, shutting down...", sig)
		nsqConsumer.Close()
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
