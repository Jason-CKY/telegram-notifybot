package core

import (
	"testing"
	"time"

	"github.com/Jason-CKY/telegram-notifybot/internal/schemas"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateExchangeRateChart_Success(t *testing.T) {
	rates := []schemas.HistoricalRate{
		{Date: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), Rate: 1.3500},
		{Date: time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC), Rate: 1.3400},
		{Date: time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC), Rate: 1.3300},
		{Date: time.Date(2026, 2, 15, 0, 0, 0, 0, time.UTC), Rate: 1.3200},
		{Date: time.Date(2026, 2, 20, 0, 0, 0, 0, time.UTC), Rate: 1.3100},
	}

	chartData, err := GenerateExchangeRateChart(rates, "USD")
	require.NoError(t, err)
	require.NotNil(t, chartData)
	assert.NotEmpty(t, *chartData)
	assert.Greater(t, len(*chartData), 1000)
}

func TestGenerateExchangeRateChart_EmptyRates(t *testing.T) {
	_, err := GenerateExchangeRateChart([]schemas.HistoricalRate{}, "USD")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no historical rates")
}

func TestGenerateExchangeRateChart_SingleRate(t *testing.T) {
	rates := []schemas.HistoricalRate{
		{Date: time.Date(2026, 2, 20, 0, 0, 0, 0, time.UTC), Rate: 1.3500},
	}

	chartData, err := GenerateExchangeRateChart(rates, "USD")
	require.NoError(t, err)
	require.NotNil(t, chartData)
	assert.NotEmpty(t, *chartData)
}

func TestGenerateExchangeRateChart_YAxisScaling(t *testing.T) {
	rates := []schemas.HistoricalRate{
		{Date: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), Rate: 1.3500},
		{Date: time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC), Rate: 1.3501},
		{Date: time.Date(2026, 1, 3, 0, 0, 0, 0, time.UTC), Rate: 1.3499},
	}

	chartData, err := GenerateExchangeRateChart(rates, "USD")
	require.NoError(t, err)
	require.NotNil(t, chartData)
	assert.NotEmpty(t, *chartData)
}

func TestFormatCurrentRateMessage(t *testing.T) {
	response := &schemas.FrankfurterLatestResponse{
		Amount: 1.0,
		Base:   "SGD",
		Date:   "2026-02-20",
		Rates:  map[string]float64{"USD": 1.2678},
	}

	msg := FormatCurrentRateMessage("USD", 0.7889, response)

	assert.Contains(t, msg, "USD/SGD Exchange Rate")
	assert.Contains(t, msg, "0.7889")
	assert.Contains(t, msg, "2026-02-20")
	assert.Contains(t, msg, "/fx_chart USD")
}

func TestFormatCurrentRateMessage_NilResponse(t *testing.T) {
	msg := FormatCurrentRateMessage("EUR", 1.4500, nil)

	assert.Contains(t, msg, "EUR/SGD Exchange Rate")
	assert.Contains(t, msg, "1.4500")
}

func TestFormatSubscriptionListMessage_Empty(t *testing.T) {
	msg := FormatSubscriptionListMessage(nil)
	assert.Contains(t, msg, "no active subscriptions")
}

func TestFormatSubscriptionListMessage_WithSubscriptions(t *testing.T) {
	subscriptions := []schemas.CurrencySubscription{
		{
			Currency:       "USD",
			ThresholdAbove: float64Ptr(1.4000),
		},
		{
			Currency: "EUR",
			Interval: float64Ptr(0.05),
		},
		{
			Currency:       "GBP",
			ThresholdBelow: float64Ptr(1.5000),
		},
	}

	msg := FormatSubscriptionListMessage(subscriptions)

	assert.Contains(t, msg, "USD")
	assert.Contains(t, msg, "EUR")
	assert.Contains(t, msg, "GBP")
	assert.Contains(t, msg, "Alert above")
	assert.Contains(t, msg, "Interval")
	assert.Contains(t, msg, "Alert below")
}

func float64Ptr(f float64) *float64 {
	return &f
}
