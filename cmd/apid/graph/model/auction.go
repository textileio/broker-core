package model

import "time"

type Auction struct {
	ID                                 string        `json:"id"`
	BatchID                            string        `json:"batchId"`
	PayloadCid                         string        `json:"payloadCid"`
	DealSize                           int64         `json:"dealSize"`
	DealDuration                       int64         `json:"dealDuration"`
	DealReplication                    int32         `json:"dealReplication"`
	DealVerified                       bool          `json:"dealVerified"`
	FilEpochDeadline                   int64         `json:"filEpochDeadline"`
	ExcludedMiners                     []string      `json:"excludedMiners"`
	Status                             AuctionStatus `json:"status"`
	StartedAt                          time.Time     `json:"startedAt"`
	UpdatedAt                          time.Time     `json:"updatedAt"`
	Duration                           int64         `json:"duration"`
	Attempts                           int32         `json:"attempts"`
	ErrorCause                         *string       `json:"errorCause"`
	BrokerAlreadyNotifiedClosedAuction bool          `json:"brokerAlreadyNotifiedClosedAuction"`
}
