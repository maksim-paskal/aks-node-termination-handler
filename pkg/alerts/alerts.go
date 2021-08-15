package alerts

import (
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	"github.com/maksim-paskal/aks-node-termination-handler/pkg/config"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

var (
	bot         *tgbotapi.BotAPI
	initialized = false
)

func InitAlerts() error {
	if len(*config.Get().TelegramToken) == 0 {
		log.Warning("not sending Telegram message, no token")

		return nil
	}

	var err error

	bot, err = tgbotapi.NewBotAPI(*config.Get().TelegramToken)
	if err != nil {
		return errors.Wrap(err, "error in NewBotAPI")
	}

	initialized = true

	log.Printf("Authorized on account %s", bot.Self.UserName)

	return nil
}

func Send(obj TemplateMessageType) error {
	if !initialized {
		log.Warning("not sending Telegram message, not initialized")

		return nil
	}

	messageText, err := TemplateMessage(obj)
	if err != nil {
		return err
	}

	msg := tgbotapi.NewMessage(int64(*config.Get().TelegramChatID), messageText)

	if _, err := bot.Send(msg); err != nil {
		return errors.Wrap(err, "error in bot.Send")
	}

	return nil
}
