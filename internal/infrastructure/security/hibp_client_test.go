package security

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHIBPClient_IsCompromised_PasswordFound(t *testing.T) {
	t.Parallel()

	// Arrange: Mock HIBP API that returns a list of hash suffixes
	// Test password: "password123" -> SHA-1: CBFDAC6008F9CAB4083784CBD1874F76618D2A97
	// Prefix: CBFDA
	// Suffix: C6008F9CAB4083784CBD1874F76618D2A97
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request
		assert.Equal(t, "/range/CBFDA", r.URL.Path)
		assert.Equal(t, hibpUserAgent, r.Header.Get("User-Agent"))

		// Return mock response with our suffix
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("C6008F9CAB4083784CBD1874F76618D2A97:12345\n" +
			"C6008F9CAB4083784CBD1874F76618D2A98:6789\n" +
			"C6008F9CAB4083784CBD1874F76618D2A99:1\n"))
	}))
	defer server.Close()

	client := &HIBPClient{
		httpClient: &http.Client{Timeout: 5 * time.Second},
		apiURL:     server.URL + "/range/",
	}

	// Act
	compromised, err := client.IsCompromised(context.Background(), "password123")

	// Assert
	require.NoError(t, err)
	assert.True(t, compromised, "password123 should be found in breach database")
}

func TestHIBPClient_IsCompromised_PasswordNotFound(t *testing.T) {
	t.Parallel()

	// Arrange: Mock HIBP API that returns different hash suffixes
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		// Return suffixes that don't match our password
		_, _ = w.Write([]byte("AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA:123\n" +
			"BBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBB:456\n"))
	}))
	defer server.Close()

	client := &HIBPClient{
		httpClient: &http.Client{Timeout: 5 * time.Second},
		apiURL:     server.URL + "/range/",
	}

	// Act: Use a strong unique password
	compromised, err := client.IsCompromised(context.Background(), "MyVeryStr0ngP@ssw0rd!2024")

	// Assert
	require.NoError(t, err)
	assert.False(t, compromised, "unique strong password should not be in breach database")
}

func TestHIBPClient_IsCompromised_APIError(t *testing.T) {
	t.Parallel()

	// Arrange: Mock server that returns HTTP error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	client := &HIBPClient{
		httpClient: &http.Client{Timeout: 5 * time.Second},
		apiURL:     server.URL + "/range/",
	}

	// Act
	compromised, err := client.IsCompromised(context.Background(), "testpassword123")

	// Assert: Should fail open (return false but with error)
	require.Error(t, err)
	assert.False(t, compromised, "should fail open on API error")
	assert.Contains(t, err.Error(), "HIBP API returned status 500")
}

func TestHIBPClient_IsCompromised_NetworkError(t *testing.T) {
	t.Parallel()

	// Arrange: Client pointing to non-existent server
	client := &HIBPClient{
		httpClient: &http.Client{Timeout: 100 * time.Millisecond},
		apiURL:     "http://localhost:99999/range/",
	}

	// Act
	compromised, err := client.IsCompromised(context.Background(), "testpassword123")

	// Assert: Should fail open (return false but with error)
	require.Error(t, err)
	assert.False(t, compromised, "should fail open on network error")
	assert.Contains(t, err.Error(), "HIBP API request failed")
}

func TestHIBPClient_IsCompromised_Timeout(t *testing.T) {
	t.Parallel()

	// Arrange: Mock server that delays response
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(200 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := &HIBPClient{
		httpClient: &http.Client{Timeout: 50 * time.Millisecond},
		apiURL:     server.URL + "/range/",
	}

	// Act
	compromised, err := client.IsCompromised(context.Background(), "testpassword123")

	// Assert: Should fail open on timeout
	require.Error(t, err)
	assert.False(t, compromised, "should fail open on timeout")
}

func TestHIBPClient_IsCompromised_EmptyResponse(t *testing.T) {
	t.Parallel()

	// Arrange: Mock server that returns empty response
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(""))
	}))
	defer server.Close()

	client := &HIBPClient{
		httpClient: &http.Client{Timeout: 5 * time.Second},
		apiURL:     server.URL + "/range/",
	}

	// Act
	compromised, err := client.IsCompromised(context.Background(), "testpassword123")

	// Assert: Empty response means password not found
	require.NoError(t, err)
	assert.False(t, compromised, "empty response should mean password not found")
}

func TestHIBPClient_IsCompromised_MalformedResponse(t *testing.T) {
	t.Parallel()

	// Arrange: Mock server that returns malformed data
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		// Response without count (missing colon)
		_, _ = w.Write([]byte("AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA\n" +
			"BBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBB\n"))
	}))
	defer server.Close()

	client := &HIBPClient{
		httpClient: &http.Client{Timeout: 5 * time.Second},
		apiURL:     server.URL + "/range/",
	}

	// Act
	compromised, err := client.IsCompromised(context.Background(), "testpassword123")

	// Assert: Should handle gracefully
	require.NoError(t, err)
	assert.False(t, compromised, "should handle malformed response gracefully")
}

func TestHIBPClient_IsCompromised_ContextCancellation(t *testing.T) {
	t.Parallel()

	// Arrange: Mock server with delay
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(200 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := &HIBPClient{
		httpClient: &http.Client{Timeout: 5 * time.Second},
		apiURL:     server.URL + "/range/",
	}

	// Create context with immediate cancellation
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	// Act
	compromised, err := client.IsCompromised(ctx, "testpassword123")

	// Assert: Should fail with context error
	require.Error(t, err)
	assert.False(t, compromised, "should fail on context cancellation")
}

func TestHIBPClient_IsCompromised_CaseInsensitive(t *testing.T) {
	t.Parallel()

	// Arrange: Test that hash comparison is case-insensitive
	// Password: "Password1!" -> SHA-1: A1733B6D75BD906A9DA80FB2D4991FA2F9B5F3E7
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		// Return suffix in lowercase (should still match)
		_, _ = w.Write([]byte("733b6d75bd906a9da80fb2d4991fa2f9b5f3e7:100\n"))
	}))
	defer server.Close()

	client := &HIBPClient{
		httpClient: &http.Client{Timeout: 5 * time.Second},
		apiURL:     server.URL + "/range/",
	}

	// Act
	compromised, err := client.IsCompromised(context.Background(), "Password1!")

	// Assert: Should match despite case difference
	require.NoError(t, err)
	assert.True(t, compromised, "hash matching should be case-insensitive")
}

func TestHIBPClient_IsCompromised_WhitespaceHandling(t *testing.T) {
	t.Parallel()

	// Arrange: Response with extra whitespace
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		// Response with trailing/leading whitespace
		_, _ = w.Write([]byte("  AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA:123  \n" +
			"\nBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBB:456\n\n"))
	}))
	defer server.Close()

	client := &HIBPClient{
		httpClient: &http.Client{Timeout: 5 * time.Second},
		apiURL:     server.URL + "/range/",
	}

	// Act
	compromised, err := client.IsCompromised(context.Background(), "testpassword123")

	// Assert: Should handle whitespace correctly
	require.NoError(t, err)
	assert.False(t, compromised)
}

func TestNewHIBPClient(t *testing.T) {
	t.Parallel()

	// Act
	client := NewHIBPClient()

	// Assert
	assert.NotNil(t, client)
	assert.NotNil(t, client.httpClient)
	assert.Equal(t, defaultTimeout, client.httpClient.Timeout)
	assert.Equal(t, hibpAPIURL, client.apiURL)
}

// Test that real passwords are actually found (integration-like test)
// This test calls the real HIBP API and should be skipped in CI if needed
func TestHIBPClient_IsCompromised_RealAPI_KnownCompromisedPassword(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	t.Parallel()

	// Arrange: Use real client
	client := NewHIBPClient()

	// Act: Test with a known compromised password
	// "password" is definitely in HIBP database
	compromised, err := client.IsCompromised(context.Background(), "password")

	// Assert
	require.NoError(t, err, "real HIBP API should be available")
	assert.True(t, compromised, "password 'password' should be in HIBP database")
}

func TestHIBPClient_IsCompromised_RealAPI_StrongPassword(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	t.Parallel()

	// Arrange: Use real client
	client := NewHIBPClient()

	// Act: Test with a strong, unique password (very unlikely to be in database)
	// Using a UUID-like string as password
	uniquePassword := "xQ9mK2nP8vL3wB6yT5cR4hJ7fD1sA0gN!@#$%"
	compromised, err := client.IsCompromised(context.Background(), uniquePassword)

	// Assert
	require.NoError(t, err, "real HIBP API should be available")
	assert.False(t, compromised, "unique strong password should not be in HIBP database")
}
