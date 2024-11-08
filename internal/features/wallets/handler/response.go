package handler

import (
	"time"

	wallets "github.com/dwiw96/vocagame-technical-test-backend/internal/features/wallets"
)

type walletResp struct {
	Balance   int32     `json:"balance"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func toWalletResp(arg *wallets.Wallet) (res walletResp) {
	res.Balance = arg.Balance
	res.CreatedAt = arg.CreatedAt
	res.UpdatedAt = arg.UpdatedAt

	return
}
