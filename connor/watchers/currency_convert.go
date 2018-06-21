package watchers

import (
	"context"
	"encoding/json"
	"sync"
)

type CurrencyData struct {
	Data struct {
		ID     int    `json:"id"`
		Name   string `json:"name"`
		Symbol string `json:"symbol"`
		Quotes struct {
			USD struct {
				Price            float64 `json:"price"`
				Volume24H        float64 `json:"volume_24h"`
				MarketCap        float64 `json:"market_cap"`
				PercentChange1H  float64 `json:"percent_change_1h"`
				PercentChange24H float64 `json:"percent_change_24h"`
				PercentChange7D  float64 `json:"percent_change_7d"`
			} `json:"USD"`
			BTC struct {
				Price            float64 `json:"price"`
				Volume24H        float64 `json:"volume_24h"`
				MarketCap        float64 `json:"market_cap"`
				PercentChange1H  float64 `json:"percent_change_1h"`
				PercentChange24H float64 `json:"percent_change_24h"`
				PercentChange7D  float64 `json:"percent_change_7d"`
			} `json:"BTC"`
		} `json:"quotes"`
		LastUpdated int `json:"last_updated"`
	} `json:"data"`
	Metadata struct {
		Timestamp int         `json:"timestamp"`
		Error     interface{} `json:"error"`
	} `json:"metadata"`
}

type ConvertBtcWatcher struct {
	mu   sync.Mutex
	url  string
	id   []string
	data map[string]*CurrencyData
}

func (p *ConvertBtcWatcher) GetCurrencyData(id string) *CurrencyData {
	return p.data[id]
}

func (p *ConvertBtcWatcher) Update(ctx context.Context) error {
	for _, id := range p.id {
		forCur, err := p.getCurrencyData(id, p.url)
		if err != nil {
			return err
		}
		p.data[id] = forCur
	}
	return nil
}

func (p *ConvertBtcWatcher) getCurrencyData(id string, url string) (*CurrencyData, error) {
	body, err := fetchBody(url + id + "/?convert=BTC")
	if err != nil {
		return nil, err
	}
	forCur := &CurrencyData{}
	err = json.Unmarshal(body, forCur)
	if err != nil {
		return nil, err
	}
	return forCur, nil
}
