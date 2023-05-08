package telegram

import (
	"context"
	"fmt"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/vm-affekt/tgytbot/internal/app"
	"github.com/vm-affekt/tgytbot/internal/dialogs"
	"github.com/vm-affekt/tgytbot/internal/logging"
	"io"
	"strings"
)

type reqUserProvider struct {
	bot             *tgbotapi.BotAPI
	userDialogState *app.UserDialogState
	container       *dialogs.Container
	from            *tgbotapi.User
}

func NewReqUserProvider(
	bot *tgbotapi.BotAPI,
	from *tgbotapi.User,
	userDialogState *app.UserDialogState,
	container *dialogs.Container,
) *reqUserProvider {
	return &reqUserProvider{
		bot:             bot,
		from:            from,
		userDialogState: userDialogState,
		container:       container,
	}
}

func (rup *reqUserProvider) User() *tgbotapi.User {
	return rup.from
}

func (rup *reqUserProvider) SendMessageWithKeyboardf(ctx context.Context, replyKeyboard *tgbotapi.ReplyKeyboardMarkup, text string, args ...interface{}) (int, error) {
	msg := rup.makeTextMsgf(text, args...)
	if replyKeyboard != nil {
		msg.ReplyMarkup = replyKeyboard
	} else {
		msg.ReplyMarkup = tgbotapi.NewRemoveKeyboard(false)
	}

	return rup.sendMessage(ctx, msg)
}

func (rup *reqUserProvider) SendAudio(ctx context.Context, stream io.Reader, fileName string) error {
	log := logging.FromContextS(ctx)
	log.Infof("Uploading audio file %q to Telegram...", fileName)
	file := tgbotapi.FileReader{
		Name:   fileName,
		Reader: stream,
	}
	audioMsg := tgbotapi.NewAudio(rup.from.ID, file)
	if _, err := rup.bot.Send(audioMsg); err != nil {
		return fmt.Errorf("failed to upload audio to telegram: %w", err)
	}
	log.Info("Uploading audio file to Telegram successfully done!")
	return nil
}

func (rup *reqUserProvider) RedirectToDialog(ctx context.Context, id app.DialogID) (newDlg app.Dialog, err error) {
	log := logging.FromContextS(ctx)
	log.Infof("Redirecting to dialog with id=%d...", id)
	if err := id.Validate(); err != nil {
		return nil, fmt.Errorf("failed to validate dialog: %w", err)
	}

	newDlg = rup.container.CreateDialog(id, rup)
	rup.userDialogState.SetDialogForUser(rup.from.ID, newDlg)
	if err := newDlg.OnEnter(ctx); err != nil {
		return nil, fmt.Errorf("failed OnEnter on new dialog: %w", err)
	}
	return newDlg, nil
}

func (rup *reqUserProvider) DeleteMessages(ctx context.Context, msgIDs ...int) error {
	log := logging.FromContextS(ctx)
	log.Infof("Removing of %d messages..", len(msgIDs))
	for _, msgID := range msgIDs {
		cfg := tgbotapi.NewDeleteMessage(rup.from.ID, msgID)
		if _, err := rup.bot.Send(cfg); err != nil {
			if !strings.Contains(err.Error(), "json: cannot unmarshal bool into Go value of type tgbotapi.Message") { // TODO: В либе ошибка, телеграмовский ответ неправильно маршалится
				return fmt.Errorf("failed to delete message with id %v: %w", msgID, err)
			}
		}
	}
	log.Info("All specified messages deleted!")
	return nil
}

func (rup *reqUserProvider) makeTextMsgf(text string, args ...interface{}) tgbotapi.MessageConfig {
	m := tgbotapi.NewMessage(rup.from.ID, fmt.Sprintf(text, args...))
	m.ParseMode = "HTML"
	return m
}

func (rup *reqUserProvider) sendMessage(ctx context.Context, msg tgbotapi.MessageConfig) (messageID int, err error) {
	select {
	case <-ctx.Done():
		return 0, fmt.Errorf("context is done while sending message: %w", ctx.Err())
	default:
	}

	logging.FromContextS(ctx).Infow("Sending message to user...",
		"text", msg.Text)

	sentMsg, err := rup.bot.Send(msg)
	if err != nil {
		return 0, fmt.Errorf("failed to send message %+v to user with telegram_id=%d: %w", msg, rup.from.ID, err)
	}
	return sentMsg.MessageID, nil
}
