package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"path/filepath"

	nsq "github.com/nsqio/go-nsq"
	pb "github.com/threat-scan/proto"
	"google.golang.org/grpc"
)

// ScanRequestMessage for NSQ publishing
type ScanRequestMessage struct {
	SHA256   string `json:"sha256"`
	FilePath string `json:"filepath"`
	FileName string `json:"filename"`
	ID       string `json:"id"`
}

// GRPCServer implements the ScanService gRPC interface
type GRPCServer struct {
	pb.UnimplementedScanServiceServer
	producer       *nsq.Producer
	resultConsumer *nsq.Consumer
	uploadPath     string
	maxWorkers     int
	semaphore      chan struct{}
	resultChans    map[string]chan *pb.ScanResponse
}

// NewGRPCServer creates a new gRPC server with NSQ producer
func NewGRPCServer(producer *nsq.Producer, nsqdAddr, uploadPath string, maxWorkers int) *GRPCServer {
	config := nsq.NewConfig()
	consumer, err := nsq.NewConsumer("threat_scan_results", "grpc_server", config)
	if err != nil {
		log.Fatalf("Failed to create NSQ consumer: %v", err)
	}
	server := &GRPCServer{
		producer:       producer,
		resultConsumer: consumer,
		uploadPath:     uploadPath,
		maxWorkers:     maxWorkers,
		semaphore:      make(chan struct{}, maxWorkers),
		resultChans:    make(map[string]chan *pb.ScanResponse),
	}
	consumer.AddHandler(nsq.HandlerFunc(server.handleResultMessage))
	if err := consumer.ConnectToNSQD(nsqdAddr); err != nil {
		log.Fatalf("Failed to connect result consumer to NSQD: %v", err)
	}
	return server
}

// Scan implements the Scan RPC method - publishes to NSQ instead of scanning directly
func (s *GRPCServer) Scan(ctx context.Context, req *pb.ScanRequest) (*pb.ScanResponse, error) {
	log.Printf("Received scan request for file: %s (SHA256: %s)", req.Filename, req.Sha256)

	// Acquire semaphore
	select {
	case s.semaphore <- struct{}{}:
		defer func() { <-s.semaphore }()
	case <-ctx.Done():
		return nil, fmt.Errorf("context cancelled before acquiring semaphore")
	}

	// Validate request
	if req.Sha256 == "" {
		return &pb.ScanResponse{
			Status:       "error",
			ErrorMessage: "SHA256 is required",
		}, nil
	}

	if req.Filepath == "" {
		return &pb.ScanResponse{
			Status:       "error",
			ErrorMessage: "Filepath is required",
		}, nil
	}

	// Construct full file path
	fullPath := filepath.Join(s.uploadPath, req.Filepath)

	// Validate file exists and size
	if err := ValidateFile(fullPath, 100*1024*1024); err != nil {
		return &pb.ScanResponse{
			Status:       "error",
			ErrorMessage: fmt.Sprintf("File validation failed: %v", err),
		}, nil
	}

	// Create scan request message
	scanMsg := ScanRequestMessage{
		SHA256:   req.Sha256,
		FilePath: req.Filepath,
		FileName: req.Filename,
		ID:       req.Sha256, // Use SHA256 as unique ID
	}

	// Marshal to JSON
	msgJSON, err := json.Marshal(scanMsg)
	if err != nil {
		return &pb.ScanResponse{
			Status:       "error",
			ErrorMessage: fmt.Sprintf("Failed to marshal message: %v", err),
		}, nil
	}

	// Prepare to receive result
	resultChan := make(chan *pb.ScanResponse, 1)
	s.resultChans[req.Sha256] = resultChan
	defer delete(s.resultChans, req.Sha256)

	// Publish to NSQ
	if err := s.producer.Publish("threat_scan", msgJSON); err != nil {
		return &pb.ScanResponse{
			Status:       "error",
			ErrorMessage: fmt.Sprintf("Failed to publish scan request: %v", err),
		}, nil
	}
	log.Printf("Published scan request for %s to NSQ topic: threat_scan", req.Filename)

	// Wait for result or context timeout
	select {
	case res := <-resultChan:
		return res, nil
	case <-ctx.Done():
		return &pb.ScanResponse{
			Status:       "error",
			ErrorMessage: "Timed out waiting for scan result",
		}, nil
	}
}

// handleResultMessage handles incoming scan results from NSQ
func (s *GRPCServer) handleResultMessage(msg *nsq.Message) error {
	var resultMsg struct {
		ID      string           `json:"id"`
		Status  string           `json:"status"`
		Results []*pb.ScanResult `json:"results"`
		Error   string           `json:"error"`
	}
	if err := json.Unmarshal(msg.Body, &resultMsg); err != nil {
		log.Printf("Failed to unmarshal scan result: %v", err)
		return err
	}
	ch, ok := s.resultChans[resultMsg.ID]
	if ok {
		ch <- &pb.ScanResponse{
			Status:       resultMsg.Status,
			Results:      resultMsg.Results,
			ErrorMessage: resultMsg.Error,
		}
	}
	return nil
}

// StartGRPCServer starts the gRPC server
func StartGRPCServer(port string, server *GRPCServer) error {
	lis, err := net.Listen("tcp", port)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %w", port, err)
	}

	s := grpc.NewServer()
	pb.RegisterScanServiceServer(s, server)

	log.Printf("Starting gRPC server on %s", port)
	return s.Serve(lis)
}
