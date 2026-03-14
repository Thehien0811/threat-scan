package service

import (
	"context"
	"fmt"
	"log"
	"net"
	"path/filepath"
	"time"

	pb "github.com/threat-scan/proto"
	"google.golang.org/grpc"
)

// GRPCServer implements the ScanService gRPC interface
type GRPCServer struct {
	pb.UnimplementedScanServiceServer
	scanner    *MultiScanner
	uploadPath string
	maxWorkers int
	semaphore  chan struct{}
}

// NewGRPCServer creates a new gRPC server
func NewGRPCServer(scanner *MultiScanner, uploadPath string, maxWorkers int) *GRPCServer {
	return &GRPCServer{
		scanner:    scanner,
		uploadPath: uploadPath,
		maxWorkers: maxWorkers,
		semaphore:  make(chan struct{}, maxWorkers),
	}
}

// Scan implements the Scan RPC method
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

	// Set scan timeout
	scanCtx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()

	// Perform scan
	results, err := s.scanner.Scan(scanCtx, fullPath)
	if err != nil {
		return &pb.ScanResponse{
			Status:       "error",
			ErrorMessage: fmt.Sprintf("Scan failed: %v", err),
		}, nil
	}

	// Convert results
	pbResults := make([]*pb.ScanResult, len(results))
	overallStatus := "clean"

	for i, result := range results {
		pbResults[i] = &pb.ScanResult{
			Engine:    result.Engine,
			Status:    result.Status,
			Detection: result.Detection,
			Details:   result.Details,
		}

		if result.Status == "infected" {
			overallStatus = "infected"
		} else if result.Status == "error" && overallStatus == "clean" {
			overallStatus = "error"
		}
	}

	log.Printf("Scan completed for %s: %s", req.Filename, overallStatus)

	return &pb.ScanResponse{
		Status:  overallStatus,
		Results: pbResults,
	}, nil
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
