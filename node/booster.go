package node

import (
	"context"
	"encoding/hex"
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
	BoosterCMDHeartbit   = uint8(6)
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
	Sub(topic string) chan interface{}
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

	Proxy *Proxy

	NodeIdleTimeout time.Duration
}

// NewBooster returns a booster instance.
func NewBooster(proxy *Proxy, balancer *Balancer, log *log.Logger, ps PubSub, tr Tracer) *Booster {
	b := new(Booster)
	b.Proxy = proxy
	b.Balancer = balancer
	b.Logger = log
	b.Dialer = new(net.Dialer)
	b.PubSub = ps
	b.Tracer = tr

	b.NodeIdleTimeout = 60 * 5 * time.Second // time before closing an idle remote node

	return b
}

// NewBoosterDefault returns a new booster instance with initialized logger, balancer, pubsub, tracer and proxy.
func NewBoosterDefault() *Booster {
	log := log.New(os.Stdout, "BOOSTER ", log.LstdFlags)
	pubsub := pubsub.New()
	tracer := tracer.New(log, pubsub)
	balancer := NewBalancer(log, pubsub)
	proxy := NewProxyBalancer(balancer, tracer, pubsub)

	return NewBooster(proxy, balancer, log, pubsub, tracer)
}

// Start starts a booster node. It is made by a booster compliant tcp server
// and a socks5 compliant tcp server.
func (b *Booster) Start(pport, bport int) error {
	errc := make(chan error)
	tracerStream := b.Sub(tracer.TopicConnDiscovered)
	wlStream := b.Sub(socks5.TopicWorkload)

	// goroutine responsible for adding new nodes when the tracer tells to do so.
	go func() {
		defer func(){
			b.Unsub(tracerStream, tracer.TopicConnDiscovered)
		}()

		for i := range tracerStream {
			m, ok := i.(tracer.Message)
			if !ok {
				b.Printf("booster: unable to recognise tracer message %v", m)
				return
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
			if rn.IsActive {
				continue
			}

			laddr := net.JoinHostPort("localhost", strconv.Itoa(bport))
			raddr := rn.BAddr.String()

			if _, err := b.Connect(context.Background(), "tcp", laddr, raddr); err != nil {
				// the node is up but we cannot open a proper Booster connection
				// to it.
				b.Print(err)
				continue
			}

			// do not trace this node anymore, as we managed to connect to it.
			b.Untrace(m.ID)
		}
	}()

	// goroutine responsible for updating rootNode's information (i.e. local proxy usage and operations).
	go func() {
		defer func() {
			b.Unsub(wlStream, socks5.TopicWorkload)
		}()

		rootNode, err := NewNode("localhost", strconv.Itoa(pport), strconv.Itoa(bport))
		if err != nil {
			errc <- errors.New("booster: unable to create local node: " + err.Error())
			return
		}
		rootNode.IsActive = true
		b.SetRootNode(rootNode)

		for i := range wlStream {
			wm, ok := i.(socks5.WorkloadMessage)
			if !ok {
				b.Printf("proxy: unable to recognise workload message: %v", wm)
				return
			}
			b.UpdateNode(rootNode, wm.Load, wm.ID)
		}
	}()

	go func() {
		errc <- b.Proxy.ListenAndServe(pport)
	}()

	go func() {
		errc <- b.ListenAndServe(bport)
	}()

	defer func() {
		defer func() {
			// maybe one of the two channels was already closed.
			recover()
		}()

		close(tracerStream)
		close(wlStream)
	}()

	return <-errc
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

	// send status messages.
	go func() {
		defer func() {
			b.Proxy.Unsub(wc, socks5.TopicWorkload)
		}()

		buf := make([]byte, 0, 4+20)
		buf = append(buf, BoosterVersion1)
		buf = append(buf, BoosterCMDStatus)
		buf = append(buf, BoosterFieldReserved)
		for i := range wc {
			wm, ok := i.(socks5.WorkloadMessage)
			if !ok {
				ec <- errors.New("booster: unable to recognise workload message")
				return
			}

			idbuf, err := hex.DecodeString(wm.ID)
			if err != nil {
				ec <- errors.New("booster: unable to parse target: " + wm.ID + ": " + err.Error())
				return
			}
			if len(idbuf) != 20 {
				ec <- errors.New("booster: unexpected status target length: " + strconv.Itoa(len(idbuf)))
				return
			}

			buf = buf[:3]
			buf = append(buf, byte(wm.Load))
			buf = append(buf, idbuf...)

			if _, err := conn.Write(buf); err != nil {
				ec <- errors.New("booster: unable to write status: " + err.Error())
			}
		}
	}()

	// read heartbit messages.
	go func() {
		buf := make([]byte, 2)
		for {
			if _, err := io.ReadFull(conn, buf); err != nil {
				ec <- errors.New("booster: unable to read heartbit message: " + err.Error())
				return
			}

			_ = buf[0] // version
			_ = buf[1] // cmd
		}
	}()

	select {
	case <-ctx.Done():
		b.Proxy.Unsub(wc, socks5.TopicWorkload)
		return errors.New("booster: serve status: " + ctx.Err().Error())
	case err := <-ec:
		b.Proxy.Unsub(wc, socks5.TopicWorkload)
		return err
	}
}

// UpdateStatus expects conn to produce booster status messages and produces
// itself heartbit messages, which must be consumed by the connected node. It then
// uses the data received to update the workload's value of the node.
//
// If the connection is closed, the data is somehow corrupted or a cancel
// signal is received, it closes the connection and sets the IsActive value
// of the node to false.
//
// Publishes a TopicNodes message when a node is updated.
func (b *Booster) UpdateStatus(ctx context.Context, node *Node, conn net.Conn) error {
	if conn == nil {
		return errors.New("remote node: found nil connection. Unable to update node status")
	}
	ctx, cancel := context.WithCancel(ctx)
	node.Lock()
	node.IsActive = true
	node.cancel = cancel
	node.Unlock()

	go func() {
		fail := func(err error) {
			conn.Close() // be sure that the connection gets closed.

			// Do not fail multiple times.
			if !node.IsActive {
				return
			}

			b.CloseNode(node.ID())
			b.Trace(node)
			b.Printf("booster: update status: deactivating remote node %v: %v", node.ID(), err)
		}

		// this function will be fired after NodeIdleTimeout
		timer := time.AfterFunc(b.NodeIdleTimeout, func() {
			fail(errors.New("node flagged as idle"))
		})

		buf := make([]byte, 4+20)
		statusc := make(chan error)
		demuxc := make(chan error)

		// keep on reading status messages until the node is closed.
		go func() {
			for {
				// check if the node is active before trying to read from the connection
				if !node.IsActive {
					return
				}

				if _, err := io.ReadFull(conn, buf); err != nil {
					statusc <- err
					return
				}

				_ = buf[0]                           // version - already checked in the hello procedure
				_ = buf[1]                           // command
				_ = buf[2]                           // reserved field
				load := buf[3]                       // workload
				target := fmt.Sprintf("%x", buf[4:]) // target

				b.UpdateNode(node, int(load), target)
				statusc <- nil
			}
		}()

		// send heartbit messages to check if the connection is still alive. Fails if
		// it's not able to send messages after two seconds.
		go func() {
			buf := make([]byte, 0, 2)
			buf = append(buf, BoosterVersion1)
			buf = append(buf, BoosterCMDHeartbit)

			for {
				// Write fails after 2 seconds.
				conn.SetWriteDeadline(time.Now().Add(2*time.Second))
				if _, err := conn.Write(buf); err != nil {
					statusc <- errors.New("unable to send heartbit message: " + err.Error())
					return
				}

				<- time.After(2*time.Second) // Wait 2 seconds before sending another message.
			}
		}()

		// demux the messages from ctx and statusc into one channel
		go func() {
			for {
				select {
				case <-ctx.Done():
					demuxc <- ctx.Err()
					return
				case err := <-statusc:
					demuxc <- err
					if err != nil {
						return
					}
				}
			}
		}()

		// read demultiplexed messages and act accordingly.
		for err := range demuxc {
			if err != nil {
				fail(err)
				return
			}

			// Reset the timer if no errors occurred.
			timer.Reset(b.NodeIdleTimeout)
		}
	}()

	return nil
}
