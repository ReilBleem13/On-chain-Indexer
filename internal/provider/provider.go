package provider

import (
	"context"

	"github.com/solana-foundation/solana-go/v2/rpc"
	"github.com/solana-foundation/solana-go/v2/rpc/ws"
)

type TransactionFetcher interface {
	GetTransaction(ctx context.Context, signature string) (*rpc.GetTransactionResult, error)
}

type LogSubscriber interface {
	SubscribeLogs(ctx context.Context, programID string) (<-chan ws.LogResult, error)
}

type Provider interface {
	TransactionFetcher
	LogSubscriber
}
