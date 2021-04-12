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
	LogName = "http-api"

	gib         = 1024 * 1024 * 1024
	maxBodySize = 32 * gib
)

var (
	log = logging.Logger(LogName)
)

func NewServer(listenAddr string, s storage.StorageRequester) (*http.Server, error) {
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

func createMux(s storage.StorageRequester) *http.ServeMux {
	mux := http.NewServeMux()

	mux.Handle("/upload", otelhttp.NewHandler(http.HandlerFunc(uploadHandler(s)), "upload"))
	return mux
}

func uploadHandler(s storage.StorageRequester) func(w http.ResponseWriter, r *http.Request) {
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
			httpError(w, fmt.Sprintf("uploading data: %s", err), http.StatusInternalServerError)
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
