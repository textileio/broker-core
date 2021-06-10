package httpapi

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/ipfs/go-cid"
	bidstore "github.com/textileio/broker-core/cmd/bidbot/service/store"
	golog "github.com/textileio/go-log/v2"
)

var (
	log = golog.Logger("bidbot/api")
)

// Service provides scoped access to the bidbot service.
type Service interface {
	ListBids(query bidstore.Query) ([]bidstore.Bid, error)
	WriteCar(ctx context.Context, cid cid.Cid) (string, error)
}

// NewServer returns a new http server for bidbot commands.
func NewServer(listenAddr string, service Service) (*http.Server, error) {
	httpServer := &http.Server{
		Addr:              listenAddr,
		ReadHeaderTimeout: time.Second * 5,
		WriteTimeout:      time.Second * 10,
		Handler:           createMux(service),
	}

	go func() {
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Errorf("stopping http server: %s", err)
		}
	}()

	log.Info("http server started")
	return httpServer, nil
}

func createMux(service Service) *http.ServeMux {
	mux := http.NewServeMux()

	mux.HandleFunc("/health", healthHandler)

	writeCarRequest := writeCarRequestHandler(service)
	mux.HandleFunc("/cid/", writeCarRequest)

	return mux
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		httpError(w, "only GET method is allowed", http.StatusBadRequest)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func writeCarRequestHandler(service Service) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			httpError(w, "only GET method is allowed", http.StatusBadRequest)
			return
		}
		urlParts := strings.SplitN(r.URL.Path, "/", 3)
		if len(urlParts) < 3 {
			httpError(w, "the url should be /cid/{cid}", http.StatusBadRequest)
			return
		}
		id, err := cid.Decode(urlParts[2])
		if err != nil {
			httpError(w, fmt.Sprintf("decoding cid: %s", err), http.StatusBadRequest)
			return
		}

		dest, err := service.WriteCar(r.Context(), id)
		if err != nil {
			httpError(w, fmt.Sprintf("writing service: %s", err), http.StatusInternalServerError)
			return
		}

		resp := fmt.Sprintf("downloaded to %s", dest)
		if _, err = w.Write([]byte(resp)); err != nil {
			httpError(w, fmt.Sprintf("writing response: %s", err), http.StatusInternalServerError)
			return
		}
	}
}

func httpError(w http.ResponseWriter, err string, status int) {
	log.Errorf("request error: %s", err)
	http.Error(w, err, status)
}
