package model

type CarUrlSource struct {
	AuctionID string `json:"auction"`
	URLString string `json:"urlString"`
}

func (CarUrlSource) IsAuctionSource() {}
