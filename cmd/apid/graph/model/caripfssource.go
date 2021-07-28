package model

type CarIpfsSource struct {
	AuctionID  string   `json:"auction"`
	Cid        string   `json:"cid"`
	Multiaddrs []string `json:"multiaddrs"`
}

func (CarIpfsSource) IsAuctionSource() {}
