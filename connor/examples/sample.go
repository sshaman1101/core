package main

import (
	"fmt"
	"os"

	"github.com/sonm-io/core/connor/modules"
	"math/big"
	"log"
)

func main() {
	actualPrice := big.NewInt(72005555555555)
	dealPrice := big.NewInt(61905555555555)
	res, err := modules.GetChangePercent(actualPrice, dealPrice)
	if err != nil {
		fmt.Errorf("cannot get change percent %v\r\n", err)
	}

	log.Printf("Result delta %.2f %% FROM :: actualPrice : %v, deal Price : %v\r\n",
		res, modules.PriceToString(actualPrice), modules.PriceToString(dealPrice))
	cmd, err := modules.CmpChangeOfPrice(res, 5)
	if err != nil {
		fmt.Printf("ti pidor")
		os.Exit(1)
	}
	fmt.Printf("result ::  %v\r\n", cmd)

}



