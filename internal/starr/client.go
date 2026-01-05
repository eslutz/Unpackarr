package starr

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/eslutz/unpackarr/internal/config"
	"github.com/eslutz/unpackarr/internal/extract"
	"golift.io/starr"
)

type Client struct {
	name      string
	config    *config.StarrApp
	queue     *extract.Queue
	timing    *config.TimingConfig
	stop      chan struct{}
	mu        sync.RWMutex
	connected bool
	queueSize int
}

type QueueItem struct {
	ID         int64
	Path       string
	Protocol   string
	Status     string
	Name       string
	Size       float64
	DownloadID string
}

func NewClient(name string, cfg *config.StarrApp, queue *extract.Queue, timing *config.TimingConfig) *Client {
	return &Client{
		name:   name,
		config: cfg,
		queue:  queue,
		timing: timing,
		stop:   make(chan struct{}),
	}
}

func (c *Client) Start(poller func(context.Context, *Client) error) {
	go c.run(poller)
}

func (c *Client) Stop() {
	close(c.stop)
}

func (c *Client) run(poller func(context.Context, *Client) error) {
	ticker := time.NewTicker(c.timing.PollInterval)
	defer ticker.Stop()

	log.Printf("[%s] Started polling %s", c.name, c.config.URL)

	for {
		select {
		case <-c.stop:
			log.Printf("[%s] Stopped", c.name)
			return
		case <-ticker.C:
			ctx, cancel := context.WithTimeout(context.Background(), c.config.Timeout)
			err := poller(ctx, c)
			cancel()

			c.mu.Lock()
			c.connected = (err == nil)
			c.mu.Unlock()

			if err != nil {
				log.Printf("[%s] Poll error: %v", c.name, err)
			}
		}
	}
}

func (c *Client) Config() *starr.Config {
	return &starr.Config{
		URL:    c.config.URL,
		APIKey: c.config.APIKey,
	}
}

func (c *Client) ShouldProcess(item *QueueItem) bool {
	if !c.config.HasPath(item.Path) {
		return false
	}
	if !c.config.HasProtocol(item.Protocol) {
		return false
	}
	return true
}

func (c *Client) QueueExtract(item *QueueItem) error {
	_, err := c.queue.Add(&extract.Request{
		Name:       item.Name,
		Path:       item.Path,
		Source:     c.name,
		DeleteOrig: false,
		Passwords:  []string{},
	})
	return err
}

func (c *Client) SetQueueSize(size int) {
	c.mu.Lock()
	c.queueSize = size
	c.mu.Unlock()
}

func (c *Client) Status() (connected bool, queueSize int) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.connected, c.queueSize
}

func (c *Client) Name() string {
	return c.name
}

func formatError(app, operation string, err error) error {
	return fmt.Errorf("%s %s: %w", app, operation, err)
}
