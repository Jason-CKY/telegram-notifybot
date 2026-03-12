// Package schemas provides exchange rate fetching via Yahoo Finance API.
//
// Yahoo Finance API Notes:
//   - Uses pairs like SGDUSD=X for SGD/USD exchange rate
//   - Most currencies have direct SGD pairs (e.g., SGDEUR=X, SGDJPY=X)
//   - Some currencies (VND, PHP) don't have direct SGD pairs and require USD intermediary:
//     SGD -> USD -> Currency conversion using both USD/SGD and Currency/USD rates
//
// Exchange Rate Calculation:
//   - For direct pairs: Rate = Yahoo returns SGD per unit of foreign currency
//   - For USD intermediary (VND, PHP):
//     SGD/XXX = (USD/SGD) / (USD/XXX) = USD_SGD_rate / XXX_USD_rate
package schemas

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"sort"
	"time"
)

const (
	YahooFinanceURL = "https://query1.finance.yahoo.com/v8/finance/chart"
)

var (
	yahooFinanceURL = YahooFinanceURL
)

func SetYahooFinanceURL(url string) {
	yahooFinanceURL = url
}

func GetYahooFinanceURL() string {
	return yahooFinanceURL
}

type YahooChartResponse struct {
	Chart struct {
		Result []*YahooChartResult `json:"result"`
		Error  interface{}         `json:"error"`
	} `json:"chart"`
}

type YahooChartResult struct {
	Meta       YahooMeta `json:"meta"`
	Timestamp  []int64   `json:"timestamp"`
	Indicators struct {
		Quote []YahooQuote `json:"quote"`
	} `json:"indicators"`
}

type YahooMeta struct {
	RegularMarketPrice float64 `json:"regularMarketPrice"`
	Currency           string  `json:"currency"`
}

type YahooQuote struct {
	Close []float64 `json:"close"`
}

type HistoricalRate struct {
	Date time.Time
	Rate float64
}

type ExchangeRateResponse struct {
	Amount float64            `json:"amount"`
	Base   string             `json:"base"`
	Date   string             `json:"date"`
	Rates  map[string]float64 `json:"rates"`
}

func FetchLatestExchangeRate(currency string) (float64, *ExchangeRateResponse, error) {
	// VND and PHP don't have direct SGD pairs in Yahoo Finance
	// Use USD as intermediary: SGD -> USD -> VND/PHP
	if currency == "VND" || currency == "PHP" {
		return fetchLatestWithUSDIntermediary(currency)
	}
	return fetchLatestFromYahoo(currency)
}

func fetchLatestFromYahoo(currency string) (float64, *ExchangeRateResponse, error) {
	return fetchLatestFromYahooWithPair("SGD", currency)
}

func fetchLatestFromYahooWithPair(base, quote string) (float64, *ExchangeRateResponse, error) {
	endpoint := fmt.Sprintf("%s/%s%s=X", yahooFinanceURL, base, quote)

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
		return 0, nil, fmt.Errorf("Yahoo Finance API error: status %d, body: %s", res.StatusCode, string(body))
	}

	var response YahooChartResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return 0, nil, err
	}

	if len(response.Chart.Result) == 0 {
		return 0, nil, fmt.Errorf("no data available for pair: %s/%s", base, quote)
	}

	result := response.Chart.Result[0]
	rate := result.Meta.RegularMarketPrice

	slog.Debug("fx rate fetched", "pair", fmt.Sprintf("%s/%s", base, quote), "rate", rate)

	if rate == 0 {
		return 0, nil, fmt.Errorf("rate not available for pair: %s/%s", base, quote)
	}

	exchangeResp := &ExchangeRateResponse{
		Amount: 1.0,
		Base:   base,
		Date:   time.Now().Format("2006-01-02"),
		Rates:  map[string]float64{quote: rate},
	}

	return rate, exchangeResp, nil
}

func fetchLatestWithUSDIntermediary(currency string) (float64, *ExchangeRateResponse, error) {
	// Fetch USD/SGD rate (how many SGD per 1 USD)
	usdSGD, _, err := fetchLatestFromYahooWithPair("USD", "SGD")
	if err != nil {
		return 0, nil, fmt.Errorf("failed to fetch USD/SGD rate: %w", err)
	}

	// Fetch USD/currency rate (how many currency per 1 USD)
	// Yahoo ticker format: USD{currency}=X (e.g., USDVND=X, USDPHP=X)
	// This returns how many VND/PHP you get per 1 USD
	usdCurrency, _, err := fetchLatestFromYahooWithPair("USD", currency)
	if err != nil {
		return 0, nil, fmt.Errorf("failed to fetch USD/%s rate: %w", currency, err)
	}

	// Convert: SGD -> USD -> Currency
	// 1 SGD = (1/usdSGD) USD
	// 1 USD = usdCurrency Currency
	// Therefore: 1 SGD = (1/usdSGD) * usdCurrency = usdCurrency / usdSGD
	rate := usdCurrency / usdSGD

	slog.Debug("fx rate via USD intermediary", "currency", currency, "usd_sgd", usdSGD, "usd_currency", usdCurrency, "result", rate)

	exchangeResp := &ExchangeRateResponse{
		Amount: 1.0,
		Base:   "SGD",
		Date:   time.Now().Format("2006-01-02"),
		Rates:  map[string]float64{currency: rate},
	}

	return rate, exchangeResp, nil
}

func FetchHistoricalExchangeRates(currency string, days int) ([]HistoricalRate, error) {
	if days <= 0 {
		days = 365
	}
	if days > 3650 {
		days = 3650
	}

	// VND and PHP require USD as intermediary because:
	// 1. Yahoo Finance doesn't provide direct SGD/VND or SGD/PHP pairs
	// 2. We fetch USD/SGD and currency/USD separately, then compute: SGD -> USD -> VND/PHP
	// 3. Formula: SGD/VND = (USD/VND) / (USD/SGD) = (1/USD/VND rate) / USD/SGD rate
	//    Alternatively: SGD/VND = 1 / (USD/SGD * VND/USD) where VND/USD = 1/(USD/VND)
	//    Simplified: SGD/VND = USD_SGD_rate / VND_USD_rate
	if currency == "VND" || currency == "PHP" {
		return fetchHistoricalWithUSDIntermediary(currency, days)
	}

	return fetchHistoricalFromYahoo(currency, days)
}

func fetchHistoricalFromYahoo(currency string, days int) ([]HistoricalRate, error) {
	return fetchHistoricalFromYahooWithPair("SGD", currency, days)
}

func fetchHistoricalFromYahooWithPair(base, quote string, days int) ([]HistoricalRate, error) {
	var rangeParam string
	if days <= 5 {
		rangeParam = "5d"
	} else if days <= 30 {
		rangeParam = "1mo"
	} else if days <= 90 {
		rangeParam = "3mo"
	} else if days <= 180 {
		rangeParam = "6mo"
	} else if days <= 365 {
		rangeParam = "1y"
	} else if days <= 730 {
		rangeParam = "2y"
	} else {
		rangeParam = "5y"
	}

	endpoint := fmt.Sprintf("%s/%s%s=X?range=%s&interval=1d", yahooFinanceURL, base, quote, rangeParam)

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
		return nil, fmt.Errorf("Yahoo Finance API error: status %d, body: %s", res.StatusCode, string(body))
	}

	var response YahooChartResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, err
	}

	if len(response.Chart.Result) == 0 {
		return nil, fmt.Errorf("no historical data available for currency: %s/%s", base, quote)
	}

	result := response.Chart.Result[0]

	if len(result.Indicators.Quote) == 0 || len(result.Indicators.Quote[0].Close) == 0 {
		return nil, fmt.Errorf("no quote data available for currency: %s/%s", base, quote)
	}

	rates := make([]HistoricalRate, 0, len(result.Timestamp))
	closes := result.Indicators.Quote[0].Close
	for i, ts := range result.Timestamp {
		if i >= len(closes) || closes[i] == 0 {
			continue
		}
		rates = append(rates, HistoricalRate{
			Date: time.Unix(ts, 0).UTC(),
			Rate: closes[i],
		})
	}

	sort.Slice(rates, func(i, j int) bool {
		return rates[i].Date.Before(rates[j].Date)
	})

	if len(rates) > days {
		rates = rates[len(rates)-days:]
	}

	return rates, nil
}

func fetchHistoricalWithUSDIntermediary(currency string, days int) ([]HistoricalRate, error) {
	usdSGD, err := fetchHistoricalFromYahooWithPair("USD", "SGD", days)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch USD/SGD rates: %w", err)
	}
	slog.Debug("USD/SGD rates fetched", "points", len(usdSGD))

	// Use USD{currency}=X format (e.g., USDVND=X, USDPHP=X)
	// This returns how many currency units per 1 USD
	usdCurrency, err := fetchHistoricalFromYahooWithPair("USD", currency, days)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch USD/%s rates: %w", currency, err)
	}
	slog.Debug("USD/currency rates fetched", "currency", currency, "points", len(usdCurrency))

	usdSGDMap := make(map[string]float64)
	for _, r := range usdSGD {
		usdSGDMap[r.Date.Format("2006-01-02")] = r.Rate
	}
	slog.Debug("USD/SGD map sample", "sample", usdSGDMap)

	rates := make([]HistoricalRate, 0, len(usdCurrency))
	for _, r := range usdCurrency {
		dateKey := r.Date.Format("2006-01-02")
		usdSgdRate, ok := usdSGDMap[dateKey]
		if !ok || usdSgdRate == 0 || r.Rate == 0 {
			continue
		}
		// usdSgdRate = SGD per 1 USD
		// r.Rate = currency per 1 USD (e.g., 25000 VND per USD)
		// To get currency per SGD: currency/SGD = (currency/USD) / (SGD/USD)
		currSgd := r.Rate / usdSgdRate
		slog.Debug("calculated rate", "date", dateKey, "usd_sgd", usdSgdRate, "currency_usd", r.Rate, "result", currSgd)
		rates = append(rates, HistoricalRate{
			Date: r.Date,
			Rate: currSgd,
		})
	}

	sort.Slice(rates, func(i, j int) bool {
		return rates[i].Date.Before(rates[j].Date)
	})

	return rates, nil
}
