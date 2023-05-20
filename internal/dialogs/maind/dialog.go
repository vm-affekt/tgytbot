package maind

import (
	"context"
	"github.com/vm-affekt/tgytbot/internal/app"
	"github.com/vm-affekt/tgytbot/internal/downloader"
	"github.com/vm-affekt/tgytbot/internal/logging"
)

type dialog struct {
	rup     app.ReqUserProvider
}

func New(rup app.ReqUserProvider) app.Dialog {
	return &dialog{rup: rup}
}

func (d *dialog) OnEnter(ctx context.Context) error {
	log := logging.FromContextS(ctx)
	log.Info("User entered to main dialog")
	return nil
}

func (d *dialog) OnMessage(ctx context.Context, text string, msgID int) error {
	if err := downloader.ValidateLink(text); err != nil {
		return app.
			NewUserError("Введите корректную ссылку на любой YouTube-ролик, чтобы получить аудиозапись").
			WithCause(err)
	}
	downloadDlg, err := d.rup.RedirectToDialog(ctx, app.DialogYoutubeDownload)
	if err != nil {
		return err
	}
	if err := downloadDlg.OnMessage(ctx, text, msgID); err != nil {
		return err
	}
	return nil
}
