package progress

import (
	"errors"
	"time"
)

func (c *Counter) EstimatedTime() (time.Duration, error) {
	if c.contentLen == 0 {
		return 0, errors.New("can't compute estimated time if total content len is unknown")
	}
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.currentDownloaded == 0 {
		return 0, errors.New("can't compute estimated time when there is not downloaded data")
	}
	currentT := time.Since(c.startTime)
	currentV := float64(c.currentDownloaded) / float64(currentT)
	remainingS := c.contentLen - c.currentDownloaded
	estimatedTime := float64(remainingS) / currentV
	return time.Duration(estimatedTime), nil
}
