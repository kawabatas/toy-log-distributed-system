package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/kawabatas/toy-log-distributed-system/internal/config"
	"github.com/kawabatas/toy-log-distributed-system/internal/server"
)

func main() {
	addr := "127.0.0.1:8443"
	tlsConfig, err := config.SetupTLSConfig(config.TLSConfig{
		CertFile:      config.ServerCertFile,
		KeyFile:       config.ServerKeyFile,
		CAFile:        config.CAFile,
		ServerAddress: addr,
		Server:        true,
	})
	if err != nil {
		log.Fatalf("Setup TLS config error: %v", err)
	}
	srv, err := server.NewHTTPServer(addr, server.WithTLSConfig(tlsConfig))
	if err != nil {
		log.Fatalf("New HTTP server error: %v", err)
	}

	log.Println("Starting...")
	go func() {
		if err := srv.ListenAndServeTLS(config.ServerCertFile, config.ServerKeyFile); !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("HTTP server error: %v", err)
		}
		log.Println("Stopped serving new connections.")
	}()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	shutdownCtx, shutdownRelease := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownRelease()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("HTTP shutdown error: %v", err)
	}
	log.Println("Graceful shutdown complete.")
}
