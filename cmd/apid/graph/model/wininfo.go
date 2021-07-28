package model

type WinInfo struct {
	BidID                   string `json:"bid"`
	AuctionID               string `json:"auction"`
	BidderID                string `json:"bidderId"`
	Acknowledged            bool   `json:"acknowledged"`
	ProposalCid             string `json:"proposalCid"`
	ProposalCidAcknowledged bool   `json:"proposalCidAcknowledged"`
}
