package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"ReilBleem13/On-chain-Indexer/internal/config"
	"ReilBleem13/On-chain-Indexer/internal/domain"
	"ReilBleem13/On-chain-Indexer/internal/launchpads"
	"ReilBleem13/On-chain-Indexer/internal/provider"
	"ReilBleem13/On-chain-Indexer/internal/subs"
)

func main() {
	cfg, err := config.New()
	if err != nil {
		log.Fatal(err)
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	solanaRPC, err := provider.NewSolanaRPC(ctx, "https"+cfg.SolanaEndpoint, "wss"+cfg.SolanaEndpoint)
	if err != nil {
		log.Fatal(err)
	}

	pumpfun := launchpads.NewPumpFunLaunchpad()
	ingestor := subs.NewIngestor(solanaRPC, pumpfun)

	events, err := ingestor.Start(ctx)
	if err != nil {
		log.Fatal(err)
	}

	for {
		select {
		case <-ctx.Done():
			return

		case ev, ok := <-events:
			if !ok {
				return
			}
			printEvent(ev)
		}
	}
}

func printEvent(event domain.Event) {
	switch p := event.Payload.(type) {
	case domain.TokenLaunched:
		fmt.Printf("\n🟢  TOKEN LAUNCHED [%s]\n", event.Type)
		fmt.Printf("    Mint      : %s\n", p.Mint)
		fmt.Printf("    Name      : %s (%s)\n", p.Name, p.Symbol)
		fmt.Printf("    Curve     : %s\n", p.BondingCurve)
		fmt.Printf("    Creator   : %s\n", p.Creator)
		fmt.Printf("    URI       : %s\n", p.MetadataURI)
		fmt.Printf("    Mayhem    : %t\n", p.IsMayhem)
		fmt.Printf("    TxHash    : %s\n", p.TxHash)
		fmt.Printf("    Slot      : %d\n", p.Slot)
		fmt.Printf("    Timestamp : %s\n", p.Timestamp.Format(time.RFC3339))
		fmt.Printf("    Detected  : %s\n", p.DetectedAt.Format(time.RFC3339))

	case domain.TokenMigrated:
		fmt.Printf("\n🟣  TOKEN MIGRATED [%s]\n", event.Type)
		fmt.Printf("    Mint      : %s\n", p.Mint)
		fmt.Printf("    Quote Mint: %s\n", p.QuoteMint)
		fmt.Printf("    Curve     : %s\n", p.BondingCurve)
		fmt.Printf("    Creator   : %s\n", p.Creator)
		fmt.Printf("    New Pool  : %s\n", p.NewPool)
		fmt.Printf("    Signature : %s\n", p.Signature)
		fmt.Printf("    Slot      : %d\n", p.Slot)
		fmt.Printf("    Timestamp : %s\n", p.Timestamp.Format(time.RFC3339))
		fmt.Printf("    Detected  : %s\n", p.DetectedAt.Format(time.RFC3339))

	default:
	}
}
