// Code generated by sqlc. DO NOT EDIT.

package db

import (
	"database/sql"
	"time"

	"github.com/textileio/bidbot/lib/auction"
	"github.com/textileio/broker-core/broker"
)

type Auction struct {
	ID                       auction.ID           `json:"id"`
	BatchID                  broker.BatchID       `json:"batchID"`
	DealSize                 int64                `json:"dealSize"`
	DealDuration             uint64               `json:"dealDuration"`
	DealReplication          int32                `json:"dealReplication"`
	DealVerified             bool                 `json:"dealVerified"`
	FilEpochDeadline         uint64               `json:"filEpochDeadline"`
	ExcludedStorageProviders []string             `json:"excludedStorageProviders"`
	PayloadCid               string               `json:"payloadCid"`
	CarUrl                   string               `json:"carUrl"`
	CarIpfsCid               string               `json:"carIpfsCid"`
	CarIpfsAddrs             []string             `json:"carIpfsAddrs"`
	Status                   broker.AuctionStatus `json:"status"`
	ErrorCause               string               `json:"errorCause"`
	Duration                 int64                `json:"duration"`
	StartedAt                time.Time            `json:"startedAt"`
	UpdatedAt                time.Time            `json:"updatedAt"`
}

type Bid struct {
	ID                       auction.BidID  `json:"id"`
	AuctionID                auction.ID     `json:"auctionID"`
	WalletAddrSig            []byte         `json:"walletAddrSig"`
	StorageProviderID        string         `json:"storageProviderID"`
	BidderID                 string         `json:"bidderID"`
	AskPrice                 int64          `json:"askPrice"`
	VerifiedAskPrice         int64          `json:"verifiedAskPrice"`
	StartEpoch               int64          `json:"startEpoch"`
	FastRetrieval            bool           `json:"fastRetrieval"`
	ReceivedAt               time.Time      `json:"receivedAt"`
	WonAt                    sql.NullTime   `json:"wonAt"`
	ProposalCid              sql.NullString `json:"proposalCid"`
	ProposalCidDeliveredAt   sql.NullTime   `json:"proposalCidDeliveredAt"`
	ProposalCidDeliveryError sql.NullString `json:"proposalCidDeliveryError"`
}
