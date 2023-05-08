package progress

import (
	"fmt"
	"sync"
	"testing"
	"time"
)

func TestCounter_EstimatedTime(t *testing.T) {
	c := &Counter{
		contentLen:        15000,
		currentDownloaded: 15,
		mu:                &sync.RWMutex{},
		startTime:         time.Now().Add(-1 * time.Second),
	}
	estimatedTime, err := c.EstimatedTime()
	if err != nil {
		fmt.Println("err:", err)
	}
	fmt.Println("Estimated time: ", estimatedTime.String())

}
