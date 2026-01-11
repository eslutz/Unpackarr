package starr

import (
	"context"
	"time"

	"github.com/eslutz/unpackarr/internal/config"
	"github.com/eslutz/unpackarr/internal/extract"
	"github.com/eslutz/unpackarr/internal/logger"
	"golift.io/starr/lidarr"
)

type LidarrClient struct {
	*Client
	client *lidarr.Lidarr
}

func NewLidarr(cfg *config.StarrApp, queue *extract.Queue, timing *config.TimingConfig, starrTimeout time.Duration) *LidarrClient {
	base := NewClient("lidarr", cfg, queue, timing, starrTimeout)
	lc := &LidarrClient{
		Client: base,
		client: lidarr.New(base.Config()),
	}
	base.Start(lc.poll)
	return lc
}

func (l *LidarrClient) poll(ctx context.Context, c *Client) error {
	queue, err := l.client.GetQueueContext(ctx, 0, 100)
	if err != nil {
		return formatError("Lidarr", "get queue", err)
	}

	c.SetQueueSize(queue.TotalRecords)
	logger.Debug("[Lidarr] Polled queue: %d total records", queue.TotalRecords)

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
			logger.Debug("[Lidarr] Filtered out %s (path=%s, protocol=%s)", item.Name, item.Path, item.Protocol)
			continue
		}

		if record.Status == "completed" {
			matched++
			if err := c.QueueExtract(item); err != nil {
				logger.Error("[Lidarr] Queue extract error for %s: %v", item.Name, err)
			} else {
				logger.Info("[Lidarr] Queued extraction: %s", item.Name)
			}
		} else {
			logger.Debug("[Lidarr] Skipped %s (status=%s)", record.Title, record.Status)
		}
	}

	logger.Debug("[Lidarr] Poll complete: matched %d items for extraction", matched)
	return nil
}
