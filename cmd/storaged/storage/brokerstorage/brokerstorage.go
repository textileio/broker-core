package brokerstorage

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"net/url"
	"time"

	"github.com/ipfs/go-cid"
	httpapi "github.com/ipfs/go-ipfs-http-client"
	"github.com/ipfs/interface-go-ipfs-core/options"
	"github.com/ipld/go-car"
	"github.com/multiformats/go-multiaddr"
	"github.com/textileio/bidbot/lib/auction"
	"github.com/textileio/broker-core/auth"
	"github.com/textileio/broker-core/broker"
	"github.com/textileio/broker-core/cmd/storaged/storage"
	"github.com/textileio/broker-core/cmd/storaged/storage/brokerstorage/uploader"
	"github.com/textileio/broker-core/ipfsutil"
	logger "github.com/textileio/go-log/v2"
)

var (
	log = logger.Logger("brokerstorage")
)

// BrokerStorage its an API implementation of the storage service.
type BrokerStorage struct {
	auth       auth.Authorizer
	uploader   uploader.Uploader
	broker     broker.Broker
	ipfsApis   []ipfsutil.IpfsAPI
	pinataAuth string
}

var _ storage.Requester = (*BrokerStorage)(nil)

// New returns a new BrokerStorage.
func New(
	auth auth.Authorizer,
	up uploader.Uploader,
	broker broker.Broker,
	ipfsEndpoints []multiaddr.Multiaddr,
	pinataJWT string,
) (*BrokerStorage, error) {
	ipfsApis := make([]ipfsutil.IpfsAPI, len(ipfsEndpoints))
	for i, endpoint := range ipfsEndpoints {
		api, err := httpapi.NewApi(endpoint)
		if err != nil {
			return nil, fmt.Errorf("creating ipfs api: %s", err)
		}
		coreapi, err := api.WithOptions(options.Api.Offline(true))
		if err != nil {
			return nil, fmt.Errorf("creating offline core api: %s", err)
		}
		ipfsApis[i] = ipfsutil.IpfsAPI{Address: endpoint, API: coreapi}
	}

	if pinataJWT != "" {
		pinataJWT = "Bearer " + pinataJWT
	}

	return &BrokerStorage{
		auth:       auth,
		uploader:   up,
		broker:     broker,
		ipfsApis:   ipfsApis,
		pinataAuth: pinataJWT,
	}, nil
}

// IsAuthorized resolves if the provided identity is authorized to use the
// service. If it isn't authorized, the second return parameter explains the cause.
func (bs *BrokerStorage) IsAuthorized(
	ctx context.Context,
	identity string) (auth.AuthorizedEntity, bool, string, error) {
	return bs.auth.IsAuthorized(ctx, identity)
}

// CreateFromReader creates a StorageRequest using data from a stream.
func (bs *BrokerStorage) CreateFromReader(
	ctx context.Context,
	or io.Reader,
) (storage.Request, error) {
	type pinataUploadResult struct {
		c   cid.Cid
		err error
	}
	pinataRes := make(chan pinataUploadResult)
	teer, teew := io.Pipe()
	if bs.pinataAuth != "" {
		or = io.TeeReader(or, teew)

		go func() {
			defer close(pinataRes)
			c, err := bs.uploadToPinata(teer)
			if err != nil {
				pinataRes <- pinataUploadResult{err: fmt.Errorf("upload to pinata: %s", err)}
				return
			}
			pinataRes <- pinataUploadResult{c: c}
		}()
	}

	c, err := bs.uploader.Store(ctx, or)
	if err != nil {
		return storage.Request{}, fmt.Errorf("storing stream: %s", err)
	}

	if bs.pinataAuth != "" {
		if err := teew.Close(); err != nil {
			return storage.Request{}, fmt.Errorf("closing pinata tee writer: %s", err)
		}
		pres := <-pinataRes
		if pres.err != nil {
			return storage.Request{}, fmt.Errorf("uploading to pinata: %s", err)
		}
		if pres.c != c {
			return storage.Request{}, fmt.Errorf("pinata and broker cids doesn't match: %s vs %s", pres.c, c)
		}
	}

	sr, err := bs.broker.Create(ctx, c)
	if err != nil {
		return storage.Request{}, fmt.Errorf("creating storage request: %s", err)
	}
	status, err := storageRequestStatusToStorageRequestStatus(sr.Status)
	if err != nil {
		return storage.Request{}, fmt.Errorf("mapping statuses: %s", err)
	}

	return storage.Request{
		ID:         string(sr.ID),
		Cid:        c,
		StatusCode: status,
	}, nil
}

func (bs *BrokerStorage) uploadToPinata(r io.Reader) (cid.Cid, error) {
	body, bodyWriter := io.Pipe()
	req, err := http.NewRequest(http.MethodPost, "https://api.pinata.cloud/pinning/pinFileToIPFS", body)
	if err != nil {
		return cid.Undef, fmt.Errorf("creating request: %s", err)
	}

	mwriter := multipart.NewWriter(bodyWriter)
	req.Header.Add("Content-Type", mwriter.FormDataContentType())
	req.Header.Add("Authorization", bs.pinataAuth)
	errCh := make(chan error)
	go func() {
		defer close(errCh)
		defer func() {
			if err := bodyWriter.Close(); err != nil {
				log.Errorf("closing body writer: %s", err)
			}
		}()
		defer func() {
			if err := mwriter.Close(); err != nil {
				log.Errorf("closing multipart writer: %s", err)
			}
		}()

		w, err := mwriter.CreateFormFile("file", "filedata")
		if err != nil {
			errCh <- err
			return
		}

		if written, err := io.Copy(w, r); err != nil {
			errCh <- fmt.Errorf("copying file stream to pinata after %d bytes: %s", written, err)
			return
		}

		if err := mwriter.Close(); err != nil {
			errCh <- err
			return
		}
	}()
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return cid.Undef, fmt.Errorf("executing request: %s", err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			log.Errorf("closing pinata response: %s", err)
		}
	}()
	merr := <-errCh
	if merr != nil {
		return cid.Undef, fmt.Errorf("writing multipart: %s", merr)
	}

	var pinataRes struct{ IpfsHash string }
	jsonRes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return cid.Undef, fmt.Errorf("reading response: %s", err)
	}
	if err := json.Unmarshal(jsonRes, &pinataRes); err != nil {
		return cid.Undef, fmt.Errorf("unmarshaling response: %s", err)
	}
	pinataCid, err := cid.Decode(pinataRes.IpfsHash)
	if err != nil {
		return cid.Undef, fmt.Errorf("decoding cid: %s", err)
	}

	return pinataCid, nil
}

// CreateFromExternalSource creates a storage request for prepared data.
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

	deadline := time.Now().Add(broker.DefaultPreparedCARDeadline)
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
	status, err := storageRequestStatusToStorageRequestStatus(sr.Status)
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
	br, err := bs.broker.GetStorageRequestInfo(ctx, broker.StorageRequestID(id))
	if err != nil {
		return storage.RequestInfo{}, fmt.Errorf("getting storage request info: %s", err)
	}

	status, err := storageRequestStatusToStorageRequestStatus(br.StorageRequest.Status)
	if err != nil {
		return storage.RequestInfo{}, fmt.Errorf("mapping statuses: %s", err)
	}
	sri := storage.RequestInfo{
		Request: storage.Request{
			ID:         string(br.StorageRequest.ID),
			Cid:        br.StorageRequest.DataCid,
			StatusCode: status,
		},
	}

	for _, d := range br.Deals {
		deal := storage.Deal{
			StorageProviderID: d.StorageProviderID,
			DealID:            d.DealID,
			Expiration:        d.Expiration,
		}
		sri.Deals = append(sri.Deals, deal)
	}

	return sri, nil
}

// GetCARHeader generates a CAR header from the provided Cid and writes it to a io.Writer.
// If the Cid can not be found, it does nothing and returns false without error.
func (bs *BrokerStorage) GetCARHeader(ctx context.Context, c cid.Cid, w io.Writer) (bool, error) {
	_, found := ipfsutil.GetNodeGetterForCid(bs.ipfsApis, c)
	if !found {
		return false, nil
	}
	h := &car.CarHeader{
		Roots:   []cid.Cid{c},
		Version: 1,
	}
	if err := car.WriteHeader(h, w); err != nil {
		return true, fmt.Errorf("failed to write car header: %s", err)
	}
	return true, nil
}

// GetCAR generates a CAR file from the provided Cid and writes it a io.Writer.
// If the Cid can not be found, it does nothing and returns false without error.
func (bs *BrokerStorage) GetCAR(ctx context.Context, c cid.Cid, w io.Writer) (bool, error) {
	ng, found := ipfsutil.GetNodeGetterForCid(bs.ipfsApis, c)
	if !found {
		return false, nil
	}
	if err := car.WriteCar(ctx, ng, []cid.Cid{c}, w); err != nil {
		return false, fmt.Errorf("fetching data cid %s: %v", c, err)
	}
	return true, nil
}

func storageRequestStatusToStorageRequestStatus(status broker.StorageRequestStatus) (storage.Status, error) {
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
