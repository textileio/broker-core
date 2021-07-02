package cast

import (
	"net/url"

	"github.com/ipfs/go-cid"
	ma "github.com/multiformats/go-multiaddr"
	"github.com/textileio/bidbot/lib/auction"
	pb "github.com/textileio/broker-core/gen/broker/auctioneer/v1"
)

// SourcesToPb converts Sources to pb.
func SourcesToPb(sources auction.Sources) *pb.Sources {
	var carIPFS *pb.Sources_CARIPFS
	if sources.CARIPFS != nil {
		var multiaddrs []string
		for _, addr := range sources.CARIPFS.Multiaddrs {
			multiaddrs = append(multiaddrs, addr.String())
		}
		carIPFS = &pb.Sources_CARIPFS{
			Cid:        sources.CARIPFS.Cid.String(),
			Multiaddrs: multiaddrs,
		}
	}
	var carURL *pb.Sources_CARURL
	if sources.CARURL != nil {
		carURL = &pb.Sources_CARURL{
			URL: sources.CARURL.URL.String(),
		}
	}
	return &pb.Sources{
		CarUrl:  carURL,
		CarIpfs: carIPFS,
	}
}

// SourcesFromPb converts Sources back from pb.
func SourcesFromPb(pbs *pb.Sources) (sources auction.Sources, err error) {
	if pbs.CarUrl != nil {
		u, err := url.Parse(pbs.CarUrl.URL)
		if err != nil {
			return auction.Sources{}, err
		}
		sources.CARURL = &auction.CARURL{URL: *u}
	}

	if pbs.CarIpfs != nil {
		id, err := cid.Parse(pbs.CarIpfs.Cid)
		if err != nil {
			return auction.Sources{}, err
		}
		var multiaddrs []ma.Multiaddr
		for _, s := range pbs.CarIpfs.Multiaddrs {
			addr, err := ma.NewMultiaddr(s)
			if err != nil {
				return auction.Sources{}, err
			}
			multiaddrs = append(multiaddrs, addr)
		}
		sources.CARIPFS = &auction.CARIPFS{Cid: id, Multiaddrs: multiaddrs}
	}
	return
}
