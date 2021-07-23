package ipfsutil

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"github.com/ipfs/go-cid"
	format "github.com/ipfs/go-ipld-format"
	iface "github.com/ipfs/interface-go-ipfs-core"
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

// GetNodeGetterForCid returns a NodeGetter that is pinning or can access a Cid block from a list.
func GetNodeGetterForCid(ipfsApis []IpfsAPI, c cid.Cid) (format.NodeGetter, error) {
	var ng format.NodeGetter

	rand.Shuffle(len(ipfsApis), func(i, j int) {
		ipfsApis[i], ipfsApis[j] = ipfsApis[j], ipfsApis[i]
	})

	for _, coreapi := range ipfsApis {
		ctx, cls := context.WithTimeout(context.Background(), time.Second*15)
		defer cls()
		_, ok, err := coreapi.API.Pin().IsPinned(ctx, ipfspath.IpfsPath(c))
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
			return nil, fmt.Errorf("node getter for cid %s not found", c)
		}
	}

	return ng, nil
}
