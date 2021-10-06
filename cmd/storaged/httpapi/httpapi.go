package httpapi

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/ipfs/go-cid"
	"github.com/textileio/bidbot/lib/auction"
	"github.com/textileio/broker-core/auth"
	"github.com/textileio/broker-core/cmd/storaged/storage"
	logging "github.com/textileio/go-log/v2"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

var (
	log = logging.Logger("http/api")
)

// NewServer returns a new http server exposing the storage API.
func NewServer(listenAddr string, skipAuth bool, s storage.Requester, maxUploadSize uint) (*http.Server, error) {
	httpServer := &http.Server{
		Addr:              listenAddr,
		ReadHeaderTimeout: time.Second * 5,
		Handler:           createMux(s, maxUploadSize, skipAuth),
	}

	log.Info("running HTTP API...")
	go func() {
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Errorf("stopping http server: %s", err)
		}
	}()

	return httpServer, nil
}

func createMux(s storage.Requester, maxUploadSize uint, skipAuth bool) *http.ServeMux {
	var reqAuth requestAuthorizer = getAuth
	if skipAuth {
		reqAuth = getAuthMock
	}
	mux := http.NewServeMux()

	uploadHandler := wrapMiddlewares(uploadHandler(reqAuth, s, maxUploadSize), "upload")
	mux.Handle("/upload", uploadHandler)

	storageRequestStatusHandler := wrapMiddlewares(storageRequestHandler(s), "storagerequest")
	mux.Handle("/storagerequest/", storageRequestStatusHandler)

	auctionDataHandler := wrapMiddlewares(auctionDataHandler(reqAuth, s), "auction-data")
	mux.Handle("/auction-data", auctionDataHandler)

	mux.HandleFunc("/car/", carDownloadHandler(s))

	mux.HandleFunc("/healthz", healthHandler)
	mux.HandleFunc("/health", healthHandler)
	mux.HandleFunc("/", healthHandler)

	return mux
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		httpError(w, "only GET method is allowed", http.StatusBadRequest)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func wrapMiddlewares(h http.HandlerFunc, name string) http.Handler {
	handler := instrumentHandler(h, name)
	handler = corsHandler(handler)

	return handler
}

func instrumentHandler(h http.Handler, name string) http.Handler {
	return otelhttp.NewHandler(h, name)
}

func corsHandler(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Accept, Accept-Language, Content-Type, Authorization")
		if r.Method == "OPTIONS" {
			return
		}

		h.ServeHTTP(w, r)
	})
}

func uploadHandler(
	reqAuth requestAuthorizer,
	s storage.Requester,
	maxUploadSize uint) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			httpError(w, "only POST method is allowed", http.StatusBadRequest)
			return
		}

		ae, status, err := reqAuth(r, s)
		if err != nil {
			httpError(w, err.Error(), status)
			return
		}

		r.Body = http.MaxBytesReader(w, r.Body, int64(maxUploadSize))

		file, err := parseMultipart(r)
		if err != nil {
			httpError(w, fmt.Sprintf("parsing multipart: %s", err), http.StatusBadRequest)
			return
		}

		sr, err := s.CreateFromReader(r.Context(), file, ae.Origin)
		if err != nil {
			httpError(w, fmt.Sprintf("upload data and create storage request: %s", err), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(sr); err != nil {
			httpError(w, fmt.Sprintf("marshaling response: %s", err), http.StatusInternalServerError)
			return
		}
	}
}

func storageRequestHandler(s storage.Requester) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			httpError(w, "only GET method is allowed", http.StatusBadRequest)
			return
		}
		urlParts := strings.SplitN(r.URL.Path, "/", 3)
		if len(urlParts) < 3 {
			httpError(w, "the url should be /storagerequest/{id}", http.StatusBadRequest)
			return
		}
		id := urlParts[2]

		sr, err := s.GetRequestInfo(r.Context(), id)
		if err != nil {
			httpError(w, fmt.Sprintf("get request info: %s", err), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(sr); err != nil {
			httpError(w, fmt.Sprintf("marshaling response: %s", err), http.StatusInternalServerError)
			return
		}
	}
}

func parseMultipart(r *http.Request) (io.Reader, error) {
	mr, err := r.MultipartReader()
	if err != nil {
		return nil, fmt.Errorf("opening multipart reader: %s", err)
	}

	var file io.Reader
Loop:
	for {
		part, err := mr.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("getting next part: %s", err)
		}
		switch part.FormName() {
		case "file":
			file = part
			break Loop
		default:
			return nil, errors.New("malformed request")
		}
	}
	if file == nil {
		return nil, fmt.Errorf("missing file part: %s", err)
	}

	return file, nil
}

func auctionDataHandler(reqAuth requestAuthorizer, s storage.Requester) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			httpError(w, "only POST method is allowed", http.StatusBadRequest)
			return
		}

		ae, status, err := reqAuth(r, s)
		if err != nil {
			httpError(w, err.Error(), status)
			return
		}

		body := http.MaxBytesReader(w, r.Body, 1<<20)

		jsonDecoder := json.NewDecoder(body)
		jsonDecoder.DisallowUnknownFields()
		var ad storage.AuctionDataRequest
		if err := jsonDecoder.Decode(&ad); err != nil {
			httpError(w, fmt.Sprintf("decoding auction-data request body: %s", err), http.StatusBadRequest)
			return
		}

		sr, err := s.CreateFromExternalSource(r.Context(), ad, ae.Origin)
		if err != nil {
			httpError(w, fmt.Sprintf("creating storage-request from external data: %s", err), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(sr); err != nil {
			httpError(w, fmt.Sprintf("marshaling response: %s", err), http.StatusInternalServerError)
			return
		}
	}
}

type requestAuthorizer func(*http.Request, storage.Requester) (auth.AuthorizedEntity, int, error)

func getAuth(r *http.Request, s storage.Requester) (auth.AuthorizedEntity, int, error) {
	authHeader, ok := r.Header["Authorization"]
	if !ok || len(authHeader) == 0 {
		return auth.AuthorizedEntity{}, http.StatusBadRequest, errors.New("authorization header is required")
	}
	if len(authHeader) > 1 {
		return auth.AuthorizedEntity{}, http.StatusBadRequest, errors.New("only one authorization header is allowed")
	}
	authParts := strings.SplitN(authHeader[0], " ", 2)
	if len(authParts) != 2 {
		return auth.AuthorizedEntity{}, http.StatusBadRequest, errors.New("invalid authorization format")
	}

	ae, ok, authErr, err := s.IsAuthorized(r.Context(), authParts[1])
	if err != nil {
		log.Errorf("authorizer: %s", err)
		return auth.AuthorizedEntity{}, http.StatusInternalServerError, fmt.Errorf("authorization failed")
	}
	if !ok {
		return auth.AuthorizedEntity{}, http.StatusUnauthorized, fmt.Errorf("authorization failed: %s", authErr)
	}

	return ae, http.StatusOK, nil
}

func getAuthMock(r *http.Request, s storage.Requester) (auth.AuthorizedEntity, int, error) {
	return auth.AuthorizedEntity{
		Identity: "identity-fake",
		Origin:   "origin-fake",
	}, http.StatusOK, nil
}

func httpError(w http.ResponseWriter, err string, status int) {
	log.Errorf("request error: %s", err)
	http.Error(w, err, status)
}

func carDownloadHandler(s storage.Requester) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			httpError(w, "only GET method is allowed", http.StatusBadRequest)
			return
		}

		urlParts := strings.SplitN(r.URL.Path, "/", 3)
		if len(urlParts) < 3 {
			httpError(w, "the url should be /car/{cid}", http.StatusBadRequest)
			return
		}
		cidStr := urlParts[2]

		w.Header().Set("Content-Disposition", "attachment; filename="+strconv.Quote(cidStr)+".car")
		w.Header().Set("Content-Type", "application/octet-stream")

		cid, err := cid.Decode(cidStr)
		if err != nil {
			httpError(w, fmt.Sprintf("decoding cid: %s", err), http.StatusInternalServerError)
			return
		}

		responder := s.GetCAR
		if r.Header.Get(auction.HTTPCarHeaderOnly) != "" {
			responder = s.GetCARHeader
		}
		if found, err := responder(r.Context(), cid, w); err != nil {
			httpError(w, fmt.Sprintf("writing car stream: %s", err), http.StatusInternalServerError)
			return
		} else if !found {
			httpError(w, fmt.Sprintf("cid %s not found", cid), http.StatusNotFound)
			return
		}
	}
}
