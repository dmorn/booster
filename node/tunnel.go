package node

import (
	"sync"
)

type Tunnel struct {
	id     string
	Target string

	sync.Mutex
	copies int // number of copies
	acks   int // number of acknoledged copies
}

func NewTunnel(target string) *Tunnel {
	return &Tunnel{
		id:     sha1Hash([]byte(target)),
		Target: target,
		copies: 1,
	}
}

func (t *Tunnel) ID() string {
	return t.id
}

func (t *Tunnel) Copies() int {
	t.Lock()
	defer t.Unlock()

	return t.copies
}

func (t *Tunnel) Acks() int {
	t.Lock()
	defer t.Unlock()

	return t.acks
}
