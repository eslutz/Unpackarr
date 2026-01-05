package health

import (
	"fmt"
	"sync"

	"github.com/eslutz/unpackarr/internal/extract"
)

type Metrics struct {
	mu                  sync.RWMutex
	extractionsTotal    map[string]map[string]int64
	extractionDurations map[string]float64
	bytesExtracted      map[string]int64
	filesExtracted      map[string]int64
	archivesProcessed   map[string]int64
}

func NewMetrics() *Metrics {
	return &Metrics{
		extractionsTotal:    make(map[string]map[string]int64),
		extractionDurations: make(map[string]float64),
		bytesExtracted:      make(map[string]int64),
		filesExtracted:      make(map[string]int64),
		archivesProcessed:   make(map[string]int64),
	}
}

func (m *Metrics) RecordExtraction(result *extract.Result) {
	m.mu.Lock()
	defer m.mu.Unlock()

	source := result.Source
	status := "success"
	if !result.Success {
		status = "failed"
	}

	if m.extractionsTotal[source] == nil {
		m.extractionsTotal[source] = make(map[string]int64)
	}
	m.extractionsTotal[source][status]++

	if result.Success {
		m.extractionDurations[source] = result.Elapsed.Seconds()
		m.bytesExtracted[source] += result.Size
		m.filesExtracted[source] += int64(result.Files)
		m.archivesProcessed[source] += int64(result.Archives)
	}
}

func (m *Metrics) ExportPrometheus() string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var out string

	out += "# HELP unpackarr_extractions_total Total number of extractions\n"
	out += "# TYPE unpackarr_extractions_total counter\n"
	for source, statuses := range m.extractionsTotal {
		for status, count := range statuses {
			out += fmt.Sprintf("unpackarr_extractions_total{source=\"%s\",status=\"%s\"} %d\n", source, status, count)
		}
	}

	out += "# HELP unpackarr_extraction_duration_seconds Last extraction duration\n"
	out += "# TYPE unpackarr_extraction_duration_seconds gauge\n"
	for source, duration := range m.extractionDurations {
		out += fmt.Sprintf("unpackarr_extraction_duration_seconds{source=\"%s\"} %.2f\n", source, duration)
	}

	out += "# HELP unpackarr_bytes_extracted_total Total bytes extracted\n"
	out += "# TYPE unpackarr_bytes_extracted_total counter\n"
	for source, bytes := range m.bytesExtracted {
		out += fmt.Sprintf("unpackarr_bytes_extracted_total{source=\"%s\"} %d\n", source, bytes)
	}

	out += "# HELP unpackarr_files_extracted_total Total files extracted\n"
	out += "# TYPE unpackarr_files_extracted_total counter\n"
	for source, files := range m.filesExtracted {
		out += fmt.Sprintf("unpackarr_files_extracted_total{source=\"%s\"} %d\n", source, files)
	}

	out += "# HELP unpackarr_archives_processed_total Total archives processed\n"
	out += "# TYPE unpackarr_archives_processed_total counter\n"
	for source, archives := range m.archivesProcessed {
		out += fmt.Sprintf("unpackarr_archives_processed_total{source=\"%s\"} %d\n", source, archives)
	}

	return out
}
