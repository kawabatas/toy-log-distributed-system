package agent

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"sync"

	"github.com/kawabatas/toy-log-distributed-system/internal/config"
	"github.com/kawabatas/toy-log-distributed-system/internal/discovery"
	"github.com/kawabatas/toy-log-distributed-system/internal/replicator"
	"github.com/kawabatas/toy-log-distributed-system/internal/server"
)

type Agent struct {
	Config

	server     *http.Server
	membership *discovery.Membership
	replicator *replicator.Replicator

	shutdown     bool
	shutdowns    chan struct{}
	shutdownLock sync.Mutex
}

type Config struct {
	BindAddr       string // for Serf
	RPCPort        int    // for HTTP
	NodeName       string
	StartJoinAddrs []string
}

func (c Config) RPCAddr() (string, error) {
	host, _, err := net.SplitHostPort(c.BindAddr)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s:%d", host, c.RPCPort), nil
}

func New(config Config) (*Agent, error) {
	a := &Agent{
		Config:    config,
		shutdowns: make(chan struct{}),
	}
	setup := []func() error{
		a.setupServer,
		a.setupMembership,
	}
	for _, fn := range setup {
		if err := fn(); err != nil {
			return nil, err
		}
	}
	return a, nil
}

func (a *Agent) setupServer() error {
	rpcAddr, err := a.Config.RPCAddr()
	if err != nil {
		return err
	}
	tlsConfig, err := config.SetupTLSConfig(config.TLSConfig{
		CertFile:      config.ServerCertFile,
		KeyFile:       config.ServerKeyFile,
		CAFile:        config.CAFile,
		ServerAddress: rpcAddr,
		Server:        true,
	})
	if err != nil {
		return fmt.Errorf("setup TLS config error: %w", err)
	}
	a.server, err = server.NewHTTPServer(rpcAddr, server.WithTLSConfig(tlsConfig))
	if err != nil {
		return fmt.Errorf("new HTTP server error: %w", err)
	}
	log.Printf("[INFO] starting HTTP server at %s\n", rpcAddr)

	go func() {
		if err := a.server.ListenAndServeTLS(config.ServerCertFile, config.ServerKeyFile); !errors.Is(err, http.ErrServerClosed) {
			_ = a.server.Shutdown(context.Background())
		}
	}()
	return err
}

func (a *Agent) setupMembership() error {
	rpcAddr, err := a.Config.RPCAddr()
	if err != nil {
		return err
	}
	a.replicator = &replicator.Replicator{
		LocalAddr: rpcAddr,
	}
	a.membership, err = discovery.New(a.replicator, discovery.Config{
		NodeName: a.Config.NodeName,
		BindAddr: a.Config.BindAddr,
		Tags: map[string]string{
			"rpc_addr": rpcAddr,
		},
		StartJoinAddrs: a.Config.StartJoinAddrs,
	})
	return err
}

func (a *Agent) Shutdown() error {
	a.shutdownLock.Lock()
	defer a.shutdownLock.Unlock()
	if a.shutdown {
		return nil
	}
	a.shutdown = true
	close(a.shutdowns)

	shutdown := []func() error{
		a.membership.Leave,
		a.replicator.Close,
		func() error {
			a.server.Shutdown(context.Background())
			return nil
		},
	}
	for _, fn := range shutdown {
		if err := fn(); err != nil {
			return err
		}
	}
	return nil
}
