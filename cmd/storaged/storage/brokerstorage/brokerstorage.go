package brokerstorage

import (
	"context"
	"fmt"
	"io"
	"math/rand"
	"time"

	"github.com/ipfs/go-cid"
	httpapi "github.com/ipfs/go-ipfs-http-client"
	format "github.com/ipfs/go-ipld-format"
	iface "github.com/ipfs/interface-go-ipfs-core"
	"github.com/ipfs/interface-go-ipfs-core/options"
	ipfspath "github.com/ipfs/interface-go-ipfs-core/path"
	"github.com/ipld/go-car"
	"github.com/multiformats/go-multiaddr"
	"github.com/textileio/broker-core/auth"
	"github.com/textileio/broker-core/broker"
	"github.com/textileio/broker-core/cmd/storaged/storage"
	"github.com/textileio/broker-core/cmd/storaged/storage/brokerstorage/uploader"
	logger "github.com/textileio/go-log/v2"
)

var (
	log = logger.Logger("brokerstorage")
)

// BrokerStorage its an API implementation of the storage service.
type BrokerStorage struct {
	auth     auth.Authorizer
	up       uploader.Uploader
	broker   broker.BrokerRequestor
	ipfsApis []ipfsAPI
}

type ipfsAPI struct {
	address multiaddr.Multiaddr
	api     iface.CoreAPI
}

var _ storage.Requester = (*BrokerStorage)(nil)

// New returns a new BrokerStorage.
func New(
	auth auth.Authorizer,
	up uploader.Uploader,
	broker broker.BrokerRequestor,
	ipfsEndpoints []multiaddr.Multiaddr,
) (*BrokerStorage, error) {
	ipfsApis := make([]ipfsAPI, len(ipfsEndpoints))
	for i, endpoint := range ipfsEndpoints {
		api, err := httpapi.NewApi(endpoint)
		if err != nil {
			return nil, fmt.Errorf("creating ipfs api: %s", err)
		}
		coreapi, err := api.WithOptions(options.Api.Offline(true))
		if err != nil {
			return nil, fmt.Errorf("creating offline core api: %s", err)
		}
		ipfsApis[i] = ipfsAPI{address: endpoint, api: coreapi}
	}
	return &BrokerStorage{
		auth:     auth,
		up:       up,
		broker:   broker,
		ipfsApis: ipfsApis,
	}, nil
}

// IsAuthorized resolves if the provided identity is authorized to use the
// service. If that isn't the case, a string is also return to exply why.
func (bs *BrokerStorage) IsAuthorized(ctx context.Context, identity string) (bool, string, error) {
	return bs.auth.IsAuthorized(ctx, identity)
}

// CreateFromReader creates a StorageRequest using data from a stream.
func (bs *BrokerStorage) CreateFromReader(
	ctx context.Context,
	r io.Reader,
	meta storage.Metadata,
) (storage.Request, error) {
	c, err := bs.up.Store(ctx, r)
	if err != nil {
		return storage.Request{}, fmt.Errorf("storing stream: %s", err)
	}

	brokerMeta := broker.Metadata{
		Region: meta.Region,
	}
	sr, err := bs.broker.Create(ctx, c, brokerMeta)
	if err != nil {
		return storage.Request{}, fmt.Errorf("creating storage request: %s", err)
	}
	status, err := brokerRequestStatusToStorageRequestStatus(sr.Status)
	if err != nil {
		return storage.Request{}, fmt.Errorf("mapping statuses: %s", err)
	}

	return storage.Request{
		ID:         string(sr.ID),
		Cid:        c,
		StatusCode: status,
	}, nil
}

// Get returns a Request by id.
func (bs *BrokerStorage) Get(ctx context.Context, id string) (storage.Request, error) {
	br, err := bs.broker.Get(ctx, broker.BrokerRequestID(id))
	if err != nil {
		return storage.Request{}, fmt.Errorf("getting broker request: %s", err)
	}

	status, err := brokerRequestStatusToStorageRequestStatus(br.Status)
	if err != nil {
		return storage.Request{}, fmt.Errorf("mapping statuses: %s", err)
	}
	sr := storage.Request{
		ID:         string(br.ID),
		Cid:        br.DataCid,
		StatusCode: status,
	}

	return sr, nil
}

func (bs *BrokerStorage) GetCAR(ctx context.Context, c cid.Cid, w io.Writer) error {
	ng, err := bs.getNodeGetterForCid(c)
	if err != nil {
		return fmt.Errorf("resolving node getter: %s", err)
	}
	if err := car.WriteCar(ctx, ng, []cid.Cid{c}, w); err != nil {
		return fmt.Errorf("fetching data cid %s: %v", c, err)
	}
	return nil
}

func (bs *BrokerStorage) getNodeGetterForCid(c cid.Cid) (format.NodeGetter, error) {
	var ng format.NodeGetter

	rand.Shuffle(len(bs.ipfsApis), func(i, j int) {
		bs.ipfsApis[i], bs.ipfsApis[j] = bs.ipfsApis[j], bs.ipfsApis[i]
	})

	log.Debug("core-api lookup for cid")
	for _, coreapi := range bs.ipfsApis {
		ctx, cls := context.WithTimeout(context.Background(), time.Second*5)
		defer cls()
		_, ok, err := coreapi.api.Pin().IsPinned(ctx, ipfspath.IpfsPath(c))
		if err != nil {
			log.Errorf("checking if %s is pinned in %s: %s", c, coreapi.address, err)
			continue
		}
		if !ok {
			continue
		}
		log.Debugf("found core-api for cid: %s", coreapi.address)
		ng = coreapi.api.Dag()
		break
	}

	if ng == nil {
		return nil, fmt.Errorf("node getter for cid not found")
	}

	return ng, nil
}

func brokerRequestStatusToStorageRequestStatus(status broker.BrokerRequestStatus) (storage.Status, error) {
	switch status {
	case broker.RequestBatching:
		return storage.StatusBatching, nil
	case broker.RequestPreparing:
		return storage.StatusPreparing, nil
	case broker.RequestAuctioning:
		return storage.StatusAuctioning, nil
	case broker.RequestDealMaking:
		return storage.StatusDealMaking, nil
	case broker.RequestSuccess:
		return storage.StatusSuccess, nil
	default:
		return storage.StatusUnknown, fmt.Errorf("unknown status: %s", status)
	}
}
