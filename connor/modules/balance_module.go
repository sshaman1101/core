package modules

import "math/big"

//Limit balance for Charge orders. Default value = 0.5
func LimitChargeSNM(balance *big.Int, partCharge float64) *big.Int {
	limitChargeSNM := balance.Div(balance, big.NewInt(100))
	limitChargeSNM = limitChargeSNM.Mul(balance, big.NewInt(int64(partCharge*100)))
	return limitChargeSNM
}

//converting snmBalance = > USD Balance
func ConvertingToUSDBalance(balanceSide *big.Int, snmPrice float64) float64 {
	bal := balanceSide.Mul(balanceSide, big.NewInt(int64(snmPrice*1e18)))
	bal = bal.Div(bal, big.NewInt(1e18))
	d, e := bal.DivMod(bal, big.NewInt(1e18), big.NewInt(0))
	f, _ := big.NewFloat(0).SetInt(e).Float64()
	res := float64(d.Int64()) + (f / 1e18)
	return res
}
