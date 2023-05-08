package telegram

import (
	"fmt"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func (p *MsgProcessor) StartLongPolling(updTimeout int32) error {
	if err := p.connect(); err != nil {
		return fmt.Errorf("failed to connect Telegram server: %w", err)
	}
	updCfg := tgbotapi.NewUpdate(0)
	updCfg.Timeout = int(updTimeout)

	p.updates = p.bot.GetUpdatesChan(updCfg)
	p.startDispatcher()

	return nil
}
