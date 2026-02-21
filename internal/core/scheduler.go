package core

import (
	"sync"
	"time"

	"github.com/Jason-CKY/telegram-notifybot/internal/schemas"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	log "github.com/sirupsen/logrus"
)

func StartFXScheduler(bot *tgbotapi.BotAPI) {
	localTimezone, err := time.LoadLocation("Asia/Singapore")
	if err != nil {
		panic(err)
	}

	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	checkAndNotify(bot, localTimezone)

	for range ticker.C {
		checkAndNotify(bot, localTimezone)
	}
}

func checkAndNotify(bot *tgbotapi.BotAPI, timezone *time.Location) {
	subscriptions, err := schemas.GetAllActiveSubscriptions()
	if err != nil {
		log.Errorf("Error fetching subscriptions: %v", err)
		return
	}

	if len(subscriptions) == 0 {
		return
	}

	currencyRates := make(map[string]float64)
	currencyHistories := make(map[string][]schemas.HistoricalRate)

	for _, sub := range subscriptions {
		if _, exists := currencyRates[sub.Currency]; !exists {
			rate, _, err := GetCurrentRate(sub.Currency)
			if err != nil {
				log.Errorf("Error fetching rate for %s: %v", sub.Currency, err)
				continue
			}
			currencyRates[sub.Currency] = rate

			history, err := GetHistoricalRates(sub.Currency, 12)
			if err != nil {
				log.Errorf("Error fetching history for %s: %v", sub.Currency, err)
			} else {
				currencyHistories[sub.Currency] = history
			}
		}
	}

	var wg sync.WaitGroup

	for _, sub := range subscriptions {
		currentRate, exists := currencyRates[sub.Currency]
		if !exists {
			continue
		}

		shouldNotify := false
		var thresholdToRemove *float64

		if sub.ShouldNotifyForThreshold(currentRate) {
			shouldNotify = true
			if sub.ThresholdAbove != nil && currentRate >= *sub.ThresholdAbove {
				thresholdToRemove = sub.ThresholdAbove
			} else if sub.ThresholdBelow != nil && currentRate <= *sub.ThresholdBelow {
				thresholdToRemove = sub.ThresholdBelow
			}
		}

		if sub.ShouldNotifyForInterval(currentRate) {
			shouldNotify = true
		}

		if !shouldNotify {
			continue
		}

		wg.Add(1)
		go func(s schemas.CurrencySubscription, rate float64, threshold *float64) {
			defer wg.Done()

			history := currencyHistories[s.Currency]
			chartBuf, err := GenerateExchangeRateChart(history, s.Currency)
			if err != nil {
				log.Errorf("Error generating chart for %s: %v", s.Currency, err)
			}

			if chartBuf != nil {
				photoFileBytes := tgbotapi.FileBytes{
					Name:  "chart",
					Bytes: *chartBuf,
				}
				photoConfig := tgbotapi.NewPhoto(s.ChatID, photoFileBytes)
				photoConfig.Caption = s.GetNotificationMessage(rate, history)
				photoConfig.ParseMode = "Markdown"

				if _, err := bot.Send(photoConfig); err != nil {
					log.Errorf("Error sending notification to chat %d: %v", s.ChatID, err)
					return
				}
			} else {
				msg := tgbotapi.NewMessage(s.ChatID, s.GetNotificationMessage(rate, history))
				msg.ParseMode = "Markdown"
				if _, err := bot.Send(msg); err != nil {
					log.Errorf("Error sending notification to chat %d: %v", s.ChatID, err)
					return
				}
			}

			s.LastNotifiedRate = rate
			s.LastNotificationTime = time.Now().In(timezone)

			if threshold != nil {
				s.ThresholdAbove = nil
				s.ThresholdBelow = nil

				if s.ThresholdAbove == nil && s.ThresholdBelow == nil && s.Interval == nil {
					s.Enabled = false
				}
			}

			if err := s.Update(); err != nil {
				log.Errorf("Error updating subscription: %v", err)
			}

			log.Infof("Sent notification to chat %d for %s at rate %.4f", s.ChatID, s.Currency, rate)
		}(sub, currentRate, thresholdToRemove)
	}

	wg.Wait()
}
