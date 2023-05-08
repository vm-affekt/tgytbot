package progress

import (
	"sync"
	"time"
)

type Counter struct {
	contentLen        int64
	currentDownloaded int64

	mu        *sync.RWMutex
	startTime time.Time
}

func NewCounter(contentLen int64) *Counter {
	return &Counter{
		contentLen: contentLen,
		mu:         new(sync.RWMutex),
		startTime:  time.Now(),
	}
}

func (c *Counter) Write(p []byte) (n int, err error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.currentDownloaded += int64(len(p))
	return len(p), nil
}

func (c *Counter) Percentage() float64 {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return (float64(c.currentDownloaded) / float64(c.contentLen)) * 100.0
}

func (c *Counter) ContentLen() int64 {
	return c.contentLen
}

func (c *Counter) CurrentDownloaded() int64 {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.currentDownloaded
}
