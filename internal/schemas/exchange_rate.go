package schemas

import (
	"encoding/json"
	"fmt"
	"io"
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

type FrankfurterLatestResponse struct {
	Amount float64            `json:"amount"`
	Base   string             `json:"base"`
	Date   string             `json:"date"`
	Rates  map[string]float64 `json:"rates"`
}

func FetchLatestExchangeRate(currency string) (float64, *FrankfurterLatestResponse, error) {
	return fetchLatestFromYahoo(currency)
}

func fetchLatestFromYahoo(currency string) (float64, *FrankfurterLatestResponse, error) {
	endpoint := fmt.Sprintf("%s/SGD%s=X", yahooFinanceURL, currency)

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
		return 0, nil, fmt.Errorf("no data available for currency: %s", currency)
	}

	result := response.Chart.Result[0]
	rate := result.Meta.RegularMarketPrice

	fmt.Printf("[DEBUG] /fx rate for %s: %f SGD/%s\n", currency, rate, currency)

	if rate == 0 {
		return 0, nil, fmt.Errorf("rate not available for currency: %s", currency)
	}

	frankfurterResp := &FrankfurterLatestResponse{
		Amount: 1.0,
		Base:   "SGD",
		Date:   time.Now().Format("2006-01-02"),
		Rates:  map[string]float64{currency: rate},
	}

	return rate, frankfurterResp, nil
}

func FetchHistoricalExchangeRates(currency string, days int) ([]HistoricalRate, error) {
	if days <= 0 {
		days = 365
	}
	if days > 3650 {
		days = 3650
	}

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
	fmt.Printf("[DEBUG] USD/SGD rates fetched: %d points\n", len(usdSGD))

	currUSD, err := fetchHistoricalFromYahooWithPair(currency, "USD", days)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch %s/USD rates: %w", currency, err)
	}
	fmt.Printf("[DEBUG] %s/USD rates fetched: %d points\n", currency, len(currUSD))

	usdSGDMap := make(map[string]float64)
	for _, r := range usdSGD {
		usdSGDMap[r.Date.Format("2006-01-02")] = r.Rate
	}
	fmt.Printf("[DEBUG] USD/SGD map sample: %v\n", usdSGDMap)

	rates := make([]HistoricalRate, 0, len(currUSD))
	for _, r := range currUSD {
		dateKey := r.Date.Format("2006-01-02")
		usdSgdRate, ok := usdSGDMap[dateKey]
		if !ok || usdSgdRate == 0 || r.Rate == 0 {
			continue
		}
		currUsdRate := r.Rate
		if currUsdRate > 1 {
			currUsdRate = 1 / currUsdRate
		}
		sgdCurr := usdSgdRate * currUsdRate
		currSgd := 1 / sgdCurr
		fmt.Printf("[DEBUG] %s: USD/SGD=%f, %s/USD raw=%f, %s/USD=%f, %s/SGD=%f\n", dateKey, usdSgdRate, currency, r.Rate, currency, currUsdRate, currency, currSgd)
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
