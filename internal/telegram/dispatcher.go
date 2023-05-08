package telegram

import (
	"context"
	"errors"
	"fmt"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/google/uuid"
	"github.com/vm-affekt/tgytbot/internal/app"
	"github.com/vm-affekt/tgytbot/internal/logging"
	"sync"
	"time"
)

func (p *MsgProcessor) startDispatcher() {
	ctx := context.Background()
	ctx, p.cancelDispatcher = context.WithCancel(ctx)
	go p.startUpdListener(ctx)
}

func (p *MsgProcessor) startUpdListener(gCtx context.Context) {
	log := logging.FromContextS(gCtx)
	log.Info("Message receiver started... The bot is ready to process new messages!")
	for upd := range p.updates {
		var (
			msg  *tgbotapi.Message
			from *tgbotapi.User
		)

		switch {
		case upd.Message != nil:
			msg = upd.Message
			from = msg.From
		default:
			continue
		}

		p.muLocker.Lock()
		mu, ok := p.lockByUserID[from.ID] // we can handle only one message from certain user at once
		if !ok {
			mu = new(sync.Mutex)
			p.lockByUserID[from.ID] = mu
		}
		p.muLocker.Unlock()

		ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second) // Parent of this context is Background, not a gCtx. Because cancellation of gCtx should'nt interrupt message handling.
		go func() {
			start := time.Now()
			mu.Lock()
			defer mu.Unlock()
			rqID := genRequestID()
			userID := from.ID
			ctx, log = logging.NewContextSL(ctx,
				"request_id", rqID,
				"user_tg_id", userID,
				"user_name", from.UserName,
			)
			text := msg.Text
			log.Infof("Received message %q", text)
			rup := NewReqUserProvider(p.bot, from, p.userDialogState, p.container)
			defer func() {
				if r := recover(); r != nil {
					log.With("recovered_obj", r).Error("!!! A PANIC occurred while handling query !!! See recovered object in recovered_obj!")
				}
				_, _ = app.SendMessagef(ctx, rup, "При обработки вашего сообщения произошла ошибка. Идентификатор запроса: %v", rqID)
				totalElapsedTime := time.Since(start)
				log.Infow("Query is proceeded.",
					"total_elapsed_time", totalElapsedTime,
				)
			}()
			defer cancel()

			currentDialog := p.userDialogState.FindDialogByUser(userID)
			if currentDialog == nil {
				var err error
				currentDialog, err = p.initUser(ctx, rup)
				if err != nil {
					log.Errorf("Failed to init user: %v", err)
					_, _ = app.SendMessagef(ctx, rup, "При регистрации вашего пользователя в системе произошла ошибка. Идентификатор запроса: %v", rqID)
					return
				}
			}
			if err := currentDialog.OnMessage(ctx, text, msg.MessageID); err != nil {
				log.Errorf("Failed to process message: %v", err)
				var usrErr *app.UserError
				if errors.As(err, &usrErr) {
					_, _ = app.SendMessagef(ctx, rup, usrErr.UserMessage)
				} else {
					_, _ = app.SendMessagef(ctx, rup, "При обработке сообщения возникла ошибка. Попробуйте попытку позже. Идентификатор запроса: %v", rqID)
				}
			}

		}()

	}

}

func (p *MsgProcessor) initUser(ctx context.Context, rup app.ReqUserProvider) (mainDlg app.Dialog, err error) {
	mainDlg, err = rup.RedirectToDialog(ctx, app.DialogMain)
	if err != nil {
		return mainDlg, fmt.Errorf("failed to redirect to main dialog: %w", err)
	}
	return mainDlg, err
}

func genRequestID() string {
	rid, _ := uuid.NewRandom()
	return rid.String()
}
