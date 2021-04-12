package uploader

import (
	"context"
	"io"

	"github.com/ipfs/go-cid"
)

// Uploader stores data in a persistence-layer.
type Uploader interface {
	Store(ctx context.Context, r io.Reader) (cid.Cid, error)
}
