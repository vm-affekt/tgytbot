package downloader

import (
	"fmt"
	"github.com/kkdai/youtube/v2"
	"strings"
)

func ValidateLink(link string) error {
	if !strings.Contains(link, "youtu.be/") && !strings.Contains(link, "youtube.com/") {
		return fmt.Errorf("string %q doesn't contain youtube host", link)
	}
	_, err := youtube.ExtractVideoID(link)
	if err != nil {
		return fmt.Errorf("failed to extract video id from link: %w", err)
	}
	return nil
}
