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

	"github.com/ipfs/go-cid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/textileio/broker-core/cmd/storaged/storage"
)

func TestSuccess(t *testing.T) {
	t.Parallel()

	req, res := makeRequestWithFile(t)

	c, _ := cid.Decode("bafybeifsc7cb4abye3cmv4s7icreryyteym6wqa4ee5bcgih36lgbmrqkq")
	expectedSR := storage.Request{ID: "ID1", Cid: c, StatusCode: storage.StatusBatching}
	usm := &uploaderMock{}
	usm.On("CreateFromReader", mock.Anything, mock.Anything, mock.Anything).Return(expectedSR, nil)
	usm.On("IsAuthorized", mock.Anything, mock.Anything).Return(true, "", nil)
	usm.On("Get", mock.Anything, mock.Anything).Return(expectedSR, nil)

	mux := createMux(usm)
	mux.ServeHTTP(res, req)
	require.Equal(t, http.StatusOK, res.Code)

	var responseSR storage.Request
	err := json.Unmarshal(res.Body.Bytes(), &responseSR)
	require.NoError(t, err)

	require.Equal(t, expectedSR, responseSR)

	// Call Get(..)
	req = httptest.NewRequest("GET", "/storagerequest/"+responseSR.ID, nil)
	req.Header.Add("Authorization", "Bearer foo")
	res = httptest.NewRecorder()
	mux.ServeHTTP(res, req)
	require.Equal(t, http.StatusOK, res.Code)
	err = json.Unmarshal(res.Body.Bytes(), &responseSR)
	require.NoError(t, err)
	require.Equal(t, expectedSR, responseSR)

	usm.AssertExpectations(t)
}

func TestFail(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name               string
		method             string
		authHeader         []string
		expectedStatusCode int
	}{
		{
			name:               "wrong method",
			method:             "GET",
			authHeader:         []string{"Bearer valid-auth"},
			expectedStatusCode: http.StatusBadRequest,
		},
		{
			name:               "required auth",
			method:             "POST",
			authHeader:         nil,
			expectedStatusCode: http.StatusUnauthorized,
		},
		{
			name:               "invalid auth",
			method:             "POST",
			authHeader:         []string{"i", "am", "playing", "with", "auth", "headers"},
			expectedStatusCode: http.StatusBadRequest,
		},
		{
			name:               "wrong auth",
			method:             "POST",
			authHeader:         []string{"Bearer invalid-auth"},
			expectedStatusCode: http.StatusUnauthorized,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			req := httptest.NewRequest(tc.method, "/upload", nil)
			req.Header["Authorization"] = tc.authHeader
			res := httptest.NewRecorder()

			usm := &uploaderMock{}
			call := usm.On("IsAuthorized", mock.Anything, mock.Anything)
			call.Run(func(args mock.Arguments) {
				auth := args.String(1)
				if auth == "valid-auth" {
					call.Return(true, "", nil)
					return
				}
				call.Return(false, "sorry, you're unauthorized", nil)
			})

			mux := createMux(usm)
			mux.ServeHTTP(res, req)

			require.Equal(t, tc.expectedStatusCode, res.Code)
		})
	}

	t.Run("uploader-failed", func(t *testing.T) {
		t.Parallel()

		req, res := makeRequestWithFile(t)

		usm := &uploaderMock{}
		usm.On("CreateFromReader", mock.Anything, mock.Anything, mock.Anything).
			Return(storage.Request{}, fmt.Errorf("oops"))
		usm.On("IsAuthorized", mock.Anything, mock.Anything).Return(true, "", nil)

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
		defer func() { _ = writer.Close() }()

		err := writer.WriteField("region", "asia")
		require.NoError(t, err)
		w, err := writer.CreateFormFile("file", "something.jpg")
		require.NoError(t, err)
		_, err = w.Write(make([]byte, 100))
		require.NoError(t, err)
	}()

	req := httptest.NewRequest("POST", "/upload", pr)
	req.Header.Add("Content-Type", writer.FormDataContentType())
	req.Header.Add("Authorization", "Bearer foo")
	res := httptest.NewRecorder()

	return req, res
}

type uploaderMock struct {
	mock.Mock
}

func (um *uploaderMock) CreateFromReader(
	ctx context.Context,
	r io.Reader,
	meta storage.Metadata,
) (storage.Request, error) {
	args := um.Called(ctx, r, meta)

	return args.Get(0).(storage.Request), args.Error(1)
}

func (um *uploaderMock) IsAuthorized(ctx context.Context, identity string) (bool, string, error) {
	args := um.Called(ctx, identity)

	return args.Bool(0), args.String(1), args.Error(2)
}

func (um *uploaderMock) Get(ctx context.Context, id string) (storage.Request, error) {
	args := um.Called(ctx, id)

	return args.Get(0).(storage.Request), args.Error(1)
}
