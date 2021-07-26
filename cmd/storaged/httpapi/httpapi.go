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
		Handler:           createMux(s, skipAuth, maxUploadSize),
	}

	log.Info("running HTTP API...")
	go func() {
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Errorf("stopping http server: %s", err)
		}
	}()

	return httpServer, nil
}

func createMux(s storage.Requester, skipAuth bool, maxUploadSize uint) *http.ServeMux {
	mux := http.NewServeMux()

	mux.HandleFunc("/health", healthHandler)

	uploadHandler := wrapMiddlewares(s, skipAuth, uploadHandler(s, maxUploadSize), "upload")
	mux.Handle("/upload", uploadHandler)

	storageRequestStatusHandler := wrapMiddlewares(s, true, storageRequestHandler(s), "storagerequest")
	mux.Handle("/storagerequest/", storageRequestStatusHandler)

	auctionDataHandler := wrapMiddlewares(s, skipAuth, auctionDataHandler(s), "auction-data")
	mux.Handle("/auction-data", auctionDataHandler)

	mux.HandleFunc("/car/", carDownloadHandler(s))

	return mux
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		httpError(w, "only GET method is allowed", http.StatusBadRequest)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func wrapMiddlewares(s storage.Requester, skipAuth bool, h http.HandlerFunc, name string) http.Handler {
	handler := instrumentHandler(h, name)
	if !skipAuth {
		handler = authenticateHandler(handler, s)
	}
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

func authenticateHandler(h http.Handler, s storage.Requester) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader, ok := r.Header["Authorization"]
		if !ok || len(authHeader) == 0 {
			httpError(w, "Authorization header is required", http.StatusUnauthorized)
			return
		}
		if len(authHeader) > 1 {
			httpError(w, "there should only be one authorization value", http.StatusBadRequest)
			return
		}
		authParts := strings.SplitN(authHeader[0], " ", 2)
		if len(authParts) != 2 {
			httpError(w, "invalid authorization value", http.StatusBadRequest)
			return
		}

		ok, authErr, err := s.IsAuthorized(r.Context(), authParts[1])
		if err != nil {
			httpError(w, fmt.Sprintf("calling authorizer: %s", err), http.StatusInternalServerError)
			return
		}
		if !ok {
			httpError(w, fmt.Sprintf("unauthorized: %s", authErr), http.StatusUnauthorized)
			return
		}

		h.ServeHTTP(w, r)
	})
}

func uploadHandler(s storage.Requester, maxUploadSize uint) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			httpError(w, "only POST method is allowed", http.StatusBadRequest)
			return
		}

		r.Body = http.MaxBytesReader(w, r.Body, int64(maxUploadSize))

		file, err := parseMultipart(r)
		if err != nil {
			httpError(w, fmt.Sprintf("parsing multipart: %s", err), http.StatusBadRequest)
			return
		}

		storageRequest, err := s.CreateFromReader(r.Context(), file)
		if err != nil {
			httpError(w, fmt.Sprintf("upload data and create broker request: %s", err), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(storageRequest); err != nil {
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

func auctionDataHandler(s storage.Requester) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			httpError(w, "only POST method is allowed", http.StatusBadRequest)
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

		sr, err := s.CreateFromExternalSource(r.Context(), ad)
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

		if found, err := s.GetCAR(r.Context(), cid, w); err != nil {
			httpError(w, fmt.Sprintf("writing car stream: %s", err), http.StatusInternalServerError)
			return
		} else if !found {
			httpError(w, fmt.Sprintf("cid %s not found", cid), http.StatusNotFound)
			return
		}
	}
}
