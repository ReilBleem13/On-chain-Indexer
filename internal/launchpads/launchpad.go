package launchpads

import (
	"ReilBleem13/On-chain-Indexer/internal/domain"

	"github.com/solana-foundation/solana-go/v2/rpc"
)

type Launchpad interface {
	Name() string
	ProgramID() string

	IsCreate(logs []string) bool
	ParseCreate(tx *rpc.GetTransactionResult) (domain.TokenLaunched, error)

	IsMigrate(logs []string) bool
	ParseMigrate(tx *rpc.GetTransactionResult) (domain.TokenMigrated, error)
}
