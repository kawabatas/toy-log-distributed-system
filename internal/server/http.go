package server

import (
	"crypto/tls"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"time"
)

func NewHTTPServer(addr string, opts ...Option) (*http.Server, error) {
	httpServer := newHTTPServer()
	mux := http.NewServeMux()
	mux.HandleFunc("POST /", httpServer.handleProduce)
	mux.HandleFunc("GET /", httpServer.handleConsume)
	mux.HandleFunc("GET /hello", func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, "Hello, world!\n")
	})

	var option option
	for _, opt := range opts {
		err := opt(&option)
		if err != nil {
			return nil, err
		}
	}

	s := &http.Server{
		Addr:              addr,
		ReadHeaderTimeout: 500 * time.Millisecond,
		ReadTimeout:       500 * time.Millisecond,
		Handler:           http.TimeoutHandler(mux, 2*time.Second, "timeout handler"),
		IdleTimeout:       time.Second,
	}
	if option.tlsConfig != nil {
		s.TLSConfig = option.tlsConfig
	}
	return s, nil
}

type option struct {
	tlsConfig *tls.Config
}

type Option func(o *option) error

func WithTLSConfig(cfg *tls.Config) Option {
	return func(o *option) error {
		o.tlsConfig = cfg
		return nil
	}
}

type httpServer struct {
	Log *Log
}

func newHTTPServer() *httpServer {
	return &httpServer{
		Log: NewLog(), // inmemory
	}
}

type ProduceRequest struct {
	Record Record `json:"record"`
}

type ProduceResponse struct {
	Offset uint64 `json:"offset"`
}

type ConsumeRequest struct {
	Offset uint64 `json:"offset"`
}

type ConsumeResponse struct {
	Record Record `json:"record"`
}

func (s *httpServer) handleProduce(w http.ResponseWriter, r *http.Request) {
	defer func() {
		err := r.Body.Close()
		if err != nil {
			log.Printf("failed to close response: %v\n", err)
		}
	}()

	var req ProduceRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	off, err := s.Log.Append(req.Record)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	res := ProduceResponse{Offset: off}
	err = json.NewEncoder(w).Encode(res)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (s *httpServer) handleConsume(w http.ResponseWriter, r *http.Request) {
	defer func() {
		err := r.Body.Close()
		if err != nil {
			log.Printf("failed to close response: %v\n", err)
		}
	}()

	var req ConsumeRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	record, err := s.Log.Read(req.Offset)
	if err != nil {
		if errors.Is(err, ErrOffsetNotFound) {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	res := ConsumeResponse{Record: record}
	err = json.NewEncoder(w).Encode(res)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}
