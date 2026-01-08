package extract

import (
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/eslutz/unpackarr/internal/config"
	"golift.io/xtractr"
)

type Queue struct {
	xtractr  *xtractr.Xtractr
	config   *config.ExtractConfig
	callback func(*Result)
	mu       sync.RWMutex
	stats    Stats
}

type Stats struct {
	Waiting    int
	Extracting int
}

type Result struct {
	Name       string
	Source     string
	Path       string
	DeleteOrig bool
	Started    time.Time
	Elapsed    time.Duration
	Archives   int
	Files      int
	Size       int64
	Success    bool
	Error      error
}

type Request struct {
	Name       string
	Path       string
	Source     string
	DeleteOrig bool
	Passwords  []string
}

func NewQueue(cfg *config.ExtractConfig, callback func(*Result)) *Queue {
	q := &Queue{
		config:   cfg,
		callback: callback,
	}

	q.xtractr = xtractr.NewQueue(&xtractr.Config{
		Parallel: cfg.Parallel,
		BuffSize: 1000,
		Logger:   q,
	})

	return q
}

func (q *Queue) Add(req *Request) (int, error) {
	passwords := append([]string{}, q.config.Passwords...)
	passwords = append(passwords, req.Passwords...)

	queueSize, err := q.xtractr.Extract(&xtractr.Xtract{
		Name:       req.Name,
		Password:   "",
		Passwords:  passwords,
		DeleteOrig: req.DeleteOrig,
		TempFolder: false,
		LogFile:    false,
		Filter: xtractr.Filter{
			Path: req.Path,
		},
		CBFunction: func(resp *xtractr.Response) {
			q.handleCallback(resp, req)
		},
	})

	if err != nil {
		return 0, fmt.Errorf("queue extract: %w", err)
	}

	q.mu.Lock()
	q.stats.Waiting++
	q.mu.Unlock()

	return queueSize, nil
}

func (q *Queue) Stats() Stats {
	q.mu.RLock()
	defer q.mu.RUnlock()
	return q.stats
}

func (q *Queue) Stop() {
	if q.xtractr != nil {
		q.xtractr.Stop()
	}
}

func (q *Queue) handleCallback(resp *xtractr.Response, req *Request) {
	if !resp.Done {
		q.mu.Lock()
		q.stats.Waiting--
		q.stats.Extracting++
		q.mu.Unlock()
		return
	}

	q.mu.Lock()
	q.stats.Extracting--
	q.mu.Unlock()

	result := &Result{
		Name:       resp.X.Name,
		Source:     req.Source,
		Path:       req.Path,
		DeleteOrig: req.DeleteOrig,
		Started:    resp.Started,
		Elapsed:    resp.Elapsed,
		Archives:   len(resp.Archives),
		Files:      len(resp.NewFiles),
		Size:       resp.Size,
		Success:    resp.Error == nil,
		Error:      resp.Error,
	}

	if q.callback != nil {
		q.callback(result)
	}
}

func (q *Queue) Printf(format string, v ...interface{}) {
	log.Printf(format, v...)
}

func (q *Queue) Debugf(format string, v ...interface{}) {
	log.Printf("[DEBUG] "+format, v...)
}
