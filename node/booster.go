package node

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strconv"

	"github.com/danielmorandini/booster/net"
	"github.com/danielmorandini/booster/net/packet"
)

type Booster struct {
	*log.Logger

	stop chan struct{}
}

func NewBooster() *Booster {
	log := log.New(os.Stdout, "BOOSTER ", log.LstdFlags)

	b := new(Booster)
	b.Logger = log
	b.stop = make(chan struct{})

	return b
}

func (b *Booster) Run(pport, bport int) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	errc := make(chan error)
	go func() {
		errc <- b.ListenAndServe(ctx, bport)
	}()

	go func() {
		c := make(chan os.Signal, 1)
		signal.Notify(c, os.Interrupt)

		for sig := range c {
			b.Printf("booster: signal (%v) received: exiting...", sig)
			b.Close()
			return
		}
	}()

	select {
	case err := <-errc:
		return err
	case <-b.stop:
		cancel()
		<-errc // wait for ListenAndServe to return
		return fmt.Errorf("booster: stopped")
	}
}

func (b *Booster) Close() error {
	b.stop <- struct{}{}
	return nil
}

func (b *Booster) ListenAndServe(ctx context.Context, port int) error {
	p := strconv.Itoa(port)
	ln, err := net.Listen("tcp", ":"+p)
	if err != nil {
		return err
	}
	defer ln.Close()

	b.Printf("listening on port: %v", p)

	errc := make(chan error)
	defer close(errc)

	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				errc <- fmt.Errorf("booster: cannot accept conn: %v", err)
				return
			}

			go b.HandleConn(ctx, conn)
		}
	}()

	select {
	case err := <-errc:
		return err
	case <-ctx.Done():
		ln.Close()
		<-errc // wait for listener to return
		return ctx.Err()
	}
}

func (b *Booster) HandleConn(ctx context.Context, conn *net.Conn) {
	pkts, err := conn.Consume()
	if err != nil {
		b.Printf("booster: cannot consume packets: %v", err)
		return
	}

	b.HandlePkts(ctx, pkts)
}

func (b *Booster) HandlePkts(ctx context.Context, pkts <-chan *packet.Packet) {
}
