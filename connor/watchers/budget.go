package watchers

import (
	"math/big"
	"sync"
)

type budgetWatcher struct {
	mu   sync.Mutex
	data *big.Int
}

//func (p *budgetWatcher) Update(api blockchain.API, ctx context.Context, ethAddr string) error {
//	p.data, _ = GetBudget(api, ctx, ethAddr)
//	return nil
//}
//
//func (p *budgetWatcher) GetBalance() (*big.Int) {
//	return p.data
//}

//func GetBudget(api blockchain.API, ctx context.Context, ethAddr string, ) (*big.Int, *big.Int, error) {
//	balanceSide, err := api.SideToken().BalanceOf(ctx, ethAddr)
//	if err != nil {
//		fmt.Errorf("cannot get balance from SideChain  %v\r\n", err)
//		os.Exit(1)
//		return nil, nil, err
//	}
//	balanceLive, err := api.LiveToken().BalanceOf(ctx, ethAddr)
//	if err != nil {
//		fmt.Errorf("cannot get balance from LiveChain %v\r\n", err)
//		os.Exit(1)
//		return nil, nil, err
//	}
//	return balanceSide, balanceLive, nil
//}
