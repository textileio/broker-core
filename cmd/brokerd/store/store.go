package store

import (
	"context"
	"crypto/rand"
	"database/sql"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/filecoin-project/go-address"
	bindata "github.com/golang-migrate/migrate/v4/source/go_bindata"
	"github.com/ipfs/go-cid"
	"github.com/jackc/pgconn"
	peer "github.com/libp2p/go-libp2p-core/peer"
	"github.com/multiformats/go-multiaddr"
	"github.com/oklog/ulid/v2"
	"github.com/textileio/bidbot/lib/auction"
	"github.com/textileio/broker-core/broker"
	"github.com/textileio/broker-core/cmd/brokerd/store/internal/db"
	"github.com/textileio/broker-core/cmd/brokerd/store/migrations"
	"github.com/textileio/broker-core/common"
	"github.com/textileio/broker-core/storeutil"
	logger "github.com/textileio/go-log/v2"
)

var (
	// ErrNotFound is returned if the storage request doesn't exist.
	ErrNotFound = fmt.Errorf("not found")

	// ErrBatchExists if the provided batch id already exists.
	ErrBatchExists = errors.New("batch-id already exists")

	// ErrBatchInAuction if the provided batch id is already in auction status.
	ErrBatchInAuction = errors.New("batch-id already in auction")

	// ErrBatchContainsUnknownStorageRequest is returned if a batch contains an
	// unknown storage request.
	ErrBatchContainsUnknownStorageRequest = fmt.Errorf("batch contains an unknown storage request")

	log = logger.Logger("store")
)

// Store provides a persistent layer for storage requests.
type Store struct {
	conn    *sql.DB
	db      *db.Queries
	lock    sync.Mutex
	entropy *ulid.MonotonicEntropy
}

// New returns a new Store backed by `postgresURI`.
func New(postgresURI string) (*Store, error) {
	as := bindata.Resource(migrations.AssetNames(),
		func(name string) ([]byte, error) {
			return migrations.Asset(name)
		})
	conn, err := storeutil.MigrateAndConnectToDB(postgresURI, as)
	if err != nil {
		return nil, fmt.Errorf("initializing db connection: %s", err)
	}

	return &Store{conn: conn, db: db.New(conn)}, nil
}

// CtxWithTx attach a database transaction to the context. It returns the
// context unchanged if there's error starting the transaction.
func (s *Store) CtxWithTx(ctx context.Context, opts ...storeutil.TxOptions) (context.Context, error) {
	return storeutil.CtxWithTx(ctx, s.conn, opts...)
}

//nolint:unparam
func (s *Store) withTx(ctx context.Context, f func(*db.Queries) error, opts ...storeutil.TxOptions) (err error) {
	return storeutil.WithTx(ctx, s.conn, func(tx *sql.Tx) error {
		return f(s.db.WithTx(tx))
	}, opts...)
}

func (s *Store) withCtxTx(ctx context.Context, f func(*db.Queries) error) (err error) {
	return storeutil.WithCtxTx(ctx,
		func(tx *sql.Tx) error { return f(s.db.WithTx(tx)) },
		func() error { return f(s.db) })
}

// CreateStorageRequest creates the provided StorageRequest in store.
func (s *Store) CreateStorageRequest(ctx context.Context, br broker.StorageRequest) error {
	return s.withCtxTx(ctx, func(q *db.Queries) error {
		return q.CreateStorageRequest(ctx,
			db.CreateStorageRequestParams{
				ID:      br.ID,
				DataCid: br.DataCid.String(),
				Status:  br.Status,
				Origin:  br.Origin,
			})
	})
}

// GetStorageRequest gets a StorageRequest with the specified `id`. If not found returns ErrNotFound.
func (s *Store) GetStorageRequest(
	ctx context.Context,
	id broker.StorageRequestID) (br broker.StorageRequest, err error) {
	err = s.withCtxTx(ctx, func(q *db.Queries) error {
		r, err := s.db.GetStorageRequest(ctx, id)
		if err == sql.ErrNoRows {
			return ErrNotFound
		}
		if err != nil {
			return err
		}

		dataCid, err := cid.Parse(r.DataCid)
		if err != nil {
			return err
		}
		br = broker.StorageRequest{
			ID:        r.ID,
			DataCid:   dataCid,
			Status:    r.Status,
			Origin:    r.Origin,
			BatchID:   broker.BatchID(r.BatchID.String),
			CreatedAt: r.CreatedAt,
			UpdatedAt: r.UpdatedAt,
		}
		return err
	})
	return
}

// GetRemoteWalletConfig gets the remote wallet configured to sign deals from a batch.
// If none was configured, it *does not return an error but a nil result*.
func (s *Store) GetRemoteWalletConfig(ctx context.Context, id broker.BatchID) (rw *broker.RemoteWallet, err error) {
	err = s.withCtxTx(ctx, func(q *db.Queries) error {
		dbrw, err := q.GetBatchRemoteWallet(ctx, id)
		if err == sql.ErrNoRows {
			return nil
		}
		if err != nil {
			return err
		}

		peerID, err := peer.Decode(dbrw.PeerID)
		if err != nil {
			return fmt.Errorf("decoding peer-id: %s", err)
		}
		waddr, err := address.NewFromString(dbrw.WalletAddr)
		if err != nil {
			return fmt.Errorf("decoding wallet address: %s", err)
		}
		if waddr.Empty() {
			return errors.New("invalid wallet address (empty)")
		}
		maddrs := make([]multiaddr.Multiaddr, len(dbrw.Multiaddrs))
		for i, strmaddr := range dbrw.Multiaddrs {
			maddr, err := multiaddr.NewMultiaddr(strmaddr)
			if err != nil {
				return fmt.Errorf("decoding multiaddr %s: %s", strmaddr, err)
			}
			maddrs[i] = maddr
		}

		rw = &broker.RemoteWallet{
			PeerID:     peerID,
			AuthToken:  dbrw.AuthToken,
			WalletAddr: waddr,
			Multiaddrs: maddrs,
		}
		return err
	})
	return
}

// CreateBatch persists a batch.
func (s *Store) CreateBatch(
	ctx context.Context,
	ba *broker.Batch,
	brIDs []broker.StorageRequestID,
	manifest []byte,
	rw *broker.RemoteWallet) error {
	if ba.ID == "" {
		return fmt.Errorf("batch id is empty")
	}
	var brStatus broker.StorageRequestStatus
	// StorageRequests status should mirror the Batch status.
	switch ba.Status {
	case broker.BatchStatusPreparing:
		brStatus = broker.RequestPreparing
	case broker.BatchStatusAuctioning:
		brStatus = broker.RequestAuctioning
	default:
		return fmt.Errorf("unexpected batch initial status %s", ba.Status)
	}

	start := time.Now()
	defer log.Debugf(
		"creating batch %s with group size %d took %dms",
		ba.ID, len(brIDs),
		time.Since(start).Milliseconds(),
	)

	// 2- Persist the Batch.
	dsources := struct {
		carURL         string
		ipfsCid        string
		ipfsMultiaddrs []string
	}{}
	if ba.Sources.CARURL != nil {
		dsources.carURL = ba.Sources.CARURL.URL.String()
	}
	if ba.Sources.CARIPFS != nil {
		dsources.ipfsCid = ba.Sources.CARIPFS.Cid.String()
		dsources.ipfsMultiaddrs = make([]string, len(ba.Sources.CARIPFS.Multiaddrs))
		for i, maddr := range ba.Sources.CARIPFS.Multiaddrs {
			dsources.ipfsMultiaddrs[i] = maddr.String()
		}
	}
	var pieceCid string
	if ba.PieceCid.Defined() {
		pieceCid = ba.PieceCid.String()
	}
	var payloadSize sql.NullInt64
	if ba.PayloadSize != nil {
		payloadSize.Valid = true
		payloadSize.Int64 = *ba.PayloadSize
	}
	isd := db.CreateBatchParams{
		ID:                         ba.ID,
		Status:                     ba.Status,
		RepFactor:                  ba.RepFactor,
		DealDuration:               ba.DealDuration,
		PayloadCid:                 ba.PayloadCid.String(),
		PayloadSize:                payloadSize,
		PieceCid:                   pieceCid,
		PieceSize:                  ba.PieceSize,
		DisallowRebatching:         ba.DisallowRebatching,
		FilEpochDeadline:           ba.FilEpochDeadline,
		ProposalStartOffsetSeconds: int64(ba.ProposalStartOffset.Seconds()),
		CarUrl:                     dsources.carURL,
		CarIpfsCid:                 dsources.ipfsCid,
		CarIpfsAddrs:               strings.Join(dsources.ipfsMultiaddrs, ","),
		Origin:                     ba.Origin,
		Providers:                  common.StringifyAddrs(ba.Providers...),
	}
	ids := make([]string, len(brIDs))
	for i, id := range brIDs {
		ids[i] = string(id)
	}

	return s.withTx(ctx, func(txn *db.Queries) error {
		if err := txn.CreateBatch(ctx, isd); err != nil {
			if err, ok := err.(*pgconn.PgError); ok {
				if err.Code == "23505" {
					return ErrBatchExists
				}
			}
			return fmt.Errorf("creating batch: %s", err)
		}
		updated, err := txn.BatchUpdateStorageRequests(ctx, db.BatchUpdateStorageRequestsParams{
			Ids: ids, Status: brStatus, BatchID: batchIDToSQL(ba.ID)})
		if err != nil {
			return fmt.Errorf("updating storage requests: %s", err)
		}
		// this check should be within the transaction to rollback when required
		if len(updated) < len(ids) {
			return fmt.Errorf("unknown storage request ids %v: %w",
				sliceDiff(ids, updated), ErrBatchContainsUnknownStorageRequest)
		}

		for key, value := range ba.Tags {
			if err := txn.CreateBatchTag(ctx, db.CreateBatchTagParams{
				BatchID: ba.ID,
				Key:     key,
				Value:   value,
			}); err != nil {
				return fmt.Errorf("creating tag: %s", err)
			}
		}

		if manifest != nil {
			param := db.CreateBatchManifestParams{
				BatchID:  string(ba.ID),
				Manifest: manifest,
			}
			if err := txn.CreateBatchManifest(ctx, param); err != nil {
				return fmt.Errorf("saving batch manifest: %s", err)
			}
		}

		if rw != nil {
			strmaddrs := make([]string, len(rw.Multiaddrs))
			for i, maddr := range rw.Multiaddrs {
				strmaddrs[i] = maddr.String()
			}
			param := db.CreateBatchRemoteWalletParams{
				BatchID:    ba.ID,
				PeerID:     rw.PeerID.String(),
				AuthToken:  rw.AuthToken,
				WalletAddr: rw.WalletAddr.String(),
				Multiaddrs: strmaddrs,
			}
			if err := txn.CreateBatchRemoteWallet(ctx, param); err != nil {
				return fmt.Errorf("saving batch remote wallet: %s", err)
			}
		}

		return nil
	})
}

// BatchToAuctioning moves a batch and the underlying storage requests
// to Auctioning status.
func (s *Store) BatchToAuctioning(
	ctx context.Context,
	id broker.BatchID,
	pieceCid cid.Cid,
	pieceSize uint64) error {
	return s.withTx(ctx, func(txn *db.Queries) error {
		if err := txn.UpdateBatch(ctx, db.UpdateBatchParams{
			ID:        id,
			Status:    broker.BatchStatusAuctioning,
			PieceCid:  pieceCid.String(),
			PieceSize: pieceSize,
		}); err != nil {
			return fmt.Errorf("update batch: %s", err)
		}

		if err := txn.UpdateStorageRequestsStatus(ctx, db.UpdateStorageRequestsStatusParams{
			Status: broker.RequestAuctioning, BatchID: batchIDToSQL(id)}); err != nil {
			return fmt.Errorf("update storage requests status: %s", err)
		}
		return nil
	})
}

// BatchError moves a batch to an error status with a specified error cause.
func (s *Store) BatchError(
	ctx context.Context,
	id broker.BatchID,
	errorCause string,
	rebatch bool) (brIDs []broker.StorageRequestID, err error) {
	err = s.withTx(ctx, func(txn *db.Queries) error {
		ba, err := txn.GetBatch(ctx, id)
		if err != nil {
			return fmt.Errorf("get batch: %s", err)
		}

		// 1. Verify some pre-state conditions.
		switch ba.Status {
		case broker.BatchStatusAuctioning, broker.BatchStatusDealMaking:
			if ba.Error != "" {
				return fmt.Errorf("error cause should be empty: %s", ba.Error)
			}
		case broker.BatchStatusError:
			brIDs, err = txn.GetStorageRequestIDs(ctx, batchIDToSQL(id))
			return err
		case broker.BatchStatusSuccess:
			log.Warnf("ignoring batch erroring due to being on success, cause: %s", ba.Error)
			return nil
		default:
			return fmt.Errorf("wrong batch %s status transition, tried from %s", ba.ID, ba.Status)
		}

		// 2. Move the Batch to BatchError with the error cause.
		if err := txn.UpdateBatchStatusAndError(ctx, db.UpdateBatchStatusAndErrorParams{
			ID: id, Error: errorCause, Status: broker.BatchStatusError}); err != nil {
			return fmt.Errorf("save batch: %s", err)
		}

		// 3. Move every underlying StorageRequest to batching again, since they will be re-batched.
		status := broker.RequestError
		if rebatch {
			status = broker.RequestBatching
		}

		if err := txn.UpdateStorageRequestsStatus(ctx, db.UpdateStorageRequestsStatusParams{
			BatchID: batchIDToSQL(id), Status: status}); err != nil {
			return fmt.Errorf("getting storage request: %s", err)
		}
		if rebatch {
			if err := txn.RebatchStorageRequests(ctx, db.RebatchStorageRequestsParams{
				BatchID: batchIDToSQL(id), ErrorCause: errorCause}); err != nil {
				return fmt.Errorf("getting storage request: %s", err)
			}
		}

		// 4. Mark the batchCid as unpinnable, since it won't be used anymore for auctions or deals.
		unpinID, err := s.newID()
		if err != nil {
			return fmt.Errorf("generating id for unpin job: %s", err)
		}

		if !ba.DisallowRebatching {
			if err := txn.CreateUnpinJob(ctx, db.CreateUnpinJobParams{
				ID: unpinID, Cid: ba.PayloadCid, Type: int16(UnpinTypeBatch)}); err != nil {
				return fmt.Errorf("saving unpin job: %s", err)
			}
		}

		brIDs, err = txn.GetStorageRequestIDs(ctx, batchIDToSQL(id))
		return err
	})
	return
}

// BatchSuccess moves a batch and the underlying storage requests to
// Success status.
func (s *Store) BatchSuccess(ctx context.Context, id broker.BatchID) error {
	return s.withTx(ctx, func(txn *db.Queries) error {
		ba, err := txn.GetBatch(ctx, id)
		if err != nil {
			return fmt.Errorf("get batch: %s", err)
		}

		if err := txn.UpdateBatchStatus(ctx, db.UpdateBatchStatusParams{
			ID: id, Status: broker.BatchStatusSuccess}); err != nil {
			return fmt.Errorf("updating batch status: %s", err)
		}

		if err := txn.UpdateStorageRequestsStatus(ctx, db.UpdateStorageRequestsStatusParams{
			BatchID: batchIDToSQL(id), Status: broker.RequestSuccess}); err != nil {
			return fmt.Errorf("updating  storage requests status: %s", err)
		}

		brs, err := txn.GetStorageRequests(ctx, batchIDToSQL(id))
		if err != nil {
			return fmt.Errorf("getting storage requests: %s", err)
		}

		if !ba.DisallowRebatching {
			for _, br := range brs {
				unpinID, err := s.newID()
				if err != nil {
					return fmt.Errorf("generating id for unpin job: %s", err)
				}
				if err := txn.CreateUnpinJob(ctx, db.CreateUnpinJobParams{
					ID: unpinID, Cid: br.DataCid, Type: int16(UnpinTypeData)}); err != nil {
					return fmt.Errorf("saving unpin job: %s", err)
				}
			}

			unpinID, err := s.newID()
			if err != nil {
				return fmt.Errorf("generating id for unpin job: %s", err)
			}
			if err := txn.CreateUnpinJob(ctx, db.CreateUnpinJobParams{
				ID: unpinID, Cid: ba.PayloadCid, Type: int16(UnpinTypeBatch)}); err != nil {
				return fmt.Errorf("saving unpin job: %s", err)
			}
		}
		return nil
	})
}

// AddDeals includes new deals from a finalized auction. Confirming the winners in the database
// can discover the fact that a concurrent auction already made some of them winners.
// The fully confirmed accepted winners are returned in the acceptedWinners return parameter.
func (s *Store) AddDeals(ctx context.Context, auction broker.ClosedAuction) (acceptedWinners []string, err error) {
	err = s.withTx(ctx, func(txn *db.Queries) error {
		ba, err := txn.GetBatch(ctx, auction.BatchID)
		if err != nil {
			return fmt.Errorf("get batch: %s", err)
		}

		// Take care of correct state transitions.
		if ba.Status != broker.BatchStatusAuctioning && ba.Status != broker.BatchStatusDealMaking {
			return fmt.Errorf("wrong storage request status transition, tried moving to %s", ba.Status)
		}

		excludedParams := db.GetExcludedStorageProvidersParams{
			PieceCid: ba.PieceCid,
			Origin:   ba.Origin,
		}
		var excludedProviders []string

		// Only consider excluding auction winners from looking at history for non direct to
		// provider auctions, since we allow the client to have total control of this.
		if len(ba.Providers) == 0 {
			excludedProviders, err = txn.GetExcludedStorageProviders(ctx, excludedParams)
			if err != nil {
				return fmt.Errorf("get excluded providers: %s", err)
			}
		}

		// Add winning bids to list of deals.
		for bidID, bid := range auction.WinningBids {
			var alreadyWon bool
			for _, excludedProvider := range excludedProviders {
				if excludedProvider == bid.StorageProviderID {
					alreadyWon = true
					break
				}
			}
			if alreadyWon {
				log.Warnf("excluding winner %s in auction %s since already won", bid.StorageProviderID, auction.ID)
				continue
			}

			acceptedWinners = append(acceptedWinners, bid.StorageProviderID)
			if err := txn.CreateDeal(ctx, db.CreateDealParams{
				BatchID:           auction.BatchID,
				AuctionID:         auction.ID,
				BidID:             bidID,
				StorageProviderID: bid.StorageProviderID,
			}); err != nil {
				return fmt.Errorf("save batch: %s", err)
			}
		}

		moveStorageRequestsToDealMaking := false
		if ba.Status == broker.BatchStatusAuctioning {
			if err := txn.UpdateBatchStatus(ctx, db.UpdateBatchStatusParams{
				ID: auction.BatchID, Status: broker.BatchStatusDealMaking}); err != nil {
				return fmt.Errorf("save batch: %s", err)
			}
			moveStorageRequestsToDealMaking = true
		}

		// If the batch was already in BatchDealMaking, then we're
		// just adding more winning bids from new auctions we created to statisfy the
		// replication factor. That case means we already updated the underlying
		// storage-requests status.
		// This conditional is to only move the storage-request on the first successful auction
		// that might happen. On further auctions, we don't need to do this again.
		if moveStorageRequestsToDealMaking {
			if err := txn.UpdateStorageRequestsStatus(ctx, db.UpdateStorageRequestsStatusParams{
				BatchID: batchIDToSQL(ba.ID), Status: broker.RequestDealMaking}); err != nil {
				return fmt.Errorf("saving storage request: %s", err)
			}
		}

		return nil
	})
	return
}

// GetBatch gets an existing batch by id. If the batch doesn't exists, it returns
// ErrNotFound.
func (s *Store) GetBatch(ctx context.Context, id broker.BatchID) (ba *broker.Batch, err error) {
	err = s.withCtxTx(ctx, func(q *db.Queries) error {
		var dbBatch db.Batch
		dbBatch, err = q.GetBatch(ctx, id)
		if err != nil {
			return err
		}
		tags, err := q.GetBatchTags(ctx, id)
		if err != nil {
			return err
		}
		ba, err = batchFromDB(&dbBatch, tags)
		return err
	})
	return
}

// GetBatchManifest gets the stored manifest of a batch. If none is found it returns ErrNotFound.
func (s *Store) GetBatchManifest(ctx context.Context, id broker.BatchID) ([]byte, error) {
	var manifest []byte
	if err := s.withCtxTx(ctx, func(q *db.Queries) error {
		bm, err := q.GetBatchManifest(ctx, string(id))
		if err == sql.ErrNoRows {
			return ErrNotFound
		} else if err != nil {
			return fmt.Errorf("db get manifest: %s", err)
		}
		manifest = bm.Manifest
		return nil
	}); err != nil {
		return nil, err
	}

	return manifest, nil
}

// GetDeals gets storage-provider deals for a batch.
func (s *Store) GetDeals(ctx context.Context, id broker.BatchID) (deals []db.Deal, err error) {
	err = s.withCtxTx(ctx, func(q *db.Queries) error {
		deals, err = q.GetDeals(ctx, id)
		return err
	})
	return
}

// GetStorageRequestIDs gets the ids of the storage requests for a batch.
func (s *Store) GetStorageRequestIDs(ctx context.Context, id broker.BatchID) (
	brIDs []broker.StorageRequestID, err error) {
	err = s.withCtxTx(ctx, func(q *db.Queries) error {
		brIDs, err = q.GetStorageRequestIDs(ctx, batchIDToSQL(id))
		return err
	})
	return
}

// OperationExists checks if the operation ID already exists in db.
func (s *Store) OperationExists(ctx context.Context, opID string) (exists bool, err error) {
	err = s.withCtxTx(ctx, func(q *db.Queries) error {
		err = q.CreateOperation(ctx, opID)
		if err, ok := err.(*pgconn.PgError); ok {
			if err.Code == "23505" {
				exists = true
				return nil
			}
		}
		exists = false
		return err
	})
	return
}

// SaveDeals saves a new finalized (succeeded or errored) auction deal
// into the batch.
func (s *Store) SaveDeals(ctx context.Context, fad broker.FinalizedDeal) error {
	return s.withCtxTx(ctx, func(q *db.Queries) error {
		rows, err := q.UpdateDeals(ctx,
			db.UpdateDealsParams{
				BatchID:           fad.BatchID,
				StorageProviderID: fad.StorageProviderID,
				DealExpiration:    fad.DealExpiration,
				DealID:            fad.DealID,
				ErrorCause:        fad.ErrorCause,
			})
		if err != nil {
			return fmt.Errorf("update deal: %s", err)
		}
		if rows == 0 {
			return fmt.Errorf("deal not found: %v", fad)
		}
		return nil
	})
}

// GetExcludedStorageProviders returns a list of all storage providers that have won
// auctions fired by the origin for the provided PieceCid. Note that this considers
// all auctions history for the PieceCid and the origin.
func (s *Store) GetExcludedStorageProviders(
	ctx context.Context,
	pieceCid cid.Cid,
	origin string) (sps []string, err error) {
	err = s.withCtxTx(ctx, func(q *db.Queries) error {
		params := db.GetExcludedStorageProvidersParams{
			PieceCid: pieceCid.String(),
			Origin:   origin,
		}
		sps, err = q.GetExcludedStorageProviders(ctx, params)
		if err != nil {
			return fmt.Errorf("calling get excluded storage providers: %s", err)
		}
		return nil
	})
	return
}

func (s *Store) newID() (string, error) {
	s.lock.Lock()
	// Not deferring unlock since can be recursive.

	if s.entropy == nil {
		s.entropy = ulid.Monotonic(rand.Reader, 0)
	}
	id, err := ulid.New(ulid.Timestamp(time.Now().UTC()), s.entropy)
	if errors.Is(err, ulid.ErrMonotonicOverflow) {
		s.entropy = nil
		s.lock.Unlock()
		return s.newID()
	} else if err != nil {
		s.lock.Unlock()
		return "", fmt.Errorf("generating id: %v", err)
	}
	s.lock.Unlock()
	return strings.ToLower(id.String()), nil
}

func batchFromDB(dbBatch *db.Batch, tags []db.BatchTag) (ba *broker.Batch, err error) {
	var payloadCid cid.Cid
	if dbBatch.PayloadCid != "" {
		payloadCid, err = cid.Parse(dbBatch.PayloadCid)
		if err != nil {
			return nil, fmt.Errorf("parsing payload CID: %s", err)
		}
	}
	var pieceCid cid.Cid
	if dbBatch.PieceCid != "" {
		pieceCid, err = cid.Parse(dbBatch.PieceCid)
		if err != nil {
			return nil, fmt.Errorf("parsing piece CID: %s", err)
		}
	}
	var sources auction.Sources
	if u, err := url.ParseRequestURI(dbBatch.CarUrl); err == nil {
		sources.CARURL = &auction.CARURL{URL: *u}
	}
	if dbBatch.CarIpfsCid != "" {
		if id, err := cid.Parse(dbBatch.CarIpfsCid); err == nil {
			carIPFS := &auction.CARIPFS{Cid: id}
			addrs := strings.Split(dbBatch.CarIpfsAddrs, ",")
			for _, addr := range addrs {
				if ma, err := multiaddr.NewMultiaddr(addr); err == nil {
					carIPFS.Multiaddrs = append(carIPFS.Multiaddrs, ma)
				} else {
					return nil, fmt.Errorf("parsing IPFS multiaddr: %s", err)
				}
			}
			sources.CARIPFS = carIPFS
		}
	}
	providers := make([]address.Address, len(dbBatch.Providers))
	for i, strProv := range dbBatch.Providers {
		providerAddr, err := address.NewFromString(strProv)
		if err != nil {
			return nil, fmt.Errorf("parsing provider %s: %s", strProv, err)
		}
		if providerAddr.Protocol() != address.ID {
			return nil, fmt.Errorf("%s should be an identity address", strProv)
		}
		providers[i] = providerAddr
	}

	mtags := make(map[string]string, len(tags))
	for _, tag := range tags {
		mtags[tag.Key] = tag.Value
	}

	var payloadSize *int64
	if dbBatch.PayloadSize.Valid {
		payloadSize = &dbBatch.PayloadSize.Int64
	}

	return &broker.Batch{
		ID:                  dbBatch.ID,
		Status:              dbBatch.Status,
		RepFactor:           dbBatch.RepFactor,
		DealDuration:        dbBatch.DealDuration,
		PayloadCid:          payloadCid,
		PayloadSize:         payloadSize,
		PieceCid:            pieceCid,
		PieceSize:           dbBatch.PieceSize,
		Sources:             sources,
		DisallowRebatching:  dbBatch.DisallowRebatching,
		FilEpochDeadline:    dbBatch.FilEpochDeadline,
		ProposalStartOffset: time.Second * time.Duration(dbBatch.ProposalStartOffsetSeconds),
		Origin:              dbBatch.Origin,
		Tags:                mtags,
		Providers:           providers,
		Error:               dbBatch.Error,
		CreatedAt:           dbBatch.CreatedAt,
		UpdatedAt:           dbBatch.UpdatedAt,
	}, nil
}

func batchIDToSQL(id broker.BatchID) sql.NullString {
	return sql.NullString{String: string(id), Valid: true}
}

func sliceDiff(full []string, subset []broker.StorageRequestID) (diff []string) {
	fullSet := make(map[string]struct{}, len(full))
	for _, s := range full {
		fullSet[s] = struct{}{}
	}
	for _, s := range subset {
		if _, found := fullSet[string(s)]; !found {
			diff = append(diff, string(s))
		}
	}
	return
}
