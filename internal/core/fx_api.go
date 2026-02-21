package core

import (
	"github.com/Jason-CKY/telegram-notifybot/internal/schemas"
)

func GetCurrentRate(currency string) (float64, *schemas.FrankfurterLatestResponse, error) {
	return schemas.FetchLatestExchangeRate(currency)
}

func GetHistoricalRates(currency string, months int) ([]schemas.HistoricalRate, error) {
	if months <= 0 {
		months = 12
	}
	if months > 120 {
		months = 120
	}
	days := months * 30
	return schemas.FetchHistoricalExchangeRates(currency, days)
}
