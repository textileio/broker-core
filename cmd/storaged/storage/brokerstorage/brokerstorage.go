package brokerstorage

import (
	"context"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"net/url"
	"time"

	"github.com/ipfs/go-cid"
	httpapi "github.com/ipfs/go-ipfs-http-client"
	format "github.com/ipfs/go-ipld-format"
	iface "github.com/ipfs/interface-go-ipfs-core"
	"github.com/ipfs/interface-go-ipfs-core/options"
	ipfspath "github.com/ipfs/interface-go-ipfs-core/path"
	"github.com/ipld/go-car"
	"github.com/multiformats/go-multiaddr"
	"github.com/textileio/bidbot/lib/auction"
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
	broker   broker.Broker
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
	broker broker.Broker,
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
// service. If it isn't authorized, the second return parameter explains the cause.
func (bs *BrokerStorage) IsAuthorized(ctx context.Context, identity string) (bool, string, error) {
	return bs.auth.IsAuthorized(ctx, identity)
}

// CreateFromReader creates a StorageRequest using data from a stream.
func (bs *BrokerStorage) CreateFromReader(
	ctx context.Context,
	r io.Reader,
) (storage.Request, error) {
	c, err := bs.up.Store(ctx, r)
	if err != nil {
		return storage.Request{}, fmt.Errorf("storing stream: %s", err)
	}

	sr, err := bs.broker.Create(ctx, c)
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

// CreateFromExternalSource creates a broker request for prepared data.
func (bs *BrokerStorage) CreateFromExternalSource(
	ctx context.Context,
	adr storage.AuctionDataRequest) (storage.Request, error) {
	// Validate PayloadCid.
	payloadCid, err := cid.Decode(adr.PayloadCid)
	if err != nil {
		return storage.Request{}, fmt.Errorf("payload-cid is invalid: %s", err)
	}

	// Validate PieceCid.
	pieceCid, err := cid.Decode(adr.PieceCid)
	if err != nil {
		return storage.Request{}, fmt.Errorf("piece-cid is invalid: %s", err)
	}
	if pieceCid.Prefix().Codec != broker.CodecFilCommitmentUnsealed {
		return storage.Request{}, fmt.Errorf("piece-cid must be have fil-commitment-unsealed codec")
	}

	// Validate PieceSize.
	if adr.PieceSize&(adr.PieceSize-1) != 0 {
		return storage.Request{}, errors.New("piece-size must be a power of two")
	}
	if adr.PieceSize > broker.MaxPieceSize {
		return storage.Request{}, fmt.Errorf("piece-size can't be greater than %d", broker.MaxPieceSize)
	}

	// Validate rep factor.
	if adr.RepFactor < 0 {
		return storage.Request{}, errors.New("rep-factor should can't be negative")
	}

	deadline := time.Now().Add(auction.DefaultDealDeadline)
	if adr.Deadline != "" {
		deadline, err = time.Parse(time.RFC3339, adr.Deadline)
		if err != nil {
			return storage.Request{}, fmt.Errorf("deadline should be in RFC3339 format: %s", err)
		}
	}

	pc := broker.PreparedCAR{
		PieceCid:  pieceCid,
		PieceSize: adr.PieceSize,
		RepFactor: adr.RepFactor,
		Deadline:  deadline,
	}

	if adr.CARURL != nil {
		url, err := url.Parse(adr.CARURL.URL)
		if err != nil {
			return storage.Request{}, fmt.Errorf("parsing CAR URL: %s", err)
		}
		if url.Scheme != "http" && url.Scheme != "https" {
			return storage.Request{}, fmt.Errorf("CAR URL scheme should be http(s)")
		}

		pc.Sources.CARURL = &auction.CARURL{
			URL: *url,
		}
	}

	if adr.CARIPFS != nil {
		carCid, err := cid.Decode(adr.CARIPFS.Cid)
		if err != nil {
			return storage.Request{}, fmt.Errorf("car cid isn't valid: %s", err)
		}
		maddrs := make([]multiaddr.Multiaddr, len(adr.CARIPFS.Multiaddrs))
		for i, smaddr := range adr.CARIPFS.Multiaddrs {
			maddr, err := multiaddr.NewMultiaddr(smaddr)
			if err != nil {
				return storage.Request{}, fmt.Errorf("invalid multiaddr %s: %s", smaddr, err)
			}
			maddrs[i] = maddr
		}
		pc.Sources.CARIPFS = &auction.CARIPFS{
			Cid:        carCid,
			Multiaddrs: maddrs,
		}
	}

	if pc.Sources.CARURL == nil && pc.Sources.CARIPFS == nil {
		return storage.Request{}, errors.New("at least one source must be specified")
	}

	sr, err := bs.broker.CreatePrepared(ctx, payloadCid, pc)
	if err != nil {
		return storage.Request{}, fmt.Errorf("creating storage request: %s", err)
	}
	status, err := brokerRequestStatusToStorageRequestStatus(sr.Status)
	if err != nil {
		return storage.Request{}, fmt.Errorf("mapping statuses: %s", err)
	}

	return storage.Request{
		ID:         string(sr.ID),
		Cid:        payloadCid,
		StatusCode: status,
	}, nil
}

// GetRequestInfo returns information about a request.
func (bs *BrokerStorage) GetRequestInfo(ctx context.Context, id string) (storage.RequestInfo, error) {
	br, err := bs.broker.GetBrokerRequestInfo(ctx, broker.BrokerRequestID(id))
	if err != nil {
		return storage.RequestInfo{}, fmt.Errorf("getting broker request info: %s", err)
	}

	status, err := brokerRequestStatusToStorageRequestStatus(br.BrokerRequest.Status)
	if err != nil {
		return storage.RequestInfo{}, fmt.Errorf("mapping statuses: %s", err)
	}
	sri := storage.RequestInfo{
		Request: storage.Request{
			ID:         string(br.BrokerRequest.ID),
			Cid:        br.BrokerRequest.DataCid,
			StatusCode: status,
		},
	}

	for _, d := range br.Deals {
		deal := storage.Deal{
			Miner:      d.Miner,
			DealID:     d.DealID,
			Expiration: d.Expiration,
		}
		sri.Deals = append(sri.Deals, deal)
	}

	return sri, nil
}

// GetCAR generates a CAR file from the provided Cid and writes it a io.Writer.
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
