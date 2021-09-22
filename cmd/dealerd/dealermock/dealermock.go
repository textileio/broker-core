package dealermock

import (
	"bytes"
	"context"
	"fmt"
	"math/rand"
	"time"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/go-state-types/big"
	"github.com/filecoin-project/go-state-types/crypto"
	"github.com/filecoin-project/specs-actors/actors/builtin/market"
	"github.com/jsign/go-filsigner/wallet"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/multiformats/go-multiaddr"
	"github.com/textileio/broker-core/broker"
	dealeri "github.com/textileio/broker-core/dealer"
	mbroker "github.com/textileio/broker-core/msgbroker"
	"github.com/textileio/go-auctions-client/propsigner"
	logger "github.com/textileio/go-log/v2"
)

var log = logger.Logger("dealermock")

// Dealer provides a mocked implementation of Dealer. It reports successful
// deals to the broker after 1 sec.
type Dealer struct {
	mb         mbroker.MsgBroker
	host       host.Host
	relayMaddr string
}

// New returns a new Dealer.
func New(mb mbroker.MsgBroker, h host.Host, relayMaddr string) *Dealer {
	return &Dealer{
		mb:         mb,
		host:       h,
		relayMaddr: relayMaddr,
	}
}

// ReadyToCreateDeals registers deals to execute.
func (d *Dealer) ReadyToCreateDeals(ctx context.Context, sdb dealeri.AuctionDeals) error {
	log.Debugf("received ready to create deals %s", sdb.BatchID)
	go d.reportToBroker(sdb)
	return nil
}

func (d *Dealer) reportToBroker(auds dealeri.AuctionDeals) {
	time.Sleep(time.Second)

	log.Debugf("faking deal execution...")
	if auds.RemoteWallet != nil {
		if err := d.callRemoteWallet(auds); err != nil {
			log.Errorf("asking for fake proposal remote signature: %s", err)
		}
	}

	for _, p := range auds.Proposals {
		fd := broker.FinalizedDeal{
			BatchID:           auds.BatchID,
			DealID:            rand.Int63(),
			DealExpiration:    uint64(rand.Int63()),
			StorageProviderID: p.StorageProviderID,
			AuctionID:         p.AuctionID,
			BidID:             p.BidID,
		}
		if err := mbroker.PublishMsgFinalizedDeal(context.Background(), d.mb, fd); err != nil {
			log.Errorf("publishing finalized-deal msg to msgbroker: %s", err)
		}
	}
}

func (d *Dealer) callRemoteWallet(auds dealeri.AuctionDeals) error {
	rw := auds.RemoteWallet
	ctx, cls := context.WithTimeout(context.Background(), time.Second*30)
	defer cls()

	if d.relayMaddr != "" {
		relayed, err := multiaddr.NewMultiaddr(d.relayMaddr + "/p2p-circuit/p2p/" + rw.PeerID.String())
		if err != nil {
			return fmt.Errorf("creating relayed maddr: %s", err)
		}
		rw.Multiaddrs = append(rw.Multiaddrs, relayed)
	}
	pi := peer.AddrInfo{
		ID:    rw.PeerID,
		Addrs: rw.Multiaddrs,
	}
	if err := d.host.Connect(ctx, pi); err != nil {
		return fmt.Errorf("connecting with remote wallet: %s", err)
	}

	minerAddr, err := address.NewFromString(auds.Proposals[0].StorageProviderID)
	if err != nil {
		return fmt.Errorf("parsing provider id: %s", err)
	}
	proposal := market.DealProposal{
		PieceCID:     auds.PieceCid,
		PieceSize:    abi.PaddedPieceSize(auds.PieceSize),
		VerifiedDeal: auds.Proposals[0].Verified,
		Client:       rw.WalletAddr,
		Provider:     minerAddr,

		Label: "this is a fake label",

		StartEpoch:           100,
		EndEpoch:             200,
		StoragePricePerEpoch: big.NewInt(3000),
	}

	log.Debugf("requesting remote signature to %s", rw.PeerID)
	sig, err := propsigner.RequestSignatureV1(ctx, d.host, rw.AuthToken, proposal, rw.PeerID)
	if err != nil {
		return fmt.Errorf("remote signing proposal: %s", err)
	}
	log.Debugf("remote signature to %s received successfully", rw.PeerID)
	if err := validateSignature(proposal, sig); err != nil {
		return fmt.Errorf("remote signature is invalid: %s", err)
	}
	log.Debugf("remote signature from %s is valid", rw.PeerID)

	return nil
}

func validateSignature(proposal market.DealProposal, sig *crypto.Signature) error {
	msg := &bytes.Buffer{}
	err := proposal.MarshalCBOR(msg)
	if err != nil {
		return fmt.Errorf("marshaling proposal: %s", err)
	}
	sigBytes, err := sig.MarshalBinary()
	if err != nil {
		return fmt.Errorf("marshaling signature: %s", err)
	}
	ok, err := wallet.WalletVerify(proposal.Client, msg.Bytes(), sigBytes)
	if err != nil {
		return fmt.Errorf("verifying signature: %s", err)
	}
	if !ok {
		return fmt.Errorf("signature is invalid: %s", err)
	}
	return nil
}
