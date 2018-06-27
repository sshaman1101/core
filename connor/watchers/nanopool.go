package watchers

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
)

type ReturningData struct {
	PoolData        *PoolData
	PoolWorkersData *ReportedHashrate
}
type PoolData struct {
	Data struct {
		Balance     string `json:"balance"`
		Hashrate    string `json:"hashrate"`
		AvgHashrate struct {
			H1  string `json:"h1"`
			H24 string `json:"h24"`
		} `json:"avgHashrate"`
	} `json:"data"`
}

type ReportedHashrate struct {
	Status bool     `json:"status"`
	Data   []RHData `json:"data"`
}

type RHData struct {
	Worker   string  `json:"worker"`
	Hashrate float64 `json:"hashrate"`
}

type nanopoolWatcher struct {
	mu   sync.Mutex
	url  string
	addr []string
	data map[string]*ReturningData
}

func NewPoolWatcher(url string, addr []string) PoolWatcher {
	return &nanopoolWatcher{
		url:  url,
		addr: addr,
		data: make(map[string]*ReturningData),
	}
}

func (p *nanopoolWatcher) GetData(addr string) (*ReturningData, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	d, ok := p.data[addr]
	if !ok {
		return nil, errors.New("no pool with given addr")
	}

	return d, nil
}

func (p *nanopoolWatcher) Update(ctx context.Context) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	for _, addr := range p.addr {
		forPool, err := p.getPoolData(addr, p.url)
		if err != nil {
			return err
		}
		p.data[addr] = forPool
	}
	return nil
}

func (p *nanopoolWatcher) getPoolData(addr string, url string) (*ReturningData, error) {
	body, err := fetchBody(url + addr)
	if err != nil {
		return nil, err
	}
	forPool := &ReturningData{}
	err = json.Unmarshal(body, forPool)
	if err != nil {
		return nil, err
	}
	return forPool, nil
}
