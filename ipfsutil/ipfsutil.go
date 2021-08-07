package ipfsutil

import (
	"context"
	"math/rand"
	"time"

	"github.com/ipfs/go-cid"
	format "github.com/ipfs/go-ipld-format"
	iface "github.com/ipfs/interface-go-ipfs-core"
	"github.com/ipfs/interface-go-ipfs-core/options"
	ipfspath "github.com/ipfs/interface-go-ipfs-core/path"
	"github.com/multiformats/go-multiaddr"
	logger "github.com/textileio/go-log/v2"
)

var (
	log = logger.Logger("ipfsutil")
)

// IpfsAPI describes an IPFS endpoint.
type IpfsAPI struct {
	Address multiaddr.Multiaddr
	API     iface.CoreAPI
}

// GetNodeGetterForCid returns a NodeGetter that is pinning or can access a Cid
// block from a list.
func GetNodeGetterForCid(ipfsApis []IpfsAPI, c cid.Cid) (ng format.NodeGetter, found bool) {
	rand.Shuffle(len(ipfsApis), func(i, j int) {
		ipfsApis[i], ipfsApis[j] = ipfsApis[j], ipfsApis[i]
	})

	for _, coreapi := range ipfsApis {
		ctx, cls := context.WithTimeout(context.Background(), time.Second*15)
		defer cls()
		_, ok, err := coreapi.API.Pin().IsPinned(ctx, ipfspath.IpfsPath(c), options.Pin.IsPinned.Recursive())
		if err != nil {
			log.Errorf("checking if %s is pinned in %s: %s", c, coreapi.Address, err)
			continue
		}
		if !ok {
			continue
		}
		log.Debugf("found core-api for cid: %s", coreapi.Address)
		ng = coreapi.API.Dag()
		break
	}

	if ng == nil {
		log.Warnf("no go-ipfs node have pinned %s, using fallback", c)
		for _, coreapi := range ipfsApis {
			ctx, cls := context.WithTimeout(context.Background(), time.Second*15)
			defer cls()
			_, err := coreapi.API.Block().Get(ctx, ipfspath.IpfsPath(c))
			if err != nil {
				log.Errorf("checking if %s is pinned in %s: %s", c, coreapi.Address, err)
				continue
			}
			log.Debugf("found core-api for cid: %s", coreapi.Address)
			ng = coreapi.API.Dag()
			break
		}

		if ng == nil {
			return nil, false
		}
	}

	return ng, true
}

// IsPinned returns if a cid is pinned in one of the provided go-ipfs nodes.
func IsPinned(ctx context.Context, ipfsApis []IpfsAPI, c cid.Cid) (bool, error) {
	for _, coreapi := range ipfsApis {
		ctx, cls := context.WithTimeout(ctx, time.Second*15)
		defer cls()
		_, ok, err := coreapi.API.Pin().IsPinned(ctx, ipfspath.IpfsPath(c), options.Pin.IsPinned.Recursive())
		if err != nil {
			log.Errorf("checking if %s is pinned in %s: %s", c, coreapi.Address, err)
			continue
		}
		if !ok {
			continue
		}
		log.Debugf("found core-api for cid: %s", coreapi.Address)
		return true, nil
	}
	return false, nil
}
