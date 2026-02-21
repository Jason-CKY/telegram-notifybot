package schemas

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"sort"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFetchLatestExchangeRate_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "SGD", r.URL.Query().Get("from"))
		assert.Equal(t, "USD", r.URL.Query().Get("to"))

		response := FrankfurterLatestResponse{
			Amount: 1.0,
			Base:   "SGD",
			Date:   "2026-02-20",
			Rates:  map[string]float64{"USD": 1.2678},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	rate, resp, err := fetchLatestExchangeRateFromURL(server.URL, "USD")
	require.NoError(t, err)
	assert.InDelta(t, 0.7889, rate, 0.001)
	assert.Equal(t, "2026-02-20", resp.Date)
}

func TestFetchLatestExchangeRate_UnsupportedCurrency(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := FrankfurterLatestResponse{
			Amount: 1.0,
			Base:   "SGD",
			Date:   "2026-02-20",
			Rates:  map[string]float64{},
		}
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	_, _, err := fetchLatestExchangeRateFromURL(server.URL, "INVALID")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not available")
}

func TestFetchLatestExchangeRate_APIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	_, _, err := fetchLatestExchangeRateFromURL(server.URL, "USD")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "status 500")
}

func fetchLatestExchangeRateFromURL(baseURL, currency string) (float64, *FrankfurterLatestResponse, error) {
	endpoint := baseURL + "?from=SGD&to=" + currency

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

	var body []byte
	body, _ = io.ReadAll(res.Body)
	if res.StatusCode != 200 {
		return 0, nil, fmt.Errorf("status %d", res.StatusCode)
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

func TestFetchHistoricalExchangeRates_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "SGD", r.URL.Query().Get("from"))
		assert.Equal(t, "USD", r.URL.Query().Get("to"))

		response := FrankfurterHistoricalResponse{
			Amount:    1.0,
			Base:      "SGD",
			StartDate: "2026-01-01",
			EndDate:   "2026-02-20",
			Rates: map[string]map[string]float64{
				"2026-02-20": {"USD": 1.2678},
				"2026-02-19": {"USD": 1.2690},
				"2026-01-15": {"USD": 1.3500},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	rates, err := fetchHistoricalFromURL(server.URL+"?from=SGD&to=USD", "USD", 60)
	require.NoError(t, err)
	assert.Len(t, rates, 3)

	assert.Equal(t, time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC), rates[0].Date)
	assert.InDelta(t, 0.7407, rates[0].Rate, 0.001)

	assert.Equal(t, time.Date(2026, 2, 20, 0, 0, 0, 0, time.UTC), rates[2].Date)
	assert.InDelta(t, 0.7889, rates[2].Rate, 0.001)
}

func fetchHistoricalFromURL(url, currency string, days int) ([]HistoricalRate, error) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
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
		return nil, fmt.Errorf("status %d: %s", res.StatusCode, string(body))
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

func TestFetchHistoricalExchangeRates_DataSortedChronologically(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := FrankfurterHistoricalResponse{
			Amount:    1.0,
			Base:      "SGD",
			StartDate: "2026-01-01",
			EndDate:   "2026-02-20",
			Rates: map[string]map[string]float64{
				"2026-02-10": {"USD": 1.2800},
				"2026-01-15": {"USD": 1.3500},
				"2026-02-20": {"USD": 1.2678},
			},
		}
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	rates, err := fetchHistoricalFromURL(server.URL+"?from=SGD&to=USD", "USD", 60)
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(rates), 3)

	for i := 1; i < len(rates); i++ {
		assert.True(t, rates[i].Date.After(rates[i-1].Date) || rates[i].Date.Equal(rates[i-1].Date),
			"rates should be sorted chronologically, got %v before %v", rates[i].Date, rates[i-1].Date)
	}
}
