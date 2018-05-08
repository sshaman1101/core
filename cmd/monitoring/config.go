package main

import (
	"time"

	"github.com/jinzhu/configor"
	"github.com/sonm-io/core/accounts"
	"github.com/sonm-io/core/insonmnia/logging"
)

type Config struct {
	Endpoint string             `yaml:"endpoint"`
	Account  accounts.EthConfig `yaml:"account"`
	Logging  logging.Config     `yaml:"logging"`
	Checks   []CheckConfig      `yaml:"checks"`
}

type CheckConfig struct {
	Type       string        `yaml:"type"`
	Target     string        `yaml:"target"`
	RequestURI string        `yaml:"request_uri"`
	Args       interface{}   `yaml:"args"`
	Namespace  string        `yaml:"namespace"`
	Subsystem  string        `yaml:"subsystem"`
	Metrics    []string      `yaml:"metrics"`
	Interval   time.Duration `yaml:"interval"`
}

func NewConfig(path string) (*Config, error) {
	cfg := &Config{}
	err := configor.Load(cfg, path)
	if err != nil {
		return nil, err
	}

	return cfg, nil
}
