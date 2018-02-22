package booster

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"strconv"
	"time"

	"github.com/danielmorandini/booster-network/pubsub"
	"github.com/danielmorandini/booster-network/socks5"
	"github.com/danielmorandini/booster-network/tracer"
	"github.com/danielmorandini/booster-network/node"
	"github.com/danielmorandini/booster-network/proxy"
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
	BoosterCMDHeartbeat  = uint8(6)
)

// Possible stream instructions
const (
	BoosterStreamStart = uint8(2)
	BoosterStreamStop  = uint8(3)
	BoosterStreamNext  = uint8(1)
)

// Possible node operations
const (
	BoosterNodeAdded   = uint8(0)
	BoosterNodeClosed  = uint8(1)
	BoosterNodeUpdated = uint8(2)
	BoosterNodeRemoved = uint8(3)
)

// Reserved field value
const (
	BoosterFieldReserved = uint8(0xff)
	BoosterFieldPing     = uint8(0)
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
	// TopicNodes is the topic where the remote node updates
	// will be published. Use Sub with it to the the messages.
	TopicNodes = "topic_rn"
)

// Tracer is a wrapper around the basic Trace and Untrace functions.
type Tracer interface {
	Trace(p tracer.Pinger) error
	Untrace(id string)
}

// PubSub describes the required functionalities of a publication/subscription object.
type PubSub interface {
	Sub(topic string) (chan interface{}, error)
	Unsub(c chan interface{}, topic string) error
	Pub(message interface{}, topic string) error
}

// Booster is capable of handling tcp connections that follow booster-network
// protocol. It can be initialized with a custom logger and load balancer.
type Booster struct {
	*log.Logger
	socks5.Dialer
	*Balancer
	PubSub
	Tracer

	Proxy proxy.Proxy
	NodeIdleTimeout time.Duration // time before closing an idle remote node
	stop chan struct{}
}

func New() *Booster {
	log := log.New(os.Stdout, "BOOSTER ", log.LstdFlags)
	ps := pubsub.New()
	tr := tracer.New(log, pubsub)
	balancer := NewBalancer(log, pubsub)
	proxy := socks5.New()
	dialer := NewDialer(balancer)

	proxy.SetDialer(dialer)

	b := new(Booster)
	b.Proxy = proxy
	b.Balancer = balancer
	b.Logger = log
	b.Dialer = new(net.Dialer)
	b.PubSub = ps
	b.Tracer = tr

	b.NacerodeIdleTimeout = 3 * time.Second
}

// ListenAndServe listens and serves tcp connections.
func (b *Booster) ListenAndServe(ctx context.Context, port int) error {
	p := strconv.Itoa(port)
	ln, err := net.Listen("tcp", ":"+p)
	if err != nil {
		return err
	}

	// TODO(daniel): make this loop exit in case of context Done()
	b.Printf("listening on port: %v", p)
	for {
		conn, err := ln.Accept()
		if err != nil {
			b.Printf("tcp accept error: %v", err)
			continue
		}

		go func() {
			if err := b.Handle(ctx, conn); err != nil {
				b.Println(err)
			}
		}()
	}
}

// Start starts a booster node. It is made by a booster compliant tcp server
// and a socks5 compliant tcp server.
func (b *Booster) Start(pport, bport int) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	errc := make(chan error)
	b.stop = make(chan struct{})

	rootNode, _ := node.New("localhost", strconv.Itoa(pport), strconv.Itoa(bport), true)
	b.SetRootNode(rootNode)

	go func() {
		errc <- b.StartUpdatingRootNode(ctx)
	}()

	go func() {
		errc <- b.StartNodeTracer(ctx, bport)
	}()

	go func() {
		errc <- b.Proxy.ListenAndServe(ctx, pport)
	}()

	go func() {
		errc <- b.ListenAndServe(ctx, bport)
	}()

	select {
	case err := <- errc:
		return err
	case <-ctx.Done():
		return errors.New("booster: start: " + ctx.Err().Error())
	case <-b.stop:
		return errors.New("booster: stopped")
	}
}

func (b *Booster) StartNodeTracer(ctx context.Context, bport int) error {
	stream, _ := b.Sub(tracer.TopicConnDiscovered)
	errc := make(chan error)
	defer func() {
		b.Unsub(stream, tracer.TopicConnDiscovered)
	}()

	go func() {
		errc <- b.startNodeTracer(stream, bport)
	}()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-errc:
		return err
	}
}


func (b *Booster) StartUpdatingRootNode(ctx context.Context) error {
	stream, _ := b.Sub(socks5.TopicWorkload)
	errc := make(chan error)
	defer func() {
		b.Unsub(stream, socks5.TopicWorkload)
	}()

	go func() {
		errc <- b.updateRootNode(stream)
	}()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-errc:
		return err
	}
}

func (b *Booster) Stop() {
	close(b.stop)
}

// Handle takes care of every connection that booster receives.
// It expects to receive only "Hello", "Connect", "Inspect" or "Disconnect" requests.
// Ends serving forever the state of the proxy.
func (b *Booster) Handle(ctx context.Context, conn net.Conn) error {
	_ctx := ctx
	ctx, cancel := context.WithCancel(ctx)
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

	errc := make(chan error)
	go func() {
		switch cmd {
		case BoosterCMDConnect:
			node, err := b.handleConnect(ctx, conn)
			if err != nil {
				errc <- err
			}

			if err := b.Ping(_ctx, node); err != nil {
				errc <- err
			}

			if err = b.Status(_ctx, node); err != nil {
				errc <- err
			}

		case BoosterCMDDisconnect:
			errc <- b.handleDisconnect(ctx, conn)

		case BoosterCMDInspect:
			errc <- b.handleInspect(ctx, conn)

		case BoosterCMDHello:
			errc <- b.handleHello(conn)

		case BoosterCMDStatus:
			errc <- b.ServeStatus(ctx, conn)

		case BoosterCMDHeartbeat:
			errc <- b.handlePing(ctx, conn)

		default:
			errc <- errors.New("booster: unknown command " + string(cmd) + "from " + conn.RemoteAddr().String())
		}
	}()

	select {
	case <-b.stop:
		return errors.New("booster: stopped")
	case <- ctx.Done():
		return errors.New("booster: " + ctx.Err().Error())
	case err := <-errc:
		return err
	}
}

func (b *Booster) startNodeTracer(c <-chan interface{}, bport int) error {
	for i := range c {
		m, ok := i.(tracer.Message)
		if !ok {
			return fmt.Errorf("booster: unable to recognise tracer message %v", m)
		}

		// means that the device is still offline.
		if m.Err != nil {
			continue
		}

		rn, err := b.GetNode(m.ID)
		if err != nil {
			b.Printf("booster: found unresolved id from tracer: %v", err)
			b.Untrace(m.ID)
			continue
		}

		// do not do anything if the node is already running.
		if rn.IsActive() {
			continue
		}

		laddr := net.JoinHostPort("localhost", strconv.Itoa(bport))
		raddr := rn.Addr().String()

		if _, err := b.Connect(context.Background(), "tcp", laddr, raddr); err != nil {
			// the node is up but we cannot open a proper Booster connection
			// to it.
			b.Print(err)
			continue
		}

		// do not trace this node anymore, as we managed to connect to it.
		b.Untrace(m.ID)
	}

	return nil
}

func (b *Booster) updateRootNode(c <-chan interface{}) error {
	for i := range c {
		wm, ok := i.(socks5.WorkloadMessage)
		if !ok {
			return fmt.Errorf("unable to recognise workload message: %v", wm)
		}

		b.Printf("booster: WM received: %+v", wm)
		target, err := net.ResolveTCPAddr("tcp", wm.Target)
		if err != nil {
			b.Printf("booster: unable to create addr: %v", err.Error())
			continue
		}

		switch wm.Event {
		case socks5.EventPush:
			if err := b.Ack(b.RootNode(), target); err != nil {
				b.Printf("booster: %v", err)
				continue
			}
		case socks5.EventPop:
			if err := b.RemoveTunnel(b.RootNode(), target); err != nil {
				b.Printf("booster: %v", err)
				continue
			}
		default:
			b.Printf("booster: unrecognised WM event: %v", wm.Event)
		}
	}

	return nil
}
