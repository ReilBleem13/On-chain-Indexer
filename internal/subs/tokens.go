package subs

import (
	"ReilBleem13/On-chain-Indexer/internal/domain"
	"ReilBleem13/On-chain-Indexer/internal/launchpads"
	"ReilBleem13/On-chain-Indexer/internal/provider"
	"context"
	"fmt"
	"log/slog"
	"sync"

	"github.com/solana-foundation/solana-go/v2/rpc"
	"github.com/solana-foundation/solana-go/v2/rpc/ws"
)

const (
	defaultWorkers = 3
)

type Ingestor struct {
	provider   provider.Provider
	launchpads map[string]launchpads.Launchpad
	workers    int
}

func NewIngestor(
	provider provider.Provider,
	lps ...launchpads.Launchpad,
) *Ingestor {
	lpMap := make(map[string]launchpads.Launchpad, len(lps))
	for _, lp := range lps {
		lpMap[lp.ProgramID()] = lp
	}

	return &Ingestor{
		provider:   provider,
		launchpads: lpMap,
		workers:    defaultWorkers,
	}
}

type logJob struct {
	Log        ws.LogResult
	Launchpads launchpads.Launchpad
}

func (i *Ingestor) Start(ctx context.Context) (<-chan domain.Event, error) {
	out := make(chan domain.Event, 64)
	jobs := make(chan logJob, 64)

	type subSource struct {
		lp     launchpads.Launchpad
		logsCh <-chan ws.LogResult
	}
	sources := make([]subSource, 0, len(i.launchpads))

	for programID, lp := range i.launchpads {
		logsCh, err := i.provider.SubscribeLogs(ctx, programID)
		if err != nil {
			return nil, fmt.Errorf("subLogs %s: %w", programID, err)
		}
		sources = append(sources, subSource{lp: lp, logsCh: logsCh})
	}

	var wg sync.WaitGroup
	for w := 0; w < i.workers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for job := range jobs {
				i.handleLog(ctx, job, out)
			}
		}()
	}

	var subWg sync.WaitGroup
	for _, src := range sources {
		subWg.Add(1)
		go func(src subSource) {
			defer subWg.Done()

			for {
				select {
				case <-ctx.Done():
					return
				case logRes, ok := <-src.logsCh:
					if !ok {
						return
					}

					select {
					case jobs <- logJob{Log: logRes, Launchpads: src.lp}:
					case <-ctx.Done():
						return
					}

				}
			}
		}(src)
	}

	go func() {
		subWg.Wait()
		close(jobs)
	}()

	go func() {
		wg.Wait()
		close(out)
	}()
	return out, nil
}

func (i *Ingestor) handleLog(ctx context.Context, job logJob, out chan<- domain.Event) {
	logs := job.Log.Value.Logs
	sig := job.Log.Value.Signature.String()
	lp := job.Launchpads

	isCreate := lp.IsCreate(logs)
	isMigrate := lp.IsMigrate(logs)

	if !isCreate && !isMigrate {
		return
	}

	txResult, err := i.provider.GetTransaction(ctx, sig)
	if err != nil {
		slog.Error("GetTransaction failed",
			slog.String("sig", sig),
			slog.String("error", err.Error()),
		)
		return
	}

	if txResult == nil || txResult.Meta == nil || txResult.Meta.Err != nil {
		slog.Warn("Not full data", slog.String("sig", sig))
		return
	}

	if isCreate {
		err = i.handleLaunchedToken(txResult, lp, out)
	} else if isMigrate {
		err = i.handleMigratedToken(txResult, lp, out)
	}

	if err != nil {
		slog.Warn("Failed to handle token", slog.String("error", err.Error()))
	}
}

func (i *Ingestor) handleLaunchedToken(
	txResult *rpc.GetTransactionResult,
	lp launchpads.Launchpad,
	out chan<- domain.Event,
) error {
	token, err := lp.ParseCreate(txResult)
	if err != nil {
		return err
	}

	out <- domain.Event{
		Type:    string(domain.EventTokenLauched),
		Payload: token,
	}
	return nil
}

func (i *Ingestor) handleMigratedToken(
	txResult *rpc.GetTransactionResult,
	lp launchpads.Launchpad,
	out chan<- domain.Event,
) error {
	token, err := lp.ParseMigrate(txResult)
	if err != nil {
		return err
	}

	out <- domain.Event{
		Type:    string(domain.EventTokenMigrated),
		Payload: token,
	}
	return nil
}
