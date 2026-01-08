package notify

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/eslutz/unpackarr/internal/config"
	"github.com/eslutz/unpackarr/internal/extract"
)

func TestNewWebhook(t *testing.T) {
	cfg := &config.WebhookConfig{
		URL:      "http://test.com/webhook",
		Template: "discord",
		Events:   []string{"extracted", "failed"},
		Timeout:  10 * time.Second,
	}

	webhook := NewWebhook(cfg)
	if webhook == nil {
		t.Fatal("NewWebhook() should not return nil")
	}
}

func TestNewWebhookEmpty(t *testing.T) {
	cfg := &config.WebhookConfig{
		URL: "",
	}

	webhook := NewWebhook(cfg)
	if webhook != nil {
		t.Error("NewWebhook() with empty URL should return nil")
	}
}

func TestDetermineEvent(t *testing.T) {
	cfg := &config.WebhookConfig{
		URL: "http://test.com",
	}
	webhook := NewWebhook(cfg)

	successResult := &extract.Result{Success: true}
	if webhook.determineEvent(successResult) != "extracted" {
		t.Error("determineEvent() for success should return 'extracted'")
	}

	failResult := &extract.Result{Success: false}
	if webhook.determineEvent(failResult) != "failed" {
		t.Error("determineEvent() for failure should return 'failed'")
	}
}

func TestShouldNotify(t *testing.T) {
	cfg := &config.WebhookConfig{
		URL:    "http://test.com",
		Events: []string{"extracted"},
	}
	webhook := NewWebhook(cfg)

	if !webhook.shouldNotify("extracted") {
		t.Error("shouldNotify('extracted') should return true")
	}
	if webhook.shouldNotify("failed") {
		t.Error("shouldNotify('failed') should return false")
	}
}

func TestDiscordPayload(t *testing.T) {
	cfg := &config.WebhookConfig{
		URL:      "http://test.com",
		Template: "discord",
	}
	webhook := NewWebhook(cfg)

	result := &extract.Result{
		Name:     "Test Movie",
		Source:   "radarr",
		Started:  time.Now(),
		Elapsed:  30 * time.Second,
		Archives: 1,
		Files:    10,
		Size:     1024 * 1024 * 100,
		Success:  true,
	}

	payload := webhook.discordPayload(result, "extracted")
	payloadStr := string(payload)

	if !strings.Contains(payloadStr, "embeds") {
		t.Error("Discord payload should contain 'embeds'")
	}
	if !strings.Contains(payloadStr, "Test Movie") {
		t.Error("Discord payload should contain movie name")
	}
}

func TestSlackPayload(t *testing.T) {
	cfg := &config.WebhookConfig{
		URL:      "http://test.com",
		Template: "slack",
	}
	webhook := NewWebhook(cfg)

	result := &extract.Result{
		Name:     "Test Show",
		Source:   "sonarr",
		Started:  time.Now(),
		Elapsed:  45 * time.Second,
		Archives: 2,
		Files:    15,
		Size:     1024 * 1024 * 200,
		Success:  true,
	}

	payload := webhook.slackPayload(result, "extracted")
	payloadStr := string(payload)

	if !strings.Contains(payloadStr, "attachments") {
		t.Error("Slack payload should contain 'attachments'")
	}
	if !strings.Contains(payloadStr, "Test Show") {
		t.Error("Slack payload should contain show name")
	}
}

func TestGotifyPayload(t *testing.T) {
	cfg := &config.WebhookConfig{
		URL:      "http://test.com",
		Template: "gotify",
	}
	webhook := NewWebhook(cfg)

	result := &extract.Result{
		Name:     "Test Album",
		Source:   "lidarr",
		Started:  time.Now(),
		Elapsed:  20 * time.Second,
		Archives: 1,
		Files:    12,
		Size:     1024 * 1024 * 50,
		Success:  true,
	}

	payload := webhook.gotifyPayload(result, "extracted")
	payloadStr := string(payload)

	if !strings.Contains(payloadStr, "title") {
		t.Error("Gotify payload should contain 'title'")
	}
	if !strings.Contains(payloadStr, "message") {
		t.Error("Gotify payload should contain 'message'")
	}
}

func TestJSONPayload(t *testing.T) {
	cfg := &config.WebhookConfig{
		URL:      "http://test.com",
		Template: "json",
	}
	webhook := NewWebhook(cfg)

	result := &extract.Result{
		Name:     "Test Book",
		Source:   "readarr",
		Started:  time.Now(),
		Elapsed:  15 * time.Second,
		Archives: 1,
		Files:    5,
		Size:     1024 * 1024 * 10,
		Success:  true,
	}

	payload := webhook.jsonPayload(result, "extracted")
	payloadStr := string(payload)

	if !strings.Contains(payloadStr, "event") {
		t.Error("JSON payload should contain 'event'")
	}
	if !strings.Contains(payloadStr, "Test Book") {
		t.Error("JSON payload should contain book name")
	}
	if !strings.Contains(payloadStr, "readarr") {
		t.Error("JSON payload should contain source")
	}
}

func TestNotifySuccess(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	cfg := &config.WebhookConfig{
		URL:      server.URL,
		Template: "json",
		Events:   []string{"extracted"},
		Timeout:  5 * time.Second,
	}
	webhook := NewWebhook(cfg)

	result := &extract.Result{
		Name:    "Test",
		Source:  "folder",
		Started: time.Now(),
		Elapsed: 10 * time.Second,
		Success: true,
	}

	webhook.Notify(result)
}

func TestNotifyNil(t *testing.T) {
	var webhook *Webhook
	result := &extract.Result{Success: true}

	webhook.Notify(result)
}

func TestNotifyFilteredEvent(t *testing.T) {
	cfg := &config.WebhookConfig{
		URL:      "http://test.com",
		Template: "json",
		Events:   []string{"extracted"},
	}
	webhook := NewWebhook(cfg)

	result := &extract.Result{
		Success: false,
	}

	webhook.Notify(result)
}

func TestBuildPayload(t *testing.T) {
	tests := []struct {
		name     string
		template string
		contains string
	}{
		{"discord", "discord", "embeds"},
		{"slack", "slack", "attachments"},
		{"gotify", "gotify", "title"},
		{"json", "json", "event"},
		{"unknown", "unknown", "event"},
	}

	result := &extract.Result{
		Name:    "Test",
		Success: true,
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.WebhookConfig{
				URL:      "http://test.com",
				Template: tt.template,
			}
			webhook := NewWebhook(cfg)
			payload := webhook.buildPayload(result, "extracted")

			if !strings.Contains(string(payload), tt.contains) {
				t.Errorf("buildPayload(%s) should contain '%s'", tt.template, tt.contains)
			}
		})
	}
}

func TestSendError(t *testing.T) {
	cfg := &config.WebhookConfig{
		URL:     "http://invalid-url-that-does-not-exist.local",
		Timeout: 1 * time.Second,
	}
	webhook := NewWebhook(cfg)

	err := webhook.send([]byte("{}"))
	if err == nil {
		t.Error("send() to invalid URL should return error")
	}
}

func TestSendResponseBodyLimited(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Send a large response body
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		// Write 2MB of data (exceeds the 1MB limit)
		for i := 0; i < 2; i++ {
			_, _ = w.Write(make([]byte, 1024*1024))
		}
	}))
	defer server.Close()

	cfg := &config.WebhookConfig{
		URL:     server.URL,
		Timeout: 5 * time.Second,
	}
	webhook := NewWebhook(cfg)

	// This should succeed even with large response body due to limiting
	err := webhook.send([]byte("{}"))
	if err != nil {
		t.Errorf("send() with large response body should succeed, got: %v", err)
	}
}

func TestSendSuccess(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("{}"))
	}))
	defer server.Close()

	cfg := &config.WebhookConfig{
		URL:     server.URL,
		Timeout: 5 * time.Second,
	}
	webhook := NewWebhook(cfg)

	err := webhook.send([]byte("{}"))
	if err != nil {
		t.Errorf("send() should succeed, got: %v", err)
	}
}
