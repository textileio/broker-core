package graph

// This file will be automatically regenerated based on the schema, any resolver implementations
// will be copied through when generating and any unknown code will be moved to the end.

import (
	"context"
	"fmt"

	"github.com/textileio/broker-core/cmd/apid/graph/generated"
	"github.com/textileio/broker-core/cmd/apid/graph/model"
)

func (r *auctionResolver) Bids(ctx context.Context, obj *model.Auction) ([]*model.Bid, error) {
	panic(fmt.Errorf("not implemented"))
}

func (r *auctionResolver) WinningBids(ctx context.Context, obj *model.Auction) ([]*model.WinInfo, error) {
	panic(fmt.Errorf("not implemented"))
}

func (r *auctionResolver) Source(ctx context.Context, obj *model.Auction) (model.AuctionSource, error) {
	panic(fmt.Errorf("not implemented"))
}

func (r *bidResolver) Auction(ctx context.Context, obj *model.Bid) (*model.Auction, error) {
	panic(fmt.Errorf("not implemented"))
}

func (r *bidResolver) WinInfo(ctx context.Context, obj *model.Bid) (*model.WinInfo, error) {
	panic(fmt.Errorf("not implemented"))
}

func (r *carIpfsSourceResolver) Auction(ctx context.Context, obj *model.CarIpfsSource) (*model.Auction, error) {
	panic(fmt.Errorf("not implemented"))
}

func (r *carUrlSourceResolver) Auction(ctx context.Context, obj *model.CarUrlSource) (*model.Auction, error) {
	panic(fmt.Errorf("not implemented"))
}

func (r *queryResolver) Auctions(ctx context.Context) ([]*model.Auction, error) {
	panic(fmt.Errorf("not implemented"))
}

func (r *winInfoResolver) Bid(ctx context.Context, obj *model.WinInfo) (*model.Bid, error) {
	panic(fmt.Errorf("not implemented"))
}

func (r *winInfoResolver) Auction(ctx context.Context, obj *model.WinInfo) (*model.Auction, error) {
	panic(fmt.Errorf("not implemented"))
}

// Auction returns generated.AuctionResolver implementation.
func (r *Resolver) Auction() generated.AuctionResolver { return &auctionResolver{r} }

// Bid returns generated.BidResolver implementation.
func (r *Resolver) Bid() generated.BidResolver { return &bidResolver{r} }

// CarIpfsSource returns generated.CarIpfsSourceResolver implementation.
func (r *Resolver) CarIpfsSource() generated.CarIpfsSourceResolver { return &carIpfsSourceResolver{r} }

// CarUrlSource returns generated.CarUrlSourceResolver implementation.
func (r *Resolver) CarUrlSource() generated.CarUrlSourceResolver { return &carUrlSourceResolver{r} }

// Query returns generated.QueryResolver implementation.
func (r *Resolver) Query() generated.QueryResolver { return &queryResolver{r} }

// WinInfo returns generated.WinInfoResolver implementation.
func (r *Resolver) WinInfo() generated.WinInfoResolver { return &winInfoResolver{r} }

type auctionResolver struct{ *Resolver }
type bidResolver struct{ *Resolver }
type carIpfsSourceResolver struct{ *Resolver }
type carUrlSourceResolver struct{ *Resolver }
type queryResolver struct{ *Resolver }
type winInfoResolver struct{ *Resolver }
