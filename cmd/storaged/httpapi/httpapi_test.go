package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ipfs/go-cid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/textileio/broker-core/auth"
	"github.com/textileio/broker-core/cmd/storaged/storage"
)

func TestCreateShortCircuit(t *testing.T) {
	t.Parallel()
	req, res := makeRequestWithFile(t)
	mux := createMux(&uploaderMock{}, 1<<20, true)
	mux.ServeHTTP(res, req)
	require.Equal(t, http.StatusNotImplemented, res.Code)
}

func TestCreatePreparedSuccess(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		pr   preparedRequest
	}{
		{name: "without remote-wallet", pr: makeRequestPrepared(t)},
		{name: "with remote-wallet", pr: makeRequestRemoteWallet(t)},
		{name: "with direct providers", pr: makeRequestPreparedWithDirectProviders(t)},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			c, _ := cid.Decode("bafybeifsc7cb4abye3cmv4s7icreryyteym6wqa4ee5bcgih36lgbmrqkq")
			expectedSR := storage.Request{ID: "ID1", Cid: c, StatusCode: storage.StatusBatching}
			expectedSRI := storage.RequestInfo{Request: expectedSR}
			usm := &uploaderMock{}
			usm.On("CreateFromExternalSource",
				mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(expectedSR, nil)
			usm.On("GetRequestInfo", mock.Anything, mock.Anything).Return(expectedSRI, nil)

			mux := createMux(usm, 1<<20, true)
			mux.ServeHTTP(test.pr.res, test.pr.req)
			require.Equal(t, http.StatusOK, test.pr.res.Code)

			var responseSR storage.Request
			err := json.Unmarshal(test.pr.res.Body.Bytes(), &responseSR)
			require.NoError(t, err)

			require.Equal(t, expectedSR, responseSR)

			// Call Get(..)
			req := httptest.NewRequest("GET", "/storagerequest/"+responseSR.ID, nil)
			req.Header.Add("Authorization", "Bearer foo")
			res := httptest.NewRecorder()
			mux.ServeHTTP(res, req)
			require.Equal(t, http.StatusOK, res.Code)
			var responseSRI storage.RequestInfo
			err = json.Unmarshal(res.Body.Bytes(), &responseSRI)
			require.NoError(t, err)
			require.Equal(t, expectedSRI, responseSRI)

			usm.AssertExpectations(t)
		})
	}
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
			expectedStatusCode: http.StatusBadRequest,
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
			req := httptest.NewRequest(tc.method, "/auction-data", nil)
			req.Header["Authorization"] = tc.authHeader
			res := httptest.NewRecorder()

			usm := &uploaderMock{}
			call := usm.On("IsAuthorized", mock.Anything, mock.Anything, mock.Anything)
			call.Run(func(args mock.Arguments) {
				a := args.String(1)
				if a == "valid-auth" {
					call.Return(auth.AuthorizedEntity{}, true, "", nil)
					return
				}
				call.Return(auth.AuthorizedEntity{}, false, "sorry, you're unauthorized", nil)
			})

			mux := createMux(usm, 1<<20, false)
			mux.ServeHTTP(res, req)

			require.Equal(t, tc.expectedStatusCode, res.Code)
		})
	}

}

func makeRequestWithFile(t *testing.T) (*http.Request, *httptest.ResponseRecorder) {
	pr, pw := io.Pipe()
	writer := multipart.NewWriter(pw)
	go func() {
		defer func() { _ = writer.Close() }()

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

type preparedRequest struct {
	req *http.Request
	res *httptest.ResponseRecorder
}

func makeRequestPrepared(t *testing.T) preparedRequest {
	adr := storage.AuctionDataRequest{
		PayloadCid: "bafybeifsc7cb4abye3cmv4s7icreryyteym6wqa4ee5bcgih36lgbmrqkq",
		PieceCid:   "baga6ea4seaqecjuu654vrpfi5ekfiequcfwgeuhiqflxo2e7nq6bfpb4ilxoqci",
		PieceSize:  64,
		RepFactor:  3,
		Deadline:   "2021-06-17",
		CARURL: &storage.CARURL{
			URL: "https://hello.com/world.car",
		},
		CARIPFS: &storage.CARIPFS{
			Cid:        "QmcCpRyHhCPNaKLVC3eMgS14L5wNfXBM6NyJafD22af5AE",
			Multiaddrs: []string{"/ip4/127.0.0.1/tcp/9999/p2p/12D3KooWA8o5KiBQew75GKWhZcpJdBKrfWjp2Zhyp7X5thxw41TE"},
		},
	}
	body, err := json.Marshal(adr)
	require.NoError(t, err)

	req := httptest.NewRequest("POST", "/auction-data", bytes.NewReader(body))
	req.Header.Add("Authorization", "Bearer foo")
	res := httptest.NewRecorder()

	return preparedRequest{req: req, res: res}
}

func makeRequestPreparedWithDirectProviders(t *testing.T) preparedRequest {
	adr := storage.AuctionDataRequest{
		PayloadCid: "bafybeifsc7cb4abye3cmv4s7icreryyteym6wqa4ee5bcgih36lgbmrqkq",
		PieceCid:   "baga6ea4seaqecjuu654vrpfi5ekfiequcfwgeuhiqflxo2e7nq6bfpb4ilxoqci",
		PieceSize:  64,
		RepFactor:  3,
		Deadline:   "2021-06-17",
		CARURL: &storage.CARURL{
			URL: "https://hello.com/world.car",
		},
		CARIPFS: &storage.CARIPFS{
			Cid:        "QmcCpRyHhCPNaKLVC3eMgS14L5wNfXBM6NyJafD22af5AE",
			Multiaddrs: []string{"/ip4/127.0.0.1/tcp/9999/p2p/12D3KooWA8o5KiBQew75GKWhZcpJdBKrfWjp2Zhyp7X5thxw41TE"},
		},
		Providers: []string{"f0001", "f0002"},
	}
	body, err := json.Marshal(adr)
	require.NoError(t, err)

	req := httptest.NewRequest("POST", "/auction-data", bytes.NewReader(body))
	req.Header.Add("Authorization", "Bearer foo")
	res := httptest.NewRecorder()

	return preparedRequest{req: req, res: res}
}

func makeRequestRemoteWallet(t *testing.T) preparedRequest {
	adr := storage.AuctionDataRequest{
		PayloadCid: "bafybeifsc7cb4abye3cmv4s7icreryyteym6wqa4ee5bcgih36lgbmrqkq",
		PieceCid:   "baga6ea4seaqecjuu654vrpfi5ekfiequcfwgeuhiqflxo2e7nq6bfpb4ilxoqci",
		PieceSize:  64,
		RepFactor:  3,
		Deadline:   "2021-06-17",
		CARURL: &storage.CARURL{
			URL: "https://hello.com/world.car",
		},
		CARIPFS: &storage.CARIPFS{
			Cid:        "QmcCpRyHhCPNaKLVC3eMgS14L5wNfXBM6NyJafD22af5AE",
			Multiaddrs: []string{"/ip4/127.0.0.1/tcp/9999/p2p/12D3KooWA8o5KiBQew75GKWhZcpJdBKrfWjp2Zhyp7X5thxw41TE"},
		},
		RemoteWallet: &storage.RemoteWallet{
			PeerID:     "Qmf12XE1bSB8SbngTc4Yy8KLNGKT2Nhp2xuj2DFEWX3N5H",
			WalletAddr: "f1fib3pv7jua2ockdugtz7viz3cyy6lkhh7rfx3sa",
			AuthToken:  "abbccddee",
			Multiaddrs: []string{"/ip4/127.0.0.1/tcp/1234/p2p/12D3KooWA8o5KiBQew75GKWhZcpJdBKrfWjp2Zhyp7X5thxw41TZ"},
		},
	}
	body, err := json.Marshal(adr)
	require.NoError(t, err)

	req := httptest.NewRequest("POST", "/auction-data", bytes.NewReader(body))
	req.Header.Add("Authorization", "Bearer foo")
	res := httptest.NewRecorder()

	return preparedRequest{req: req, res: res}
}

type uploaderMock struct {
	mock.Mock
}

func (um *uploaderMock) CreateFromReader(
	ctx context.Context,
	r io.Reader,
	origin string,
) (storage.Request, error) {
	args := um.Called(ctx, r)

	return args.Get(0).(storage.Request), args.Error(1)
}

func (um *uploaderMock) IsAuthorized(
	ctx context.Context,
	identity string) (auth.AuthorizedEntity, bool, string, error) {
	args := um.Called(ctx, identity)

	return args.Get(0).(auth.AuthorizedEntity), args.Bool(1), args.String(2), args.Error(3)
}

func (um *uploaderMock) GetRequestInfo(ctx context.Context, id string) (storage.RequestInfo, error) {
	args := um.Called(ctx, id)

	return args.Get(0).(storage.RequestInfo), args.Error(1)
}

func (um *uploaderMock) GetCAR(ctx context.Context, c cid.Cid, w io.Writer) (bool, error) {
	args := um.Called(ctx, c, w)

	return args.Bool(0), args.Error(1)
}

func (um *uploaderMock) GetCARHeader(ctx context.Context, c cid.Cid, w io.Writer) (bool, error) {
	args := um.Called(ctx, c, w)

	return args.Bool(0), args.Error(1)
}

func (um *uploaderMock) CreateFromExternalSource(
	ctx context.Context,
	adr storage.AuctionDataRequest,
	origin string,
) (storage.Request, error) {
	args := um.Called(ctx, adr)

	return args.Get(0).(storage.Request), args.Error(1)
}
