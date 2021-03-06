package gcpblob

import (
	"context"
	"errors"
	"fmt"
	"io"
	"time"

	"cloud.google.com/go/storage"
	logger "github.com/textileio/go-log/v2"
	"google.golang.org/api/iterator"
)

var (
	log = logger.Logger("gcpblob")
)

// GCPBlob provides blob storage services for car files.
type GCPBlob struct {
	client *storage.Client
	bucket *storage.BucketHandle
}

// New returns a new GCP blob storage.
func New(projectID string) (*GCPBlob, error) {
	if projectID == "" {
		return nil, errors.New("project id is empty")
	}
	ctx, cls := context.WithTimeout(context.Background(), time.Second*15)
	defer cls()
	client, err := storage.NewClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("creating client: %s", err)
	}

	gcpb := GCPBlob{
		client: client,
		bucket: client.Bucket("textile-cars"),
	}
	return &gcpb, nil
}

// Store stores a file and returns a public URL.
func (b *GCPBlob) Store(ctx context.Context, name string, r io.Reader) (string, error) {
	_, err := b.bucket.Objects(ctx, &storage.Query{Prefix: name}).Next()
	if err != iterator.Done {
		log.Debugf("deleting existing %s", name)
		err := b.bucket.Object(name).Delete(ctx)
		if err != nil {
			log.Warnf("deleting file %s", err)
		}
	}

	log.Debugf("creating %s in the bucket", name)
	w := b.bucket.Object(name).NewWriter(ctx)
	if _, err := io.Copy(w, r); err != nil {
		return "", fmt.Errorf("uploading data: %s", err)
	}
	if err := w.Close(); err != nil {
		return "", fmt.Errorf("closing uploader: %s", err)
	}

	return "https://storage.googleapis.com/textile-cars/" + name, nil
}

// Close closes the service.
func (b *GCPBlob) Close() error {
	if err := b.client.Close(); err != nil {
		return fmt.Errorf("closing client: %s", err)
	}
	return nil
}
