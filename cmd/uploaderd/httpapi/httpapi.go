package httpapi

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	logging "github.com/ipfs/go-log/v2"
	"github.com/textileio/broker-core/cmd/uploaderd/storage"
)

const (
	LogName = "http-api"

	httpErrMalformedRequest = "malformed request"
	httpErrWrongMethodName  = "wrong method name"
	httpErrMissingData      = "no data was provided"

	gib         = 1024 * 1024 * 1024
	maxBodySize = 32 * gib
)

var (
	log = logging.Logger(LogName)
)

func NewServer(listenAddr string, s storage.Storage) (*http.Server, error) {
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

func createMux(s storage.Storage) *http.ServeMux {
	mux := http.NewServeMux()

	mux.HandleFunc("/upload", uploadHandler(s))
	return mux
}

func uploadHandler(s storage.Storage) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			httpError(w, httpErrWrongMethodName, http.StatusBadRequest)

		}

		// Be safe.
		r.Body = http.MaxBytesReader(w, r.Body, maxBodySize)

		mr, err := r.MultipartReader()
		if err != nil {
			httpError(w, "opening multipart reader", http.StatusBadRequest)
			return
		}

		var (
			region string
			file   io.Reader
		)
		for {
			part, err := mr.NextPart()
			if err == io.EOF {
				break
			}
			switch part.FormName() {
			case "region":
				buf := &strings.Builder{}
				if _, err := io.Copy(buf, part); err != nil {
					httpError(w, "reading part", http.StatusBadRequest)
					return
				}
				region = buf.String()
			case "file":
				file = part
				break
			default:
				httpError(w, httpErrMalformedRequest, http.StatusBadRequest)
				return
			}
		}
		if file == nil {
			httpError(w, httpErrMissingData, http.StatusBadRequest)
			return
		}

		meta := storage.Metadata{
			Region: region,
		}
		storageRequest, err := s.CreateStorageRequest(r.Context(), file, meta)
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

func httpError(w http.ResponseWriter, err string, status int) {
	log.Errorf("request error: %s", err)
	http.Error(w, err, status)
}
