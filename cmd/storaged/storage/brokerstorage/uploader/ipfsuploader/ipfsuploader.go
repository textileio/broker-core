package ipfsuploader

import (
	"context"
	"fmt"
	"io"

	"github.com/ipfs/go-cid"
	ipfsfiles "github.com/ipfs/go-ipfs-files"
	httpapi "github.com/ipfs/go-ipfs-http-client"
	logger "github.com/ipfs/go-log/v2"
	"github.com/ipfs/interface-go-ipfs-core/options"
	"github.com/multiformats/go-multiaddr"
)

var (
	log = logger.Logger("ipfs-uploader")
)

type IpfsUploader struct {
	client *httpapi.HttpApi
}

func New(ipfsAPIMultiaddr string) (*IpfsUploader, error) {
	ma, err := multiaddr.NewMultiaddr(ipfsAPIMultiaddr)
	if err != nil {
		return nil, fmt.Errorf("parsing ipfs client multiaddress: %s", err)
	}
	client, err := httpapi.NewApi(ma)
	if err != nil {
		return nil, fmt.Errorf("creating ipfs client: %s", err)
	}

	return &IpfsUploader{
		client: client,
	}, nil
}

func (iu *IpfsUploader) Store(ctx context.Context, r io.Reader) (cid.Cid, error) {
	p, err := iu.client.Unixfs().Add(ctx, ipfsfiles.NewReaderFile(r), options.Unixfs.Pin(true))
	if err != nil {
		return cid.Undef, fmt.Errorf("adding data to ipfs: %s", err)
	}

	return p.Cid(), nil

}
