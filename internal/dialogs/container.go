package dialogs

import (
	"github.com/vm-affekt/tgytbot/internal/app"
	"github.com/vm-affekt/tgytbot/internal/dialogs/download"
	"github.com/vm-affekt/tgytbot/internal/dialogs/maind"
	"time"
)

// Container is DI-container of app
type Container struct {
	downloadService    app.DownloadService
	downloadTimeout    time.Duration
	audioMaxFileSizeMB int64
}

func NewContainer(downloadService app.DownloadService, downloadTimeout time.Duration, audioMaxFileSizeMB int64) *Container {
	return &Container{
		downloadService:    downloadService,
		downloadTimeout:    downloadTimeout,
		audioMaxFileSizeMB: audioMaxFileSizeMB,
	}
}

func (c *Container) CreateDialog(id app.DialogID, rup app.ReqUserProvider) app.Dialog {
	switch id {
	case app.DialogMain:
		return maind.New(rup)
	case app.DialogYoutubeDownload:
		return download.New(rup, c.downloadService, c.downloadTimeout, c.audioMaxFileSizeMB)
	}
	return nil
}
