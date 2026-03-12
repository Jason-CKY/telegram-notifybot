package core

import (
	"fmt"
	"strings"

	"github.com/Jason-CKY/telegram-notifybot/internal/schemas"
	"github.com/vicanso/go-charts/v2"
)

func GenerateExchangeRateChart(rates []schemas.HistoricalRate, currency string, inverse bool) (*[]byte, error) {
	if len(rates) == 0 {
		return nil, fmt.Errorf("no historical rates available")
	}

	values := make([]float64, len(rates))
	dates := make([]string, len(rates))

	for i, r := range rates {
		if inverse {
			values[i] = 1.0 / r.Rate
		} else {
			values[i] = r.Rate
		}
		dates[i] = r.Date.Format("Jan 06")
	}

	minVal := values[0]
	maxVal := values[0]
	for _, v := range values {
		if v < minVal {
			minVal = v
		}
		if v > maxVal {
			maxVal = v
		}
	}

	padding := (maxVal - minVal) * 0.1
	if padding == 0 {
		padding = maxVal * 0.05
	}
	minWithPadding := minVal - padding
	maxWithPadding := maxVal + padding

	title := fmt.Sprintf("%s/SGD Exchange Rate History", currency)
	if inverse {
		title = fmt.Sprintf("SGD/%s Exchange Rate History (Inverse)", currency)
	}

	chartOption := charts.ChartOption{
		Width:  1000,
		Height: 400,
		SeriesList: []charts.Series{
			{
				Type:  charts.ChartTypeLine,
				Data:  charts.NewSeriesDataFromValues(values),
				Label: charts.SeriesLabel{Show: *charts.FalseFlag()},
			},
		},
		Title: charts.TitleOption{
			Text: title,
		},
		Padding: charts.Box{
			Top:    20,
			Left:   20,
			Right:  20,
			Bottom: 20,
		},
		Legend: charts.NewLegendOption([]string{
			"Exchange Rate",
		}, charts.PositionRight),
		XAxis: charts.NewXAxisOption(dates),
		YAxisOptions: []charts.YAxisOption{
			{
				Min: &minWithPadding,
				Max: &maxWithPadding,
			},
		},
		ValueFormatter: func(f float64) string {
			return fmt.Sprintf("%.4f", f)
		},
	}

	p, err := charts.Render(chartOption)
	if err != nil {
		return nil, err
	}

	buf, err := p.Bytes()
	if err != nil {
		return nil, err
	}

	return &buf, nil
}

func FormatCurrentRateMessage(currency string, rate float64, response *schemas.ExchangeRateResponse) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("💱 %s/SGD Exchange Rate\n\n", currency))
	sb.WriteString(fmt.Sprintf("1 SGD → %.4f %s\n", rate, currency))
	sb.WriteString(fmt.Sprintf("1 %s → %.4f SGD\n\n", currency, 1/rate))

	if response != nil {
		sb.WriteString(fmt.Sprintf("Data as of: %s\n", response.Date))
	}

	sb.WriteString("\nUse /fx_chart ")
	sb.WriteString(currency)
	sb.WriteString(" for historical chart")

	return sb.String()
}

func FormatSubscriptionListMessage(subscriptions []schemas.CurrencySubscription) string {
	if len(subscriptions) == 0 {
		return "You have no active subscriptions.\n\nUse /fx_subscribe to create one."
	}

	var sb strings.Builder
	sb.WriteString("📋 Your Currency Subscriptions\n\n")

	for _, sub := range subscriptions {
		sb.WriteString(fmt.Sprintf("💱 %s/SGD\n", sub.Currency))
		if sub.ThresholdAbove != nil {
			sb.WriteString(fmt.Sprintf("  • Alert above: %.4f SGD (1 SGD → %.4f %s)\n", *sub.ThresholdAbove, 1.0/(*sub.ThresholdAbove), sub.Currency))
		}
		if sub.ThresholdBelow != nil {
			sb.WriteString(fmt.Sprintf("  • Alert below: %.4f SGD (1 SGD → %.4f %s)\n", *sub.ThresholdBelow, 1.0/(*sub.ThresholdBelow), sub.Currency))
		}
		if sub.Interval != nil {
			sb.WriteString(fmt.Sprintf("  • Interval: %.4f SGD (1 SGD → %.4f %s)\n", *sub.Interval, 1.0/(*sub.Interval), sub.Currency))
		}
		sb.WriteString("\n")
	}

	sb.WriteString("Use /fx_unsubscribe <currency> to remove a subscription.")
	return sb.String()
}
