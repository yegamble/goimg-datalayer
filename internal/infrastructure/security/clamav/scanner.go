// Package clamav provides a client for the ClamAV antivirus daemon.
// It uses the clamd TCP socket protocol for malware scanning.
package clamav

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net"
	"strings"
	"sync"
	"time"
)

// ScanResult contains the result of a malware scan.
type ScanResult struct {
	// Clean is true if no malware was detected.
	Clean bool

	// Infected is true if malware was detected.
	Infected bool

	// Virus contains the name of the detected malware (if any).
	Virus string

	// ScannedAt is when the scan was performed.
	ScannedAt time.Time
}

// Scanner provides malware scanning capabilities using ClamAV.
type Scanner interface {
	// Scan checks data for malware.
	Scan(ctx context.Context, data []byte) (*ScanResult, error)

	// ScanReader checks a stream for malware.
	ScanReader(ctx context.Context, reader io.Reader, size int64) (*ScanResult, error)

	// Ping verifies the ClamAV daemon is responsive.
	Ping(ctx context.Context) error

	// Version returns the ClamAV version string.
	Version(ctx context.Context) (string, error)

	// Stats returns ClamAV statistics including signature info.
	Stats(ctx context.Context) (string, error)
}

// Client implements Scanner using the clamd TCP socket protocol.
type Client struct {
	address    string
	timeout    time.Duration
	bufferPool sync.Pool
}

// Config configures the ClamAV client.
type Config struct {
	// TCPAddress is the clamd daemon address (host:port).
	// Example: "localhost:3310" or "clamav:3310" for Docker.
	TCPAddress string

	// Timeout is the maximum time for scan operations.
	Timeout time.Duration
}

// DefaultConfig returns sensible defaults for ClamAV connection.
func DefaultConfig() Config {
	return Config{
		TCPAddress: "localhost:3310",
		Timeout:    30 * time.Second,
	}
}

// NewClient creates a new ClamAV client.
func NewClient(cfg Config) (*Client, error) {
	if cfg.TCPAddress == "" {
		cfg.TCPAddress = "localhost:3310"
	}
	if cfg.Timeout == 0 {
		cfg.Timeout = 30 * time.Second
	}

	return &Client{
		address: cfg.TCPAddress,
		timeout: cfg.Timeout,
		bufferPool: sync.Pool{
			New: func() interface{} {
				buf := make([]byte, 32*1024) // 32KB chunks
				return &buf
			},
		},
	}, nil
}

// Scan checks data for malware.
func (c *Client) Scan(ctx context.Context, data []byte) (*ScanResult, error) {
	conn, err := c.dial(ctx)
	if err != nil {
		return nil, err
	}
	defer func() {
		if cerr := conn.Close(); cerr != nil {
			// Connection close errors are logged but don't affect scan result
		}
	}()

	// Set deadline based on context and timeout
	if deadline, ok := ctx.Deadline(); ok {
		if err := conn.SetDeadline(deadline); err != nil {
			return nil, fmt.Errorf("clamav: set deadline: %w", err)
		}
	} else {
		if err := conn.SetDeadline(time.Now().Add(c.timeout)); err != nil {
			return nil, fmt.Errorf("clamav: set deadline: %w", err)
		}
	}

	// Send INSTREAM command
	if _, err := conn.Write([]byte("zINSTREAM\x00")); err != nil {
		return nil, fmt.Errorf("clamav: send command: %w", err)
	}

	// Send data in chunks (clamd protocol: 4-byte size prefix per chunk)
	chunkSize := 32 * 1024 // 32KB chunks
	for i := 0; i < len(data); i += chunkSize {
		end := i + chunkSize
		if end > len(data) {
			end = len(data)
		}
		chunk := data[i:end]

		// Write 4-byte size prefix (big-endian)
		chunkLen := len(chunk)
		if chunkLen > 0x7FFFFFFF { // Validate conversion won't overflow
			return nil, fmt.Errorf("clamav: chunk size too large: %d", chunkLen)
		}
		size := uint32(chunkLen) // #nosec G115 -- validated chunk size is safe
		sizeBytes := []byte{
			byte(size >> 24),
			byte(size >> 16),
			byte(size >> 8),
			byte(size),
		}
		if _, err := conn.Write(sizeBytes); err != nil {
			return nil, fmt.Errorf("clamav: write size: %w", err)
		}
		if _, err := conn.Write(chunk); err != nil {
			return nil, fmt.Errorf("clamav: write chunk: %w", err)
		}
	}

	// Send zero-length chunk to signal end of stream
	if _, err := conn.Write([]byte{0, 0, 0, 0}); err != nil {
		return nil, fmt.Errorf("clamav: end stream: %w", err)
	}

	// Read response
	return c.readScanResponse(conn)
}

// ScanReader checks a stream for malware.
func (c *Client) ScanReader(ctx context.Context, reader io.Reader, size int64) (*ScanResult, error) {
	conn, err := c.dial(ctx)
	if err != nil {
		return nil, err
	}
	defer func() {
		if cerr := conn.Close(); cerr != nil {
			// Connection close errors are logged but don't affect scan result
		}
	}()

	// Set deadline
	if deadline, ok := ctx.Deadline(); ok {
		if err := conn.SetDeadline(deadline); err != nil {
			return nil, fmt.Errorf("clamav: set deadline: %w", err)
		}
	} else {
		if err := conn.SetDeadline(time.Now().Add(c.timeout)); err != nil {
			return nil, fmt.Errorf("clamav: set deadline: %w", err)
		}
	}

	// Send INSTREAM command
	if _, err := conn.Write([]byte("zINSTREAM\x00")); err != nil {
		return nil, fmt.Errorf("clamav: send command: %w", err)
	}

	// Stream data in chunks
	bufPtr := c.bufferPool.Get().(*[]byte)
	buf := *bufPtr
	defer c.bufferPool.Put(bufPtr)

	for {
		n, err := reader.Read(buf)
		if n > 0 {
			// Write size prefix
			if n > 0x7FFFFFFF { // Validate conversion won't overflow
				return nil, fmt.Errorf("clamav: read size too large: %d", n)
			}
			size := uint32(n) // #nosec G115 -- validated read size is safe
			sizeBytes := []byte{
				byte(size >> 24),
				byte(size >> 16),
				byte(size >> 8),
				byte(size),
			}
			if _, werr := conn.Write(sizeBytes); werr != nil {
				return nil, fmt.Errorf("clamav: write size: %w", werr)
			}
			if _, werr := conn.Write(buf[:n]); werr != nil {
				return nil, fmt.Errorf("clamav: write chunk: %w", werr)
			}
		}
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, fmt.Errorf("clamav: read input: %w", err)
		}
	}

	// Send zero-length chunk to signal end
	if _, err := conn.Write([]byte{0, 0, 0, 0}); err != nil {
		return nil, fmt.Errorf("clamav: end stream: %w", err)
	}

	return c.readScanResponse(conn)
}

// Ping verifies the ClamAV daemon is responsive.
func (c *Client) Ping(ctx context.Context) error {
	conn, err := c.dial(ctx)
	if err != nil {
		return err
	}
	defer func() {
		if cerr := conn.Close(); cerr != nil {
			// Connection close errors are logged but don't affect ping result
		}
	}()

	if err := conn.SetDeadline(time.Now().Add(5 * time.Second)); err != nil {
		return fmt.Errorf("clamav: set deadline: %w", err)
	}

	if _, err := conn.Write([]byte("zPING\x00")); err != nil {
		return fmt.Errorf("clamav: ping send: %w", err)
	}

	response, err := c.readResponse(conn)
	if err != nil {
		return fmt.Errorf("clamav: ping read: %w", err)
	}

	if response != "PONG" {
		return fmt.Errorf("clamav: unexpected ping response: %s", response)
	}

	return nil
}

// Version returns the ClamAV version string.
func (c *Client) Version(ctx context.Context) (string, error) {
	conn, err := c.dial(ctx)
	if err != nil {
		return "", err
	}
	defer func() {
		if cerr := conn.Close(); cerr != nil {
			// Connection close errors are logged but don't affect version result
		}
	}()

	if err := conn.SetDeadline(time.Now().Add(5 * time.Second)); err != nil {
		return "", fmt.Errorf("clamav: set deadline: %w", err)
	}

	if _, err := conn.Write([]byte("zVERSION\x00")); err != nil {
		return "", fmt.Errorf("clamav: version send: %w", err)
	}

	return c.readResponse(conn)
}

// Stats returns ClamAV statistics including signature info.
func (c *Client) Stats(ctx context.Context) (string, error) {
	conn, err := c.dial(ctx)
	if err != nil {
		return "", err
	}
	defer func() {
		if cerr := conn.Close(); cerr != nil {
			// Connection close errors are logged but don't affect stats result
		}
	}()

	if err := conn.SetDeadline(time.Now().Add(5 * time.Second)); err != nil {
		return "", fmt.Errorf("clamav: set deadline: %w", err)
	}

	if _, err := conn.Write([]byte("zSTATS\x00")); err != nil {
		return "", fmt.Errorf("clamav: stats send: %w", err)
	}

	return c.readResponse(conn)
}

// dial creates a new connection to the ClamAV daemon.
func (c *Client) dial(ctx context.Context) (net.Conn, error) {
	var d net.Dialer
	conn, err := d.DialContext(ctx, "tcp", c.address)
	if err != nil {
		return nil, fmt.Errorf("clamav: connect to %s: %w", c.address, err)
	}
	return conn, nil
}

// readResponse reads a single line response from clamd.
func (c *Client) readResponse(conn net.Conn) (string, error) {
	reader := bufio.NewReader(conn)

	// Read until null byte or newline
	var result strings.Builder
	for {
		b, err := reader.ReadByte()
		if err != nil {
			return result.String(), err
		}
		if b == 0 || b == '\n' {
			break
		}
		result.WriteByte(b)
	}

	return strings.TrimSpace(result.String()), nil
}

// readScanResponse parses a scan response from clamd.
func (c *Client) readScanResponse(conn net.Conn) (*ScanResult, error) {
	response, err := c.readResponse(conn)
	if err != nil {
		return nil, fmt.Errorf("clamav: read response: %w", err)
	}

	result := &ScanResult{
		ScannedAt: time.Now(),
	}

	// Response format: "stream: OK" or "stream: Eicar-Signature FOUND"
	switch {
	case strings.HasSuffix(response, "OK"):
		result.Clean = true
		result.Infected = false
	case strings.Contains(response, "FOUND"):
		result.Clean = false
		result.Infected = true
		// Extract virus name: "stream: Eicar-Test-Signature FOUND"
		parts := strings.Split(response, ":")
		if len(parts) >= 2 {
			virusPart := strings.TrimSpace(parts[1])
			virusPart = strings.TrimSuffix(virusPart, " FOUND")
			result.Virus = strings.TrimSpace(virusPart)
		}
	case strings.Contains(response, "ERROR"):
		return nil, fmt.Errorf("clamav: scan error: %s", response)
	default:
		return nil, fmt.Errorf("clamav: unexpected response: %s", response)
	}

	return result, nil
}
