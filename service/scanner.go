package service

import (
	"context"
	"fmt"
	"os"
	"time"
)

// ScanResult holds the result from a single AV engine
type ScanResult struct {
	Engine    string
	Status    string // "clean", "infected", "error"
	Detection string
	Details   string
}

// Scanner interface for AV engines
type Scanner interface {
	Scan(ctx context.Context, filepath string) (*ScanResult, error)
	Health(ctx context.Context) error
}

// MultiScanner coordinates scanning across multiple AV engines
type MultiScanner struct {
	scanners map[string]Scanner
}

// NewMultiScanner creates a new multi-scanner
func NewMultiScanner() *MultiScanner {
	return &MultiScanner{
		scanners: make(map[string]Scanner),
	}
}

// RegisterScanner registers an AV engine
func (m *MultiScanner) RegisterScanner(name string, scanner Scanner) {
	m.scanners[name] = scanner
}

// Scan scans a file with all registered AV engines
func (m *MultiScanner) Scan(ctx context.Context, filePath string) ([]ScanResult, error) {
	// Verify file exists
	if _, err := os.Stat(filePath); err != nil {
		return nil, fmt.Errorf("file not found: %w", err)
	}

	results := make([]ScanResult, 0)

	for name, scanner := range m.scanners {
		result, err := m.scanWithTimeout(ctx, scanner, filePath, time.Minute)
		if err != nil {
			results = append(results, ScanResult{
				Engine:  name,
				Status:  "error",
				Details: err.Error(),
			})
			continue
		}
		results = append(results, *result)
	}

	return results, nil
}

// scanWithTimeout scans with timeout
func (m *MultiScanner) scanWithTimeout(ctx context.Context, scanner Scanner, filePath string, timeout time.Duration) (*ScanResult, error) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	type resultType struct {
		result *ScanResult
		err    error
	}

	ch := make(chan resultType)
	go func() {
		result, err := scanner.Scan(ctx, filePath)
		ch <- resultType{result, err}
	}()

	select {
	case r := <-ch:
		return r.result, r.err
	case <-ctx.Done():
		return nil, fmt.Errorf("scan timeout: %w", ctx.Err())
	}
}

// CheckHealth verifies all scanners are healthy
func (m *MultiScanner) CheckHealth(ctx context.Context) map[string]error {
	health := make(map[string]error)

	for name, scanner := range m.scanners {
		if err := scanner.Health(ctx); err != nil {
			health[name] = err
		} else {
			health[name] = nil
		}
	}

	return health
}

// ValidateFile validates a file before scanning
func ValidateFile(filePath string, maxSize int64) error {
	info, err := os.Stat(filePath)
	if err != nil {
		return fmt.Errorf("file stat error: %w", err)
	}

	if info.IsDir() {
		return fmt.Errorf("path is a directory, not a file")
	}

	if info.Size() > maxSize {
		return fmt.Errorf("file size %d exceeds maximum %d", info.Size(), maxSize)
	}

	return nil
}
