package main

import (
	"github.com/golang/mock/gomock"
	"github.com/sonm-io/core/proto"
	"math/big"
	"testing"
	"fmt"
)

func mockDWH(ctrl *gomock.Controller, t sonm.OrderType) sonm.DWHClient {
	id := sonm.NewBigInt(big.NewInt(100918))
	id2 := sonm.NewBigInt(big.NewInt(1009189))
	id3 := sonm.NewBigInt(big.NewInt(1009189283))
	orders := []*sonm.DWHOrder{
		{Order: &sonm.Order{OrderType: t, Id:id , Price: sonm.NewBigIntFromInt(111)}},
		{Order: &sonm.Order{OrderType: t, Id: id2, Price: sonm.NewBigIntFromInt(222)}},
		{Order: &sonm.Order{OrderType: t, Id: id3, Price: sonm.NewBigIntFromInt(333)}},
	}

	dwh := sonm.NewMockDWHClient(ctrl)
	dwh.EXPECT().GetOrders(gomock.Any(), gomock.Any()).AnyTimes().
		Return(&sonm.DWHOrdersReply{Orders: orders}, nil)
		fmt.Printf("orders: #%v\r\n", orders)
	return dwh
}
func TestDWH(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockDWH(mockCtrl, 2)
}