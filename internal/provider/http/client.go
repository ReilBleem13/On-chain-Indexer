package http

import (
	"context"
	"fmt"
	"time"

	"github.com/solana-foundation/solana-go/v2"
	"github.com/solana-foundation/solana-go/v2/rpc"
	"golang.org/x/time/rate"
)

type Options struct {
	RPS        float64
	Burst      int
	Commitment rpc.CommitmentType
}
type Client struct {
	rpc     *rpc.Client
	limiter *rate.Limiter
	opts    Options
}

type Option func(*Options)

func NewClient(httpURL string, opts ...Option) *Client {
	clientOpts := Options{
		RPS:        3,
		Burst:      5,
		Commitment: rpc.CommitmentConfirmed,
	}

	for _, opt := range opts {
		opt(&clientOpts)
	}

	return &Client{
		rpc:     rpc.New(httpURL),
		limiter: rate.NewLimiter(rate.Limit(clientOpts.RPS), clientOpts.Burst),
		opts:    clientOpts,
	}
}

func WithRPS(a float64) func(*Options) {
	return func(o *Options) {
		o.RPS = a
	}
}

func WithBurst(a int) func(*Options) {
	return func(o *Options) {
		o.Burst = a
	}
}

func WithCommitment(a rpc.CommitmentType) func(*Options) {
	return func(o *Options) {
		o.Commitment = a
	}
}

func (c *Client) GetTransaction(ctx context.Context, signature string) (*rpc.GetTransactionResult, error) {
	if err := c.limiter.Wait(ctx); err != nil {
		return nil, fmt.Errorf("rate limit: %w", err)
	}

	sig, err := solana.SignatureFromBase58(signature)
	if err != nil {
		return nil, fmt.Errorf("invalid signature: %w", err)
	}

	tx, err := c.rpc.GetTransaction(ctx, sig, &rpc.GetTransactionOpts{
		MaxSupportedTransactionVersion: new(uint64(0)),
		Commitment:                     c.opts.Commitment,
	})
	if err != nil {
		return nil, fmt.Errorf("getTransaction: %w", err)
	}
	return tx, nil
}

func (c *Client) GetTransactionWithRetry(
	ctx context.Context,
	signature string,
	maxAttempts int,
	initialBackoff time.Duration,
) (*rpc.GetTransactionResult, error) {

	if maxAttempts <= 0 {
		maxAttempts = 3
	}
	if initialBackoff <= 0 {
		initialBackoff = 100 * time.Millisecond
	}

	var err error
	backoff := initialBackoff

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		var tx *rpc.GetTransactionResult
		tx, err = c.GetTransaction(ctx, signature)
		if err == nil && tx != nil {
			return tx, nil
		}

		if attempt == maxAttempts {
			break
		}

		timer := time.NewTimer(backoff)
		select {
		case <-ctx.Done():
			timer.Stop()
			return nil, ctx.Err()
		case <-timer.C:
		}

		backoff *= 2
		if backoff > 2*time.Second {
			backoff = 2 * time.Second
		}
	}

	return nil, fmt.Errorf("getTransaction after %d attempts: %w", maxAttempts, err)
}

func (c *Client) Raw() *rpc.Client {
	return c.rpc
}
