package starr

import (
	"context"
	"time"

	"github.com/eslutz/unpackarr/internal/config"
	"github.com/eslutz/unpackarr/internal/extract"
	"github.com/eslutz/unpackarr/internal/logger"
	"golift.io/starr/sonarr"
)

type SonarrClient struct {
	*Client
	client *sonarr.Sonarr
}

func NewSonarr(cfg *config.StarrApp, queue *extract.Queue, timing *config.TimingConfig, starrTimeout time.Duration) *SonarrClient {
	base := NewClient("sonarr", cfg, queue, timing, starrTimeout)
	sc := &SonarrClient{
		Client: base,
		client: sonarr.New(base.Config()),
	}
	base.Start(sc.poll)
	return sc
}

func (s *SonarrClient) poll(ctx context.Context, c *Client) error {
	queue, err := s.client.GetQueueContext(ctx, 0, 100)
	if err != nil {
		return formatError("Sonarr", "get queue", err)
	}

	c.SetQueueSize(queue.TotalRecords)
	logger.Debug("[Sonarr] Polled queue: %d total records", queue.TotalRecords)

	matched := 0
	for _, record := range queue.Records {
		item := &QueueItem{
			ID:         record.ID,
			Path:       record.OutputPath,
			Protocol:   string(record.Protocol),
			Status:     record.Status,
			Name:       record.Title,
			Size:       record.Size,
			DownloadID: record.DownloadID,
		}

		if !c.ShouldProcess(item) {
			logger.Debug("[Sonarr] Filtered out %s (path=%s, protocol=%s)", item.Name, item.Path, item.Protocol)
			continue
		}

		// Queue extraction when download is completed and waiting for import (archived files)
		if record.Status == "completed" && record.TrackedDownloadState == "importPending" {
			matched++
			if err := c.QueueExtract(item); err != nil {
				logger.Error("[Sonarr] Queue extract error for %s: %v", item.Name, err)
			} else {
				logger.Info("[Sonarr] Queued extraction: %s", item.Name)
			}
		} else {
			logger.Debug("[Sonarr] Skipped %s (status=%s, state=%s)", record.Title, record.Status, record.TrackedDownloadState)
		}
	}

	logger.Debug("[Sonarr] Poll complete: matched %d items for extraction", matched)
	return nil
}
