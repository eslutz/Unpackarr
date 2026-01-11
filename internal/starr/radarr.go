package starr

import (
	"context"
	"log"
	"time"

	"github.com/eslutz/unpackarr/internal/config"
	"github.com/eslutz/unpackarr/internal/extract"
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

		// Queue extraction when download is completed and waiting for import (archived files)
		if record.Status == "completed" && record.TrackedDownloadState == "importPending" {
			if err := c.QueueExtract(item); err != nil {
				log.Printf("[Radarr] Queue extract error for %s: %v", item.Name, err)
			} else {
				log.Printf("[Radarr] Queued extraction: %s", item.Name)
			}
		}
	}

	return nil
}
