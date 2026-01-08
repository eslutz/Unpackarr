package starr

import (
	"context"
	"log"

	"github.com/eslutz/unpackarr/internal/config"
	"github.com/eslutz/unpackarr/internal/extract"
	"golift.io/starr/lidarr"
)

type LidarrClient struct {
	*Client
	client *lidarr.Lidarr
}

func NewLidarr(cfg *config.StarrApp, queue *extract.Queue, timing *config.TimingConfig) *LidarrClient {
	base := NewClient("lidarr", cfg, queue, timing)
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
			continue
		}

		if record.Status == "completed" {
			if err := c.QueueExtract(item); err != nil {
				log.Printf("[Lidarr] Queue extract error for %s: %v", item.Name, err)
			} else {
				log.Printf("[Lidarr] Queued extraction: %s", item.Name)
			}
		}
	}

	return nil
}
