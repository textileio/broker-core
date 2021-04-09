package httpapi

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/textileio/broker-core/cmd/uploaderd/storage"
)

func TestSuccess(t *testing.T) {
	t.Parallel()

	req, res := makeRequestWithFile(t)

	expectedSR := storage.StorageRequest{ID: "ID1", StatusCode: storage.StorageRequestStatusPendingPrepare}
	usm := &uploaderMock{}
	usm.On("CreateStorageRequest", mock.Anything, mock.Anything, mock.Anything).Return(expectedSR, nil)

	mux := createMux(usm)
	mux.ServeHTTP(res, req)

	var responseSR storage.StorageRequest
	err := json.Unmarshal(res.Body.Bytes(), &responseSR)
	require.NoError(t, err)

	require.Equal(t, expectedSR, responseSR)
	usm.AssertExpectations(t)
}

func TestFail(t *testing.T) {
	t.Parallel()

	t.Run("wrong-method", func(t *testing.T) {
		t.Parallel()

		req := httptest.NewRequest("GET", "/upload", nil)
		res := httptest.NewRecorder()
		mux := createMux(nil)
		mux.ServeHTTP(res, req)

		require.Equal(t, http.StatusBadRequest, res.Code)
	})

	t.Run("uploader-failed", func(t *testing.T) {
		t.Parallel()

		req, res := makeRequestWithFile(t)

		usm := &uploaderMock{}
		usm.On("CreateStorageRequest", mock.Anything, mock.Anything, mock.Anything).Return(storage.StorageRequest{}, fmt.Errorf("oops"))

		mux := createMux(usm)
		mux.ServeHTTP(res, req)
		require.Equal(t, http.StatusInternalServerError, res.Code)
		usm.AssertExpectations(t)
	})
}

func makeRequestWithFile(t *testing.T) (*http.Request, *httptest.ResponseRecorder) {
	pr, pw := io.Pipe()
	writer := multipart.NewWriter(pw)
	go func() {
		defer writer.Close()

		writer.WriteField("region", "asia")
		w, err := writer.CreateFormFile("file", "something.jpg")
		require.NoError(t, err)
		w.Write(make([]byte, 100))

	}()

	req := httptest.NewRequest("POST", "/upload", pr)
	req.Header.Add("Content-Type", writer.FormDataContentType())
	res := httptest.NewRecorder()

	return req, res
}

// Mocks
type uploaderMock struct {
	mock.Mock
}

func (um *uploaderMock) CreateStorageRequest(ctx context.Context, r io.Reader, meta storage.Metadata) (storage.StorageRequest, error) {
	args := um.Called(ctx, r, meta)

	return args.Get(0).(storage.StorageRequest), args.Error(1)
}

func (um *uploaderMock) IsStorageAuthorized(ctx context.Context, identity string) (bool, string, error) {
	args := um.Called(ctx, identity)

	return args.Bool(0), args.String(1), args.Error(2)
}
