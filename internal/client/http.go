package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/kawabatas/toy-log-distributed-system/internal/config"
	"github.com/kawabatas/toy-log-distributed-system/internal/server"
)

type HttpClient struct {
	client *http.Client
}

func NewClient() (*HttpClient, error) {
	tlsConfig, err := config.SetupTLSConfig(config.TLSConfig{
		CAFile:   config.CAFile,
		CertFile: config.RootClientCertFile,
		KeyFile:  config.RootClientKeyFile,
	})
	if err != nil {
		return nil, err
	}

	return &HttpClient{
		client: &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: tlsConfig,
			},
		},
	}, nil
}

func (c *HttpClient) Consume(addr string, offset int) (server.Record, error) {
	var empty server.Record
	param, err := json.Marshal(&server.ConsumeRequest{
		Offset: uint64(offset),
	})
	if err != nil {
		return empty, err
	}
	endpoint := fmt.Sprintf("https://%s", addr)
	ctx := context.Background()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, bytes.NewBuffer(param))
	if err != nil {
		return empty, err
	}
	req.Header.Add("Content-Type", "application/json")
	res, err := c.client.Do(req)
	if err != nil {
		return empty, err
	}
	defer func() {
		err := res.Body.Close()
		if err != nil {
			log.Printf("failed to close response: %v\n", err)
		}
	}()

	if res.StatusCode >= 400 {
		return empty, err
	}

	var resData server.ConsumeResponse
	if err := json.NewDecoder(res.Body).Decode(&resData); err != nil {
		return empty, err
	}
	return resData.Record, nil
}

func (c *HttpClient) Produce(addr string, record server.Record) (int, error) {
	param, err := json.Marshal(&server.ProduceRequest{
		Record: record,
	})
	if err != nil {
		return 0, err
	}
	endpoint := fmt.Sprintf("https://%s", addr)
	ctx := context.Background()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewBuffer(param))
	if err != nil {
		return 0, err
	}
	req.Header.Add("Content-Type", "application/json")
	res, err := c.client.Do(req)
	if err != nil {
		return 0, err
	}
	defer func() {
		err := res.Body.Close()
		if err != nil {
			log.Printf("failed to close response: %v\n", err)
		}
	}()

	if res.StatusCode >= 400 {
		return 0, err
	}

	var resData server.ProduceResponse
	if err := json.NewDecoder(res.Body).Decode(&resData); err != nil {
		return 0, err
	}
	return int(resData.Offset), nil
}
