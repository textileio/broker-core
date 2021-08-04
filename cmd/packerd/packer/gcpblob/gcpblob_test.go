package gcpblob

import (
	"bytes"
	"context"
	"crypto/rand"
	"io"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestStore(t *testing.T) {
	gcpb, err := New("textile-310716")
	require.NoError(t, err)

	buf := make([]byte, 1000)
	_, err = io.ReadFull(rand.Reader, buf)
	require.NoError(t, err)
	url, err := gcpb.Store(context.Background(), "jorge", bytes.NewReader(buf))
	require.NoError(t, err)
	require.Equal(t, "http://storage.googleapis.com/textile-cars/jorge", url)

	err = gcpb.Close()
	require.NoError(t, err)
}
