package replicator

import (
	"errors"
	"log"
	"sync"
	"time"

	"github.com/kawabatas/toy-log-distributed-system/internal/client"
	"github.com/kawabatas/toy-log-distributed-system/internal/server"
	"github.com/oklog/ulid/v2"
)

type Replicator struct {
	LocalAddr string

	mu      sync.Mutex
	servers map[string]chan struct{}
	closed  bool
	close   chan struct{}
}

func (r *Replicator) Join(name, addr string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.init()

	if r.closed {
		return nil
	}

	if _, ok := r.servers[name]; ok {
		// すでにレプリケーションを行っているのでスキップ
		return nil
	}
	r.servers[name] = make(chan struct{})

	httpClient, err := client.NewClient()
	if err != nil {
		return err
	}
	go r.replicate(httpClient, addr, r.servers[name])

	return nil
}

func (r *Replicator) replicate(c *client.HttpClient, addr string, leave chan struct{}) {
	records := make(chan server.Record)

	go func() {
		timerDuration := 1 * time.Second
		timer := time.NewTimer(timerDuration)
		var lastRecordID ulid.ULID
		offset := 0

		for {
			timer.Reset(timerDuration)
			<-timer.C
			res, err := c.Consume(addr, offset)
			if err != nil {
				if errors.Is(err, server.ErrOffsetNotFound) {
					break
				}
				r.logError(err, "failed to consume", addr)
			}
			if res.Value != "" {
				if lastRecordID.Compare(res.ID) < 0 {
					records <- res
					offset++
					lastRecordID = res.ID
				}
			}
		}
	}()

	for {
		select {
		case <-r.close:
			return
		case <-leave:
			return
		case record := <-records:
			log.Printf("[DEBUG] record: %v\n", record)
			_, err := c.Produce(r.LocalAddr, record)
			if err != nil {
				r.logError(err, "failed to produce", r.LocalAddr)
				return
			}
		}
	}
}

func (r *Replicator) Leave(name string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.init()
	if _, ok := r.servers[name]; !ok {
		return nil
	}
	close(r.servers[name])
	delete(r.servers, name)
	return nil
}

func (r *Replicator) init() {
	if r.servers == nil {
		r.servers = make(map[string]chan struct{})
	}
	if r.close == nil {
		r.close = make(chan struct{})
	}
}

func (r *Replicator) Close() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.init()

	if r.closed {
		return nil
	}
	r.closed = true
	close(r.close)
	return nil
}

func (m *Replicator) logError(err error, msg, addr string) {
	log.Printf("[ERR] msg: %s, addr: %s, err: %s", msg, addr, err)
}
