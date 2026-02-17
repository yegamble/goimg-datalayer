package handlers

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestDecodeJSONBody_Limit(t *testing.T) {
	// Create a JSON payload slightly larger than 1MB
	// 1MB = 1048576 bytes
	// The overhead of JSON structure is small, so adding 1MB string guarantees overflow
	largePayload := `{"data":"` + strings.Repeat("x", 1048576) + `"}`
	req := httptest.NewRequest(http.MethodPost, "/test", bytes.NewBufferString(largePayload))

	var v interface{}
	err := DecodeJSONBody(req, &v)

	if err == nil {
		t.Error("Expected error for payload > 1MB, got nil")
	}

	expectedErr := "request body too large"
	if err != nil && !strings.Contains(err.Error(), expectedErr) {
		t.Errorf("Expected error containing %q, got %q", expectedErr, err.Error())
	}
}

func TestDecodeJSONBody_UnderLimit(t *testing.T) {
	// Create a JSON payload under 1MB
	smallPayload := `{"data":"small"}`
	req := httptest.NewRequest(http.MethodPost, "/test", bytes.NewBufferString(smallPayload))

	var v map[string]string
	err := DecodeJSONBody(req, &v)

	if err != nil {
		t.Errorf("Expected no error for small payload, got %v", err)
	}
	if v["data"] != "small" {
		t.Errorf("Expected data 'small', got %q", v["data"])
	}
}
