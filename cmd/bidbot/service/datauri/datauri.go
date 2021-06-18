package datauri

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"

	"github.com/ipfs/go-cid"
	"github.com/ipld/go-car"
	golog "github.com/textileio/go-log/v2"
)

var (
	log = golog.Logger("bidbot/getter")

	// ErrSchemeNotSupported indicates a given URI scheme is not supported.
	ErrSchemeNotSupported = errors.New("scheme not supported")

	// ErrInvalidCarFile indicates a given URI points to an invalid car file.
	ErrInvalidCarFile = errors.New("invalid car file")
)

// URI describes a data car file for a storage deal.
type URI interface {
	fmt.Stringer
	Cid() cid.Cid
	Write(context.Context, io.Writer) error
	Validate(ctx context.Context) error
}

// NewURI returns a new URI for the given string uri.
// ErrSchemeNotSupported is returned if the scheme is not supported.
func NewURI(uri string) (URI, error) {
	parsed, err := url.Parse(uri)
	if err != nil {
		return nil, fmt.Errorf("parsing uri '%s': %v", uri, err)
	}
	switch parsed.Scheme {
	case "http", "https":
		id, err := cid.Decode(path.Base(parsed.Path))
		if err != nil {
			return nil, fmt.Errorf("parsing uri cid '%s': %v", uri, err)
		}
		return &HTTPURI{uri: uri, cid: id}, nil
	default:
		return nil, fmt.Errorf("parsing uri '%s': %w", uri, ErrSchemeNotSupported)
	}
}

// HTTPURI is used to get http/https resources.
type HTTPURI struct {
	uri string
	cid cid.Cid
}

// Cid returns the data cid referenced by the uri.
func (u *HTTPURI) Cid() cid.Cid {
	return u.cid
}

// Validate checks the integrity of the car file.
// The cid associated with the uri must be the one and only root of the car file.
func (u *HTTPURI) Validate(ctx context.Context) error {
	res, err := u.getRequest(ctx)
	if err != nil {
		return fmt.Errorf("get request: %v", err)
	}
	defer func() {
		if err := res.Body.Close(); err != nil {
			log.Errorf("closing http get request: %v", err)
		}
	}()

	// Ensure cid is the one and only root of the car file
	ch, _, err := car.ReadHeader(bufio.NewReader(res.Body))
	if err != nil {
		return fmt.Errorf("reading car header: %v", err)
	}
	if len(ch.Roots) != 1 {
		return fmt.Errorf("car file must have only one root: %w", ErrInvalidCarFile)
	}
	if !ch.Roots[0].Equals(u.cid) {
		return fmt.Errorf("car file root does not match uri: %w", ErrInvalidCarFile)
	}
	return nil
}

// Write the uri's car file to writer.
func (u *HTTPURI) Write(ctx context.Context, writer io.Writer) error {
	res, err := u.getRequest(ctx)
	if err != nil {
		return fmt.Errorf("get request: %v", err)
	}
	defer func() {
		if err := res.Body.Close(); err != nil {
			log.Errorf("closing http get request: %v", err)
		}
	}()

	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("http request returned bad status: %d", res.StatusCode)
	}

	if _, err := io.Copy(writer, res.Body); err != nil {
		return fmt.Errorf("writing http get response: %v", err)
	}
	return nil
}

// String returns the uri as a string.
func (u *HTTPURI) String() string {
	return u.uri
}

func (u *HTTPURI) getRequest(ctx context.Context) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", u.uri, nil)
	if err != nil {
		return nil, fmt.Errorf("building http request: %v", err)
	}
	client := &http.Client{}
	res, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("sending http request: %v", err)
	}
	return res, nil
}
