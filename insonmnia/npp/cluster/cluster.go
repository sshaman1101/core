package cluster

import (
	"encoding/hex"
	"time"

	"github.com/hashicorp/memberlist"
	"github.com/sonm-io/core/util/netutil"
	"go.uber.org/zap"
)

func NewCluster(cfg Config, notified memberlist.EventDelegate, logger *zap.Logger) (*memberlist.Memberlist, error) {
	key, err := hex.DecodeString(cfg.SecretKey)
	if err != nil {
		return nil, err
	}

	keyring, err := memberlist.NewKeyring([][]byte{}, key)
	if err != nil {
		return nil, err
	}

	addr, port, err := netutil.SplitHostPort(cfg.Endpoint)
	if err != nil {
		return nil, err
	}

	config := memberlist.DefaultWANConfig()
	config.Name = cfg.Name
	config.BindAddr = addr.String()
	config.BindPort = int(port)

	if len(cfg.Announce) > 0 {
		announceAddr, announcePort, err := netutil.SplitHostPort(cfg.Announce)
		if err != nil {
			return nil, err
		}

		config.AdvertiseAddr = announceAddr.String()
		config.AdvertisePort = int(announcePort)
	}
	config.Events = notified
	config.Keyring = keyring
	config.LogOutput = newLogAdapter(logger)
	config.ProbeInterval = time.Second

	mlist, err := memberlist.Create(config)
	if err != nil {
		return nil, err
	}

	return mlist, nil
}
