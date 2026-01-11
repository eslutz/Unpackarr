package starr

import (
	"context"
	"time"

	"github.com/eslutz/unpackarr/internal/config"
	"github.com/eslutz/unpackarr/internal/extract"
	"github.com/eslutz/unpackarr/internal/logger"
	"golift.io/starr/radarr"
)

type RadarrClient struct {
	*Client
	client *radarr.Radarr
}

func NewRadarr(cfg *config.StarrApp, queue *extract.Queue, timing *config.TimingConfig, starrTimeout time.Duration) *RadarrClient {
	base := NewClient("radarr", cfg, queue, timing, starrTimeout)
	rc := &RadarrClient{
		Client: base,
		client: radarr.New(base.Config()),
	}
	base.Start(rc.poll)
	return rc
}

func (r *RadarrClient) poll(ctx context.Context, c *Client) error {
	queue, err := r.client.GetQueueContext(ctx, 0, 100)
	if err != nil {
		return formatError("Radarr", "get queue", err)
	}

	c.SetQueueSize(queue.TotalRecords)
	logger.Debug("[Radarr] Polled queue: %d total records", queue.TotalRecords)
	logger.Debug("[Radarr] Configured paths: %v, protocols: %v", c.config.Paths, c.config.Protocols)

	matched := 0
	for _, record := range queue.Records {
		logger.Debug("[Radarr] Processing: %s (status=%s, state=%s, trackedStatus=%s, path=%s, protocol=%s)",
			record.Title, record.Status, record.TrackedDownloadState, record.TrackedDownloadStatus, record.OutputPath, record.Protocol)

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
			logger.Debug("[Radarr] Filtered out %s (ShouldProcess returned false)", item.Name)
			continue
		}

		logger.Debug("[Radarr] %s passed path/protocol checks", item.Name)

		// Queue extraction when download is completed and waiting for import (archived files)
		if record.Status == "completed" && record.TrackedDownloadState == "importPending" {
			logger.Debug("[Radarr] %s matches extraction criteria (completed + importPending)", item.Name)
			matched++
			if err := c.QueueExtract(item); err != nil {
				logger.Error("[Radarr] Queue extract error for %s: %v", item.Name, err)
			} else {
				logger.Info("[Radarr] Queued extraction: %s", item.Name)
			}
		} else {
			logger.Debug("[Radarr] Skipped %s (status=%s, state=%s, does not match completed+importPending)", record.Title, record.Status, record.TrackedDownloadState)
		}
	}

	logger.Debug("[Radarr] Poll complete: matched %d items for extraction", matched)
	return nil
}
