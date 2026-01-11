package starr

import (
	"context"
	"log"
	"time"

	"github.com/eslutz/unpackarr/internal/config"
	"github.com/eslutz/unpackarr/internal/extract"
	"golift.io/starr/readarr"
)

type ReadarrClient struct {
	*Client
	client *readarr.Readarr
}

func NewReadarr(cfg *config.StarrApp, queue *extract.Queue, timing *config.TimingConfig, starrTimeout time.Duration) *ReadarrClient {
	base := NewClient("readarr", cfg, queue, timing, starrTimeout)
	rc := &ReadarrClient{
		Client: base,
		client: readarr.New(base.Config()),
	}
	base.Start(rc.poll)
	return rc
}

func (r *ReadarrClient) poll(ctx context.Context, c *Client) error {
	queue, err := r.client.GetQueueContext(ctx, 0, 100)
	if err != nil {
		return formatError("Readarr", "get queue", err)
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
				log.Printf("[Readarr] Queue extract error for %s: %v", item.Name, err)
			} else {
				log.Printf("[Readarr] Queued extraction: %s", item.Name)
			}
		}
	}

	return nil
}
