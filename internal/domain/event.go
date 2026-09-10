package domain

import (
	"time"
)

type EventType string

const (
	EventTokenLauched  EventType = "token.launched"
	EventTokenMigrated EventType = "token.migrated"
)

type Event struct {
	Type    string
	Payload any
}

type TokenLaunched struct {
	Mint              string
	BondingCurve      string
	Creator           string
	Slot              uint64
	TxHash            string
	Timestamp         time.Time
	DetectedAt        time.Time
	Name              string
	Symbol            string
	MetadataURI       string
	IsMayhem          bool
	IsCashbackEnabled bool // пока не достаю.
}

type TokenMigrated struct {
	Mint         string
	BondingCurve string
	NewPool      string
	QuoteMint    string
	Creator      string
	Signature    string
	Slot         uint64
	DetectedAt   time.Time
	Timestamp    time.Time
}
