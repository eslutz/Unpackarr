package notify

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/eslutz/unpackarr/internal/config"
	"github.com/eslutz/unpackarr/internal/extract"
	"github.com/eslutz/unpackarr/internal/logger"
)

type Webhook struct {
	config *config.WebhookConfig
	client *http.Client
}

func NewWebhook(cfg *config.WebhookConfig) *Webhook {
	if cfg.URL == "" {
		return nil
	}

	return &Webhook{
		config: cfg,
		client: &http.Client{
			Timeout: cfg.Timeout,
		},
	}
}

func (w *Webhook) Notify(result *extract.Result) {
	if w == nil {
		return
	}

	event := w.determineEvent(result)
	if !w.shouldNotify(event) {
		return
	}

	payload := w.buildPayload(result, event)
	if err := w.send(payload); err != nil {
		logger.Error("[Webhook] Send error: %v", err)
	}
}

func (w *Webhook) determineEvent(result *extract.Result) string {
	if result.Success {
		return "extracted"
	}
	return "failed"
}

func (w *Webhook) shouldNotify(event string) bool {
	for _, e := range w.config.Events {
		if e == event {
			return true
		}
	}
	return false
}

func (w *Webhook) buildPayload(result *extract.Result, event string) []byte {
	switch w.config.Template {
	case "discord":
		return w.discordPayload(result, event)
	case "slack":
		return w.slackPayload(result, event)
	case "gotify":
		return w.gotifyPayload(result, event)
	default:
		return w.jsonPayload(result, event)
	}
}

func (w *Webhook) discordPayload(result *extract.Result, _ string) []byte {
	color := 3066993
	if !result.Success {
		color = 15158332
	}

	status := "✅ Extracted"
	if !result.Success {
		status = "❌ Failed"
	}

	payload := map[string]any{
		"embeds": []map[string]any{
			{
				"title":       fmt.Sprintf("%s: %s", status, result.Name),
				"description": fmt.Sprintf("Source: %s\nDuration: %s", result.Source, result.Elapsed.Round(time.Second)),
				"color":       color,
				"timestamp":   result.Started.Format(time.RFC3339),
				"fields": []map[string]any{
					{"name": "Archives", "value": fmt.Sprint(result.Archives), "inline": true},
					{"name": "Files", "value": fmt.Sprint(result.Files), "inline": true},
					{"name": "Size", "value": fmt.Sprintf("%.1f MiB", float64(result.Size)/(1024*1024)), "inline": true},
				},
			},
		},
	}

	if !result.Success && result.Error != nil {
		payload["embeds"].([]map[string]any)[0]["fields"] = append(
			payload["embeds"].([]map[string]any)[0]["fields"].([]map[string]any),
			map[string]any{"name": "Error", "value": result.Error.Error()},
		)
	}

	data, _ := json.Marshal(payload)
	return data
}

func (w *Webhook) slackPayload(result *extract.Result, _ string) []byte {
	color := "good"
	if !result.Success {
		color = "danger"
	}

	status := "Extracted"
	if !result.Success {
		status = "Failed"
	}

	text := fmt.Sprintf("*%s:* %s\n*Source:* %s\n*Duration:* %s\n*Archives:* %d | *Files:* %d | *Size:* %.1f MiB",
		status, result.Name, result.Source, result.Elapsed.Round(time.Second),
		result.Archives, result.Files, float64(result.Size)/(1024*1024))

	if !result.Success && result.Error != nil {
		text += fmt.Sprintf("\n*Error:* %s", result.Error.Error())
	}

	payload := map[string]any{
		"attachments": []map[string]any{
			{
				"color": color,
				"text":  text,
				"ts":    result.Started.Unix(),
			},
		},
	}

	data, _ := json.Marshal(payload)
	return data
}

func (w *Webhook) gotifyPayload(result *extract.Result, _ string) []byte {
	priority := 5
	if !result.Success {
		priority = 8
	}

	status := "Extracted"
	if !result.Success {
		status = "Failed"
	}

	message := fmt.Sprintf("Source: %s\nDuration: %s\nArchives: %d | Files: %d | Size: %.1f MiB",
		result.Source, result.Elapsed.Round(time.Second),
		result.Archives, result.Files, float64(result.Size)/(1024*1024))

	if !result.Success && result.Error != nil {
		message += fmt.Sprintf("\nError: %s", result.Error.Error())
	}

	payload := map[string]any{
		"title":    fmt.Sprintf("%s: %s", status, result.Name),
		"message":  message,
		"priority": priority,
	}

	data, _ := json.Marshal(payload)
	return data
}

func (w *Webhook) jsonPayload(result *extract.Result, event string) []byte {
	payload := map[string]any{
		"event":    event,
		"name":     result.Name,
		"source":   result.Source,
		"success":  result.Success,
		"started":  result.Started.Format(time.RFC3339),
		"elapsed":  result.Elapsed.String(),
		"archives": result.Archives,
		"files":    result.Files,
		"size":     result.Size,
	}

	if result.Error != nil {
		payload["error"] = result.Error.Error()
	}

	data, _ := json.Marshal(payload)
	return data
}

func (w *Webhook) send(payload []byte) error {
	ctx, cancel := context.WithTimeout(context.Background(), w.config.Timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "POST", w.config.URL, bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := w.client.Do(req)
	if err != nil {
		return fmt.Errorf("send request: %w", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	// Limit response body to prevent memory exhaustion from malicious endpoints
	limitedBody := io.LimitReader(resp.Body, 1024*1024) // 1MB limit
	_, _ = io.ReadAll(limitedBody)

	if resp.StatusCode >= 400 {
		return fmt.Errorf("webhook returned status %d", resp.StatusCode)
	}

	return nil
}

func (w *Webhook) Test() error {
	if w == nil {
		return fmt.Errorf("webhook not configured")
	}

	testResult := &extract.Result{
		Name:     "test-extraction",
		Source:   "test",
		Started:  time.Now(),
		Elapsed:  30 * time.Second,
		Archives: 1,
		Files:    10,
		Size:     104857600,
		Success:  true,
	}

	payload := w.buildPayload(testResult, "extracted")
	return w.send(payload)
}
