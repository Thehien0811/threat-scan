package service

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net"
	"os"
	"strings"
	"time"
)

// ClamAVScanner implements Scanner interface for ClamAV
type ClamAVScanner struct {
	host    string
	port    int
	timeout time.Duration
}

// NewClamAVScanner creates a new ClamAV scanner
func NewClamAVScanner(host string, port int, timeout time.Duration) *ClamAVScanner {
	return &ClamAVScanner{
		host:    host,
		port:    port,
		timeout: timeout,
	}
}

// Scan scans a file with ClamAV
func (c *ClamAVScanner) Scan(ctx context.Context, filePath string) (*ScanResult, error) {
	// Create connection with timeout
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("%s:%d", c.host, c.port), c.timeout)
	if err != nil {
		return nil, fmt.Errorf("clamav connection failed: %w", err)
	}
	defer conn.Close()

	// Set read/write deadline
	deadline := time.Now().Add(c.timeout)
	conn.SetDeadline(deadline)

	// Send INSTREAM command for file scanning
	cmd := fmt.Sprintf("INSTREAM\n")
	if _, err := conn.Write([]byte(cmd)); err != nil {
		return nil, fmt.Errorf("clamav write command failed: %w", err)
	}

	// Read file and send in chunks
	if err := sendFileContent(conn, filePath); err != nil {
		return nil, fmt.Errorf("clamav file send failed: %w", err)
	}

	// Read response
	scanner := bufio.NewScanner(conn)
	if !scanner.Scan() {
		return nil, fmt.Errorf("clamav no response")
	}

	response := scanner.Text()
	return c.parseResponse(response), nil
}

// Health checks ClamAV connectivity
func (c *ClamAVScanner) Health(ctx context.Context) error {
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("%s:%d", c.host, c.port), c.timeout)
	if err != nil {
		return fmt.Errorf("clamav health check failed: %w", err)
	}
	defer conn.Close()

	conn.SetDeadline(time.Now().Add(c.timeout))

	// Send PING command
	if _, err := conn.Write([]byte("PING\n")); err != nil {
		return fmt.Errorf("clamav ping failed: %w", err)
	}

	scanner := bufio.NewScanner(conn)
	if !scanner.Scan() {
		return fmt.Errorf("clamav ping no response")
	}

	if strings.Contains(scanner.Text(), "PONG") {
		return nil
	}

	return fmt.Errorf("clamav unhealthy response")
}

// parseResponse parses ClamAV response
func (c *ClamAVScanner) parseResponse(response string) *ScanResult {
	result := &ScanResult{
		Engine: "clamav",
		Status: "clean",
	}

	// ClamAV response format: "stream: <status> <detection>"
	// Clean: "stream: OK"
	// Infected: "stream: Eicar-Test-File FOUND"

	parts := strings.Split(response, ":")
	if len(parts) < 2 {
		result.Status = "error"
		result.Details = "invalid response format"
		return result
	}

	content := strings.TrimSpace(parts[1])

	if strings.Contains(content, "FOUND") {
		result.Status = "infected"
		// Extract detection name
		detectionParts := strings.Split(content, " ")
		if len(detectionParts) > 0 {
			result.Detection = detectionParts[0]
		}
		result.Details = content
	} else if strings.Contains(content, "OK") {
		result.Status = "clean"
	} else {
		result.Status = "error"
		result.Details = content
	}

	return result
}

// sendFileContent sends file content to ClamAV in chunks
func sendFileContent(conn net.Conn, filePath string) error {
	file, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	const chunkSize = 32768 // 32KB chunks
	buffer := make([]byte, chunkSize)

	for {
		n, err := file.Read(buffer)
		if err != nil && err != io.EOF {
			return err
		}

		if n == 0 {
			break
		}

		// Send chunk size (4 bytes big-endian)
		size := uint32(n)
		sizeBytes := []byte{byte(size >> 24), byte(size >> 16), byte(size >> 8), byte(size)}
		if _, err := conn.Write(sizeBytes); err != nil {
			return err
		}

		// Send chunk data
		if _, err := conn.Write(buffer[:n]); err != nil {
			return err
		}
	}

	// Send terminating chunk (0 bytes)
	if _, err := conn.Write([]byte{0, 0, 0, 0}); err != nil {
		return err
	}

	return nil
}
