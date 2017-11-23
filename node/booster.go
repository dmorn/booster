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

const (
	// Booster5IP4 is a version-4 IP address, with a length of 4 octets
	BoosterAddrIP4 = uint8(1)

	// Booster5FQDN field contains a fully-qualified domain name. The first
	// octet of the address field contains the number of octets of name that
	// follow, there is no terminating NUL octet.
	BoosterAddrFQDN = uint8(3)

	// AddrTypeIPV6 is a version-6 IP address, with a length of 16 octets.
	BoosterAddrIP6 = uint8(4)
)

type Booster struct {
	*log.Logger
	Proxy    *Proxy
	balancer LoadBalancer

	sync.Mutex
}

// Conn is a wrapper around io.ReadWriteCloser.
type Conn interface {
	io.ReadWriteCloser
	RemoteAddr() net.Addr
}

func NewBooster(proxy *Proxy, balancer LoadBalancer, log *log.Logger) *Booster {
	b := new(Booster)
	b.Proxy = proxy
	b.balancer = balancer
	b.Logger = log

	return b
}

func BOOSTER() *Booster {
	balancer := NewBalancer()
	proxy := PROXY(balancer)
	log := log.New(os.Stdout, "BOOSTER ", log.LstdFlags)

	return NewBooster(proxy, balancer, log)
}

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

	return b.Hello(ctx, "tcp", addr)
}

// Register performs the steps required to register a remote node.
// laddr is the local booster address to dial with. raddr is the remote
// node address that as to be registered.
func (b *Booster) Register(ctx context.Context, network, laddr, raddr string) error {
	_ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	conn, err := new(net.Dialer).DialContext(_ctx, network, laddr)
	if err != nil {
		return errors.New("booster: unable to contact booster " + laddr + " : " + err.Error())
	}

	host, portStr, err := net.SplitHostPort(raddr)
	if err != nil {
		return errors.New("booster: unrecognised address format : " + raddr + " : " + err.Error())
	}

	port, err := strconv.Atoi(portStr)
	if err != nil {
		return errors.New("booster: failed to parse port number: " + portStr)
	}
	if port < 1 || port > 0xffff {
		return errors.New("booster: port number out of range: " + portStr)
	}

	buf := make([]byte, 0, 5+len(host))
	buf = append(buf, BoosterVersion)
	buf = append(buf, BoosterCMDRegister)
	buf = append(buf, BoosterFieldReserved)

	if ip := net.ParseIP(host); ip != nil {
		if ip4 := ip.To4(); ip4 != nil {
			buf = append(buf, BoosterAddrIP4)
			ip = ip4
		} else {
			buf = append(buf, BoosterAddrIP6)
		}
		buf = append(buf, ip...)
	} else {
		if len(host) > 255 {
			return errors.New("booster: destination host name too long: " + host)
		}
		buf = append(buf, BoosterAddrFQDN)
		buf = append(buf, byte(len(host)))
		buf = append(buf, host...)
	}
	buf = append(buf, byte(port>>8), byte(port))

	if _, err := conn.Write(buf); err != nil {
		return errors.New("booster: unable to register: " + err.Error())
	}

	return nil
}

// Hello dials with the remote address, expecting it to be a booster server.
// Right after having enstablished the connection, it performs a "Hello" request.
// If the response is successfull, the remote booster is added to the list of
// helpers.
func (b *Booster) Hello(ctx context.Context, network, addr string) error {
	_ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	conn, err := new(net.Dialer).DialContext(_ctx, network, addr)
	if err != nil {
		return errors.New("booster: unable to contact remote instance: " + err.Error())
	}

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

	paddr := net.JoinHostPort(host, strconv.Itoa(port))
	b.balancer.AddProxy(paddr, conn)

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
