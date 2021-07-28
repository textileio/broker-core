package model

import "time"

type Bid struct {
	ID                  string    `json:"id"`
	MinerAddr           string    `json:"minerAddr"`
	WalletAddeSigBase64 string    `json:"walletAddeSigBase64"`
	BidderID            string    `json:"bidderId"`
	AskPrice            int64     `json:"askPrice"`
	VerifiedAskPrice    *int64    `json:"verifiedAskPrice"`
	StartEpoch          int64     `json:"startEpoch"`
	FastRetrieval       bool      `json:"fastRetrieval"`
	ReceivedAt          time.Time `json:"receivedAt"`
	AuctionID           string    `json:"auction"`
}
