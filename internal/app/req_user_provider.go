package app

import (
	"context"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"io"
)

type ReqUserProvider interface {
	User() *tgbotapi.User

	SendMessageWithKeyboardf(ctx context.Context, replyKeyboard *tgbotapi.ReplyKeyboardMarkup, text string, args ...interface{}) (messageID int, err error)
	SendAudio(ctx context.Context, stream io.Reader, fileName string) error

	RedirectToDialog(ctx context.Context, id DialogID) (newDlg Dialog, err error)
	DeleteMessages(ctx context.Context, msgIDs ...int) error
}

func SendMessagef(ctx context.Context, rup ReqUserProvider, text string, args ...interface{}) (messageID int, err error) {
	return rup.SendMessageWithKeyboardf(ctx, nil, text, args...)
}
