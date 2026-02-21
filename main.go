package main

import (
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/Jason-CKY/telegram-notifybot/internal/core"
	"github.com/Jason-CKY/telegram-notifybot/internal/handler"
	"github.com/Jason-CKY/telegram-notifybot/internal/utils"
	"github.com/joho/godotenv"
	log "github.com/sirupsen/logrus"
)

func main() {
	err := godotenv.Load()

	if err != nil {
		log.Infof("Error loading .env file: %v\nUsing environment variables instead...", err)
	}

	utils.LogLevel = utils.LookupEnvString("LOG_LEVEL")
	utils.DirectusHost = utils.LookupEnvString("DIRECTUS_HOST")
	utils.DirectusToken = utils.LookupEnvString("DIRECTUS_TOKEN")
	utils.BotToken = utils.LookupEnvString("TELEGRAM_BOT_TOKEN")
	utils.WhitelistedUsernames = utils.LookupEnvStringArray("ALLOWED_USERNAMES")

	log.SetReportCaller(true)
	log.SetFormatter(&log.TextFormatter{
		FullTimestamp:          true,
		DisableLevelTruncation: true,
	})
	logLevel, _ := log.ParseLevel(utils.LogLevel)
	log.SetLevel(logLevel)

	log.Info("connecting to telegram bot")

	bot, err := tgbotapi.NewBotAPI(utils.BotToken)
	bot.Debug = utils.LogLevel == "debug"
	log.Infof("Authorized on account %s", bot.Self.UserName)

	if err != nil {
		panic(err)
	}

	go core.StartFXScheduler(bot)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60
	updates := bot.GetUpdatesChan(u)
	for update := range updates {
		handler.HandleUpdate(&update, bot)
	}
}
