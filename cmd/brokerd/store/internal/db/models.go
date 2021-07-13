// Code generated by sqlc. DO NOT EDIT.

package db

import (
	"database/sql"
	"time"

	"github.com/textileio/bidbot/lib/auction"
	"github.com/textileio/broker-core/broker"
)

type BrokerRequest struct {
	ID            broker.BrokerRequestID     `json:"id"`
	DataCid       string                     `json:"dataCid"`
	StorageDealID broker.StorageDealID       `json:"storageDealID"`
	Status        broker.BrokerRequestStatus `json:"status"`
	RebatchCount  int32                      `json:"rebatchCount"`
	ErrorCause    string                     `json:"errorCause"`
	CreatedAt     time.Time                  `json:"createdAt"`
	UpdatedAt     time.Time                  `json:"updatedAt"`
}

type MinerDeal struct {
	StorageDealID  broker.StorageDealID `json:"storageDealID"`
	AuctionID      auction.AuctionID    `json:"auctionID"`
	BidID          auction.BidID        `json:"bidID"`
	MinerAddr      string               `json:"minerAddr"`
	DealID         int64                `json:"dealID"`
	DealExpiration uint64               `json:"dealExpiration"`
	ErrorCause     string               `json:"errorCause"`
	CreatedAt      time.Time            `json:"createdAt"`
	UpdatedAt      time.Time            `json:"updatedAt"`
}

type StorageDeal struct {
	ID                 broker.StorageDealID     `json:"id"`
	Status             broker.StorageDealStatus `json:"status"`
	RepFactor          int                      `json:"repFactor"`
	DealDuration       int                      `json:"dealDuration"`
	PayloadCid         string                   `json:"payloadCid"`
	PieceCid           string                   `json:"pieceCid"`
	PieceSize          uint64                   `json:"pieceSize"`
	CarUrl             string                   `json:"carUrl"`
	CarIpfsCid         string                   `json:"carIpfsCid"`
	CarIpfsAddrs       string                   `json:"carIpfsAddrs"`
	DisallowRebatching bool                     `json:"disallowRebatching"`
	AuctionRetries     int                      `json:"auctionRetries"`
	FilEpochDeadline   uint64                   `json:"filEpochDeadline"`
	Error              string                   `json:"error"`
	CreatedAt          time.Time                `json:"createdAt"`
	UpdatedAt          time.Time                `json:"updatedAt"`
}

type UnpinJob struct {
	ID        string       `json:"id"`
	Executing sql.NullBool `json:"executing"`
	Cid       string       `json:"cid"`
	Type      int16        `json:"type"`
	ReadyAt   time.Time    `json:"readyAt"`
	CreatedAt time.Time    `json:"createdAt"`
	UpdatedAt time.Time    `json:"updatedAt"`
}