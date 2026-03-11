package utils

import (
	"strings"
)

var (
	LogLevel             string
	DirectusHost         string
	DirectusToken        string
	BotToken             string
	WhitelistedUsernames []string
)

const HELP_MESSAGE string = `This bot notifies you on currency exchange rates against SGD. Rates are updated daily.

Available Commands:
/fx <currency> - Show current exchange rate
/fx_chart <currency> [months] [-inverse] - Show historical chart (default: 12 months)
/fx_subscribe <currency> -above <rate> - Notify when rate goes above threshold
/fx_subscribe <currency> -below <rate> - Notify when rate goes below threshold
/fx_interval <currency> <interval> - Notify every X SGD change
/fx_list - List all your subscriptions
/fx_unsubscribe <currency> - Remove subscription for currency

Supported Currencies:
USD, EUR, GBP, JPY, MYR, HKD, AUD, KRW, TWD, VND, IDR, THB, CNY, INR, PHP

Note: /fx_chart not available for TWD, VND (no free historical data)
`

const DEFAULT_TIMEZONE = "Asia/Singapore"

var SupportedCurrencies = []string{"USD", "EUR", "GBP", "JPY", "MYR", "HKD", "AUD", "KRW", "TWD", "VND", "IDR", "THB", "CNY", "INR", "PHP"}

var CurrenciesWithoutHistorical = map[string]bool{"TWD": true, "VND": true}

func IsHistoricalSupported(currency string) bool {
	return !CurrenciesWithoutHistorical[currency]
}

func IsCurrencySupported(currency string) bool {
	upperCurrency := strings.ToUpper(currency)
	for _, c := range SupportedCurrencies {
		if c == upperCurrency {
			return true
		}
	}
	return false
}
