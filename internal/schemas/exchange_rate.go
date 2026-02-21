package schemas

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const MASExchangeRatesResourceID = "10eafb90-11a2-4fbd-b7a7-ac15a42d60b6"
const MASExchangeRatesAPIURL = "https://eservices.mas.gov.sg/api/action/datastore/search.json"

type ExchangeRateRecord struct {
	EndOfMonth string `json:"end_of_month"`
	USDSGD     string `json:"usd_sgd"`
	EURSGD     string `json:"eur_sgd"`
	GBPSGD     string `json:"gbp_sgd"`
	JPYSGD100  string `json:"jpy_sgd_100"`
	MYRSGD     string `json:"myr_sgd"`
	HKDSGD     string `json:"hkd_sgd"`
	AUDSGD     string `json:"aud_sgd"`
	KRWSGD100  string `json:"krw_sgd_100"`
	TWDSGD     string `json:"twd_sgd"`
	IDRSGD1000 string `json:"idr_sgd_1000"`
	THBSGD     string `json:"thb_sgd"`
	CNYSGD     string `json:"cny_sgd"`
	INRSGD     string `json:"inr_sgd"`
	PHPSGD     string `json:"php_sgd"`
}

func (r ExchangeRateRecord) GetRate(currency string) (float64, error) {
	var rateStr string
	var divisor float64 = 1.0

	switch strings.ToUpper(currency) {
	case "USD":
		rateStr = r.USDSGD
	case "EUR":
		rateStr = r.EURSGD
	case "GBP":
		rateStr = r.GBPSGD
	case "JPY":
		rateStr = r.JPYSGD100
		divisor = 100.0
	case "MYR":
		rateStr = r.MYRSGD
	case "HKD":
		rateStr = r.HKDSGD
	case "AUD":
		rateStr = r.AUDSGD
	case "KRW":
		rateStr = r.KRWSGD100
		divisor = 100.0
	case "TWD":
		rateStr = r.TWDSGD
	case "IDR":
		rateStr = r.IDRSGD1000
		divisor = 1000.0
	case "THB":
		rateStr = r.THBSGD
	case "CNY":
		rateStr = r.CNYSGD
	case "INR":
		rateStr = r.INRSGD
	case "PHP":
		rateStr = r.PHPSGD
	default:
		return 0, fmt.Errorf("unsupported currency: %s", currency)
	}

	if rateStr == "" || rateStr == "-" {
		return 0, fmt.Errorf("rate not available for currency: %s", currency)
	}

	rate, err := strconv.ParseFloat(rateStr, 64)
	if err != nil {
		return 0, fmt.Errorf("error parsing rate for %s: %v", currency, err)
	}

	return rate / divisor, nil
}

func (r ExchangeRateRecord) GetMonth() (time.Time, error) {
	return time.Parse("2006-01", r.EndOfMonth)
}

type MASExchangeRatesResponse struct {
	Success bool `json:"success"`
	Result  struct {
		Total   int                  `json:"total"`
		Records []ExchangeRateRecord `json:"records"`
	} `json:"result"`
}

type HistoricalRate struct {
	Date time.Time
	Rate float64
}

func FetchLatestExchangeRate(currency string) (float64, *ExchangeRateRecord, error) {
	endpoint := fmt.Sprintf("%s?resource_id=%s&limit=1&sort=end_of_month desc",
		MASExchangeRatesAPIURL, MASExchangeRatesResourceID)

	req, err := http.NewRequest(http.MethodGet, endpoint, nil)
	if err != nil {
		return 0, nil, err
	}
	req.Header.Set("User-Agent", "Telegram-NotifyBot/1.0")

	client := &http.Client{}
	res, err := client.Do(req)
	if err != nil {
		return 0, nil, err
	}
	defer res.Body.Close()

	body, _ := io.ReadAll(res.Body)
	if res.StatusCode != 200 {
		return 0, nil, fmt.Errorf("MAS API error: status %d, body: %s", res.StatusCode, string(body))
	}

	var response MASExchangeRatesResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return 0, nil, err
	}

	if !response.Success || len(response.Result.Records) == 0 {
		return 0, nil, fmt.Errorf("no exchange rate data available")
	}

	record := response.Result.Records[0]
	rate, err := record.GetRate(currency)
	if err != nil {
		return 0, nil, err
	}

	return rate, &record, nil
}

func FetchHistoricalExchangeRates(currency string, months int) ([]HistoricalRate, error) {
	endpoint := fmt.Sprintf("%s?resource_id=%s&limit=%d&sort=end_of_month desc",
		MASExchangeRatesAPIURL, MASExchangeRatesResourceID, months)

	req, err := http.NewRequest(http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Telegram-NotifyBot/1.0")

	client := &http.Client{}
	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	body, _ := io.ReadAll(res.Body)
	if res.StatusCode != 200 {
		return nil, fmt.Errorf("MAS API error: status %d, body: %s", res.StatusCode, string(body))
	}

	var response MASExchangeRatesResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, err
	}

	if !response.Success {
		return nil, fmt.Errorf("MAS API returned success=false")
	}

	rates := make([]HistoricalRate, 0, len(response.Result.Records))
	for _, record := range response.Result.Records {
		rate, err := record.GetRate(currency)
		if err != nil {
			continue
		}
		date, err := record.GetMonth()
		if err != nil {
			continue
		}
		rates = append(rates, HistoricalRate{
			Date: date,
			Rate: rate,
		})
	}

	for i, j := 0, len(rates)-1; i < j; i, j = i+1, j-1 {
		rates[i], rates[j] = rates[j], rates[i]
	}

	return rates, nil
}
