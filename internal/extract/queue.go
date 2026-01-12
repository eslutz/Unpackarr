package extract

import (
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/eslutz/unpackarr/internal/config"
	"github.com/eslutz/unpackarr/internal/logger"
	"golift.io/xtractr"
)

type Queue struct {
	xtractr     *xtractr.Xtractr
	config      *config.ExtractConfig
	callback    func(*Result)
	mu          sync.RWMutex
	stats       Stats
	activePaths map[string]struct{} // tracks paths currently queued or extracting
	progress    *ProgressManager    // tracks extraction progress
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
	progressCfg := ProgressConfig{
		ReportInterval: cfg.ProgressInterval,
		StallTimeout:   cfg.StallTimeout,
	}
	// Use defaults if not configured
	if progressCfg.ReportInterval == 0 {
		progressCfg.ReportInterval = 30 * time.Second
	}
	if progressCfg.StallTimeout == 0 {
		progressCfg.StallTimeout = 5 * time.Minute
	}

	q := &Queue{
		config:      cfg,
		callback:    callback,
		activePaths: make(map[string]struct{}),
		progress:    NewProgressManager(progressCfg),
	}

	q.xtractr = xtractr.NewQueue(&xtractr.Config{
		Parallel: cfg.Parallel,
		BuffSize: 1000,
		Logger:   q,
	})

	return q
}

func (q *Queue) Add(req *Request) (queueSize int, added bool, err error) {
	logger.Debug("[Queue] Adding extraction request: name=%s, path=%s, source=%s, deleteOrig=%t",
		req.Name, req.Path, req.Source, req.DeleteOrig)

	// Check if this path is already queued or being extracted
	q.mu.RLock()
	_, isActive := q.activePaths[req.Path]
	q.mu.RUnlock()

	if isActive {
		logger.Debug("[Queue] Skipping %s: path already queued or extracting", req.Name)
		return q.stats.Waiting + q.stats.Extracting, false, nil
	}

	passwords := append([]string{}, q.config.Passwords...)
	passwords = append(passwords, req.Passwords...)

	logger.Debug("[Queue] Using %d password(s) for %s", len(passwords), req.Name)

	var xtractrErr error
	queueSize, xtractrErr = q.xtractr.Extract(&xtractr.Xtract{
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

	if xtractrErr != nil {
		logger.Debug("[Queue] Failed to add %s: %v", req.Name, xtractrErr)
		return 0, false, fmt.Errorf("queue extract: %w", xtractrErr)
	}

	q.mu.Lock()
	q.stats.Waiting++
	q.activePaths[req.Path] = struct{}{}
	q.mu.Unlock()

	logger.Debug("[Queue] Successfully added %s (queue size: %d, waiting: %d)", req.Name, queueSize, q.stats.Waiting)

	return queueSize, true, nil
}

func (q *Queue) Stats() Stats {
	q.mu.RLock()
	defer q.mu.RUnlock()
	return q.stats
}

// IsActive returns true if the given path is currently queued or being extracted
func (q *Queue) IsActive(path string) bool {
	q.mu.RLock()
	defer q.mu.RUnlock()
	_, active := q.activePaths[path]
	return active
}

// ActiveCount returns the number of paths currently being tracked
func (q *Queue) ActiveCount() int {
	q.mu.RLock()
	defer q.mu.RUnlock()
	return len(q.activePaths)
}

func (q *Queue) Stop() {
	if q.xtractr != nil {
		q.xtractr.Stop()
	}
}

func (q *Queue) handleCallback(resp *xtractr.Response, req *Request) {
	if !resp.Done {
		logger.Debug("[Queue] Extraction started for %s", resp.X.Name)
		q.mu.Lock()
		q.stats.Waiting--
		q.stats.Extracting++
		q.mu.Unlock()
		logger.Debug("[Queue] Stats updated: waiting=%d, extracting=%d", q.stats.Waiting, q.stats.Extracting)

		// Start progress tracking
		totalFiles := 0
		totalBytes := int64(0)
		for _, archives := range resp.Archives {
			totalFiles += len(archives)
			// Try to get size estimate from archive files
			for _, archive := range archives {
				if info, err := getFileSize(archive); err == nil {
					totalBytes += info
				}
			}
		}
		q.progress.StartTracking(req.Name, req.Path, totalFiles, totalBytes)

		return
	}

	logger.Debug("[Queue] Extraction completed for %s (success=%t, archives=%d, files=%d, size=%d)",
		resp.X.Name, resp.Error == nil, len(resp.Archives), len(resp.NewFiles), resp.Size)

	// Stop progress tracking
	q.progress.StopTracking(req.Path, resp.Error == nil, resp.Size, len(resp.NewFiles), resp.Error)

	q.mu.Lock()
	q.stats.Extracting--
	delete(q.activePaths, req.Path)
	q.mu.Unlock()

	logger.Debug("[Queue] Stats updated: waiting=%d, extracting=%d", q.stats.Waiting, q.stats.Extracting)

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

	if resp.Error != nil {
		logger.Debug("[Queue] Extraction error for %s: %v", resp.X.Name, resp.Error)
	}

	if q.callback != nil {
		logger.Debug("[Queue] Invoking callback for %s", resp.X.Name)
		q.callback(result)
	}
}

func (q *Queue) Printf(format string, v ...any) {
	logger.Info(format, v...)
}

func (q *Queue) Debugf(format string, v ...any) {
	logger.Debug(format, v...)
}

// getFileSize returns the size of a file in bytes
func getFileSize(path string) (int64, error) {
	info, err := os.Stat(path)
	if err != nil {
		return 0, err
	}
	return info.Size(), nil
}
