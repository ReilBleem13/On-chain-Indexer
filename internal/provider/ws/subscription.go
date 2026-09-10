package ws

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/solana-foundation/solana-go/v2"
	"github.com/solana-foundation/solana-go/v2/rpc"
	"github.com/solana-foundation/solana-go/v2/rpc/ws"
)

type LogSubscription struct {
	conn       *Conn
	program    solana.PublicKey
	commitment rpc.CommitmentType
}

func (c *Conn) SubscribeLogs(
	ctx context.Context,
	programID solana.PublicKey,
	commitment rpc.CommitmentType,
) (<-chan ws.LogResult, error) {
	out := make(chan ws.LogResult, 256)

	go c.runSubscription(ctx, programID, commitment, out)
	return out, nil
}

func (c *Conn) runSubscription(
	ctx context.Context,
	programID solana.PublicKey,
	commitment rpc.CommitmentType,
	out chan<- ws.LogResult,
) {
	defer close(out)

	backoff := c.opts.InitialBackoff
	attempt := 0

	for {
		if ctx.Err() != nil {
			return
		}

		err := c.subAndRead(ctx, programID, commitment, out)
		if err != nil {
			if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
				return
			}

			slog.Warn("Logs sub lost or failed, reconnecting",
				slog.String("program", programID.String()),
				slog.String("error", err.Error()),
			)
		}

		if !c.opts.Reconnect {
			return
		}

		attempt++
		if c.opts.MaxRetries > 0 && attempt >= c.opts.MaxRetries {
			slog.Error("Max reconnect attempts reached", slog.String("program", programID.String()))
			return
		}

		if c.opts.OnReconnect != nil {
			c.opts.OnReconnect(attempt)
		}

		sleep(ctx, backoff)
		backoff = min(backoff*2, c.opts.MaxBackoff)
	}
}

func (c *Conn) subAndRead(
	ctx context.Context,
	programID solana.PublicKey,
	commitment rpc.CommitmentType,
	out chan<- ws.LogResult,
) error {
	if err := c.Connect(ctx); err != nil {
		return fmt.Errorf("connect failed: %w", err)
	}

	client := c.Client()
	if client == nil {
		return errors.New("ws client is null")
	}

	sub, err := client.LogsSubscribeMentions(programID, commitment)
	if err != nil {
		return fmt.Errorf("subscribe failed: %w", err)
	}
	defer sub.Unsubscribe()

	slog.Info("Logs sub active", slog.String("program", programID.String()))

	stopSignal := make(chan struct{})
	defer close(stopSignal)

	go func() {
		select {
		case <-ctx.Done():
			sub.Unsubscribe()
		case <-stopSignal:
		}
	}()

	for {
		msg, err := sub.Recv(ctx)
		if err != nil {
			return fmt.Errorf("recv error: %w", err)
		}

		if msg == nil {
			continue
		}

		select {
		case out <- *msg:
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

func sleep(ctx context.Context, d time.Duration) {
	timer := time.NewTimer(d)
	defer timer.Stop()

	select {
	case <-ctx.Done():
	case <-timer.C:
	}
}
