package schemas

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"time"
)

const FrankfurterAPIURL = "https://api.frankfurter.dev/v1"

type FrankfurterLatestResponse struct {
	Amount float64            `json:"amount"`
	Base   string             `json:"base"`
	Date   string             `json:"date"`
	Rates  map[string]float64 `json:"rates"`
}

type FrankfurterHistoricalResponse struct {
	Amount    float64                       `json:"amount"`
	Base      string                        `json:"base"`
	StartDate string                        `json:"start_date"`
	EndDate   string                        `json:"end_date"`
	Rates     map[string]map[string]float64 `json:"rates"`
}

type HistoricalRate struct {
	Date time.Time
	Rate float64
}

func FetchLatestExchangeRate(currency string) (float64, *FrankfurterLatestResponse, error) {
	endpoint := fmt.Sprintf("%s/latest?from=SGD&to=%s", FrankfurterAPIURL, currency)

	req, err := http.NewRequest(http.MethodGet, endpoint, nil)
	if err != nil {
		return 0, nil, err
	}
	req.Header.Set("User-Agent", "Telegram-NotifyBot/1.0")

	client := &http.Client{Timeout: 30 * time.Second}
	res, err := client.Do(req)
	if err != nil {
		return 0, nil, err
	}
	defer res.Body.Close()

	body, _ := io.ReadAll(res.Body)
	if res.StatusCode != 200 {
		return 0, nil, fmt.Errorf("Frankfurter API error: status %d, body: %s", res.StatusCode, string(body))
	}

	var response FrankfurterLatestResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return 0, nil, err
	}

	rate, ok := response.Rates[currency]
	if !ok {
		return 0, nil, fmt.Errorf("rate not available for currency: %s", currency)
	}

	return 1.0 / rate, &response, nil
}

func FetchHistoricalExchangeRates(currency string, days int) ([]HistoricalRate, error) {
	if days <= 0 {
		days = 365
	}
	if days > 3650 {
		days = 3650
	}

	endDate := time.Now()
	startDate := endDate.AddDate(0, 0, -days)

	endpoint := fmt.Sprintf("%s/%s..%s?from=SGD&to=%s",
		FrankfurterAPIURL,
		startDate.Format("2006-01-02"),
		endDate.Format("2006-01-02"),
		currency)

	req, err := http.NewRequest(http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Telegram-NotifyBot/1.0")

	client := &http.Client{Timeout: 30 * time.Second}
	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	body, _ := io.ReadAll(res.Body)
	if res.StatusCode != 200 {
		return nil, fmt.Errorf("Frankfurter API error: status %d, body: %s", res.StatusCode, string(body))
	}

	var response FrankfurterHistoricalResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, err
	}

	rates := make([]HistoricalRate, 0, len(response.Rates))
	for dateStr, rateMap := range response.Rates {
		rate, ok := rateMap[currency]
		if !ok {
			continue
		}
		date, err := time.Parse("2006-01-02", dateStr)
		if err != nil {
			continue
		}
		rates = append(rates, HistoricalRate{
			Date: date,
			Rate: 1.0 / rate,
		})
	}

	sort.Slice(rates, func(i, j int) bool {
		return rates[i].Date.Before(rates[j].Date)
	})

	return rates, nil
}
