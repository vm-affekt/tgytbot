package app

import (
	"context"
	"io"
)

type DownloadResult struct {
	ContentLen int64
	Name       string
	Stream     io.ReadCloser
}

type DownloadService interface {
	DownloadAudio(ctx context.Context, link string) (DownloadResult, error)
	DownloadVideo(ctx context.Context, link string) (DownloadResult, error)
}
