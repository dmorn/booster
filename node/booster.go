package node

import (
	"context"
	"errors"
	"io"
	"log"
	"net"
	"os"
	"strconv"

	"github.com/danielmorandini/booster-network/pubsub"
	"github.com/danielmorandini/booster-network/socks5"
)

// Booster versions
const (
	BoosterVersion1 = uint8(1)
)

// Possible booster CMD field values
const (
	BoosterCMDConnect    = uint8(1)
	BoosterCMDDisconnect = uint8(2)
	BoosterCMDHello      = uint8(3)
	BoosterCMDStatus     = uint8(4)
	BoosterCMDInspect    = uint8(5)
)

// Possible stream instructions
const (
	BoosterStreamStart = uint8(2)
	BoosterStreamStop  = uint8(3)
	BoosterStreamNext  = uint8(1)
)

// Reserved field value
const (
	BoosterFieldReserved = uint8(0xff)
)

// Possible booster RESP field values
const (
	BoosterRespGeneralFailure = uint8(0)
	BoosterRespSuccess        = uint8(1)
)

const (
	// BoosterAddrIP4 is a version-4 IP address, with a length of 4 octets
	BoosterAddrIP4 = uint8(1)

	// BoosterAddrFQDN field contains a fully-qualified domain name. The first
	// octet of the address field contains the number of octets of name that
	// follow, there is no terminating NUL octet.
	BoosterAddrFQDN = uint8(3)

	// BoosterAddrIP6 is a version-6 IP address, with a length of 16 octets.
	BoosterAddrIP6 = uint8(4)
)

const (
	// TopicRemoteNodes is the topic where the remote node updates
	// will be published. Use Sub with it to the the messages.
	TopicRemoteNodes = "topic_rn"
)

// Booster is capable of handling tcp connections that follow booster-network
// protocol. It can be initialized with a custom logger and load balancer.
type Booster struct {
	*log.Logger
	socks5.Dialer
	*Balancer
	*pubsub.PubSub

	Proxy *Proxy
}

// NewBooster returns a booster instance.
func NewBooster(proxy *Proxy, balancer *Balancer, log *log.Logger) *Booster {
	b := new(Booster)
	b.Proxy = proxy
	b.Balancer = balancer
	b.Logger = log
	b.Dialer = new(net.Dialer)
	b.PubSub = pubsub.New()

	return b
}

// NewBoosterDefault returns a new booster instance with initialized logger, balancer and proxy.
func NewBoosterDefault() *Booster {
	log := log.New(os.Stdout, "BOOSTER ", log.LstdFlags)
	balancer := NewBalancer(log)
	proxy := NewProxyBalancer(balancer)

	return NewBooster(proxy, balancer, log)
}

// Start starts a booster node. It is made by a booster compliant tcp server
// and a socks5 compliant tcp server.
func (b *Booster) Start(pport, bport int) error {
	c := make(chan error)

	go func() {
		c <- b.Proxy.ListenAndServe(pport)
	}()

	go func() {
		c <- b.ListenAndServe(bport)
	}()

	return <-c
}

// ListenAndServe listens and serves tcp connections.
func (b *Booster) ListenAndServe(port int) error {
	p := strconv.Itoa(port)
	ln, err := net.Listen("tcp", ":"+p)
	if err != nil {
		return err
	}

	b.Printf("listening on port: %v", p)

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
// It expects to receive only "Hello", "Connect", "Inspect" or "Disconnect" requests.
// Ends serving forever the state of the proxy.
func (b *Booster) Handle(conn net.Conn) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	defer conn.Close()

	buf := make([]byte, 3)
	if _, err := io.ReadFull(conn, buf); err != nil {
		return errors.New("booster: unable to parse request")
	}

	v := buf[0] // version
	if v != BoosterVersion1 {
		return errors.New("booster: unsupported version: " + strconv.Itoa(int(v)))
	}

	cmd := buf[1] // command
	_ = buf[2]    // reserved field

	switch cmd {
	case BoosterCMDConnect:
		return b.handleConnect(ctx, conn)

	case BoosterCMDDisconnect:
		return b.handleDisconnect(ctx, conn)

	case BoosterCMDInspect:
		return b.handleInspect(ctx, conn)

	case BoosterCMDHello:
		if err := b.handleHello(conn); err != nil {
			return err
		}

	default:
		return errors.New("booster: unknown command " + string(cmd) + "from " + conn.RemoteAddr().String())
	}

	return b.ServeStatus(ctx, conn)
}

// ServeStatus writes the proxy's status to the connection, whenever it changes.
func (b *Booster) ServeStatus(ctx context.Context, conn net.Conn) error {
	ec := make(chan error)
	wc := b.Proxy.Sub(socks5.TopicWorkload)

	go func() {
		buf := make([]byte, 0, 4)
		buf = append(buf, BoosterVersion1)
		buf = append(buf, BoosterCMDStatus)
		buf = append(buf, BoosterFieldReserved)
		for i := range wc {
			wl, _ := i.(int)
			buf = buf[:3]
			buf = append(buf, byte(wl))

			if _, err := conn.Write(buf); err != nil {
				ec <- errors.New("booster: unable to write status: " + err.Error())
			}
		}
	}()

	select {
	case <-ctx.Done():
		b.Proxy.Unsub(wc, socks5.TopicWorkload)
		return ctx.Err()
	case err := <-ec:
		b.Proxy.Unsub(wc, socks5.TopicWorkload)
		return err
	}
}
