package provider

import (
	"context"
	"time"

	myhttp "ReilBleem13/On-chain-Indexer/internal/provider/http"
	myws "ReilBleem13/On-chain-Indexer/internal/provider/ws"

	"github.com/solana-foundation/solana-go/v2"
	"github.com/solana-foundation/solana-go/v2/rpc"
	"github.com/solana-foundation/solana-go/v2/rpc/ws"
	"golang.org/x/time/rate"
)

/*
	Mainnet rate limits
	Maximum number of requests per 10 seconds per IP: 100
	Maximum number of requests per 10 seconds per IP for a single RPC: 40
	Maximum concurrent connections per IP: 40
	Maximum connection rate per 10 seconds per IP: 40
	Maximum amount of data per 30 seconds: 100 MB
*/

type SolanaRPCProvider struct {
	http    *myhttp.Client
	ws      *myws.Conn
	limiter *rate.Limiter
}

func NewSolanaRPC(ctx context.Context, httpURL, wsURL string) (*SolanaRPCProvider, error) {
	httpClient := myhttp.NewClient(httpURL)
	wsClient := myws.NewConn(wsURL)

	return &SolanaRPCProvider{
		http: httpClient,
		ws:   wsClient,
	}, nil
}

func (p *SolanaRPCProvider) GetTransaction(ctx context.Context, signature string) (*rpc.GetTransactionResult, error) {
	return p.http.GetTransactionWithRetry(ctx, signature, 3, 100*time.Millisecond)
}

func (c *SolanaRPCProvider) SubscribeLogs(ctx context.Context, progID string) (<-chan ws.LogResult, error) {
	programID := solana.MustPublicKeyFromBase58(progID)
	return c.ws.SubscribeLogs(ctx, programID, rpc.CommitmentConfirmed)
}
