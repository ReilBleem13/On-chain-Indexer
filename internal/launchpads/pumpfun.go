package launchpads

import (
	"ReilBleem13/On-chain-Indexer/internal/config"
	"ReilBleem13/On-chain-Indexer/internal/domain"
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/solana-foundation/solana-go/v2"
	"github.com/solana-foundation/solana-go/v2/rpc"
)

var (
	CreateDisc  = [8]byte{0xd6, 0x90, 0x4c, 0xec, 0x5f, 0x8b, 0x31, 0xb4}
	MigrateDisc = [8]byte{0xbb, 0xcb, 0x12, 0x1f, 0xce, 0xed, 0xfe, 0x29}
)

type PumpFunLaunchpad struct{}

func NewPumpFunLaunchpad() *PumpFunLaunchpad {
	return &PumpFunLaunchpad{}
}

func (p *PumpFunLaunchpad) Name() string {
	return "PumpFun"
}

func (p *PumpFunLaunchpad) ProgramID() string {
	return config.AppConfig().PumpFunProgramID
}

func (p *PumpFunLaunchpad) IsCreate(logs []string) bool {
	for _, l := range logs {
		if strings.Contains(strings.ToLower(l), "instruction: createv2") {
			return true
		}
	}
	return false
}

func (p *PumpFunLaunchpad) IsMigrate(logs []string) bool {
	for _, l := range logs {
		if strings.Contains(strings.ToLower(l), "instruction: migratev2") {
			return true
		}
	}
	return false
}

func (p *PumpFunLaunchpad) ParseCreate(tx *rpc.GetTransactionResult) (domain.TokenLaunched, error) {
	var result domain.TokenLaunched

	solTx, err := tx.Transaction.GetTransaction()
	if err != nil {
		return result, errors.New("get solana.Transaction")
	}

	result.Slot = tx.Slot
	result.DetectedAt = time.Now().UTC()
	if tx.BlockTime != nil {
		result.Timestamp = tx.BlockTime.Time().UTC()
	}

	if len(solTx.Signatures) > 0 {
		result.TxHash = solTx.Signatures[0].String()
	}

	accountKeys := mergeAccountKeys(solTx.Message.AccountKeys, tx.Meta)

	for i := range solTx.Message.Instructions {
		ix := &solTx.Message.Instructions[i]

		if int(ix.ProgramIDIndex) >= len(accountKeys) {
			continue
		}

		if accountKeys[ix.ProgramIDIndex].String() != p.ProgramID() {
			continue
		}

		if len(ix.Data) < 8 || !bytes.Equal(ix.Data[:8], CreateDisc[:]) {
			continue
		}

		result.Mint = accountKeys[int(ix.Accounts[0])].String()
		result.BondingCurve = accountKeys[int(ix.Accounts[2])].String()

		args, err := borchDecoder(ix.Data)
		if err != nil {
			return result, fmt.Errorf("parse create args: %w", err)
		}

		result.Name = args.Name
		result.Symbol = args.Symbol
		result.MetadataURI = args.URI
		result.IsMayhem = args.IsMayhemMode
		result.Creator = args.Creator.String()
		return result, nil
	}
	return result, errors.New("pump create instruction not found")
}

func (p *PumpFunLaunchpad) ParseMigrate(tx *rpc.GetTransactionResult) (domain.TokenMigrated, error) {
	var result domain.TokenMigrated

	solTx, err := tx.Transaction.GetTransaction()
	if err != nil {
		return result, errors.New("get solana.Transaction")
	}

	result.Slot = tx.Slot
	result.DetectedAt = time.Now().UTC()
	if tx.BlockTime != nil {
		result.Timestamp = tx.BlockTime.Time().UTC()
	}

	if len(solTx.Signatures) > 0 {
		result.Signature = solTx.Signatures[0].String()
	}

	accountKeys := mergeAccountKeys(solTx.Message.AccountKeys, tx.Meta)

	for _, ix := range solTx.Message.Instructions {
		if int(ix.ProgramIDIndex) >= len(accountKeys) {
			continue
		}

		if accountKeys[ix.ProgramIDIndex].String() != config.AppConfig().PumpFunProgramID {
			continue
		}

		if len(ix.Data) < 8 || !bytes.Equal(ix.Data[:8], MigrateDisc[:]) {
			continue
		}

		result.Mint = accountKeys[int(ix.Accounts[2])].String()
		result.QuoteMint = accountKeys[int(ix.Accounts[3])].String()
		result.BondingCurve = accountKeys[int(ix.Accounts[4])].String()
		result.Creator = accountKeys[int(ix.Accounts[7])].String()
		result.NewPool = accountKeys[int(ix.Accounts[10])].String()
		return result, nil
	}
	return result, errors.New("pump migrate instruction not found")
}

func mergeAccountKeys(staticKeys []solana.PublicKey, meta *rpc.TransactionMeta) []solana.PublicKey {
	if meta == nil {
		return staticKeys
	}

	keys := make([]solana.PublicKey, len(staticKeys), len(staticKeys)+len(meta.LoadedAddresses.Writable)+len(meta.LoadedAddresses.ReadOnly))
	copy(keys, staticKeys)

	keys = append(keys, meta.LoadedAddresses.Writable...)
	keys = append(keys, meta.LoadedAddresses.ReadOnly...)
	return keys
}

type createArgs struct {
	Name         string
	Symbol       string
	URI          string
	Creator      solana.PublicKey
	IsMayhemMode bool
}

func borchDecoder(data []byte) (createArgs, error) {
	var (
		args   createArgs
		offset int = 8
		err    error
	)

	readString := func() (string, error) {
		if offset+4 > len(data) {
			return "", errors.New("not enough bytes for string len")
		}

		l := binary.LittleEndian.Uint32(data[offset : offset+4])
		offset += 4

		if offset+int(l) > len(data) {
			return "", fmt.Errorf("not enough bytes for string of len %d", l)
		}

		s := string(data[offset : offset+int(l)])
		offset += int(l)
		return s, nil
	}

	args.Name, err = readString()
	if err != nil {
		return args, fmt.Errorf("name: %w", err)
	}

	args.Symbol, err = readString()
	if err != nil {
		return args, fmt.Errorf("symbol: %w", err)
	}

	args.URI, err = readString()
	if err != nil {
		return args, fmt.Errorf("uri: %w", err)
	}

	if offset+32 > len(data) {
		return args, errors.New("not enough bytes for creator")
	}

	copy(args.Creator[:], data[offset:offset+32])
	offset += 32

	if offset < len(data) {
		args.IsMayhemMode = data[offset] != 0
		offset++
	}
	return args, nil
}
