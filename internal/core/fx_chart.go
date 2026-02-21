package core

import (
	"fmt"
	"strings"

	"github.com/Jason-CKY/telegram-notifybot/internal/schemas"
	"github.com/vicanso/go-charts/v2"
)

func GenerateExchangeRateChart(rates []schemas.HistoricalRate, currency string) (*[]byte, error) {
	if len(rates) == 0 {
		return nil, fmt.Errorf("no historical rates available")
	}

	values := make([]float64, len(rates))
	dates := make([]string, len(rates))

	for i, r := range rates {
		values[i] = r.Rate
		dates[i] = r.Date.Format("Jan 06")
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
			Text: fmt.Sprintf("%s/SGD Exchange Rate History", currency),
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

func FormatCurrentRateMessage(currency string, rate float64, record *schemas.ExchangeRateRecord) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("💱 *%s/SGD Exchange Rate*\n\n", currency))
	sb.WriteString(fmt.Sprintf("Current Rate: *%.4f SGD*\n\n", rate))

	if record != nil {
		sb.WriteString(fmt.Sprintf("Data as of: %s\n", record.EndOfMonth))
	}

	sb.WriteString("\n_Use /fx_chart ")
	sb.WriteString(currency)
	sb.WriteString(" for historical chart_")

	return sb.String()
}

func FormatSubscriptionListMessage(subscriptions []schemas.CurrencySubscription) string {
	if len(subscriptions) == 0 {
		return "You have no active subscriptions.\n\nUse /fx_subscribe to create one."
	}

	var sb strings.Builder
	sb.WriteString("📋 *Your Currency Subscriptions*\n\n")

	for _, sub := range subscriptions {
		sb.WriteString(fmt.Sprintf("💱 *%s/SGD*\n", sub.Currency))
		if sub.ThresholdAbove != nil {
			sb.WriteString(fmt.Sprintf("  • Alert above: %.4f SGD\n", *sub.ThresholdAbove))
		}
		if sub.ThresholdBelow != nil {
			sb.WriteString(fmt.Sprintf("  • Alert below: %.4f SGD\n", *sub.ThresholdBelow))
		}
		if sub.Interval != nil {
			sb.WriteString(fmt.Sprintf("  • Interval: %.4f SGD\n", *sub.Interval))
		}
		sb.WriteString("\n")
	}

	sb.WriteString("Use /fx_unsubscribe <currency> to remove a subscription.")
	return sb.String()
}
