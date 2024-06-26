package server

import (
	"errors"
	"sync"

	"github.com/oklog/ulid/v2"
)

var ErrOffsetNotFound = errors.New("offset not found")
var emptyID ulid.ULID

type Log struct {
	mu      sync.Mutex
	records []Record
}

func NewLog() *Log {
	return &Log{}
}

func (c *Log) Append(record Record) (uint64, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if record.ID == emptyID {
		record.ID = ulid.Make()
	}
	record.Offset = uint64(len(c.records))
	c.records = append(c.records, record)
	return record.Offset, nil
}

func (c *Log) Read(offset uint64) (Record, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if offset >= uint64(len(c.records)) {
		return Record{}, ErrOffsetNotFound
	}
	return c.records[offset], nil
}

type Record struct {
	ID     ulid.ULID `json:"id"`
	Value  string    `json:"value"`
	Offset uint64    `json:"offset"`
}
