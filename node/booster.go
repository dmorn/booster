package node

import (
	"context"
	"fmt"
	"log"
	"os"
	"strconv"

	"github.com/danielmorandini/booster/net"
)

type Packet interface {
	Module(id string) (Module, error)
}

type Module interface {
	ID() string
	Payload() []byte
	Encoding() string
}

type Conn interface {
	Consume() (<-chan Packet, error)
	Send(Packet) error
	Close() error
	Err() error
}

type Booster struct {
	*log.Logger

	stop chan struct{}
}

func New() *Booster {
	log := log.New(os.Stdout, "BOOSTER ", log.LstdFlags)

	b := new(Booster)
	b.Logger = log
	stop = make(chan struct{})
}

// Run starts a booster node. It is made by a booster compliant tcp server
// and a socks5 compliant tcp server.
func (b *Booster) Run(pport, bport int) error {
	errc := make(chan error)

	go func() {
		errc <- b.ListenAndServe(bport)
	}()

	select {
	case err := <-errc:
		return err
	case <-b.stop:
		<-errc // wait for ListenAndServe to return
		return fmt.Errorf("booster: stopped")
	}
}

func (b *Booster) Close() error {
	b.stop <- struct{}{}
	return nil
}

// ListenAndServe listens and serves tcp connections.
func (b *Booster) ListenAndServe(port int) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	p := strconv.Itoa(port)
	ln, err := net.Listen("tcp", ":"+p)
	if err != nil {
		return err
	}

	b.Printf("listening on port: %v", p)

	errc := make(chan error)

	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				errc <- fmt.Errorf("booster: cannot accept conn: %v", err)
				return
			}

			pkts, err := conn.Consume()
			if err != nil {
				errc <- fmt.Errorf("booster: cannot consume packets: %v", err)
			}

			go b.Handle(ctx, pkts)
		}
	}()

	select {
	case err := <-errc:
		return err
	case <-ctx.Done():
		<-errc // wait for listener to return
		return ctx.Err()
	}
}

func (b *Booster) Handle(ctx context.Context, pkts <-chan Packet) {
}
