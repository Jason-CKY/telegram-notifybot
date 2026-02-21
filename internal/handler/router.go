package handler

import (
	"github.com/Jason-CKY/telegram-notifybot/internal/schemas"
	"github.com/Jason-CKY/telegram-notifybot/internal/utils"
	log "github.com/sirupsen/logrus"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func HandleUpdate(update *tgbotapi.Update, bot *tgbotapi.BotAPI) {
	if update.Message != nil && utils.IsUsernameAllowed(update.Message.From.UserName) {
		if update.Message.IsCommand() {
			HandleCommand(update, bot)
		}
	}
}

func HandleCommand(update *tgbotapi.Update, bot *tgbotapi.BotAPI) {
	msg := tgbotapi.NewMessage(update.Message.Chat.ID, "")

	switch update.Message.Command() {
	case "help":
		msg.Text = utils.HELP_MESSAGE
		msg.ParseMode = "Markdown"
	case "start":
		_, _, err := schemas.InsertChatSettingsIfNotPresent(update.Message.Chat.ID)
		if err != nil {
			log.Error(err)
			return
		}
		msg.Text = "Welcome to NotifyBot! Use /help to see available commands."
	case "fx":
		HandleFXCommand(update, bot)
		return
	case "fx_chart":
		HandleFXChartCommand(update, bot)
		return
	case "fx_subscribe":
		HandleFXSubscribeCommand(update, bot)
		return
	case "fx_interval":
		HandleFXIntervalCommand(update, bot)
		return
	case "fx_list":
		HandleFXListCommand(update, bot)
		return
	case "fx_unsubscribe":
		HandleFXUnsubscribeCommand(update, bot)
		return
	default:
		return
	}

	if _, err := bot.Request(msg); err != nil {
		log.Error(err)
		return
	}
}
