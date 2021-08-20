package queue

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

	bindata "github.com/golang-migrate/migrate/v4/source/go_bindata"
	"github.com/ipfs/go-cid"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/multiformats/go-multiaddr"
	"github.com/oklog/ulid/v2"
	"github.com/textileio/bidbot/lib/auction"
	"github.com/textileio/broker-core/auctioneer"
	"github.com/textileio/broker-core/broker"
	"github.com/textileio/broker-core/cmd/auctioneerd/auctioneer/queue/internal/db"
	"github.com/textileio/broker-core/cmd/auctioneerd/auctioneer/queue/migrations"
	"github.com/textileio/broker-core/storeutil"
	golog "github.com/textileio/go-log/v2"
)

var (
	log = golog.Logger("auctioneer/queue")

	// StartDelay is the time delay before the queue will process queued auctions on start.
	StartDelay = time.Second * 10

	// stuckSeconds is the seconds elapsed before an started auction is
	// considered stuck and can be rescheduled.
	stuckSeconds = int64(600)

	// MaxConcurrency is the maximum number of auctions that will be handled concurrently.
	MaxConcurrency = 1

	// ErrAuctionNotFound indicates the requested auction was not found.
	ErrAuctionNotFound = errors.New("auction not found")

	// ErrBidNotFound indicates the requested bid was not found.
	ErrBidNotFound = errors.New("bid not found")

	// ErrProposalDelivered indicates the requested bid already has proposal CID delivered.
	ErrProposalDelivered = errors.New("prooposal cid was delivered")
)

// Handler is called when an auction moves from "queued" to "started".
type Handler func(
	ctx context.Context,
	auction auctioneer.Auction,
	addBid func(bid auctioneer.Bid) (auction.BidID, error),
) (map[auction.BidID]auctioneer.WinningBid, error)

// Finalizer is called when an auction moves from "started" to "finalized".
type Finalizer func(ctx context.Context, auction *auctioneer.Auction) error

// Queue is a persistent worker-based task queue.
type Queue struct {
	conn *sql.DB
	db   *db.Queries

	handler   Handler
	finalizer Finalizer
	jobCh     chan *auctioneer.Auction
	tickCh    chan struct{}
	entropy   *ulid.MonotonicEntropy

	ctx    context.Context
	cancel context.CancelFunc

	wg sync.WaitGroup
	lk sync.Mutex
}

// NewQueue returns a new Queue using handler to process auctions.
func NewQueue(postgresURI string, handler Handler, finalizer Finalizer) (*Queue, error) {
	ctx, cancel := context.WithCancel(context.Background())
	as := bindata.Resource(migrations.AssetNames(),
		func(name string) ([]byte, error) {
			return migrations.Asset(name)
		})
	conn, err := storeutil.MigrateAndConnectToDB(postgresURI, as)
	if err != nil {
		cancel()
		return nil, err
	}

	q := &Queue{
		conn:      conn,
		db:        db.New(conn),
		handler:   handler,
		finalizer: finalizer,
		jobCh:     make(chan *auctioneer.Auction, MaxConcurrency),
		tickCh:    make(chan struct{}, MaxConcurrency),
		ctx:       ctx,
		cancel:    cancel,
	}

	// Create queue workers
	q.wg.Add(MaxConcurrency)
	for i := 0; i < MaxConcurrency; i++ {
		go q.worker(i + 1)
	}

	go q.start()
	return q, nil
}

// Close the queue. This will wait for "started" auctions.
func (q *Queue) Close() error {
	q.cancel()
	q.wg.Wait()
	return nil
}

// CreateAuction adds a new auction to the queue.
// The new auction will be handled immediately if workers are not busy.
func (q *Queue) CreateAuction(ctx context.Context, a auctioneer.Auction) error {
	if err := validate(a); err != nil {
		return fmt.Errorf("invalid auction data: %s", err)
	}
	params := db.CreateAuctionParams{
		ID:               a.ID,
		BatchID:          a.BatchID,
		DealSize:         int64(a.DealSize),
		DealDuration:     a.DealDuration,
		DealReplication:  int32(a.DealReplication),
		DealVerified:     a.DealVerified,
		FilEpochDeadline: a.FilEpochDeadline,
		PayloadCid:       a.PayloadCid.String(),
		Status:           broker.AuctionStatusQueued,
		Duration:         int64(a.Duration),
	}
	if a.ExcludedStorageProviders == nil {
		params.ExcludedStorageProviders = []string{}
	}
	if a.Sources.CARURL != nil {
		params.CarUrl = a.Sources.CARURL.URL.String()
	}
	params.CarIpfsAddrs = []string{}
	if a.Sources.CARIPFS != nil {
		params.CarIpfsCid = a.Sources.CARIPFS.Cid.String()
		for _, addr := range a.Sources.CARIPFS.Multiaddrs {
			params.CarIpfsAddrs = append(params.CarIpfsAddrs, addr.String())
		}
	}

	if err := q.db.CreateAuction(ctx, params); err != nil {
		return fmt.Errorf("creating auction: %v", err)
	}
	log.Debugf("created auction %s", a.ID)
	q.enqueue(&a)
	return nil
}

func validate(a auctioneer.Auction) error {
	if a.ID == "" {
		return errors.New("auction id is empty")
	}
	if a.BatchID == "" {
		return errors.New("batch id is empty")
	}
	if !a.PayloadCid.Defined() {
		return errors.New("payload cid is empty")
	}
	if a.DealSize == 0 {
		return errors.New("deal size must be greater than zero")
	}
	if a.DealDuration == 0 {
		return errors.New("deal duration must be greater than zero")
	}
	if a.DealReplication == 0 {
		return errors.New("deal replication must be greater than zero")
	}
	if err := a.Sources.Validate(); err != nil {
		return err
	}
	if a.Status != broker.AuctionStatusUnspecified {
		return errors.New("invalid initial auction status")
	}
	if len(a.Bids) != 0 {
		return errors.New("initial bids must be empty")
	}
	if len(a.WinningBids) != 0 {
		return errors.New("initial winning bids must be empty")
	}
	if !a.StartedAt.IsZero() {
		return errors.New("initial started at must be zero")
	}
	if !a.UpdatedAt.IsZero() {
		return errors.New("initial updated at must be zero")
	}
	if a.Duration == 0 {
		return errors.New("duration must be greater than zero")
	}
	if a.ErrorCause != "" {
		return errors.New("initial error cause must be empty")
	}
	return nil
}

// newID returns new monotonically increasing auction ids.
func (q *Queue) newID(t time.Time) (auction.ID, error) {
	q.lk.Lock() // entropy is not safe for concurrent use

	if q.entropy == nil {
		q.entropy = ulid.Monotonic(rand.Reader, 0)
	}
	id, err := ulid.New(ulid.Timestamp(t.UTC()), q.entropy)
	if errors.Is(err, ulid.ErrMonotonicOverflow) {
		q.entropy = nil
		q.lk.Unlock()
		return q.newID(t)
	} else if err != nil {
		q.lk.Unlock()
		return "", fmt.Errorf("generating id: %v", err)
	}
	q.lk.Unlock()
	return auction.ID(strings.ToLower(id.String())), nil
}

// GetAuction returns an auction by id.
// If an auction is not found for id, ErrAuctionNotFound is returned.
func (q *Queue) GetAuction(ctx context.Context, id auction.ID) (a *auctioneer.Auction, err error) {
	err = storeutil.WithTx(ctx, q.conn, func(tx *sql.Tx) error {
		txn := q.db.WithTx(tx)
		record, err := txn.GetAuction(ctx, id)
		if err == sql.ErrNoRows {
			return ErrAuctionNotFound
		} else if err != nil {
			return fmt.Errorf("getting auction: %v", err)
		}
		a, err = auctionFromDb(record)
		if err != nil {
			return fmt.Errorf("auction from db: %v", err)
		}
		bids, err := txn.GetAuctionBids(ctx, id)
		if err != nil {
			return fmt.Errorf("getting auction bids: %v", err)
		}
		a.Bids = make(map[auction.BidID]auctioneer.Bid)
		a.WinningBids = make(map[auction.BidID]auctioneer.WinningBid)
		for _, b := range bids {
			bid, err := bidFromDb(b)
			if err != nil {
				return err
			}
			a.Bids[b.ID] = *bid
			if !bid.WonAt.IsZero() {
				a.WinningBids[b.ID] = auctioneer.WinningBid{
					BidderID:    bid.BidderID,
					ProposalCid: bid.ProposalCid,
					ErrorCause:  bid.ProposalCidDeliveryError,
				}
			}
		}
		return nil
	})
	return
}

// GetBidderID returns the bidder ID for a bid.
// If an auction is not found for id, ErrAuctionNotFound is returned.
// If a bid is not found or id, ErrBidNotFound is returned.
// If an proposal CID is already delivered for the bid, ErrProposalDelivered is returned.
func (q *Queue) GetBidderID(
	ctx context.Context,
	id auction.ID,
	bid auction.BidID,
) (peer.ID, error) {
	a, err := q.GetAuction(ctx, id)
	if err != nil {
		return "", err
	}
	// Check if auction is in good standing
	if a.Status != broker.AuctionStatusFinalized {
		return "", errors.New("auction is not finalized")
	}
	if a.ErrorCause != "" {
		return "", errors.New("auction finalized with error; can't set proposal cid")
	}

	var matched *auctioneer.Bid
	for _, b := range a.Bids {
		if b.ID == bid {
			matched = &b
			break
		}
	}
	if matched == nil {
		return "", ErrBidNotFound
	}
	if matched.ProposalCid.Defined() {
		return "", ErrProposalDelivered
	}
	return matched.BidderID, nil
}

// SetProposalCidDelivered saves the proposal CID for the bid.
func (q *Queue) SetProposalCidDelivered(
	ctx context.Context,
	auctionID auction.ID,
	bidID auction.BidID,
	pcid cid.Cid) error {
	return q.db.UpdateProposalCid(ctx, db.UpdateProposalCidParams{
		ID:          bidID,
		AuctionID:   auctionID,
		ProposalCid: sql.NullString{String: pcid.String(), Valid: true},
	})
}

// SetProposalCidDeliveryError saves the error happened when delivering the
// proposal cid to the winner.
func (q *Queue) SetProposalCidDeliveryError(
	ctx context.Context,
	auctionID auction.ID,
	bidID auction.BidID,
	errCause string) error {
	return q.db.UpdateProposalCidDeliveryError(ctx, db.UpdateProposalCidDeliveryErrorParams{
		ID:                       bidID,
		AuctionID:                auctionID,
		ProposalCidDeliveryError: sql.NullString{String: errCause, Valid: true},
	})
}

func (q *Queue) enqueue(a *auctioneer.Auction) {
	if err := q.saveAndTransitionStatus(q.ctx, a, broker.AuctionStatusStarted); err != nil {
		log.Errorf("updating status (started): %v", err)
		return
	}
	select {
	case q.jobCh <- a:
	default:
		log.Debugf("workers are busy; queueing %s ", a.ID)
		if err := q.saveAndTransitionStatus(q.ctx, a, broker.AuctionStatusQueued); err != nil {
			log.Errorf("updating status (queued): %v", err)
		}
	}
}

func (q *Queue) worker(num int) {
	defer q.wg.Done()

	for {
		select {
		case <-q.ctx.Done():
			return

		case a := <-q.jobCh:
			log.Infof("worker %d started auction %s", num, a.ID)
			// Handle the auction with the handler func
			wbs, err := q.handler(q.ctx, *a, func(bid auctioneer.Bid) (auction.BidID, error) {
				return q.addBid(a, bid)
			})
			if err != nil {
				a.ErrorCause = err.Error()
				log.Debugf("auction %s failed: %s", a.ID, a.ErrorCause)
			}

			// Update winning bids; some bids may have been processed even if there was an error
			var ids []string
			for id := range wbs {
				ids = append(ids, string(id))
			}
			updated, err := q.db.UpdateBidsWonAt(q.ctx, db.UpdateBidsWonAtParams{
				BidIds:    ids,
				AuctionID: a.ID,
			})
			// failing to write db is not the end of the world. just move on.
			if err != nil {
				log.Errorf("error update winning bids: %v", err)
			}
			if len(updated) != len(ids) {
				log.Errorf("should have updated %d winning bids for auction %s, only updated %d", len(wbs), a.ID, len(updated))
			}
			a.WinningBids = wbs
			q.saveAndFinalizeAuction(a)
			select {
			case q.tickCh <- struct{}{}:
			default:
			}
		}
	}
}

func (q *Queue) saveAndFinalizeAuction(a *auctioneer.Auction) {
	if err := q.saveAndTransitionStatus(q.ctx, a, broker.AuctionStatusFinalized); err != nil {
		log.Errorf("updating status (%s): %v", broker.AuctionStatusFinalized, err)
		return
	}

	// Finish auction with the finalizer func
	if err := q.finalizer(q.ctx, a); err != nil {
		a.ErrorCause = err.Error()

		// Save error
		if err := q.saveAndTransitionStatus(q.ctx, a, a.Status); err != nil {
			log.Errorf("saving finalizer error: %v", err)
		}
	}
}

func (q *Queue) addBid(a *auctioneer.Auction, bid auctioneer.Bid) (auction.BidID, error) {
	if a.Status != broker.AuctionStatusStarted {
		return "", errors.New("auction has not started")
	}
	id, err := q.newID(bid.ReceivedAt)
	if err != nil {
		return "", fmt.Errorf("generating bid id: %v", err)
	}
	if a.Bids == nil {
		a.Bids = make(map[auction.BidID]auctioneer.Bid)
	}
	bidID := auction.BidID(id)
	a.Bids[bidID] = bid

	if err := q.db.CreateBid(q.ctx, db.CreateBidParams{
		ID:                bidID,
		AuctionID:         a.ID,
		StorageProviderID: bid.StorageProviderID,
		WalletAddrSig:     bid.WalletAddrSig,
		BidderID:          peer.Encode(bid.BidderID),
		AskPrice:          bid.AskPrice,
		VerifiedAskPrice:  bid.VerifiedAskPrice,
		StartEpoch:        int64(bid.StartEpoch),
		FastRetrieval:     bid.FastRetrieval,
		ReceivedAt:        bid.ReceivedAt,
	}); err != nil {
		return "", fmt.Errorf("saving bid: %v", err)
	}
	return auction.BidID(id), nil
}

func (q *Queue) start() {
	t := time.NewTimer(StartDelay)
	for {
		select {
		case <-q.ctx.Done():
			t.Stop()
			return
		case <-t.C:
			q.processNext()
		case <-q.tickCh:
			q.processNext()
		}
	}
}

func (q *Queue) processNext() {
	a, err := q.db.GetNextReadyToExecute(q.ctx, stuckSeconds)
	if err == sql.ErrNoRows {
		return
	} else if err != nil {
		log.Errorf("getting next in queue: %v", err)
		return
	}
	auction, err := auctionFromDb(a)
	if err != nil {
		log.Errorf("auction from DB: %v", err)
		return
	}
	q.enqueue(auction)
}

// saveAndTransitionStatus sets a new status, updating the started time if needed.
// Do not directly edit the auction status because it is needed to determine the correct status transition.
// Pass the desired new status with newStatus.
func (q *Queue) saveAndTransitionStatus(ctx context.Context, a *auctioneer.Auction,
	newStatus broker.AuctionStatus) error {
	if a.Status != newStatus {
		a.Status = newStatus
		return q.db.UpdateAuctionStatusAndError(ctx, db.UpdateAuctionStatusAndErrorParams{
			ID:         a.ID,
			Status:     a.Status,
			ErrorCause: a.ErrorCause,
		})
	}
	return nil
}

func auctionFromDb(a db.Auction) (*auctioneer.Auction, error) {
	payloadCid, err := cid.Parse(a.PayloadCid)
	if err != nil {
		return nil, fmt.Errorf("parsing payload cid: %v", err)
	}
	var sources auction.Sources
	if a.CarUrl != "" {
		u, err := url.Parse(a.CarUrl)
		if err != nil {
			return nil, fmt.Errorf("parsing car url: %v", err)
		}
		sources.CARURL = &auction.CARURL{URL: *u}
	}
	if a.CarIpfsCid != "" && a.CarIpfsAddrs != nil {
		ipfsCid, err := cid.Parse(a.CarIpfsCid)
		if err != nil {
			return nil, fmt.Errorf("parsing car ipfs cid: %v", err)
		}
		var maddrs []multiaddr.Multiaddr
		for _, s := range a.CarIpfsAddrs {
			maddr, err := multiaddr.NewMultiaddr(s)
			if err != nil {
				return nil, fmt.Errorf("parsing car ipfs multiaddr: %v", err)
			}
			maddrs = append(maddrs, maddr)
		}
		sources.CARIPFS = &auction.CARIPFS{Cid: ipfsCid, Multiaddrs: maddrs}
	}
	if err := sources.Validate(); err != nil {
		return nil, fmt.Errorf("validating sources: %v", err)
	}
	return &auctioneer.Auction{
		ID:                       a.ID,
		BatchID:                  a.BatchID,
		DealSize:                 uint64(a.DealSize),
		DealDuration:             a.DealDuration,
		DealReplication:          uint32(a.DealReplication),
		DealVerified:             a.DealVerified,
		FilEpochDeadline:         a.FilEpochDeadline,
		ExcludedStorageProviders: a.ExcludedStorageProviders,
		PayloadCid:               payloadCid,
		Sources:                  sources,
		Status:                   a.Status,
		StartedAt:                a.StartedAt,
		UpdatedAt:                a.UpdatedAt,
		Duration:                 time.Duration(a.Duration),
		ErrorCause:               a.ErrorCause,
	}, nil
}

func bidFromDb(bid db.Bid) (*auctioneer.Bid, error) {
	bidderID, err := peer.Decode(bid.BidderID)
	if err != nil {
		return nil, fmt.Errorf("invalid bidder ID: %v", err)
	}
	b := &auctioneer.Bid{
		ID:                bid.ID,
		StorageProviderID: bid.StorageProviderID,
		WalletAddrSig:     bid.WalletAddrSig,
		BidderID:          bidderID,
		AskPrice:          bid.AskPrice,
		VerifiedAskPrice:  bid.VerifiedAskPrice,
		StartEpoch:        uint64(bid.StartEpoch),
		FastRetrieval:     bid.FastRetrieval,
		ReceivedAt:        bid.ReceivedAt,
	}
	if bid.WonAt.Valid {
		b.WonAt = bid.WonAt.Time
	}
	if bid.ProposalCid.Valid {
		proposalCid, err := cid.Parse(bid.ProposalCid.String)
		if err != nil {
			return nil, fmt.Errorf("parsing proposal cid: %v", err)
		}
		b.ProposalCid = proposalCid
	}
	if bid.ProposalCidDeliveryError.Valid {
		b.ProposalCidDeliveryError = bid.ProposalCidDeliveryError.String
	}

	if bid.ProposalCidDeliveredAt.Valid {
		b.ProposalCidDeliveredAt = bid.ProposalCidDeliveredAt.Time
	}

	return b, nil
}
