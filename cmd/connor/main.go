package main

import (
	"context"
	"fmt"

	"github.com/sonm-io/core/cmd"
	"github.com/sonm-io/core/connor"
)

var (
	configFlag  string
	versionFlag bool
	appVersion  string
)

func main() {
	cmd.NewCmd("connor", appVersion, &configFlag, &versionFlag, run).Execute()
}

func run() error {
	cfg, err := connor.NewConfig(configFlag)
	if err != nil {
		return fmt.Errorf("cannot load config file: %s\r\n", err)
	}

	key, err := cfg.Eth.LoadKey()
	if err != nil {
		return fmt.Errorf("cannot open keys: %v\r\n", err)
	}
	ctx := context.Background()

	c, err := connor.NewConnor(ctx, key, cfg)
	if err != nil {
		return err
	}

	if err := c.Serve(ctx); err != nil {
		return fmt.Errorf("termination: %s", err)
	}

	return nil
}
