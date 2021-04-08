package httpapi

import (
	"fmt"
	"io"
	"net/http"
	"time"

	logging "github.com/ipfs/go-log/v2"
)

var (
	LogName = "http-api"
	log     = logging.Logger(LogName)
)

func NewServer(listenAddr string) (*http.Server, error) {
	mux := http.NewServeMux()
	httpServer := &http.Server{
		Addr:              listenAddr,
		ReadHeaderTimeout: time.Second * 5,
		WriteTimeout:      time.Second * 10,
		Handler:           mux,
	}

	mux.HandleFunc("/upload", uploadHandler)

	log.Infof("Running server...")
	go func() {
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal(err)
		}
	}()

	return httpServer, nil
}

func uploadHandler(w http.ResponseWriter, r *http.Request) {
	mr, err := r.MultipartReader()
	if err != nil {
		http.Error(w, "opening multipart reader", http.StatusBadRequest)
		return
	}

	for {
		part, err := mr.NextPart()
		if err == io.EOF {
			break
		}
		total, err := io.Copy(io.Discard, part)
		if err != nil {
			http.Error(w, "reading part", http.StatusBadRequest)
			return
		}
		fmt.Printf("%s %s %d\n", part.FileName(), part.FormName(), total)
	}
}
