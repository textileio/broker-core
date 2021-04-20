package httpapi

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	logging "github.com/ipfs/go-log/v2"
	"github.com/textileio/broker-core/cmd/storaged/storage"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

const (
	gib         = 1024 * 1024 * 1024
	maxBodySize = 32 * gib
)

var (
	log = logging.Logger("http/api")
)

// NewServer returns a new http server exposing the storage API.
func NewServer(listenAddr string, s storage.Requester) (*http.Server, error) {
	httpServer := &http.Server{
		Addr:              listenAddr,
		ReadHeaderTimeout: time.Second * 5,
		WriteTimeout:      time.Second * 10,
		Handler:           createMux(s),
	}

	log.Infof("Running HTTP API...")
	go func() {
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Errorf("stopping http server: %s", err)
		}
	}()

	return httpServer, nil
}

func createMux(s storage.Requester) *http.ServeMux {
	mux := http.NewServeMux()

	uh := wrapMiddlewares(s, uploadHandler(s), "upload")
	mux.Handle("/upload", uh)

	return mux
}

func wrapMiddlewares(s storage.Requester, h http.HandlerFunc, name string) http.Handler {
	handler := instrumentHandler(h, name)
	handler = authenticateHandler(handler, s)

	return handler
}

func instrumentHandler(h http.HandlerFunc, name string) http.Handler {
	return otelhttp.NewHandler(h, name)
}

func authenticateHandler(h http.Handler, s storage.Requester) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if origin := r.Header.Get("Origin"); origin != "" {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Accept, Accept-Language, Content-Type, Authorization")
		}
		// Preflight OPTIONS request
		if r.Method == "OPTIONS" {
			return
		}
		authHeader, ok := r.Header["Authorization"]
		if !ok || len(authHeader) == 0 {
			httpError(w, "Authorization header is required", http.StatusUnauthorized)
			return
		}
		if len(authHeader) > 1 {
			httpError(w, "there should only be one authorization value", http.StatusBadRequest)
			return
		}

		ok, authErr, err := s.IsAuthorized(r.Context(), authHeader[0])
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

func uploadHandler(s storage.Requester) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			httpError(w, "only POST method is allowed", http.StatusBadRequest)
			return
		}

		r.Body = http.MaxBytesReader(w, r.Body, maxBodySize)

		region, file, err := parseMultipart(r)
		if err != nil {
			httpError(w, fmt.Sprintf("parsing multipart: %s", err), http.StatusBadRequest)
			return
		}

		meta := storage.Metadata{
			Region: region,
		}
		storageRequest, err := s.CreateFromReader(r.Context(), file, meta)
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

func parseMultipart(r *http.Request) (string, io.Reader, error) {
	mr, err := r.MultipartReader()
	if err != nil {
		return "", nil, fmt.Errorf("opening multipart reader: %s", err)
	}

	var region string
	var file io.Reader
Loop:
	for {
		part, err := mr.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", nil, fmt.Errorf("getting next part: %s", err)
		}
		switch part.FormName() {
		case "region":
			buf := &strings.Builder{}
			if _, err := io.Copy(buf, part); err != nil {
				return "", nil, fmt.Errorf("getting region part: %s", err)
			}
			region = buf.String()
		case "file":
			file = part
			break Loop
		default:
			return "", nil, errors.New("malformed request")
		}
	}
	if file == nil {
		return "", nil, fmt.Errorf("missing file part: %s", err)
	}

	return region, file, nil
}

func httpError(w http.ResponseWriter, err string, status int) {
	log.Errorf("request error: %s", err)
	http.Error(w, err, status)
}
