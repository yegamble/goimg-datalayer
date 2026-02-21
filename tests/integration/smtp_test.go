//go:build integration

package integration_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/yegamble/goimg-datalayer/internal/infrastructure/email"
	"github.com/yegamble/goimg-datalayer/tests/integration/containers"
)

type mailpitMessages struct {
	Messages []struct {
		ID      string `json:"ID"`
		Subject string `json:"Subject"`
		To      []struct {
			Address string `json:"Address"`
		} `json:"To"`
		From struct {
			Address string `json:"Address"`
		} `json:"From"`
	} `json:"messages"`
	Total int `json:"total"`
}

func fetchMailpitMessages(t *testing.T, apiEndpoint string) *mailpitMessages {
	t.Helper()

	resp, err := http.Get(apiEndpoint + "/api/v1/messages")
	require.NoError(t, err, "failed to query Mailpit API")
	defer resp.Body.Close()

	require.Equal(t, http.StatusOK, resp.StatusCode, "Mailpit API should return 200")

	var msgs mailpitMessages
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&msgs))
	return &msgs
}

func deleteMailpitMessages(t *testing.T, apiEndpoint string) {
	t.Helper()

	req, err := http.NewRequest(http.MethodDelete, apiEndpoint+"/api/v1/messages", nil)
	require.NoError(t, err)
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
}

func newTestSMTPSender(t *testing.T, c *containers.SMTPContainer) *email.SMTPSender {
	t.Helper()

	port, err := strconv.Atoi(c.SMTPPort)
	require.NoError(t, err, "SMTP port must be numeric")

	cfg := email.Config{
		Host:        c.SMTPHost,
		Port:        port,
		Username:    "",
		Password:    "",
		FromAddress: "noreply@goimg.test",
		FromName:    "goimg Test",
		UseTLS:      false,
		Timeout:     10 * time.Second,
		RateLimit:   50,
		Enabled:     true,
	}

	logger := zerolog.Nop()
	sender, err := email.NewSMTPSender(cfg, logger)
	require.NoError(t, err, "failed to create SMTP sender")
	return sender
}

func TestSMTPSender_Send(t *testing.T) {
	ctx := context.Background()

	smtpC, err := containers.NewSMTPContainer(ctx, t)
	require.NoError(t, err)
	require.NotNil(t, smtpC)
	t.Cleanup(func() { _ = smtpC.Terminate(ctx) })

	deleteMailpitMessages(t, smtpC.APIEndpoint)

	sender := newTestSMTPSender(t, smtpC)

	to := "recipient@example.com"
	subject := "Integration Test Email"
	htmlBody := "<h1>Hello from SMTP integration test</h1>"
	textBody := "Hello from SMTP integration test"

	err = sender.Send(ctx, to, subject, htmlBody, textBody)
	require.NoError(t, err, "Send() should deliver successfully to Mailpit")

	var msgs *mailpitMessages
	require.Eventually(t, func() bool {
		msgs = fetchMailpitMessages(t, smtpC.APIEndpoint)
		return msgs.Total > 0
	}, 5*time.Second, 200*time.Millisecond, "Mailpit should receive exactly one message")

	require.Equal(t, 1, msgs.Total, "exactly one message should be received")
	assert.Equal(t, subject, msgs.Messages[0].Subject, "subject should match")
	assert.Equal(t, to, msgs.Messages[0].To[0].Address, "recipient should match")
	assert.Equal(t, "noreply@goimg.test", msgs.Messages[0].From.Address, "from address should match")
	t.Logf("SMTP verified: message '%s' delivered to Mailpit (ID: %s)", subject, msgs.Messages[0].ID)
}

func TestSMTPSender_SendNewFollowerEmail(t *testing.T) {
	ctx := context.Background()

	smtpC, err := containers.NewSMTPContainer(ctx, t)
	require.NoError(t, err)
	require.NotNil(t, smtpC)
	t.Cleanup(func() { _ = smtpC.Terminate(ctx) })

	deleteMailpitMessages(t, smtpC.APIEndpoint)

	sender := newTestSMTPSender(t, smtpC)

	err = sender.SendNewFollowerEmail(ctx, "user@example.com", "follower_user")
	require.NoError(t, err, "SendNewFollowerEmail() should deliver successfully")

	var msgs *mailpitMessages
	require.Eventually(t, func() bool {
		msgs = fetchMailpitMessages(t, smtpC.APIEndpoint)
		return msgs.Total > 0
	}, 5*time.Second, 200*time.Millisecond, "Mailpit should receive the follower email")

	require.Equal(t, 1, msgs.Total)
	assert.Contains(t, msgs.Messages[0].Subject, "follower_user",
		"follower email subject should contain the follower's username")
	t.Logf("Follower email delivered: subject=%q", msgs.Messages[0].Subject)
}

func TestSMTPSender_SendMalwareDetectedEmail(t *testing.T) {
	ctx := context.Background()

	smtpC, err := containers.NewSMTPContainer(ctx, t)
	require.NoError(t, err)
	require.NotNil(t, smtpC)
	t.Cleanup(func() { _ = smtpC.Terminate(ctx) })

	deleteMailpitMessages(t, smtpC.APIEndpoint)

	sender := newTestSMTPSender(t, smtpC)

	err = sender.SendMalwareDetectedEmail(ctx, "user@example.com", "testuser", "malicious.jpg")
	require.NoError(t, err, "SendMalwareDetectedEmail() should deliver successfully")

	var msgs *mailpitMessages
	require.Eventually(t, func() bool {
		msgs = fetchMailpitMessages(t, smtpC.APIEndpoint)
		return msgs.Total > 0
	}, 5*time.Second, 200*time.Millisecond, "Mailpit should receive the malware alert email")

	require.Equal(t, 1, msgs.Total)
	assert.Contains(t, msgs.Messages[0].Subject, "Malware",
		"malware email subject should contain 'Malware'")
	assert.Equal(t, "user@example.com", msgs.Messages[0].To[0].Address,
		"recipient should match")
	t.Logf("Malware alert email delivered: subject=%q", msgs.Messages[0].Subject)
}

func TestSMTPSender_Disabled(t *testing.T) {
	ctx := context.Background()

	cfg := email.Config{
		Host:        "localhost",
		Port:        1025,
		FromAddress: "noreply@goimg.test",
		RateLimit:   10,
		Timeout:     5 * time.Second,
		Enabled:     false,
	}

	logger := zerolog.Nop()
	sender, err := email.NewSMTPSender(cfg, logger)
	require.NoError(t, err)

	err = sender.Send(ctx, "to@example.com", "subject", "<p>body</p>", "body")
	assert.ErrorIs(t, err, email.ErrSMTPDisabled, "disabled sender should return ErrSMTPDisabled")
}

func TestSMTPSender_RateLimit(t *testing.T) {
	ctx := context.Background()

	smtpC, err := containers.NewSMTPContainer(ctx, t)
	require.NoError(t, err)
	require.NotNil(t, smtpC)
	t.Cleanup(func() { _ = smtpC.Terminate(ctx) })

	port, err := strconv.Atoi(smtpC.SMTPPort)
	require.NoError(t, err)

	cfg := email.Config{
		Host:        smtpC.SMTPHost,
		Port:        port,
		FromAddress: "noreply@goimg.test",
		FromName:    "goimg Test",
		UseTLS:      false,
		Timeout:     10 * time.Second,
		RateLimit:   2,
		Enabled:     true,
	}

	logger := zerolog.Nop()
	sender, err := email.NewSMTPSender(cfg, logger)
	require.NoError(t, err)

	for i := 0; i < 2; i++ {
		sendErr := sender.Send(ctx, fmt.Sprintf("user%d@example.com", i),
			"Rate limit test", "<p>test</p>", "test")
		require.NoError(t, sendErr, "send %d should succeed within rate limit", i+1)
	}

	sendErr := sender.Send(ctx, "overflow@example.com", "Rate limit test", "<p>test</p>", "test")
	assert.ErrorIs(t, sendErr, email.ErrRateLimited, "third send should be rate-limited")
	t.Log("Rate limiting verified: third send correctly returned ErrRateLimited")
}
