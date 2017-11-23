package node

import (
	"context"
	"errors"
	"io"
	"log"
	"net"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/danielmorandini/booster-network/socks5"
)

const (
	BoosterVersion = uint8(1)
)

const (
	BoosterCMDRegister = uint8(1)
	BoosterCMDHello    = uint8(2)
)

const (
	BoosterFieldReserved = uint8(0xff)
)

const (
	BoosterRespRefused  = uint8(0)
	BoosterRespAccepted = uint8(1)
)

type Booster struct {
	*log.Logger
	Proxy    *Proxy
	balancer *Balancer

	sync.Mutex
}

// Conn is a wrapper around io.ReadWriteCloser.
type Conn interface {
	io.ReadWriteCloser
	RemoteAddr() net.Addr
}

func NewBooster() *Booster {
	b := new(Booster)
	bal := new(Balancer)
	b.balancer = bal
	b.Proxy = NewProxy(bal)
	b.Proxy.Logger = log.New(os.Stdout, "PROXY ", log.LstdFlags)

	return b
}

func (b *Booster) ListenAndServe() error {
	ln, err := net.Listen("tcp", ":8448")
	if err != nil {
		return err
	}

	b.Printf("listening on port: 8448")

	for {
		conn, err := ln.Accept()
		if err != nil {
			b.Printf("tcp accept error: %v\n", err)
			continue
		}

		go func() {
			if err := b.Handle(conn); err != nil {
				b.Println(err)
			}
		}()
	}
}

// Handle takes care of every connection that booster receives.
// It expects to receive only "Register" or "Hello" requests.
// Ends serving forever the state of the proxy.
func (b *Booster) Handle(conn Conn) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	defer conn.Close()

	buf := make([]byte, 3)
	if _, err := io.ReadFull(conn, buf); err != nil {
		return errors.New("booster: unable to parse request")
	}

	v := buf[0] // version
	if v != BoosterVersion {
		return errors.New("booster: unsupported version: " + strconv.Itoa(int(v)))
	}

	cmd := buf[1] // command
	_ = buf[2]    // reserved field

	switch cmd {
	case BoosterCMDRegister:
		return b.handleRegister(ctx, conn)

	case BoosterCMDHello:
		if err := b.handleHello(conn); err != nil {
			return err
		}

	default:
		return errors.New("booster: unknown command " + string(cmd) + "from " + conn.RemoteAddr().String())
	}

	return b.ServeStatus(ctx, conn)
}

func (b *Booster) handleHello(conn Conn) error {
	// TODO(daniel): there could be some cases where the hello request shuold be refused.
	// Atm we always reply ok to this request.
	port := b.Proxy.Port()

	buf := make([]byte, 0, 5)
	buf = append(buf, BoosterVersion)
	buf = append(buf, BoosterRespAccepted)
	buf = append(buf, BoosterFieldReserved)
	buf = append(buf, byte(port>>8), byte(port))

	if _, err := conn.Write(buf); err != nil {
		return errors.New("booster: unable to write hello response: " + err.Error())
	}

	return nil
}

func (b *Booster) handleRegister(ctx context.Context, conn Conn) error {
	addr, err := socks5.ReadAddress(conn)
	if err != nil {
		return errors.New("booster: " + err.Error())
	}

	return b.Register(ctx, "tcp", addr)
}

// Register dials with the remote address, expecting it to be a booster server.
// Right after having enstablished the connection, it performs a "Hello" request.
// If the response is successfull, the remote booster is added to the list of
// helpers.
func (b *Booster) Register(ctx context.Context, network, addr string) error {
	_ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	conn, err := new(net.Dialer).DialContext(_ctx, network, addr)
	if err != nil {
		return errors.New("booster: unable to contact remote instance: " + err.Error())
	}

	return b.Hello(conn)
}

// Hello performs the "Hello" procedure with the connection.
func (b *Booster) Hello(conn Conn) error {
	buf := make([]byte, 0, 3)
	buf = append(buf, BoosterVersion)
	buf = append(buf, BoosterCMDHello)
	buf = append(buf, BoosterFieldReserved)

	if _, err := conn.Write(buf); err != nil {
		return errors.New("booster: unable to perform hello request: " + err.Error())
	}

	buf = make([]byte, 5)
	if _, err := io.ReadFull(conn, buf); err != nil {
		return errors.New("booster: unable read hello response: " + err.Error())
	}

	v := buf[0] // version
	if v != BoosterVersion {
		return errors.New("booster: unsupported version " + strconv.Itoa(int(v)))
	}

	resp := buf[1]                       // response
	_ = buf[2]                           // reserved field
	port := int(buf[3])<<8 | int(buf[4]) // proxy listening port

	if resp != BoosterRespAccepted {
		return errors.New("booster: remote instance refused hello request")
	}

	host, _, err := net.SplitHostPort(conn.RemoteAddr().String())
	if err != nil {
		return errors.New("booster: " + err.Error())
	}
	addr := net.JoinHostPort(host, strconv.Itoa(port))
	b.balancer.AddProxy(addr, conn)

	return nil
}

// ServeStatus writes the proxy's status to the connection, whenever it changes.
func (b *Booster) ServeStatus(ctx context.Context, conn Conn) error {
	wc := make(chan int)
	ec := make(chan error)
	id := conn.RemoteAddr().String()

	if err := b.Proxy.RegisterWorkloadListener(id, wc); err != nil {
		return err
	}

	go func() {
		buf := make([]byte, 0, 3)
		buf = append(buf, BoosterVersion)
		buf = append(buf, BoosterFieldReserved)
		for workload := range wc {
			buf = buf[:2]
			buf = append(buf, byte(workload))
			if _, err := conn.Write(buf); err != nil {
				ec <- errors.New("booster: unable to write status: " + err.Error())
			}
		}
	}()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-ec:
		b.Proxy.RemoveWorkloadListener(id)
		return err
	}
}
