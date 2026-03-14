package service

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// ClamAVScanner implements Scanner interface for ClamAV
type ClamAVScanner struct {
	timeout time.Duration
}

// NewClamAVScanner creates a new ClamAV scanner
func NewClamAVScanner(host string, port int, timeout time.Duration) *ClamAVScanner {
	return &ClamAVScanner{
		timeout: timeout,
	}
}

// Scan scans a file with ClamAV using clamdscan command
func (c *ClamAVScanner) Scan(ctx context.Context, filePath string) (*ScanResult, error) {
	// Get absolute path
	absPath, err := filepath.Abs(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to get absolute path: %w", err)
	}

	// Verify file exists
	if _, err := os.Stat(absPath); err != nil {
		return nil, fmt.Errorf("file not found: %w", err)
	}

	result := &ScanResult{
		Engine: "clamav",
	}

	// Create context with timeout
	scanCtx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	// Run clamdscan command with absolute path (requires clamd daemon)
	cmd := exec.CommandContext(scanCtx, "clamdscan", "--no-summary", absPath)
	output, err := cmd.CombinedOutput()
	outputStr := string(output)

	// Parse output
	if err != nil {
		fmt.Println(err)
		// Check if it's a context timeout
		if scanCtx.Err() == context.DeadlineExceeded {
			result.Status = "error"
			result.Details = "scan timeout"
			return result, nil
		}

		// Check for various error conditions
		if strings.Contains(outputStr, "Can't access file") || strings.Contains(outputStr, "Permission denied") {
			result.Status = "error"
			result.Details = "file access error: " + outputStr
			return result, nil
		}

		// Check if file is infected
		if strings.Contains(outputStr, "FOUND") {
			result.Status = "infected"
			result.Details = outputStr
			// Extract threat name
			for _, line := range strings.Split(outputStr, "\n") {
				if strings.Contains(line, "FOUND") {
					// Format: /path/to/file: ThreatName FOUND
					parts := strings.Split(line, ":")
					if len(parts) >= 2 {
						threatPart := strings.TrimSpace(parts[len(parts)-1])
						threatName := strings.TrimSuffix(threatPart, " FOUND")
						result.Detection = threatName
					}
					break
				}
			}
			return result, nil
		}

		// Other errors
		result.Status = "error"
		result.Details = fmt.Sprintf("scan failed: %v - %s", err, outputStr)
		return result, nil
	}

	// Success - file is clean
	result.Status = "clean"
	result.Details = "file is clean"
	return result, nil
}

// Health checks ClamAV daemon availability
func (c *ClamAVScanner) Health(ctx context.Context) error {
	scanCtx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	// Check if clamdscan can connect to clamd
	cmd := exec.CommandContext(scanCtx, "clamdscan", "--version")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("clamav health check failed: %w", err)
	}

	return nil
}
