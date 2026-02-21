package handler

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/Jason-CKY/telegram-notifybot/internal/core"
	"github.com/Jason-CKY/telegram-notifybot/internal/schemas"
	"github.com/Jason-CKY/telegram-notifybot/internal/utils"
	log "github.com/sirupsen/logrus"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func HandleFXCommand(update *tgbotapi.Update, bot *tgbotapi.BotAPI) {
	args := strings.Fields(update.Message.CommandArguments())
	if len(args) == 0 {
		msg := tgbotapi.NewMessage(update.Message.Chat.ID,
			"Usage: /fx <currency>\nExample: /fx USD")
		bot.Send(msg)
		return
	}

	currency := strings.ToUpper(args[0])
	if !utils.IsCurrencySupported(currency) {
		msg := tgbotapi.NewMessage(update.Message.Chat.ID,
			fmt.Sprintf("Unsupported currency: %s\n\nSupported currencies: %s",
				currency, strings.Join(utils.SupportedCurrencies, ", ")))
		bot.Send(msg)
		return
	}

	rate, record, err := core.GetCurrentRate(currency)
	if err != nil {
		log.Error(err)
		msg := tgbotapi.NewMessage(update.Message.Chat.ID,
			fmt.Sprintf("Error fetching exchange rate: %v", err))
		bot.Send(msg)
		return
	}

	msg := tgbotapi.NewMessage(update.Message.Chat.ID,
		core.FormatCurrentRateMessage(currency, rate, record))
	msg.ParseMode = "Markdown"
	bot.Send(msg)
}

func HandleFXChartCommand(update *tgbotapi.Update, bot *tgbotapi.BotAPI) {
	args := strings.Fields(update.Message.CommandArguments())
	if len(args) == 0 {
		msg := tgbotapi.NewMessage(update.Message.Chat.ID,
			"Usage: /fx_chart <currency> [months]\nExample: /fx_chart USD 6")
		bot.Send(msg)
		return
	}

	currency := strings.ToUpper(args[0])
	if !utils.IsCurrencySupported(currency) {
		msg := tgbotapi.NewMessage(update.Message.Chat.ID,
			fmt.Sprintf("Unsupported currency: %s\n\nSupported currencies: %s",
				currency, strings.Join(utils.SupportedCurrencies, ", ")))
		bot.Send(msg)
		return
	}

	months := 12
	if len(args) > 1 {
		if m, err := strconv.Atoi(args[1]); err == nil && m > 0 {
			months = m
		}
	}

	rates, err := core.GetHistoricalRates(currency, months)
	if err != nil {
		log.Error(err)
		msg := tgbotapi.NewMessage(update.Message.Chat.ID,
			fmt.Sprintf("Error fetching historical rates: %v", err))
		bot.Send(msg)
		return
	}

	chartBuf, err := core.GenerateExchangeRateChart(rates, currency)
	if err != nil {
		log.Error(err)
		msg := tgbotapi.NewMessage(update.Message.Chat.ID,
			fmt.Sprintf("Error generating chart: %v", err))
		bot.Send(msg)
		return
	}

	photoFileBytes := tgbotapi.FileBytes{
		Name:  "chart",
		Bytes: *chartBuf,
	}
	photoConfig := tgbotapi.NewPhoto(update.Message.Chat.ID, photoFileBytes)
	photoConfig.Caption = fmt.Sprintf("📊 %s/SGD Exchange Rate (%d months)", currency, months)
	bot.Send(photoConfig)
}

func HandleFXSubscribeCommand(update *tgbotapi.Update, bot *tgbotapi.BotAPI) {
	args := update.Message.CommandArguments()

	if args == "" {
		msg := tgbotapi.NewMessage(update.Message.Chat.ID,
			"Usage: /fx_subscribe <currency> --above <rate> OR --below <rate>\n\n"+
				"Examples:\n"+
				"/fx_subscribe USD --above 1.40\n"+
				"/fx_subscribe EUR --below 1.45")
		bot.Send(msg)
		return
	}

	currency := ""
	var thresholdAbove, thresholdBelow *float64

	parts := strings.Fields(args)
	for i, part := range parts {
		if strings.ToUpper(part) == "--ABOVE" && i+1 < len(parts) {
			currency = strings.ToUpper(parts[0])
			if val, err := strconv.ParseFloat(parts[i+1], 64); err == nil {
				thresholdAbove = &val
			}
		} else if strings.ToUpper(part) == "--BELOW" && i+1 < len(parts) {
			currency = strings.ToUpper(parts[0])
			if val, err := strconv.ParseFloat(parts[i+1], 64); err == nil {
				thresholdBelow = &val
			}
		}
	}

	if currency == "" && len(parts) > 0 {
		currency = strings.ToUpper(parts[0])
	}

	if currency == "" {
		msg := tgbotapi.NewMessage(update.Message.Chat.ID,
			"Please specify a currency.\nExample: /fx_subscribe USD --above 1.40")
		bot.Send(msg)
		return
	}

	if !utils.IsCurrencySupported(currency) {
		msg := tgbotapi.NewMessage(update.Message.Chat.ID,
			fmt.Sprintf("Unsupported currency: %s\n\nSupported currencies: %s",
				currency, strings.Join(utils.SupportedCurrencies, ", ")))
		bot.Send(msg)
		return
	}

	if thresholdAbove == nil && thresholdBelow == nil {
		msg := tgbotapi.NewMessage(update.Message.Chat.ID,
			"Please specify --above or --below with a rate.\nExample: /fx_subscribe USD --above 1.40")
		bot.Send(msg)
		return
	}

	sub, err := schemas.CreateOrUpdateSubscription(update.Message.Chat.ID, currency, thresholdAbove, thresholdBelow, nil)
	if err != nil {
		log.Error(err)
		msg := tgbotapi.NewMessage(update.Message.Chat.ID,
			fmt.Sprintf("Error creating subscription: %v", err))
		bot.Send(msg)
		return
	}

	var response string
	if thresholdAbove != nil {
		response = fmt.Sprintf("✅ Subscribed to %s/SGD notifications.\nYou will be notified when the rate goes *above* %.4f SGD.\n\nNote: This is a one-time notification and will be removed after triggered.", currency, *thresholdAbove)
	} else {
		response = fmt.Sprintf("✅ Subscribed to %s/SGD notifications.\nYou will be notified when the rate goes *below* %.4f SGD.\n\nNote: This is a one-time notification and will be removed after triggered.", currency, *thresholdBelow)
	}

	msg := tgbotapi.NewMessage(update.Message.Chat.ID, response)
	msg.ParseMode = "Markdown"
	bot.Send(msg)

	currentRate, _, err := core.GetCurrentRate(currency)
	if err == nil {
		sub.LastNotifiedRate = currentRate
		sub.Update()
	}
}

func HandleFXIntervalCommand(update *tgbotapi.Update, bot *tgbotapi.BotAPI) {
	args := strings.Fields(update.Message.CommandArguments())
	if len(args) < 2 {
		msg := tgbotapi.NewMessage(update.Message.Chat.ID,
			"Usage: /fx_interval <currency> <interval>\n\n"+
				"Example: /fx_interval USD 0.05\n"+
				"This will notify you every time the rate changes by 0.05 SGD or more.")
		bot.Send(msg)
		return
	}

	currency := strings.ToUpper(args[0])
	if !utils.IsCurrencySupported(currency) {
		msg := tgbotapi.NewMessage(update.Message.Chat.ID,
			fmt.Sprintf("Unsupported currency: %s\n\nSupported currencies: %s",
				currency, strings.Join(utils.SupportedCurrencies, ", ")))
		bot.Send(msg)
		return
	}

	interval, err := strconv.ParseFloat(args[1], 64)
	if err != nil || interval <= 0 {
		msg := tgbotapi.NewMessage(update.Message.Chat.ID,
			"Please provide a valid positive number for the interval.\nExample: /fx_interval USD 0.05")
		bot.Send(msg)
		return
	}

	sub, err := schemas.CreateOrUpdateSubscription(update.Message.Chat.ID, currency, nil, nil, &interval)
	if err != nil {
		log.Error(err)
		msg := tgbotapi.NewMessage(update.Message.Chat.ID,
			fmt.Sprintf("Error creating subscription: %v", err))
		bot.Send(msg)
		return
	}

	msg := tgbotapi.NewMessage(update.Message.Chat.ID,
		fmt.Sprintf("✅ Subscribed to %s/SGD interval notifications.\nYou will be notified every time the rate changes by %.4f SGD or more.", currency, interval))
	bot.Send(msg)

	currentRate, _, err := core.GetCurrentRate(currency)
	if err == nil {
		sub.LastNotifiedRate = currentRate
		sub.Update()
	}
}

func HandleFXListCommand(update *tgbotapi.Update, bot *tgbotapi.BotAPI) {
	subscriptions, err := schemas.GetCurrencySubscriptionsByChatID(update.Message.Chat.ID)
	if err != nil {
		log.Error(err)
		msg := tgbotapi.NewMessage(update.Message.Chat.ID,
			fmt.Sprintf("Error fetching subscriptions: %v", err))
		bot.Send(msg)
		return
	}

	msg := tgbotapi.NewMessage(update.Message.Chat.ID,
		core.FormatSubscriptionListMessage(subscriptions))
	msg.ParseMode = "Markdown"
	bot.Send(msg)
}

func HandleFXUnsubscribeCommand(update *tgbotapi.Update, bot *tgbotapi.BotAPI) {
	args := strings.Fields(update.Message.CommandArguments())
	if len(args) == 0 {
		msg := tgbotapi.NewMessage(update.Message.Chat.ID,
			"Usage: /fx_unsubscribe <currency>\nExample: /fx_unsubscribe USD")
		bot.Send(msg)
		return
	}

	currency := strings.ToUpper(args[0])
	if !utils.IsCurrencySupported(currency) {
		msg := tgbotapi.NewMessage(update.Message.Chat.ID,
			fmt.Sprintf("Unsupported currency: %s\n\nSupported currencies: %s",
				currency, strings.Join(utils.SupportedCurrencies, ", ")))
		bot.Send(msg)
		return
	}

	sub, err := schemas.GetCurrencySubscription(update.Message.Chat.ID, currency)
	if err != nil {
		log.Error(err)
		msg := tgbotapi.NewMessage(update.Message.Chat.ID,
			fmt.Sprintf("Error fetching subscription: %v", err))
		bot.Send(msg)
		return
	}

	if sub == nil {
		msg := tgbotapi.NewMessage(update.Message.Chat.ID,
			fmt.Sprintf("You don't have a subscription for %s.", currency))
		bot.Send(msg)
		return
	}

	if err := sub.Delete(); err != nil {
		log.Error(err)
		msg := tgbotapi.NewMessage(update.Message.Chat.ID,
			fmt.Sprintf("Error removing subscription: %v", err))
		bot.Send(msg)
		return
	}

	msg := tgbotapi.NewMessage(update.Message.Chat.ID,
		fmt.Sprintf("✅ Unsubscribed from %s/SGD notifications.", currency))
	bot.Send(msg)
}
