package starr

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/eslutz/unpackarr/internal/config"
	"github.com/eslutz/unpackarr/internal/extract"
	"github.com/eslutz/unpackarr/internal/logger"
	"golift.io/starr"
)

type Client struct {
	name        string
	config      *config.StarrApp
	queue       *extract.Queue
	timing      *config.TimingConfig
	starrTimeout time.Duration
	stop        chan struct{}
	mu          sync.RWMutex
	connected   bool
	queueSize   int
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

func NewClient(name string, cfg *config.StarrApp, queue *extract.Queue, timing *config.TimingConfig, starrTimeout time.Duration) *Client {
	return &Client{
		name:         name,
		config:       cfg,
		queue:        queue,
		timing:       timing,
		starrTimeout: starrTimeout,
		stop:         make(chan struct{}),
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

	logger.Info("[%s] Started polling %s", c.name, c.config.URL)
	logger.Debug("[%s] Poll interval: %v", c.name, c.timing.PollInterval)

	for {
		select {
		case <-c.stop:
			logger.Info("[%s] Stopped", c.name)
			return
		case <-ticker.C:
			logger.Debug("[%s] Ticker fired, starting poll", c.name)
			ctx, cancel := context.WithTimeout(context.Background(), c.starrTimeout)
			err := poller(ctx, c)
			cancel()

			c.mu.Lock()
			c.connected = (err == nil)
			c.mu.Unlock()

			if err != nil {
				logger.Error("[%s] Poll error: %v", c.name, err)
			} else {
				logger.Debug("[%s] Poll completed successfully", c.name)
			}
		}
	}
}

func (c *Client) Config() *starr.Config {
	return &starr.Config{
		URL:    c.config.URL,
		APIKey: c.config.APIKey,
		Client: starr.Client(c.starrTimeout, false),
	}
}

func (c *Client) ShouldProcess(item *QueueItem) bool {
	if !c.config.HasPath(item.Path) {
		logger.Debug("[%s] Path check failed for %s: path=%s not in configured paths %v", c.name, item.Name, item.Path, c.config.Paths)
		return false
	}
	if !c.config.HasProtocol(item.Protocol) {
		logger.Debug("[%s] Protocol check failed for %s: protocol=%s not in configured protocols %v", c.name, item.Name, item.Protocol, c.config.Protocols)
		return false
	}
	return true
}

func (c *Client) QueueExtract(item *QueueItem) error {
	logger.Debug("[%s] Attempting to queue extraction: name=%s, path=%s", c.name, item.Name, item.Path)
	_, err := c.queue.Add(&extract.Request{
		Name:       item.Name,
		Path:       item.Path,
		Source:     c.name,
		DeleteOrig: false,
		Passwords:  []string{},
	})
	if err != nil {
		logger.Debug("[%s] Queue.Add failed: %v", c.name, err)
	} else {
		logger.Debug("[%s] Queue.Add succeeded", c.name)
	}
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
