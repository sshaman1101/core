package main

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/sonm-io/core/accounts"
	"github.com/sonm-io/core/insonmnia/auth"
	"github.com/sonm-io/core/insonmnia/npp"
	"github.com/sonm-io/core/insonmnia/npp/rendezvous"
	"github.com/sonm-io/core/util"
)

func main() {
	ctx := context.Background()

	rvCfg := rendezvous.Config{
		Endpoints: []auth.Addr{
			auth.NewAddrRaw(common.HexToAddress("0x8125721c2413d99a33e351e1f6bb4e56b6b633fd"), "127.0.0.5:14099"),
		},
		MaxConnectionAttempts: 100,
		Timeout:               time.Minute * 10,
	}

	key, err := (&accounts.EthConfig{Keystore: "./keys", Passphrase: "any"}).LoadKey()
	if err != nil {
		log.Printf("failed to load private key: %s", err)
		os.Exit(1)
	}

	_, TLSConfig, err := util.NewHitlessCertRotator(ctx, key)
	if err != nil {
		log.Printf("failed to create rotator: %s", err)
		os.Exit(1)
	}

	dialer, err := npp.NewDialer(ctx, npp.WithRendezvous(rvCfg, util.NewTLS(TLSConfig)))
	if err != nil {
		log.Printf("failed to create Dialer: %s", err)
	}

	conn, err := dialer.Dial(
		auth.NewAddrRaw(common.HexToAddress("0x8125721c2413d99a33e351e1f6bb4e56b6b633fd"), "127.0.0.10:8080"))
	if err != nil {
		log.Printf("failed to dial: %s", err)
		os.Exit(1)
	}

	conn.Write([]byte("Hello!"))
	conn.Close()
}
