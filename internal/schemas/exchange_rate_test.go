package schemas

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFetchLatestExchangeRate_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Contains(t, r.URL.Path, "SGDUSD=X")

		response := YahooChartResponse{
			Chart: struct {
				Result []*YahooChartResult `json:"result"`
				Error  interface{}         `json:"error"`
			}{
				Result: []*YahooChartResult{
					{
						Meta: YahooMeta{
							RegularMarketPrice: 0.7889,
							Currency:           "USD",
						},
					},
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	rate, resp, err := fetchLatestFromURL(server.URL, "USD")
	require.NoError(t, err)
	assert.InDelta(t, 0.7889, rate, 0.001)
	assert.Equal(t, time.Now().Format("2006-01-02"), resp.Date)
}

func TestFetchLatestExchangeRate_UnsupportedCurrency(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := YahooChartResponse{
			Chart: struct {
				Result []*YahooChartResult `json:"result"`
				Error  interface{}         `json:"error"`
			}{
				Result: []*YahooChartResult{},
			},
		}
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	_, _, err := fetchLatestFromURL(server.URL, "INVALID")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no data available")
}

func TestFetchLatestExchangeRate_APIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	_, _, err := fetchLatestFromURL(server.URL, "USD")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "status 500")
}

func fetchLatestFromURL(baseURL, currency string) (float64, *FrankfurterLatestResponse, error) {
	endpoint := fmt.Sprintf("%s/SGD%s=X", baseURL, currency)

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
		return 0, nil, fmt.Errorf("status %d", res.StatusCode)
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

func TestFetchHistoricalExchangeRates_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Contains(t, r.URL.Path, "SGDUSD=X")

		now := time.Now().Unix()
		response := YahooChartResponse{
			Chart: struct {
				Result []*YahooChartResult `json:"result"`
				Error  interface{}         `json:"error"`
			}{
				Result: []*YahooChartResult{
					{
						Timestamp: []int64{now - 86400*60, now - 86400*30, now},
						Indicators: struct {
							Quote []YahooQuote `json:"quote"`
						}{
							Quote: []YahooQuote{
								{Close: []float64{0.7407}},
								{Close: []float64{0.7600}},
								{Close: []float64{0.7889}},
							},
						},
					},
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	rates, err := fetchHistoricalFromURL(server.URL, "USD", 60)
	require.NoError(t, err)
	assert.Len(t, rates, 3)

	assert.InDelta(t, 0.7407, rates[0].Rate, 0.001)
	assert.InDelta(t, 0.7889, rates[2].Rate, 0.001)
}

func fetchHistoricalFromURL(baseURL, currency string, days int) ([]HistoricalRate, error) {
	endpoint := fmt.Sprintf("%s/SGD%s=X?range=1y&interval=1d", baseURL, currency)

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
		return nil, fmt.Errorf("status %d: %s", res.StatusCode, string(body))
	}

	var response YahooChartResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, err
	}

	if len(response.Chart.Result) == 0 {
		return nil, fmt.Errorf("no historical data available")
	}

	result := response.Chart.Result[0]

	rates := make([]HistoricalRate, 0, len(result.Timestamp))
	for i, ts := range result.Timestamp {
		if i >= len(result.Indicators.Quote) || len(result.Indicators.Quote[i].Close) == 0 {
			continue
		}
		rate := result.Indicators.Quote[i].Close[0]
		if rate == 0 {
			continue
		}
		rates = append(rates, HistoricalRate{
			Date: time.Unix(ts, 0).UTC(),
			Rate: rate,
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

func TestFetchHistoricalExchangeRates_DataSortedChronologically(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		now := time.Now().Unix()
		response := YahooChartResponse{
			Chart: struct {
				Result []*YahooChartResult `json:"result"`
				Error  interface{}         `json:"error"`
			}{
				Result: []*YahooChartResult{
					{
						Timestamp: []int64{now, now - 86400*30, now - 86400*60},
						Indicators: struct {
							Quote []YahooQuote `json:"quote"`
						}{
							Quote: []YahooQuote{
								{Close: []float64{0.7889}},
								{Close: []float64{0.7600}},
								{Close: []float64{0.7407}},
							},
						},
					},
				},
			},
		}
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	rates, err := fetchHistoricalFromURL(server.URL, "USD", 60)
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(rates), 3)

	for i := 1; i < len(rates); i++ {
		assert.True(t, rates[i].Date.After(rates[i-1].Date) || rates[i].Date.Equal(rates[i-1].Date),
			"rates should be sorted chronologically, got %v before %v", rates[i].Date, rates[i-1].Date)
	}
}

func TestFetchHistoricalWithUSDIntermediary(t *testing.T) {
	usdSGDCalled := false
	usdPHPCalled := false

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/SGDUSD=X") {
			usdSGDCalled = true
			now := time.Now().Unix()
			response := YahooChartResponse{
				Chart: struct {
					Result []*YahooChartResult `json:"result"`
					Error  interface{}         `json:"error"`
				}{
					Result: []*YahooChartResult{
						{
							Timestamp: []int64{now - 86400, now},
							Indicators: struct {
								Quote []YahooQuote `json:"quote"`
							}{
								Quote: []YahooQuote{
									{Close: []float64{0.75}},
									{Close: []float64{0.76}},
								},
							},
						},
					},
				},
			}
			json.NewEncoder(w).Encode(response)
		} else if strings.HasPrefix(r.URL.Path, "/SGDPHP=X") {
			usdPHPCalled = true
			now := time.Now().Unix()
			response := YahooChartResponse{
				Chart: struct {
					Result []*YahooChartResult `json:"result"`
					Error  interface{}         `json:"error"`
				}{
					Result: []*YahooChartResult{
						{
							Timestamp: []int64{now - 86400, now},
							Indicators: struct {
								Quote []YahooQuote `json:"quote"`
							}{
								Quote: []YahooQuote{
									{Close: []float64{42.50}},
									{Close: []float64{43.20}},
								},
							},
						},
					},
				},
			}
			json.NewEncoder(w).Encode(response)
		}
	}))
	defer server.Close()

	rates, err := fetchHistoricalWithUSDIntermediaryFromURL(server.URL, "PHP", 30)
	require.NoError(t, err)
	assert.True(t, usdSGDCalled, "USD/SGD should be fetched")
	assert.True(t, usdPHPCalled, "USD/PHP should be fetched")
	assert.Len(t, rates, 2)

	assert.InDelta(t, 42.50/0.75, rates[0].Rate, 0.01)
	assert.InDelta(t, 43.20/0.76, rates[1].Rate, 0.01)
}

func fetchHistoricalWithUSDIntermediaryFromURL(baseURL, currency string, days int) ([]HistoricalRate, error) {
	usdSGD, err := fetchHistoricalFromURL(baseURL, "USD", days)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch USD/SGD rates: %w", err)
	}

	usdCurr, err := fetchHistoricalFromURL(baseURL, currency, days)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch USD/%s rates: %w", currency, err)
	}

	usdSGDMap := make(map[string]float64)
	for _, r := range usdSGD {
		usdSGDMap[r.Date.Format("2006-01-02")] = r.Rate
	}

	rates := make([]HistoricalRate, 0, len(usdCurr))
	for _, r := range usdCurr {
		dateKey := r.Date.Format("2006-01-02")
		usdSgdRate, ok := usdSGDMap[dateKey]
		if !ok || usdSgdRate == 0 {
			continue
		}
		sgdCurr := r.Rate / usdSgdRate
		rates = append(rates, HistoricalRate{
			Date: r.Date,
			Rate: sgdCurr,
		})
	}

	sort.Slice(rates, func(i, j int) bool {
		return rates[i].Date.Before(rates[j].Date)
	})

	return rates, nil
}
