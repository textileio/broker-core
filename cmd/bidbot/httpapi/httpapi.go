package httpapi

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/libp2p/go-libp2p-core/crypto"
	"github.com/libp2p/go-libp2p-core/host"
	ma "github.com/multiformats/go-multiaddr"
	"github.com/textileio/broker-core/broker"
	"github.com/textileio/broker-core/cmd/bidbot/service/datauri"
	bidstore "github.com/textileio/broker-core/cmd/bidbot/service/store"
	golog "github.com/textileio/go-log/v2"
)

var (
	log = golog.Logger("bidbot/api")
)

// Service provides scoped access to the bidbot service.
type Service interface {
	Host() host.Host
	ListBids(query bidstore.Query) ([]*bidstore.Bid, error)
	GetBid(id broker.BidID) (*bidstore.Bid, error)
	WriteDataURI(uri string) (string, error)
}

// NewServer returns a new http server for bidbot commands.
func NewServer(listenAddr string, service Service) (*http.Server, error) {
	httpServer := &http.Server{
		Addr:    listenAddr,
		Handler: createMux(service),
	}

	go func() {
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Errorf("stopping http server: %s", err)
		}
	}()

	log.Infof("http server started at %s", listenAddr)
	return httpServer, nil
}

func createMux(service Service) *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("/health", getOnly(healthHandler))
	mux.HandleFunc("/id", getOnly(idHandler(service)))
	// allow both with and without trailing slash
	deals := getOnly(dealsHandler(service))
	mux.HandleFunc("/deals", deals)
	mux.HandleFunc("/deals/", deals)
	dataURI := getOnly(dataURIRequestHandler(service))
	mux.HandleFunc("/datauri", dataURI)
	mux.HandleFunc("/datauri/", dataURI)
	return mux
}

func getOnly(f http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			httpError(w, "only GET method is allowed", http.StatusBadRequest)
			return
		}
		f(w, r)
	}
}

func healthHandler(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
}

func idHandler(service Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		host := service.Host()
		var pkey string
		if pk := host.Peerstore().PubKey(host.ID()); pk != nil {
			pkb, err := crypto.MarshalPublicKey(pk)
			if err != nil {
				httpError(w, fmt.Sprintf("marshaling public key: %s", err), http.StatusInternalServerError)
				return
			}
			pkey = base64.StdEncoding.EncodeToString(pkb)
		}

		v := struct {
			ID        string
			PublicKey string
			Addresses []ma.Multiaddr
		}{
			host.ID().String(),
			pkey,
			host.Addrs(),
		}
		data, err := json.MarshalIndent(v, "", "\t")
		if err != nil {
			httpError(w, fmt.Sprintf("marshaling id: %s", err), http.StatusInternalServerError)
			return
		}
		_, err = w.Write(data)
		if err != nil {
			log.Errorf("write failed: %v", err)
		}
	}
}

func dealsHandler(service Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var bids []*bidstore.Bid
		urlParts := strings.SplitN(r.URL.Path, "/", 3)
		if len(urlParts) < 3 || urlParts[2] == "" {
			statusFilters := strings.Split(r.URL.Query().Get("status"), ",")
			var proceed bool
			if bids, proceed = listBids(w, service, statusFilters); !proceed {
				return
			}
		} else {
			bid, err := service.GetBid(broker.BidID(urlParts[2]))
			if err == nil {
				bids = append(bids, bid)
			} else if err != bidstore.ErrBidNotFound {
				httpError(w, fmt.Sprintf("get bid: %s", err), http.StatusInternalServerError)
				return
			}
		}
		data, err := json.Marshal(bids)
		if err != nil {
			httpError(w, fmt.Sprintf("json encoding: %s", err), http.StatusInternalServerError)
			return
		}
		_, err = w.Write(data)
		if err != nil {
			log.Errorf("write failed: %v", err)
		}
	}
}

func listBids(w http.ResponseWriter, service Service, statusFilters []string) (bids []*bidstore.Bid, proceed bool) {
	var filters []bidstore.BidStatus
	for _, s := range statusFilters {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		bs, err := bidstore.BidStatusByString(s)
		if err != nil {
			httpError(w, fmt.Sprintf("%s: %s", s, err), http.StatusBadRequest)
			return nil, false
		}
		filters = append(filters, bs)
	}
	// for simplicity we apply filters after retrieving. if performance
	// becomes an issue, we can add query filters.
	fullList, err := service.ListBids(bidstore.Query{Limit: -1})
	if err != nil {
		httpError(w, fmt.Sprintf("listing bids: %s", err), http.StatusInternalServerError)
		return nil, false
	}
	if len(filters) == 0 {
		return fullList, true
	}
	for _, bid := range fullList {
		for _, status := range filters {
			if bid.Status == status {
				bids = append(bids, bid)
				break
			}
		}
	}
	return bids, true
}

func dataURIRequestHandler(service Service) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			httpError(w, "only GET method is allowed", http.StatusBadRequest)
			return
		}

		query := r.URL.Query().Get("uri")
		if query == "" {
			httpError(w, "missing 'uri' query param", http.StatusBadRequest)
			return
		}
		uri, err := url.QueryUnescape(query)
		if err != nil {
			httpError(w, fmt.Sprintf("parsing query: %s", err), http.StatusBadRequest)
			return
		}

		dest, err := service.WriteDataURI(uri)
		if errors.Is(err, datauri.ErrSchemeNotSupported) ||
			errors.Is(err, datauri.ErrCarFileUnavailable) ||
			errors.Is(err, datauri.ErrInvalidCarFile) {
			httpError(w, fmt.Sprintf("writing data uri: %s", err), http.StatusBadRequest)
			return
		} else if err != nil {
			httpError(w, fmt.Sprintf("writing data uri: %s", err), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		resp := fmt.Sprintf("wrote to %s", dest)
		if _, err = w.Write([]byte(resp)); err != nil {
			httpError(w, fmt.Sprintf("writing response: %s", err), http.StatusInternalServerError)
			return
		}
	}
}

func httpError(w http.ResponseWriter, err string, status int) {
	log.Debugf("request error: %s", err)
	http.Error(w, err, status)
}
