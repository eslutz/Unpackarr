package health

import (
	"strings"
	"testing"
	"time"

	"github.com/eslutz/unpackarr/internal/extract"
)

func TestNewMetrics(t *testing.T) {
	m := NewMetrics()
	if m == nil {
		t.Fatal("NewMetrics() should not return nil")
	}
}

func TestRecordExtraction(t *testing.T) {
	m := NewMetrics()

	result := &extract.Result{
		Name:     "test",
		Source:   "sonarr",
		Started:  time.Now(),
		Elapsed:  30 * time.Second,
		Archives: 1,
		Files:    10,
		Size:     1024 * 1024,
		Success:  true,
	}

	m.RecordExtraction(result)

	prom := m.ExportPrometheus()
	if !strings.Contains(prom, "unpackarr_extractions_total") {
		t.Error("ExportPrometheus() should contain unpackarr_extractions_total")
	}
	if !strings.Contains(prom, "source=\"sonarr\"") {
		t.Error("ExportPrometheus() should contain source label")
	}
	if !strings.Contains(prom, "status=\"success\"") {
		t.Error("ExportPrometheus() should contain status label")
	}
}

func TestRecordExtractionFailure(t *testing.T) {
	m := NewMetrics()

	result := &extract.Result{
		Name:    "test",
		Source:  "radarr",
		Started: time.Now(),
		Elapsed: 10 * time.Second,
		Success: false,
		Error:   nil,
	}

	m.RecordExtraction(result)

	prom := m.ExportPrometheus()
	if !strings.Contains(prom, "status=\"failed\"") {
		t.Error("ExportPrometheus() should contain status=\"failed\" for failures")
	}
}

func TestExportPrometheus(t *testing.T) {
	m := NewMetrics()

	m.RecordExtraction(&extract.Result{
		Name:     "test1",
		Source:   "sonarr",
		Started:  time.Now(),
		Elapsed:  20 * time.Second,
		Archives: 1,
		Files:    5,
		Size:     1024,
		Success:  true,
	})

	m.RecordExtraction(&extract.Result{
		Name:     "test2",
		Source:   "radarr",
		Started:  time.Now(),
		Elapsed:  15 * time.Second,
		Archives: 2,
		Files:    8,
		Size:     2048,
		Success:  true,
	})

	prom := m.ExportPrometheus()

	expectedMetrics := []string{
		"unpackarr_extractions_total",
		"unpackarr_extraction_duration_seconds",
		"unpackarr_bytes_extracted_total",
		"unpackarr_files_extracted_total",
		"unpackarr_archives_processed_total",
	}

	for _, metric := range expectedMetrics {
		if !strings.Contains(prom, metric) {
			t.Errorf("ExportPrometheus() should contain %s", metric)
		}
	}
}

func TestMultipleExtractions(t *testing.T) {
	m := NewMetrics()

	for i := 0; i < 5; i++ {
		m.RecordExtraction(&extract.Result{
			Name:     "test",
			Source:   "folder",
			Started:  time.Now(),
			Elapsed:  10 * time.Second,
			Archives: 1,
			Files:    3,
			Size:     512,
			Success:  true,
		})
	}

	prom := m.ExportPrometheus()
	if !strings.Contains(prom, "source=\"folder\"") {
		t.Error("ExportPrometheus() should track folder source")
	}
}
func TestRecordWebhook(t *testing.T) {
	m := NewMetrics()

	m.RecordWebhook("extracted", true, 100*time.Millisecond)

	prom := m.ExportPrometheus()
	if !strings.Contains(prom, "unpackarr_webhooks_total") {
		t.Error("ExportPrometheus() should contain unpackarr_webhooks_total")
	}
	if !strings.Contains(prom, "event=\"extracted\"") {
		t.Error("ExportPrometheus() should contain event label")
	}
	if !strings.Contains(prom, "status=\"success\"") {
		t.Error("ExportPrometheus() should contain status=\"success\" for successful webhooks")
	}
}

func TestRecordWebhookFailure(t *testing.T) {
	m := NewMetrics()

	m.RecordWebhook("failed", false, 50*time.Millisecond)

	prom := m.ExportPrometheus()
	if !strings.Contains(prom, "event=\"failed\"") {
		t.Error("ExportPrometheus() should contain event=\"failed\"")
	}
	if !strings.Contains(prom, "status=\"failed\"") {
		t.Error("ExportPrometheus() should contain status=\"failed\" for failed webhooks")
	}
}

func TestWebhookMetricsExport(t *testing.T) {
	m := NewMetrics()

	m.RecordWebhook("extracted", true, 100*time.Millisecond)
	m.RecordWebhook("extracted", true, 110*time.Millisecond)
	m.RecordWebhook("failed", false, 50*time.Millisecond)

	prom := m.ExportPrometheus()

	expectedMetrics := []string{
		"unpackarr_webhooks_total",
		"unpackarr_webhook_duration_seconds",
	}

	for _, metric := range expectedMetrics {
		if !strings.Contains(prom, metric) {
			t.Errorf("ExportPrometheus() should contain %s", metric)
		}
	}
}
