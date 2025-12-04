// Package mocks provides mock implementations for security testing.
package mocks

import (
	"context"
	"io"
	"strings"
	"time"

	"github.com/yegamble/goimg-datalayer/internal/infrastructure/security/clamav"
)

// MockClamAVScanner is a mock implementation of clamav.Scanner for testing.
type MockClamAVScanner struct {
	// ScanFunc allows customizing scan behavior for specific tests
	ScanFunc func(ctx context.Context, data []byte) (*clamav.ScanResult, error)

	// ScanReaderFunc allows customizing stream scan behavior
	ScanReaderFunc func(ctx context.Context, reader io.Reader, size int64) (*clamav.ScanResult, error)

	// PingFunc allows customizing ping behavior
	PingFunc func(ctx context.Context) error

	// VersionFunc allows customizing version behavior
	VersionFunc func(ctx context.Context) (string, error)

	// StatsFunc allows customizing stats behavior
	StatsFunc func(ctx context.Context) (string, error)

	// Call tracking
	ScanCalls       int
	ScanReaderCalls int
	PingCalls       int
	VersionCalls    int
	StatsCalls      int
}

// NewMockClamAVScanner creates a new mock scanner with default behavior.
// By default, all scans return clean results.
func NewMockClamAVScanner() *MockClamAVScanner {
	return &MockClamAVScanner{
		ScanFunc: func(ctx context.Context, data []byte) (*clamav.ScanResult, error) {
			return &clamav.ScanResult{
				Clean:     true,
				Infected:  false,
				Virus:     "",
				ScannedAt: time.Now(),
			}, nil
		},
		ScanReaderFunc: func(ctx context.Context, reader io.Reader, size int64) (*clamav.ScanResult, error) {
			return &clamav.ScanResult{
				Clean:     true,
				Infected:  false,
				Virus:     "",
				ScannedAt: time.Now(),
			}, nil
		},
		PingFunc: func(ctx context.Context) error {
			return nil
		},
		VersionFunc: func(ctx context.Context) (string, error) {
			return "ClamAV 1.0.0/Mock", nil
		},
		StatsFunc: func(ctx context.Context) (string, error) {
			return "POOLS: 1\nSTATE: VALID\n", nil
		},
	}
}

// NewMalwareDetectingScanner creates a mock that detects malware in data containing "EICAR".
func NewMalwareDetectingScanner() *MockClamAVScanner {
	mock := NewMockClamAVScanner()
	mock.ScanFunc = func(ctx context.Context, data []byte) (*clamav.ScanResult, error) {
		// Detect EICAR test signature
		if strings.Contains(string(data), "EICAR-STANDARD-ANTIVIRUS-TEST-FILE") {
			return &clamav.ScanResult{
				Clean:     false,
				Infected:  true,
				Virus:     "Eicar-Signature",
				ScannedAt: time.Now(),
			}, nil
		}
		return &clamav.ScanResult{
			Clean:     true,
			Infected:  false,
			Virus:     "",
			ScannedAt: time.Now(),
		}, nil
	}
	return mock
}

// Scan implements clamav.Scanner.
func (m *MockClamAVScanner) Scan(ctx context.Context, data []byte) (*clamav.ScanResult, error) {
	m.ScanCalls++
	if m.ScanFunc != nil {
		return m.ScanFunc(ctx, data)
	}
	return &clamav.ScanResult{
		Clean:     true,
		Infected:  false,
		ScannedAt: time.Now(),
	}, nil
}

// ScanReader implements clamav.Scanner.
func (m *MockClamAVScanner) ScanReader(ctx context.Context, reader io.Reader, size int64) (*clamav.ScanResult, error) {
	m.ScanReaderCalls++
	if m.ScanReaderFunc != nil {
		return m.ScanReaderFunc(ctx, reader, size)
	}
	return &clamav.ScanResult{
		Clean:     true,
		Infected:  false,
		ScannedAt: time.Now(),
	}, nil
}

// Ping implements clamav.Scanner.
func (m *MockClamAVScanner) Ping(ctx context.Context) error {
	m.PingCalls++
	if m.PingFunc != nil {
		return m.PingFunc(ctx)
	}
	return nil
}

// Version implements clamav.Scanner.
func (m *MockClamAVScanner) Version(ctx context.Context) (string, error) {
	m.VersionCalls++
	if m.VersionFunc != nil {
		return m.VersionFunc(ctx)
	}
	return "ClamAV 1.0.0/Mock", nil
}

// Stats implements clamav.Scanner.
func (m *MockClamAVScanner) Stats(ctx context.Context) (string, error) {
	m.StatsCalls++
	if m.StatsFunc != nil {
		return m.StatsFunc(ctx)
	}
	return "POOLS: 1\nSTATE: VALID\n", nil
}

// Reset clears all call counters.
func (m *MockClamAVScanner) Reset() {
	m.ScanCalls = 0
	m.ScanReaderCalls = 0
	m.PingCalls = 0
	m.VersionCalls = 0
	m.StatsCalls = 0
}
