package main

import (
	"fmt"
	"io"
	"log"
	"net/http"

	"github.com/kawabatas/toy-log-distributed-system/internal/config"
)

func main() {
	tlsConfig, err := config.SetupTLSConfig(config.TLSConfig{
		CAFile:   config.CAFile,
		CertFile: config.RootClientCertFile,
		KeyFile:  config.RootClientKeyFile,
	})
	if err != nil {
		log.Fatalf("Setup TLS config error: %v", err)
	}

	httpClient := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: tlsConfig,
		},
	}

	r, err := httpClient.Get("https://localhost:8443/hello")
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		err := r.Body.Close()
		if err != nil {
			log.Printf("failed to close response: %v\n", err)
		}
	}()

	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("%s\n", body)
}
