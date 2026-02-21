package schemas

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/Jason-CKY/telegram-notifybot/internal/utils"
)

type CurrencySubscription struct {
	ID                   string    `json:"id,omitempty"`
	ChatID               int64     `json:"chat_id"`
	Currency             string    `json:"currency"`
	ThresholdAbove       *float64  `json:"threshold_above"`
	ThresholdBelow       *float64  `json:"threshold_below"`
	Interval             *float64  `json:"interval"`
	LastNotifiedRate     float64   `json:"last_notified_rate"`
	LastNotificationTime time.Time `json:"last_notification_time"`
	Enabled              bool      `json:"enabled"`
}

func (cs CurrencySubscription) MarshalJSON() ([]byte, error) {
	type Alias CurrencySubscription

	aux := &struct {
		ChatID string `json:"chat_id"`
		*Alias
	}{
		ChatID: strconv.FormatInt(cs.ChatID, 10),
		Alias:  (*Alias)(&cs),
	}
	return json.Marshal(aux)
}

func (cs *CurrencySubscription) UnmarshalJSON(data []byte) error {
	type Alias CurrencySubscription

	aux := &struct {
		ChatID string `json:"chat_id"`
		*Alias
	}{
		Alias: (*Alias)(cs),
	}

	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	chatID, err := strconv.ParseInt(aux.ChatID, 10, 64)
	if err != nil {
		return err
	}
	cs.ChatID = chatID
	return nil
}

func (sub *CurrencySubscription) Create() error {
	endpoint := fmt.Sprintf("%v/items/notifybot_currency_subscriptions", utils.DirectusHost)
	reqBody, _ := json.Marshal(sub)
	req, httpErr := http.NewRequest(http.MethodPost, endpoint, bytes.NewBuffer(reqBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %v", utils.DirectusToken))
	if httpErr != nil {
		return httpErr
	}
	client := &http.Client{}
	res, httpErr := client.Do(req)
	if httpErr != nil {
		return httpErr
	}
	body, _ := io.ReadAll(res.Body)
	defer res.Body.Close()
	if res.StatusCode != 200 && res.StatusCode != 201 {
		return fmt.Errorf("error creating subscription: %v", string(body))
	}
	var response struct {
		Data CurrencySubscription `json:"data"`
	}
	if err := json.Unmarshal(body, &response); err != nil {
		return err
	}
	sub.ID = response.Data.ID
	return nil
}

func (sub *CurrencySubscription) Update() error {
	if sub.ID == "" {
		return fmt.Errorf("cannot update subscription without ID")
	}
	endpoint := fmt.Sprintf("%v/items/notifybot_currency_subscriptions/%v", utils.DirectusHost, sub.ID)
	reqBody, _ := json.Marshal(sub)
	req, httpErr := http.NewRequest(http.MethodPatch, endpoint, bytes.NewBuffer(reqBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %v", utils.DirectusToken))
	if httpErr != nil {
		return httpErr
	}
	client := &http.Client{}
	res, httpErr := client.Do(req)
	if httpErr != nil {
		return httpErr
	}
	body, _ := io.ReadAll(res.Body)
	defer res.Body.Close()
	if res.StatusCode != 200 {
		return fmt.Errorf("error updating subscription: %v", string(body))
	}
	return nil
}

func (sub *CurrencySubscription) Delete() error {
	if sub.ID == "" {
		return fmt.Errorf("cannot delete subscription without ID")
	}
	endpoint := fmt.Sprintf("%v/items/notifybot_currency_subscriptions/%v", utils.DirectusHost, sub.ID)
	req, httpErr := http.NewRequest(http.MethodDelete, endpoint, nil)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %v", utils.DirectusToken))
	if httpErr != nil {
		return httpErr
	}
	client := &http.Client{}
	res, httpErr := client.Do(req)
	if httpErr != nil {
		return httpErr
	}
	body, _ := io.ReadAll(res.Body)
	defer res.Body.Close()
	if res.StatusCode != 204 && res.StatusCode != 200 {
		return fmt.Errorf("error deleting subscription: %v", string(body))
	}
	return nil
}

func GetCurrencySubscription(chatID int64, currency string) (*CurrencySubscription, error) {
	endpoint := fmt.Sprintf("%v/items/notifybot_currency_subscriptions", utils.DirectusHost)
	reqBody := []byte(fmt.Sprintf(`{
		"query": {
			"filter": {
				"_and": [
					{"chat_id": {"_eq": "%v"}},
					{"currency": {"_eq": "%v"}}
				]
			}
		}
	}`, chatID, currency))
	req, httpErr := http.NewRequest("SEARCH", endpoint, bytes.NewBuffer(reqBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %v", utils.DirectusToken))
	if httpErr != nil {
		return nil, httpErr
	}
	client := &http.Client{}
	res, httpErr := client.Do(req)
	if httpErr != nil {
		return nil, httpErr
	}
	defer res.Body.Close()
	body, _ := io.ReadAll(res.Body)
	if res.StatusCode != 200 {
		return nil, fmt.Errorf("error getting subscription: %v", string(body))
	}
	var response map[string][]CurrencySubscription
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, err
	}
	if len(response["data"]) == 0 {
		return nil, nil
	}
	return &response["data"][0], nil
}

func GetCurrencySubscriptionsByChatID(chatID int64) ([]CurrencySubscription, error) {
	endpoint := fmt.Sprintf("%v/items/notifybot_currency_subscriptions", utils.DirectusHost)
	reqBody := []byte(fmt.Sprintf(`{
		"query": {
			"filter": {
				"chat_id": {"_eq": "%v"}
			}
		}
	}`, chatID))
	req, httpErr := http.NewRequest("SEARCH", endpoint, bytes.NewBuffer(reqBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %v", utils.DirectusToken))
	if httpErr != nil {
		return nil, httpErr
	}
	client := &http.Client{}
	res, httpErr := client.Do(req)
	if httpErr != nil {
		return nil, httpErr
	}
	defer res.Body.Close()
	body, _ := io.ReadAll(res.Body)
	if res.StatusCode != 200 {
		return nil, fmt.Errorf("error getting subscriptions: %v", string(body))
	}
	var response map[string][]CurrencySubscription
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, err
	}
	return response["data"], nil
}

func GetAllActiveSubscriptions() ([]CurrencySubscription, error) {
	endpoint := fmt.Sprintf("%v/items/notifybot_currency_subscriptions", utils.DirectusHost)
	reqBody := []byte(`{
		"query": {
			"filter": {
				"enabled": {"_eq": true}
			}
		}
	}`)
	req, httpErr := http.NewRequest("SEARCH", endpoint, bytes.NewBuffer(reqBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %v", utils.DirectusToken))
	if httpErr != nil {
		return nil, httpErr
	}
	client := &http.Client{}
	res, httpErr := client.Do(req)
	if httpErr != nil {
		return nil, httpErr
	}
	defer res.Body.Close()
	body, _ := io.ReadAll(res.Body)
	if res.StatusCode != 200 {
		return nil, fmt.Errorf("error getting subscriptions: %v", string(body))
	}
	var response map[string][]CurrencySubscription
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, err
	}
	return response["data"], nil
}

func CreateOrUpdateSubscription(chatID int64, currency string, thresholdAbove, thresholdBelow, interval *float64) (*CurrencySubscription, error) {
	existing, err := GetCurrencySubscription(chatID, currency)
	if err != nil {
		return nil, err
	}

	if existing != nil {
		if thresholdAbove != nil {
			existing.ThresholdAbove = thresholdAbove
		}
		if thresholdBelow != nil {
			existing.ThresholdBelow = thresholdBelow
		}
		if interval != nil {
			existing.Interval = interval
		}
		if err := existing.Update(); err != nil {
			return nil, err
		}
		return existing, nil
	}

	sub := &CurrencySubscription{
		ChatID:           chatID,
		Currency:         currency,
		ThresholdAbove:   thresholdAbove,
		ThresholdBelow:   thresholdBelow,
		Interval:         interval,
		LastNotifiedRate: 0,
		Enabled:          true,
	}
	if err := sub.Create(); err != nil {
		return nil, err
	}
	return sub, nil
}

func (sub *CurrencySubscription) ShouldNotifyForThreshold(currentRate float64) bool {
	if sub.ThresholdAbove != nil && currentRate >= *sub.ThresholdAbove {
		return true
	}
	if sub.ThresholdBelow != nil && currentRate <= *sub.ThresholdBelow {
		return true
	}
	return false
}

func (sub *CurrencySubscription) ShouldNotifyForInterval(currentRate float64) bool {
	if sub.Interval == nil || sub.LastNotifiedRate == 0 {
		return false
	}
	diff := currentRate - sub.LastNotifiedRate
	if diff < 0 {
		diff = -diff
	}
	return diff >= *sub.Interval
}

func (sub *CurrencySubscription) GetNotificationMessage(currentRate float64, rates []HistoricalRate) string {
	var thresholdMsg string
	if sub.ThresholdAbove != nil && currentRate >= *sub.ThresholdAbove {
		thresholdMsg = fmt.Sprintf("📊 Threshold: Above %.4f ✓ triggered\n", *sub.ThresholdAbove)
	} else if sub.ThresholdBelow != nil && currentRate <= *sub.ThresholdBelow {
		thresholdMsg = fmt.Sprintf("📊 Threshold: Below %.4f ✓ triggered\n", *sub.ThresholdBelow)
	}

	var changeMsg string
	if sub.LastNotifiedRate > 0 {
		change := currentRate - sub.LastNotifiedRate
		changeSymbol := "+"
		if change < 0 {
			changeSymbol = ""
		}
		changeMsg = fmt.Sprintf("Change from last: %s%.4f SGD\n", changeSymbol, change)
	}

	var minRate, maxRate float64
	if len(rates) > 0 {
		minRate, maxRate = rates[0].Rate, rates[0].Rate
		for _, r := range rates {
			if r.Rate < minRate {
				minRate = r.Rate
			}
			if r.Rate > maxRate {
				maxRate = r.Rate
			}
		}
	}

	return fmt.Sprintf(
		"💱 *%s/SGD Rate Alert*\n\n"+
			"Current Rate: %.4f SGD\n"+
			"%s"+
			"%s"+
			"📈 12-Month Range: %.4f - %.4f\n",
		sub.Currency, currentRate, changeMsg, thresholdMsg, minRate, maxRate,
	)
}
