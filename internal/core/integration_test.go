package core

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/Jason-CKY/telegram-notifybot/internal/schemas"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIntegration_FetchHistoricalRatesAndGenerateChart(t *testing.T) {
	originalURL := schemas.GetYahooFinanceURL()
	defer func() {
		schemas.SetYahooFinanceURL(originalURL)
	}()

	now := time.Now().Unix()
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Contains(t, r.URL.Path, "SGDUSD=X")

		response := schemas.YahooChartResponse{
			Chart: struct {
				Result []*schemas.YahooChartResult `json:"result"`
				Error  interface{}                 `json:"error"`
			}{
				Result: []*schemas.YahooChartResult{
					{
						Timestamp: []int64{
							now - 86400*180,
							now - 86400*150,
							now - 86400*120,
							now - 86400*90,
							now - 86400*60,
							now - 86400*30,
							now - 86400*15,
							now,
						},
						Indicators: struct {
							Quote []schemas.YahooQuote `json:"quote"`
						}{
							Quote: []schemas.YahooQuote{
								{
									Close: []float64{0.7200, 0.7350, 0.7400, 0.7450, 0.7500, 0.7600, 0.7700, 0.7800},
								},
							},
						},
					},
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer mockServer.Close()

	schemas.SetYahooFinanceURL(mockServer.URL)

	rates, err := schemas.FetchHistoricalExchangeRates("USD", 180)
	require.NoError(t, err)
	require.Len(t, rates, 8, "should have 8 data points")

	for i := 1; i < len(rates); i++ {
		assert.True(t, rates[i].Date.After(rates[i-1].Date) || rates[i].Date.Equal(rates[i-1].Date),
			"rates should be sorted chronologically")
	}

	chartBuf, err := GenerateExchangeRateChart(rates, "USD", false)
	require.NoError(t, err)
	require.NotNil(t, chartBuf)
	assert.Greater(t, len(*chartBuf), 1000, "chart should have substantial PNG data")

	assert.Equal(t, 0.7200, rates[0].Rate)
	assert.Equal(t, 0.7800, rates[7].Rate)
}

func TestIntegration_FetchHistoricalRatesMultipleCurrencies(t *testing.T) {
	originalURL := schemas.GetYahooFinanceURL()
	defer func() {
		schemas.SetYahooFinanceURL(originalURL)
	}()

	now := time.Now().Unix()
	currencies := []string{"EUR", "GBP", "JPY"}

	for _, currency := range currencies {
		t.Run(currency, func(t *testing.T) {
			mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Contains(t, r.URL.Path, fmt.Sprintf("SGD%s=X", currency))

				response := schemas.YahooChartResponse{
					Chart: struct {
						Result []*schemas.YahooChartResult `json:"result"`
						Error  interface{}                 `json:"error"`
					}{
						Result: []*schemas.YahooChartResult{
							{
								Timestamp: []int64{now - 86400*30, now - 86400*15, now},
								Indicators: struct {
									Quote []schemas.YahooQuote `json:"quote"`
								}{
									Quote: []schemas.YahooQuote{
										{Close: []float64{0.65, 0.67, 0.69}},
									},
								},
							},
						},
					},
				}
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(response)
			}))
			defer mockServer.Close()

			schemas.SetYahooFinanceURL(mockServer.URL)

			rates, err := schemas.FetchHistoricalExchangeRates(currency, 30)
			require.NoError(t, err)
			require.Len(t, rates, 3)

			chartBuf, err := GenerateExchangeRateChart(rates, currency, false)
			require.NoError(t, err)
			assert.Greater(t, len(*chartBuf), 500)
		})
	}
}

func TestIntegration_FetchHistoricalRatesWithInverseChart(t *testing.T) {
	originalURL := schemas.GetYahooFinanceURL()
	defer func() {
		schemas.SetYahooFinanceURL(originalURL)
	}()

	now := time.Now().Unix()
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Contains(t, r.URL.Path, "SGDEUR=X")

		response := schemas.YahooChartResponse{
			Chart: struct {
				Result []*schemas.YahooChartResult `json:"result"`
				Error  interface{}                 `json:"error"`
			}{
				Result: []*schemas.YahooChartResult{
					{
						Timestamp: []int64{now - 86400*60, now - 86400*30, now},
						Indicators: struct {
							Quote []schemas.YahooQuote `json:"quote"`
						}{
							Quote: []schemas.YahooQuote{
								{Close: []float64{1.50, 1.55, 1.60}},
							},
						},
					},
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer mockServer.Close()

	schemas.SetYahooFinanceURL(mockServer.URL)

	rates, err := schemas.FetchHistoricalExchangeRates("EUR", 60)
	require.NoError(t, err)

	chartBuf, err := GenerateExchangeRateChart(rates, "EUR", true)
	require.NoError(t, err)
	assert.Greater(t, len(*chartBuf), 500)
}

func TestIntegration_HistoricalRatesDateRangeFiltering(t *testing.T) {
	originalURL := schemas.GetYahooFinanceURL()
	defer func() {
		schemas.SetYahooFinanceURL(originalURL)
	}()

	now := time.Now().Unix()
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := schemas.YahooChartResponse{
			Chart: struct {
				Result []*schemas.YahooChartResult `json:"result"`
				Error  interface{}                 `json:"error"`
			}{
				Result: []*schemas.YahooChartResult{
					{
						Timestamp: []int64{
							now - 86400*200,
							now - 86400*150,
							now - 86400*100,
							now - 86400*50,
							now,
						},
						Indicators: struct {
							Quote []schemas.YahooQuote `json:"quote"`
						}{
							Quote: []schemas.YahooQuote{
								{Close: []float64{0.70, 0.72, 0.74, 0.76, 0.78}},
							},
						},
					},
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer mockServer.Close()

	schemas.SetYahooFinanceURL(mockServer.URL)

	rates90, err := schemas.FetchHistoricalExchangeRates("USD", 90)
	require.NoError(t, err)
	assert.LessOrEqual(t, len(rates90), 5, "should filter to requested days")

	rates180, err := schemas.FetchHistoricalExchangeRates("USD", 180)
	require.NoError(t, err)
	assert.LessOrEqual(t, len(rates180), 5, "should include more data points")
}

func TestIntegration_FetchHistoricalRatesUnsupportedCurrency(t *testing.T) {
	originalURL := schemas.GetYahooFinanceURL()
	defer func() {
		schemas.SetYahooFinanceURL(originalURL)
	}()

	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := schemas.YahooChartResponse{
			Chart: struct {
				Result []*schemas.YahooChartResult `json:"result"`
				Error  interface{}                 `json:"error"`
			}{
				Result: []*schemas.YahooChartResult{},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer mockServer.Close()

	schemas.SetYahooFinanceURL(mockServer.URL)

	_, err := schemas.FetchHistoricalExchangeRates("INVALID", 30)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no historical data available")
}

func TestIntegration_FetchHistoricalRatesAPIErrors(t *testing.T) {
	originalURL := schemas.GetYahooFinanceURL()
	defer func() {
		schemas.SetYahooFinanceURL(originalURL)
	}()

	t.Run("server error", func(t *testing.T) {
		mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer mockServer.Close()

		schemas.SetYahooFinanceURL(mockServer.URL)

		_, err := schemas.FetchHistoricalExchangeRates("USD", 30)
		assert.Error(t, err)
	})

	t.Run("malformed response", func(t *testing.T) {
		mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{invalid json`))
		}))
		defer mockServer.Close()

		schemas.SetYahooFinanceURL(mockServer.URL)

		_, err := schemas.FetchHistoricalExchangeRates("USD", 30)
		assert.Error(t, err)
	})

	t.Run("empty result", func(t *testing.T) {
		mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			response := schemas.YahooChartResponse{
				Chart: struct {
					Result []*schemas.YahooChartResult `json:"result"`
					Error  interface{}                 `json:"error"`
				}{
					Result: []*schemas.YahooChartResult{},
				},
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(response)
		}))
		defer mockServer.Close()

		schemas.SetYahooFinanceURL(mockServer.URL)

		_, err := schemas.FetchHistoricalExchangeRates("USD", 30)
		assert.Error(t, err)
	})
}

func TestIntegration_ChartGenerationWithDifferentRanges(t *testing.T) {
	rates := []schemas.HistoricalRate{
		{Date: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC), Rate: 1.3000},
		{Date: time.Date(2025, 2, 1, 0, 0, 0, 0, time.UTC), Rate: 1.3100},
		{Date: time.Date(2025, 3, 1, 0, 0, 0, 0, time.UTC), Rate: 1.3050},
		{Date: time.Date(2025, 4, 1, 0, 0, 0, 0, time.UTC), Rate: 1.3200},
		{Date: time.Date(2025, 5, 1, 0, 0, 0, 0, time.UTC), Rate: 1.3150},
		{Date: time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC), Rate: 1.3300},
	}

	chartBuf, err := GenerateExchangeRateChart(rates, "USD", false)
	require.NoError(t, err)
	require.NotNil(t, chartBuf)
	assert.Greater(t, len(*chartBuf), 1000)

	chartBufInverse, err := GenerateExchangeRateChart(rates, "USD", true)
	require.NoError(t, err)
	assert.Greater(t, len(*chartBufInverse), 1000)
}

func TestIntegration_ChartGenerationEdgeCases(t *testing.T) {
	t.Run("single data point", func(t *testing.T) {
		rates := []schemas.HistoricalRate{
			{Date: time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC), Rate: 1.3500},
		}

		chartBuf, err := GenerateExchangeRateChart(rates, "USD", false)
		require.NoError(t, err)
		assert.Greater(t, len(*chartBuf), 500)
	})

	t.Run("two data points", func(t *testing.T) {
		rates := []schemas.HistoricalRate{
			{Date: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC), Rate: 1.3000},
			{Date: time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC), Rate: 1.3500},
		}

		chartBuf, err := GenerateExchangeRateChart(rates, "USD", false)
		require.NoError(t, err)
		assert.Greater(t, len(*chartBuf), 500)
	})

	t.Run("very similar values", func(t *testing.T) {
		rates := []schemas.HistoricalRate{
			{Date: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC), Rate: 1.3500},
			{Date: time.Date(2025, 1, 2, 0, 0, 0, 0, time.UTC), Rate: 1.3501},
			{Date: time.Date(2025, 1, 3, 0, 0, 0, 0, time.UTC), Rate: 1.3499},
			{Date: time.Date(2025, 1, 4, 0, 0, 0, 0, time.UTC), Rate: 1.3502},
		}

		chartBuf, err := GenerateExchangeRateChart(rates, "USD", false)
		require.NoError(t, err)
		assert.Greater(t, len(*chartBuf), 500)
	})

	t.Run("empty rates", func(t *testing.T) {
		_, err := GenerateExchangeRateChart([]schemas.HistoricalRate{}, "USD", false)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "no historical rates")
	})
}

func TestIntegration_FullWorkflow(t *testing.T) {
	originalURL := schemas.GetYahooFinanceURL()
	defer func() {
		schemas.SetYahooFinanceURL(originalURL)
	}()

	now := time.Now().Unix()
	serverCalled := false

	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		serverCalled = true

		path := r.URL.Path
		hasInterval := strings.Contains(r.URL.RawQuery, "interval=1d")

		var timestamps []int64
		var closes []float64

		if hasInterval {
			timestamps = []int64{
				now - 86400*180,
				now - 86400*120,
				now - 86400*60,
				now - 86400*30,
				now,
			}
			closes = []float64{0.70, 0.73, 0.75, 0.77, 0.80}
		} else {
			timestamps = []int64{now}
			closes = []float64{0.78}
			_ = path
		}

		response := schemas.YahooChartResponse{
			Chart: struct {
				Result []*schemas.YahooChartResult `json:"result"`
				Error  interface{}                 `json:"error"`
			}{
				Result: []*schemas.YahooChartResult{
					{
						Meta: struct {
							RegularMarketPrice float64 `json:"regularMarketPrice"`
							Currency           string  `json:"currency"`
						}{
							RegularMarketPrice: closes[0],
							Currency:           "USD",
						},
						Timestamp: timestamps,
						Indicators: struct {
							Quote []schemas.YahooQuote `json:"quote"`
						}{
							Quote: []schemas.YahooQuote{
								{Close: closes},
							},
						},
					},
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer mockServer.Close()

	schemas.SetYahooFinanceURL(mockServer.URL)

	rate, response, err := schemas.FetchLatestExchangeRate("USD")
	require.NoError(t, err)
	require.NotNil(t, response)
	assert.Greater(t, rate, 0.0)

	assert.True(t, serverCalled, "mock server should have been called")

	rates, err := schemas.FetchHistoricalExchangeRates("USD", 180)
	require.NoError(t, err)
	require.NotEmpty(t, rates)

	sort.Slice(rates, func(i, j int) bool {
		return rates[i].Date.Before(rates[j].Date)
	})

	assert.Equal(t, 5, len(rates))
	assert.Equal(t, 0.70, rates[0].Rate)
	assert.Equal(t, 0.80, rates[4].Rate)

	chartBuf, err := GenerateExchangeRateChart(rates, "USD", false)
	require.NoError(t, err)
	require.NotNil(t, chartBuf)
	assert.Greater(t, len(*chartBuf), 1000, "should generate valid PNG chart")

	expectedHeaders := []byte{0x89, 0x50, 0x4E, 0x47}
	assert.Equal(t, expectedHeaders, (*chartBuf)[:4], "should be valid PNG file")
}

func TestIntegration_GetHistoricalRates(t *testing.T) {
	originalURL := schemas.GetYahooFinanceURL()
	defer func() {
		schemas.SetYahooFinanceURL(originalURL)
	}()

	now := time.Now().Unix()
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Contains(t, r.URL.Path, "SGDJPY=X")

		response := schemas.YahooChartResponse{
			Chart: struct {
				Result []*schemas.YahooChartResult `json:"result"`
				Error  interface{}                 `json:"error"`
			}{
				Result: []*schemas.YahooChartResult{
					{
						Timestamp: []int64{now - 86400*30, now - 86400*15, now},
						Indicators: struct {
							Quote []schemas.YahooQuote `json:"quote"`
						}{
							Quote: []schemas.YahooQuote{
								{Close: []float64{95.0, 96.0, 97.0}},
							},
						},
					},
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer mockServer.Close()

	schemas.SetYahooFinanceURL(mockServer.URL)

	rates, err := GetHistoricalRates("JPY", 1)
	require.NoError(t, err)
	require.Len(t, rates, 3)

	chartBuf, err := GenerateExchangeRateChart(rates, "JPY", false)
	require.NoError(t, err)
	assert.Greater(t, len(*chartBuf), 500)
}

func TestIntegration_FetchWithUSDIntermediary(t *testing.T) {
	originalURL := schemas.GetYahooFinanceURL()
	defer func() {
		schemas.SetYahooFinanceURL(originalURL)
	}()

	now := time.Now().Unix()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path

		var response interface{}
		if strings.Contains(path, "PHPUSD=X") {
			response = schemas.YahooChartResponse{
				Chart: struct {
					Result []*schemas.YahooChartResult `json:"result"`
					Error  interface{}                 `json:"error"`
				}{
					Result: []*schemas.YahooChartResult{
						{
							Timestamp: []int64{now - 86400, now},
							Indicators: struct {
								Quote []schemas.YahooQuote `json:"quote"`
							}{
								Quote: []schemas.YahooQuote{
									{Close: []float64{50.0, 52.0}},
								},
							},
						},
					},
				},
			}
		} else if strings.Contains(path, "USDSGD=X") {
			response = schemas.YahooChartResponse{
				Chart: struct {
					Result []*schemas.YahooChartResult `json:"result"`
					Error  interface{}                 `json:"error"`
				}{
					Result: []*schemas.YahooChartResult{
						{
							Timestamp: []int64{now - 86400, now},
							Indicators: struct {
								Quote []schemas.YahooQuote `json:"quote"`
							}{
								Quote: []schemas.YahooQuote{
									{Close: []float64{1.30, 1.32}},
								},
							},
						},
					},
				},
			}
		} else {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	schemas.SetYahooFinanceURL(server.URL)

	rates, err := schemas.FetchHistoricalExchangeRates("PHP", 30)
	require.NoError(t, err)
	require.Len(t, rates, 2)

	chartBuf, err := GenerateExchangeRateChart(rates, "PHP", false)
	require.NoError(t, err)
	assert.Greater(t, len(*chartBuf), 500)
}

func TestIntegration_VNDConversion(t *testing.T) {
	originalURL := schemas.GetYahooFinanceURL()
	defer func() {
		schemas.SetYahooFinanceURL(originalURL)
	}()

	now := time.Now().Unix()

	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path

		var response interface{}
		if strings.Contains(path, "VNDUSD") {
			response = schemas.YahooChartResponse{
				Chart: struct {
					Result []*schemas.YahooChartResult `json:"result"`
					Error  interface{}                 `json:"error"`
				}{
					Result: []*schemas.YahooChartResult{
						{
							Timestamp: []int64{now - 86400, now},
							Indicators: struct {
								Quote []schemas.YahooQuote `json:"quote"`
							}{
								Quote: []schemas.YahooQuote{
									{Close: []float64{25000.0, 26000.0}},
								},
							},
						},
					},
				},
			}
		} else if strings.Contains(path, "USDSGD") {
			response = schemas.YahooChartResponse{
				Chart: struct {
					Result []*schemas.YahooChartResult `json:"result"`
					Error  interface{}                 `json:"error"`
				}{
					Result: []*schemas.YahooChartResult{
						{
							Timestamp: []int64{now - 86400, now},
							Indicators: struct {
								Quote []schemas.YahooQuote `json:"quote"`
							}{
								Quote: []schemas.YahooQuote{
									{Close: []float64{1.30, 1.35}},
								},
							},
						},
					},
				},
			}
		} else {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer mockServer.Close()

	schemas.SetYahooFinanceURL(mockServer.URL)

	rates, err := schemas.FetchHistoricalExchangeRates("VND", 30)
	require.NoError(t, err)
	require.Len(t, rates, 2)

	assert.InDelta(t, 1.30/25000.0, rates[0].Rate, 0.001, "SGD/VND should be USD/SGD divided by VND/USD")
	assert.InDelta(t, 1.35/26000.0, rates[1].Rate, 0.001)

	chartBuf, err := GenerateExchangeRateChart(rates, "VND", false)
	require.NoError(t, err)
	assert.Greater(t, len(*chartBuf), 500)
}
