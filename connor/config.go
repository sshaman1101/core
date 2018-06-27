package connor

import (
	"github.com/jinzhu/configor"
	"github.com/sonm-io/core/accounts"
	"github.com/sonm-io/core/insonmnia/logging"
	"github.com/sonm-io/core/proto"
)

type marketConfig struct {
	Endpoint string `yaml:"endpoint" required:"true"`
}

type usingToken struct {
	Token string `yaml:"token"`
}
type poolAddressesConfig struct {
	ZecPoolAddr string `yaml:"zec_pool_addr"`
	XmrPoolAddr string `yaml:"xmr_pool_addr"`
	EthPoolAddr string `yaml:"eth_pool_addr"`
}
type stepsConfig struct {
	StepForETH float64 `yaml:"stepETH"`
	StepForZEC float64 `yaml:"stepZEC"`
	StepForXMR float64 `yaml:"stepXMR"`
}
type chargeOrdersETHConfig struct {
	Start       float64 `yaml:"start"`
	Destination float64 `yaml:"destination"`
}
type chargeOrdersZECConfig struct {
	Start       float64 `yaml:"start"`
	Destination float64 `yaml:"destination"`
}
type chargeOrdersXMRConfig struct {
	Start       float64 `yaml:"start"`
	Destination float64 `yaml:"destination"`
}
type imageConfig struct {
	Image string `yaml:"image"`
}
type sensitivityConfig struct {
	SensitivityForOrders     int     `yaml:"reaction_to_aging_of_orders"`
	MarginAccounting         float64 `yaml:"margin_accounting"`
	PartCharge               float64 `yaml:"part_charge"`
	PartResale               float64 `yaml:"part_resale"`
	PartBuffer               float64 `yaml:"part_buffer"`
	OrdersChangePercent      float64 `yaml:"orders_change_percent"`
	DealsChangePercent       float64 `yaml:"deals_change_percent"`
	WorkerLimitChangePercent int     `yaml:"worker_limit_change_percent"`
}
type otherParameters struct {
	IdentityForBid sonm.IdentityLevel `yaml:"identityForBid"`
}

type Config struct {
	Market            marketConfig          `yaml:"market" required:"true"`
	PoolAddress       poolAddressesConfig   `yaml:"pool accounts" required:"false"`
	UsingToken        usingToken            `yaml:"using token"`
	Distances         stepsConfig           `yaml:"step for token"`
	ChargeIntervalETH chargeOrdersETHConfig `yaml:"charge orders interval"`
	ChargeIntervalZEC chargeOrdersZECConfig `yaml:"charge orders ZEC interval"`
	ChargeIntervalXMR chargeOrdersXMRConfig `yaml:"charge orders XMR interval"`
	Sensitivity       sensitivityConfig     `yaml:"sensitivity"`
	Images            imageConfig           `yaml:"images"`
	OtherParameters   otherParameters       `yaml:"other parameters"`
	Eth               accounts.EthConfig    `yaml:"ethereum" required:"true"`
	Log               logging.Config        `yaml:"log"`
}

func NewConfig(path string) (*Config, error) {
	cfg := &Config{}
	err := configor.Load(cfg, path)
	if err != nil {
		return nil, err
	}
	return cfg, nil
}
