package connor

import (
	"github.com/jinzhu/configor"
	"github.com/sonm-io/core/accounts"
	"github.com/sonm-io/core/insonmnia/logging"
	"github.com/sonm-io/core/proto"
)

type marketConfig struct {
	Endpoint string `required:"true" yaml:"endpoint"`
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
	SensitivityForOrders int     `yaml:"reaction_to_aging_of_orders"`
	MarginAccounting     float64 `yaml:"margin_accounting"`
	PartCharge           float64 `yaml:"part_charge"`
	PartResale           float64 `yaml:"part_resale"`
	PartBuffer           float64 `yaml:"part_buffer"`
}
type otherParameters struct {
	IdentityForBid sonm.IdentityLevel `yaml:"identityForBid"`
}

type Config struct {
	Market            marketConfig          `required:"true" yaml:"market"`
	PoolAddress       poolAddressesConfig   `required:"false" yaml:"pool accounts"`
	Distances         stepsConfig           `yaml:"stepForToken"`
	ChargeIntervalETH chargeOrdersETHConfig `yaml:"chargeOrdersInterval"`
	ChargeIntervalZEC chargeOrdersZECConfig `yaml:"chargeOrdersZECInterval"`
	ChargeIntervalXMR chargeOrdersXMRConfig `yaml:"chargeOrdersXMRInterval"`
	Sensitivity       sensitivityConfig     `yaml:"sensitivity"`
	Images            imageConfig           `yaml:"images"`
	OtherParameters   otherParameters       `yaml:"otherParameters"`
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
