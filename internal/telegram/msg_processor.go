package telegram

import (
	"errors"
	"fmt"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/vm-affekt/tgytbot/internal/app"
	"github.com/vm-affekt/tgytbot/internal/dialogs"
	"sync"
)

type MsgProcessor struct {
	apiKey    string
	debugMode bool

	bot             *tgbotapi.BotAPI
	container       *dialogs.Container
	userDialogState *app.UserDialogState

	updates          tgbotapi.UpdatesChannel
	cancelDispatcher func()

	muLocker     sync.Mutex
	lockByUserID map[int64]*sync.Mutex
}

func NewMsgProcessor(apiKey string, debugMode bool, container *dialogs.Container) *MsgProcessor {
	return &MsgProcessor{
		apiKey:          apiKey,
		debugMode:       debugMode,
		container:       container,
		userDialogState: app.NewUserDialogState(),
		lockByUserID:    make(map[int64]*sync.Mutex),
	}
}

func (p *MsgProcessor) connect() (err error) {
	if p.apiKey == "" {
		return errors.New("bot api key is not specified")
	}
	p.bot, err = tgbotapi.NewBotAPI(p.apiKey)
	if err != nil {
		return fmt.Errorf("can't create bot api: %w", err)
	}
	p.bot.Debug = p.debugMode
	return nil
}
